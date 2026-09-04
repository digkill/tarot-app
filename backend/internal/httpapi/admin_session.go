package httpapi

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/digkill/tarot-app/backend/internal/auth"
)

const (
	adminCookieName                 = "tarot_admin"
	adminLoginCSRFCookie            = "tarot_admin_lc"
	ctxAdminKey          contextKey = "admin"
)

type adminSession struct {
	UserID string
	Email  string
	CSRF   string
}

func (h *Handler) cookieSecure(r *http.Request) bool {
	return r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

func (h *Handler) setCookie(w http.ResponseWriter, r *http.Request, name, value string, ttl time.Duration, httpOnly bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/admin",
		MaxAge:   int(ttl.Seconds()),
		Expires:  time.Now().Add(ttl),
		HttpOnly: httpOnly,
		Secure:   h.cookieSecure(r),
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *Handler) clearCookie(w http.ResponseWriter, r *http.Request, name string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/admin",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Secure:   h.cookieSecure(r),
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *Handler) issueAdminSession(w http.ResponseWriter, r *http.Request, userID, email string) error {
	csrf, err := auth.RandomHex(16)
	if err != nil {
		return err
	}
	token, err := auth.GenerateAdminToken(userID, email, csrf, h.cfg.JWTSecret, h.cfg.AdminSessionTTL)
	if err != nil {
		return err
	}
	h.setCookie(w, r, adminCookieName, token, h.cfg.AdminSessionTTL, true)
	return nil
}

func (h *Handler) adminFromRequest(r *http.Request) *adminSession {
	c, err := r.Cookie(adminCookieName)
	if err != nil || c.Value == "" {
		return nil
	}
	claims, err := auth.ParseAdminToken(c.Value, h.cfg.JWTSecret)
	if err != nil {
		return nil
	}
	return &adminSession{UserID: claims.Subject, Email: claims.Email, CSRF: claims.CSRF}
}

func adminFromCtx(ctx context.Context) *adminSession {
	v, _ := ctx.Value(ctxAdminKey).(*adminSession)
	return v
}

func (h *Handler) adminMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess := h.adminFromRequest(r)
		if sess == nil {
			if r.Method == http.MethodGet {
				http.Redirect(w, r, "/admin/login?next="+url.QueryEscape(r.URL.RequestURI()), http.StatusFound)
				return
			}
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if r.Method == http.MethodPost {
			var err error
			if strings.HasSuffix(r.URL.Path, "/import") {
				err = r.ParseMultipartForm(512 << 20)
			} else {
				err = r.ParseForm()
			}
			if err != nil {
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}
			if r.FormValue("csrf") != sess.CSRF {
				http.Error(w, "csrf", http.StatusForbidden)
				return
			}
		}
		ctx := context.WithValue(r.Context(), ctxAdminKey, sess)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func safeAdminNext(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "/admin"
	}
	u, err := url.Parse(raw)
	if err != nil || u.IsAbs() || u.Host != "" {
		return "/admin"
	}
	if !strings.HasPrefix(u.Path, "/admin") || strings.HasPrefix(u.Path, "/admin/login") {
		return "/admin"
	}
	if u.RawQuery != "" {
		return u.Path + "?" + u.RawQuery
	}
	return u.Path
}

type loginBucket struct {
	n     int
	until time.Time
}

var (
	loginMu   sync.Mutex
	loginHits = map[string]*loginBucket{}
)

func allowAdminLogin(ip string) bool {
	now := time.Now()
	loginMu.Lock()
	defer loginMu.Unlock()
	b, ok := loginHits[ip]
	if !ok || now.After(b.until) {
		loginHits[ip] = &loginBucket{n: 1, until: now.Add(15 * time.Minute)}
		return true
	}
	if b.n >= 12 {
		return false
	}
	b.n++
	return true
}

package httpapi

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/digkill/tarot-app/backend/internal/auth"
	"github.com/digkill/tarot-app/backend/internal/billing"
	"github.com/digkill/tarot-app/backend/internal/storage"
)

func (h *Handler) AdminLoginPage(w http.ResponseWriter, r *http.Request) {
	if h.adminFromRequest(r) != nil {
		http.Redirect(w, r, safeAdminNext(r.URL.Query().Get("next")), http.StatusFound)
		return
	}
	csrf, err := auth.RandomHex(16)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	h.setCookie(w, r, adminLoginCSRFCookie, csrf, 30*time.Minute, true)
	flash, errMsg := flashFromQuery(r)
	h.renderAdmin(w, "login", http.StatusOK, map[string]any{
		"CSRF":  csrf,
		"Next":  r.URL.Query().Get("next"),
		"Error": errMsg,
		"Flash": flash,
	})
}

func (h *Handler) AdminLogin(w http.ResponseWriter, r *http.Request) {
	if !allowAdminLogin(clientIP(r)) {
		h.renderAdmin(w, "login", http.StatusTooManyRequests, map[string]any{
			"Error": "Слишком много попыток. Подождите 15 минут.",
			"Next":  r.FormValue("next"),
		})
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	lc, _ := r.Cookie(adminLoginCSRFCookie)
	if lc == nil || lc.Value == "" || r.FormValue("csrf") != lc.Value {
		h.renderAdmin(w, "login", http.StatusForbidden, map[string]any{
			"Error": "Сессия формы устарела, обновите страницу.",
			"Next":  r.FormValue("next"),
		})
		return
	}

	email := strings.ToLower(strings.TrimSpace(r.FormValue("email")))
	password := r.FormValue("password")
	user, err := h.users.GetByEmail(r.Context(), email)
	ok := err == nil && user.IsAdmin() && auth.CheckPassword(password, user.PasswordHash)
	if !ok {
		if err != nil {
			auth.CheckPasswordDummy(password)
		}
		csrf, _ := auth.RandomHex(16)
		if csrf != "" {
			h.setCookie(w, r, adminLoginCSRFCookie, csrf, 30*time.Minute, true)
		}
		h.renderAdmin(w, "login", http.StatusUnauthorized, map[string]any{
			"CSRF":  csrf,
			"Error": "Неверный email или пароль.",
			"Next":  r.FormValue("next"),
		})
		return
	}

	if err = h.issueAdminSession(w, r, user.ID, user.Email); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	h.clearCookie(w, r, adminLoginCSRFCookie)
	http.Redirect(w, r, safeAdminNext(r.FormValue("next")), http.StatusFound)
}

func (h *Handler) AdminLogout(w http.ResponseWriter, r *http.Request) {
	h.clearCookie(w, r, adminCookieName)
	http.Redirect(w, r, "/admin/login?ok=logged_out", http.StatusFound)
}

func (h *Handler) adminBase(r *http.Request, title, nav string) adminBase {
	sess := adminFromCtx(r.Context())
	ok, errMsg := flashFromQuery(r)
	email := ""
	csrf := ""
	if sess != nil {
		email = sess.Email
		csrf = sess.CSRF
	}
	return adminBase{Title: title, AdminEmail: email, CSRF: csrf, Flash: ok, Error: errMsg, Nav: nav}
}

func (h *Handler) AdminDashboard(w http.ResponseWriter, r *http.Request) {
	stats, err := h.stats.Dashboard(r.Context())
	if err != nil {
		slog.Error("admin dashboard stats", "error", err)
		http.Error(w, "failed to load stats", http.StatusInternalServerError)
		return
	}
	tx, _, err := h.txns.List(r.Context(), storage.TxListFilter{Limit: 8})
	if err != nil {
		slog.Error("admin dashboard tx", "error", err)
		http.Error(w, "failed to load transactions", http.StatusInternalServerError)
		return
	}
	audit, err := h.audit.Recent(r.Context(), 15)
	if err != nil {
		slog.Error("admin dashboard audit", "error", err)
		http.Error(w, "failed to load audit", http.StatusInternalServerError)
		return
	}
	base := h.adminBase(r, "Сводка", "dash")
	h.renderAdmin(w, "dashboard", http.StatusOK, struct {
		adminBase
		Stats    *storage.DashboardStats
		RecentTx []storage.Transaction
		Audit    []storage.AuditEntry
	}{base, stats, tx, audit})
}

func (h *Handler) AdminUsers(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	page := parsePage(r)
	const limit = 50
	users, total, err := h.users.ListForAdmin(r.Context(), q, limit, (page-1)*limit)
	if err != nil {
		slog.Error("admin list users", "error", err)
		http.Error(w, "failed to list users", http.StatusInternalServerError)
		return
	}
	base := h.adminBase(r, "Пользователи", "users")
	pg := makePager(page, limit, total, "/admin/users", map[string]string{"q": q})
	h.renderAdmin(w, "users", http.StatusOK, struct {
		adminBase
		pagerView
		Users []storage.AdminUser
		Query string
	}{base, pg, users, q})
}

func (h *Handler) AdminUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	user, err := h.users.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "failed to load user", http.StatusInternalServerError)
		return
	}
	readings, err := h.stats.CountReadingsForUser(r.Context(), user.ID)
	if err != nil {
		http.Error(w, "failed to load readings", http.StatusInternalServerError)
		return
	}
	tx, _, err := h.txns.List(r.Context(), storage.TxListFilter{UserID: user.ID, Limit: 50})
	if err != nil {
		http.Error(w, "failed to load transactions", http.StatusInternalServerError)
		return
	}
	shopDecks, _ := h.decks.ListAll(r.Context())
	// Apple-side state answers the most common support question: "why did my
	// premium disappear".
	appleSubs, err := h.appleSubs.ListByUser(r.Context(), user.ID)
	if err != nil {
		slog.Error("load apple subscriptions for admin", "error", err, "user", user.ID)
	}
	base := h.adminBase(r, user.Email, "users")
	h.renderAdmin(w, "user", http.StatusOK, struct {
		adminBase
		User      *storage.User
		Readings  int
		Tx        []storage.Transaction
		Products  []billing.Product
		Decks     []storage.Deck
		AppleSubs []storage.AppleSubscription
	}{base, user, readings, tx, billing.Products, shopDecks, appleSubs})
}

func (h *Handler) AdminSetPremium(w http.ResponseWriter, r *http.Request) {
	sess := adminFromCtx(r.Context())
	id := chi.URLParam(r, "id")
	on := r.FormValue("premium") == "1"
	if err := h.users.SetPremium(r.Context(), id, on); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			http.Redirect(w, r, "/admin/users?err=not_found", http.StatusFound)
			return
		}
		http.Error(w, "failed to update premium", http.StatusInternalServerError)
		return
	}
	action := "premium_off"
	ok := "premium_off"
	if on {
		action = "premium_on"
		ok = "premium_on"
	}
	_ = h.audit.Insert(r.Context(), sess.UserID, sess.Email, action, "user", id, "")
	http.Redirect(w, r, "/admin/users/"+id+"?ok="+ok, http.StatusFound)
}

func (h *Handler) AdminTransactions(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	page := parsePage(r)
	const limit = 50
	tx, total, err := h.txns.List(r.Context(), storage.TxListFilter{
		Query: q, Status: status, Limit: limit, Offset: (page - 1) * limit,
	})
	if err != nil {
		slog.Error("admin list tx", "error", err)
		http.Error(w, "failed to list transactions", http.StatusInternalServerError)
		return
	}
	shopDecks, _ := h.decks.ListAll(r.Context())
	base := h.adminBase(r, "Транзакции", "tx")
	pg := makePager(page, limit, total, "/admin/transactions", map[string]string{"q": q, "status": status})
	h.renderAdmin(w, "transactions", http.StatusOK, struct {
		adminBase
		pagerView
		Tx       []storage.Transaction
		Query    string
		Status   string
		Products []billing.Product
		Decks    []storage.Deck
	}{base, pg, tx, q, status, billing.Products, shopDecks})
}

func (h *Handler) AdminTransaction(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	tx, err := h.txns.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "failed to load transaction", http.StatusInternalServerError)
		return
	}
	base := h.adminBase(r, "Транзакция", "tx")
	h.renderAdmin(w, "transaction", http.StatusOK, struct {
		adminBase
		Tx *storage.Transaction
	}{base, tx})
}

func (h *Handler) AdminCreateTransaction(w http.ResponseWriter, r *http.Request) {
	sess := adminFromCtx(r.Context())
	email := strings.ToLower(strings.TrimSpace(r.FormValue("email")))
	productID := strings.TrimSpace(r.FormValue("product_id"))
	provider := billing.NormalizeProvider(r.FormValue("provider"))
	redirect := r.FormValue("redirect")
	if !strings.HasPrefix(redirect, "/admin/") {
		redirect = "/admin/transactions"
	}

	product, isPremium := billing.Lookup(productID)
	var deck *storage.Deck
	if !isPremium {
		var derr error
		deck, derr = h.decks.GetByProductID(r.Context(), productID)
		if derr != nil {
			http.Redirect(w, r, redirect+"?err=validation", http.StatusFound)
			return
		}
	}
	if (!isPremium && deck == nil) || !isValidEmail(email) {
		http.Redirect(w, r, redirect+"?err=validation", http.StatusFound)
		return
	}
	user, err := h.users.GetByEmail(r.Context(), email)
	if err != nil {
		http.Redirect(w, r, redirect+"?err=not_found", http.StatusFound)
		return
	}

	amount := 0
	kind := billing.KindOneTime
	storeProductID := productID
	if isPremium {
		amount = product.AmountKop
		kind = product.Kind
		storeProductID = product.ID
	} else {
		amount = deck.PriceKop
		storeProductID = deck.ProductID()
	}
	if raw := strings.TrimSpace(r.FormValue("amount_rub")); raw != "" {
		raw = strings.ReplaceAll(raw, ",", ".")
		rub, perr := strconv.ParseFloat(raw, 64)
		if perr != nil || rub < 0 {
			http.Redirect(w, r, redirect+"?err=validation", http.StatusFound)
			return
		}
		amount = int(rub * 100)
	}

	if provider == billing.ProviderPromo {
		kind = billing.KindPromo
	}

	tx, err := h.txns.Create(r.Context(), storage.CreateTransactionParams{
		UserID:            user.ID,
		ProductID:         storeProductID,
		Kind:              kind,
		Status:            billing.StatusPaid,
		Provider:          provider,
		ProviderInvoiceID: storage.NullIfBlank(r.FormValue("invoice_id")),
		AmountKop:         amount,
		Currency:          "RUB",
		Notes:             strings.TrimSpace(r.FormValue("notes")),
	})
	if err != nil {
		if errors.Is(err, storage.ErrConflict) {
			http.Redirect(w, r, redirect+"?err=conflict", http.StatusFound)
			return
		}
		slog.Error("admin create tx", "error", err)
		http.Error(w, "failed to create transaction", http.StatusInternalServerError)
		return
	}
	if isPremium {
		if err = h.users.GrantPremium(r.Context(), user.ID, storage.PremiumGrant{
			ProductID: product.ID,
			Source:    provider,
			ExpiresAt: billing.ExpiresAt(product, time.Now()),
		}); err != nil {
			slog.Error("admin grant premium after tx", "error", err)
		}
	} else if deck != nil {
		tid := tx.ID
		if err = h.decks.Grant(r.Context(), user.ID, deck.ID, provider, &tid); err != nil {
			slog.Error("admin grant deck after tx", "error", err)
		}
	}
	_ = h.audit.Insert(r.Context(), sess.UserID, sess.Email, "tx_create", "transaction", tx.ID, user.Email+" "+storeProductID)
	http.Redirect(w, r, redirect+"?ok=created", http.StatusFound)
}

func (h *Handler) AdminRefund(w http.ResponseWriter, r *http.Request) {
	sess := adminFromCtx(r.Context())
	id := chi.URLParam(r, "id")
	reason := strings.TrimSpace(r.FormValue("reason"))
	if reason == "" {
		http.Redirect(w, r, "/admin/transactions/"+id+"?err=validation", http.StatusFound)
		return
	}
	tx, revoked, err := h.txns.Refund(r.Context(), id, sess.Email, reason)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			http.Redirect(w, r, "/admin/transactions?err=not_found", http.StatusFound)
			return
		}
		if errors.Is(err, storage.ErrNotRefundable) {
			http.Redirect(w, r, "/admin/transactions/"+id+"?err=not_refundable", http.StatusFound)
			return
		}
		slog.Error("admin refund", "error", err)
		http.Error(w, "failed to refund", http.StatusInternalServerError)
		return
	}
	details := tx.UserEmail
	if revoked {
		details += " · premium revoked"
	}
	_ = h.audit.Insert(r.Context(), sess.UserID, sess.Email, "tx_refund", "transaction", id, details)
	if deck, derr := h.decks.GetByProductID(r.Context(), tx.ProductID); derr == nil {
		_ = h.decks.Revoke(r.Context(), tx.UserID, deck.ID)
	}
	http.Redirect(w, r, "/admin/transactions/"+id+"?ok=refunded", http.StatusFound)
}

package httpapi

import (
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const maxRateKeys = 20_000

type rateWindow struct {
	times []time.Time
}

type rateLimiter struct {
	mu      sync.Mutex
	windows map[string]*rateWindow
}

func newRateLimiter() *rateLimiter {
	return &rateLimiter{windows: make(map[string]*rateWindow)}
}

func (l *rateLimiter) allow(key string, limit int, window time.Duration) bool {
	if limit <= 0 {
		return false
	}
	now := time.Now()
	cutoff := now.Add(-window)

	l.mu.Lock()
	defer l.mu.Unlock()

	w := l.windows[key]
	if w == nil {
		if len(l.windows) >= maxRateKeys {
			l.evictLocked(cutoff)
			if len(l.windows) >= maxRateKeys {
				return false
			}
		}
		w = &rateWindow{}
		l.windows[key] = w
	}

	kept := w.times[:0]
	for _, ts := range w.times {
		if ts.After(cutoff) {
			kept = append(kept, ts)
		}
	}
	w.times = kept
	if len(w.times) >= limit {
		return false
	}
	w.times = append(w.times, now)
	return true
}

func (l *rateLimiter) evictLocked(cutoff time.Time) {
	for key, w := range l.windows {
		alive := false
		for _, ts := range w.times {
			if ts.After(cutoff) {
				alive = true
				break
			}
		}
		if !alive {
			delete(l.windows, key)
		}
	}
}

func writeRateLimit(w http.ResponseWriter, retryAfter time.Duration, message string) {
	secs := int(retryAfter.Seconds())
	if secs < 1 {
		secs = 1
	}
	w.Header().Set("Retry-After", strconv.Itoa(secs))
	writeError(w, http.StatusTooManyRequests, "rate_limited", message)
}

func (h *Handler) ipRateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)
		path := r.URL.Path

		if !h.limiter.allow("ip:"+ip, 90, time.Minute) {
			writeRateLimit(w, time.Minute, "too many requests")
			return
		}

		if isAuthSensitive(path) && !h.limiter.allow("auth:"+ip, 8, 15*time.Minute) {
			writeRateLimit(w, 15*time.Minute, "too many authentication attempts")
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (h *Handler) userRateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := userIDFromCtx(r.Context())
		if userID == "" {
			next.ServeHTTP(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/interpretations") {
			if !h.limiter.allow("interp:"+userID, 6, time.Minute) {
				writeRateLimit(w, time.Minute, "too many interpretation requests")
				return
			}
		} else if !h.limiter.allow("user:"+userID, 120, time.Minute) {
			writeRateLimit(w, time.Minute, "too many requests")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func isAuthSensitive(path string) bool {
	switch {
	case strings.HasSuffix(path, "/auth/login"),
		strings.HasSuffix(path, "/auth/register"),
		strings.HasSuffix(path, "/auth/forgot-password"),
		strings.HasSuffix(path, "/auth/reset-password"),
		strings.HasSuffix(path, "/auth/verify-email"),
		strings.HasSuffix(path, "/auth/resend-verification"):
		return true
	default:
		return false
	}
}

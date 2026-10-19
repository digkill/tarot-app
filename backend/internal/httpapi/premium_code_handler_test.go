package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPremiumCodeRequiresAccount(t *testing.T) {
	h := &Handler{limiter: newRateLimiter()}
	w := httptest.NewRecorder()
	h.RedeemPremiumCode(w, postJSON("/api/v1/billing/premium-code", `{"code":"TARO73"}`))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d", w.Code)
	}
}

func TestPremiumCodeRejectsInvalidAndLimitsAttemptsBeforeDB(t *testing.T) {
	h := &Handler{limiter: newRateLimiter()}
	for i := 0; i < 6; i++ {
		r := postJSON("/api/v1/billing/premium-code", `{"code":"bad"}`)
		r = r.WithContext(context.WithValue(r.Context(), ctxUserID, "user-test"))
		w := httptest.NewRecorder()
		h.RedeemPremiumCode(w, r)
		want := http.StatusUnprocessableEntity
		if i == 5 {
			want = http.StatusTooManyRequests
		}
		if w.Code != want {
			t.Fatalf("attempt %d: %d want %d", i, w.Code, want)
		}
	}
}

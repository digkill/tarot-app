package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/digkill/tarot-app/backend/internal/appstore"
	"github.com/digkill/tarot-app/backend/internal/appstore/appstoretest"
	"github.com/digkill/tarot-app/backend/internal/billing"
	"github.com/digkill/tarot-app/backend/internal/config"
)

// Every handler below is built with nil repositories on purpose: if a request
// reaches the database before its signature has been checked, the test panics
// instead of quietly passing. Same trick as cloudpayments_handler_test.go.
func appleHandler(t *testing.T, chain *appstoretest.Chain) *Handler {
	t.Helper()
	c := appstore.New(appstore.Config{
		BundleID:     appstoretest.DefaultBundleID,
		AllowSandbox: true,
		RootCA:       chain.Root(),
	})
	if c == nil {
		t.Fatal("appstore.New returned nil")
	}
	return &Handler{cfg: &config.Config{}, as: c}
}

func postJSON(path, body string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func TestAppleNotificationsUnconfigured(t *testing.T) {
	h := &Handler{cfg: &config.Config{}}
	rec := httptest.NewRecorder()

	h.AppleNotifications(rec, postJSON("/api/v1/billing/apple/notifications", `{"signedPayload":"x"}`))

	// A 503 makes Apple retry, which is what we want for a misconfiguration.
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status %d, want 503", rec.Code)
	}
}

func TestAppleVerifyUnconfigured(t *testing.T) {
	h := &Handler{cfg: &config.Config{}}
	rec := httptest.NewRecorder()

	h.AppleVerifyPurchase(rec, postJSON("/api/v1/billing/apple/verify", `{"originalTransactionId":"2000000812345678"}`))

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status %d, want 503", rec.Code)
	}
}

// A forged notification must be rejected before any lookup — the handler has
// nil repositories, so reaching one would panic.
func TestAppleNotificationsRejectsForgedSignature(t *testing.T) {
	chain := appstoretest.New(t)
	attacker := appstoretest.New(t)
	h := appleHandler(t, chain)

	body, _ := json.Marshal(map[string]string{
		"signedPayload": attacker.Sign(t, appstoretest.NotificationClaims(
			attacker.Sign(t, appstoretest.TxClaims()), "")),
	})
	rec := httptest.NewRecorder()

	h.AppleNotifications(rec, postJSON("/api/v1/billing/apple/notifications", string(body)))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status %d, want 401; body %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "invalid_signature") {
		t.Fatalf("body %s", rec.Body.String())
	}
}

func TestAppleNotificationsRejectsGarbageBody(t *testing.T) {
	chain := appstoretest.New(t)
	h := appleHandler(t, chain)

	for name, body := range map[string]string{
		"not json":            `}{`,
		"missing payload":     `{}`,
		"blank payload":       `{"signedPayload":"   "}`,
		"unsigned payload":    `{"signedPayload":"not.a.jws"}`,
		"wrong type entirely": `{"signedPayload":123}`,
	} {
		t.Run(name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h.AppleNotifications(rec, postJSON("/api/v1/billing/apple/notifications", body))
			if rec.Code != http.StatusBadRequest && rec.Code != http.StatusUnauthorized {
				t.Fatalf("status %d, want 400 or 401; body %s", rec.Code, rec.Body.String())
			}
		})
	}
}

// A validly signed TEST notification must ack without touching the database,
// which is what App Store Connect's "request a test notification" exercises.
func TestAppleNotificationsAcksTestNotification(t *testing.T) {
	chain := appstoretest.New(t)
	h := appleHandler(t, chain)

	body, _ := json.Marshal(map[string]string{
		"signedPayload": chain.Sign(t, appstoretest.TestNotificationClaims()),
	})
	rec := httptest.NewRecorder()

	h.AppleNotifications(rec, postJSON("/api/v1/billing/apple/notifications", string(body)))

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, want 200; body %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"ok":true`) {
		t.Fatalf("body %s", rec.Body.String())
	}
}

// A notification for another app must not be acted on, even though its
// signature chains to our root.
func TestAppleNotificationsRejectsAnotherApp(t *testing.T) {
	chain := appstoretest.New(t)
	h := appleHandler(t, chain)

	body, _ := json.Marshal(map[string]string{
		"signedPayload": chain.Sign(t, appstoretest.NotificationClaims("", "", func(m jwt.MapClaims) {
			m["data"].(map[string]any)["bundleId"] = "com.evil.app"
		})),
	})
	rec := httptest.NewRecorder()

	h.AppleNotifications(rec, postJSON("/api/v1/billing/apple/notifications", string(body)))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status %d, want 401; body %s", rec.Code, rec.Body.String())
	}
}

func TestAppleVerifyRequiresAnIdentifier(t *testing.T) {
	chain := appstoretest.New(t)
	h := appleHandler(t, chain)

	for name, body := range map[string]string{
		"empty object": `{}`,
		"blank fields": `{"signedTransaction":"  ","originalTransactionId":" "}`,
		"bad json":     `not json`,
	} {
		t.Run(name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h.AppleVerifyPurchase(rec, postJSON("/api/v1/billing/apple/verify", body))
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status %d, want 400; body %s", rec.Code, rec.Body.String())
			}
		})
	}
}

// A forged transaction must fail before the user is loaded.
func TestAppleVerifyRejectsForgedTransaction(t *testing.T) {
	chain := appstoretest.New(t)
	attacker := appstoretest.New(t)
	h := appleHandler(t, chain)

	body, _ := json.Marshal(map[string]string{
		"signedTransaction": attacker.Sign(t, appstoretest.TxClaims()),
	})
	rec := httptest.NewRecorder()

	// No user id in the context and nil repos: anything past signature
	// verification would panic.
	h.AppleVerifyPurchase(rec, postJSON("/api/v1/billing/apple/verify", string(body)))

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status %d, want 422; body %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "invalid_signature") {
		t.Fatalf("body %s", rec.Body.String())
	}
}

// The web checkout being reachable for Apple is precisely what got the app
// rejected under guideline 3.1.1, so keep it refused.
func TestCreateCheckoutRejectsAppleProvider(t *testing.T) {
	h := &Handler{cfg: &config.Config{}}
	for _, provider := range []string{"apple", "Apple", "ios"} {
		rec := httptest.NewRecorder()
		h.CreateCheckout(rec, postJSON("/api/v1/billing/checkout",
			`{"productId":"premium_monthly","provider":"`+provider+`"}`))
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("provider %q: status %d, want 422; body %s", provider, rec.Code, rec.Body.String())
		}
	}
}

func TestAppleAmountPrefersApplePriceThenCatalog(t *testing.T) {
	yearly, _ := billing.Lookup(billing.ProductYearly)

	withPrice := &appstore.Transaction{Price: 59990, Currency: "usd"}
	if minor, currency := appleAmount(withPrice, yearly, nil); minor != 5999 || currency != "USD" {
		t.Fatalf("apple price: %d %s", minor, currency)
	}

	// price and currency are absent from older payloads, so the fallback is
	// not optional.
	noPrice := &appstore.Transaction{}
	if minor, currency := appleAmount(noPrice, yearly, nil); minor != yearly.AmountUSDCents || currency != billing.CurrencyUSD {
		t.Fatalf("catalog fallback: %d %s", minor, currency)
	}
}

func TestAppleNotesStayWithinTheColumn(t *testing.T) {
	chain := appstoretest.New(t)
	c := appstore.New(appstore.Config{
		BundleID: appstoretest.DefaultBundleID, AllowSandbox: true, RootCA: chain.Root(),
	})
	tx, err := c.VerifyTransaction(chain.Sign(t, appstoretest.TxClaims(func(m jwt.MapClaims) {
		m["environment"] = appstore.EnvSandbox
	})))
	if err != nil {
		t.Fatal(err)
	}
	notes := appleNotes(tx, &appstore.Notification{
		NotificationType: "DID_RENEW",
		Subtype:          "BILLING_RECOVERY",
		NotificationUUID: "9ad8c4b1-0000-4000-8000-000000000002",
	})
	for _, want := range []string{"apple", "DID_RENEW", "BILLING_RECOVERY", "env=Sandbox", "orig=2000000812345678", "wo=100000123", "notif="} {
		if !strings.Contains(notes, want) {
			t.Fatalf("notes %q missing %q", notes, want)
		}
	}
	if len([]rune(truncateRunes(notes, 200))) > 200 {
		t.Fatal("notes must be truncated to fit the column")
	}
}

func TestAppleInactiveReason(t *testing.T) {
	active := appstore.SubStatusActive
	expired := appstore.SubStatusExpired
	revoked := appstore.SubStatusRevoked
	retry := appstore.SubStatusBillingRetry
	grace := appstore.SubStatusGracePeriod

	future := &appstore.Transaction{ExpiresDate: nowMs() + 86_400_000}
	past := &appstore.Transaction{ExpiresDate: nowMs() - 86_400_000}
	gone := &appstore.Transaction{RevocationDate: nowMs() - 1000}

	cases := []struct {
		name   string
		tx     *appstore.Transaction
		status *int
		want   string
	}{
		{"active subscription", future, &active, ""},
		{"grace period still counts as active", past, &grace, ""},
		{"expired per Apple", future, &expired, "expired"},
		{"revoked per Apple", future, &revoked, "revoked"},
		{"billing retry", future, &retry, "billing_retry"},
		{"revocation date wins over status", gone, &active, "revoked"},
		{"no status, expiry in the past", past, nil, "expired"},
		{"no status, expiry in the future", future, nil, ""},
		{"non-consumable has no expiry", &appstore.Transaction{}, nil, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := appleInactiveReason(tc.tx, tc.status); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func nowMs() int64 { return time.Now().UnixMilli() }

package appstore_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/digkill/tarot-app/backend/internal/appstore"
	"github.com/digkill/tarot-app/backend/internal/appstore/appstoretest"
)

func newKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return key
}

func TestEmbeddedRootIsAppleRootCAG3(t *testing.T) {
	root, err := appstore.EmbeddedRoot()
	if err != nil {
		t.Fatalf("embedded root: %v", err)
	}
	if root.Subject.CommonName != "Apple Root CA - G3" {
		t.Fatalf("CN = %q", root.Subject.CommonName)
	}
	if !root.IsCA {
		t.Fatal("embedded root is not a CA")
	}
	if !root.NotAfter.After(time.Now()) {
		t.Fatalf("embedded root expired at %v", root.NotAfter)
	}
	if root.Subject.CommonName != root.Issuer.CommonName {
		t.Fatal("embedded root is not self-signed")
	}
}

func TestEnabledMatrix(t *testing.T) {
	var nilClient *appstore.Client
	if nilClient.Enabled() || nilClient.ServerAPIEnabled() {
		t.Fatal("a nil client must be disabled")
	}

	bundleOnly := appstore.New(appstore.Config{BundleID: appstoretest.DefaultBundleID})
	if !bundleOnly.Enabled() {
		t.Fatal("a bundle id alone enables signature verification and notifications")
	}
	if bundleOnly.ServerAPIEnabled() {
		t.Fatal("the Server API needs the In-App Purchase key")
	}

	key := newKey(t)
	full := appstore.New(appstore.Config{
		BundleID: appstoretest.DefaultBundleID, IssuerID: "issuer", KeyID: "KEYID", PrivateKey: key,
	})
	if !full.Enabled() || !full.ServerAPIEnabled() {
		t.Fatal("a fully configured client must enable both")
	}

	// Each credential is individually load-bearing for the Server API.
	for _, cfg := range []appstore.Config{
		{BundleID: "b", KeyID: "k", PrivateKey: key},
		{BundleID: "b", IssuerID: "i", PrivateKey: key},
		{BundleID: "b", IssuerID: "i", KeyID: "k"},
	} {
		if appstore.New(cfg).ServerAPIEnabled() {
			t.Fatalf("partial credentials must not enable the Server API: %+v", cfg)
		}
	}
}

// The developer token is checked as Apple sees it: captured off the wire.
func TestDeveloperTokenClaims(t *testing.T) {
	chain := appstoretest.New(t)
	key := newKey(t)
	// Real time, truncated: the synthetic certificates are valid from a day
	// before the real now, so a fixed date here stops verifying a day later.
	now := time.Now().UTC().Truncate(time.Second)

	var seen string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		_, _ = w.Write([]byte(`{"signedTransactionInfo":"` + chain.Sign(t, appstoretest.TxClaims()) + `"}`))
	}))
	defer srv.Close()

	c := clientFor(t, chain, func(cfg *appstore.Config) {
		cfg.IssuerID, cfg.KeyID, cfg.PrivateKey = "57246542-96fe-1a63-e053-0824d011072a", "2X9R4HXF34", key
		cfg.BaseProd, cfg.BaseSandbox = srv.URL, srv.URL
		cfg.Now = func() time.Time { return now }
	})
	if _, err := c.GetTransactionInfo(context.Background(), "2000000812345679", appstore.EnvProduction); err != nil {
		t.Fatalf("transaction info: %v", err)
	}
	if seen == "" {
		t.Fatal("no bearer token was sent")
	}

	parsed, err := jwt.Parse(seen, func(*jwt.Token) (any, error) { return &key.PublicKey, nil },
		jwt.WithValidMethods([]string{"ES256"}),
		jwt.WithTimeFunc(func() time.Time { return now }))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if kid, _ := parsed.Header["kid"].(string); kid != "2X9R4HXF34" {
		t.Fatalf("kid = %v", parsed.Header["kid"])
	}
	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		t.Fatalf("claims type %T", parsed.Claims)
	}
	if claims["aud"] != "appstoreconnect-v1" {
		t.Fatalf("aud = %v", claims["aud"])
	}
	if claims["iss"] != "57246542-96fe-1a63-e053-0824d011072a" {
		t.Fatalf("iss = %v", claims["iss"])
	}
	if claims["bid"] != appstoretest.DefaultBundleID {
		t.Fatalf("bid = %v", claims["bid"])
	}
	iat, _ := claims["iat"].(float64)
	exp, _ := claims["exp"].(float64)
	if lifetime := time.Duration(exp-iat) * time.Second; lifetime <= 0 || lifetime > time.Hour {
		t.Fatalf("token lifetime %v, Apple's maximum is 1h", lifetime)
	}
}

func TestServerAPIUnconfigured(t *testing.T) {
	c := appstore.New(appstore.Config{BundleID: appstoretest.DefaultBundleID})
	if _, err := c.GetSubscriptionStatuses(context.Background(), "2000000812345678", ""); !errors.Is(err, appstore.ErrNotConfigured) {
		t.Fatalf("err = %v, want ErrNotConfigured", err)
	}
	if _, err := c.GetTransactionInfo(context.Background(), "2000000812345679", ""); !errors.Is(err, appstore.ErrNotConfigured) {
		t.Fatalf("err = %v, want ErrNotConfigured", err)
	}
}

func TestVerifyNotificationWithNestedJWS(t *testing.T) {
	chain := appstoretest.New(t)
	c := clientFor(t, chain)

	txJWS := chain.Sign(t, appstoretest.TxClaims())
	renewalJWS := chain.Sign(t, jwt.MapClaims{
		"originalTransactionId": "2000000812345678",
		"productId":             "org.mediarise.tarot.premium.yearly",
		"autoRenewProductId":    "org.mediarise.tarot.premium.yearly",
		"autoRenewStatus":       float64(1),
		"environment":           appstore.EnvProduction,
		"signedDate":            float64(time.Now().UnixMilli()),
	})

	n, err := c.VerifyNotification(chain.Sign(t, appstoretest.NotificationClaims(txJWS, renewalJWS)))
	if err != nil {
		t.Fatalf("verify notification: %v", err)
	}
	if n.NotificationType != "DID_RENEW" || n.NotificationUUID == "" {
		t.Fatalf("notification: %+v", n)
	}
	if n.Transaction == nil || n.Transaction.TransactionID != "2000000812345679" {
		t.Fatalf("nested transaction not decoded: %+v", n.Transaction)
	}
	if n.Renewal == nil || n.Renewal.AutoRenewStatus != 1 {
		t.Fatalf("nested renewal not decoded: %+v", n.Renewal)
	}
	if got := n.OriginalTransactionID(); got != "2000000812345678" {
		t.Fatalf("OriginalTransactionID = %q", got)
	}
}

// A valid outer signature must not rescue a forged inner payload.
func TestVerifyNotificationRejectsBadNestedSignature(t *testing.T) {
	chain := appstoretest.New(t)
	attacker := appstoretest.New(t)
	c := clientFor(t, chain)

	payload := chain.Sign(t, appstoretest.NotificationClaims(attacker.Sign(t, appstoretest.TxClaims()), ""))
	if _, err := c.VerifyNotification(payload); err == nil {
		t.Fatal("a nested payload signed by another chain must fail the notification")
	}
}

func TestVerifyNotificationRejectsWrongBundle(t *testing.T) {
	chain := appstoretest.New(t)
	c := clientFor(t, chain)

	payload := chain.Sign(t, appstoretest.NotificationClaims("", "", func(m jwt.MapClaims) {
		m["data"].(map[string]any)["bundleId"] = "com.evil.app"
	}))
	if _, err := c.VerifyNotification(payload); !errors.Is(err, appstore.ErrBundleMismatch) {
		t.Fatalf("err = %v, want ErrBundleMismatch", err)
	}
}

func TestVerifyNotificationRejectsWrongAppAppleID(t *testing.T) {
	chain := appstoretest.New(t)
	c := clientFor(t, chain, func(cfg *appstore.Config) { cfg.AppAppleID = 111 })

	payload := chain.Sign(t, appstoretest.NotificationClaims("", "", func(m jwt.MapClaims) {
		m["data"].(map[string]any)["appAppleId"] = float64(222)
	}))
	if _, err := c.VerifyNotification(payload); !errors.Is(err, appstore.ErrBundleMismatch) {
		t.Fatalf("err = %v, want ErrBundleMismatch", err)
	}
}

// A TEST notification carries no data at all, so it must short-circuit before
// the bundle check — otherwise App Store Connect's "request a test
// notification" button could never succeed.
func TestVerifyNotificationAcceptsTest(t *testing.T) {
	chain := appstoretest.New(t)
	c := clientFor(t, chain)

	n, err := c.VerifyNotification(chain.Sign(t, appstoretest.TestNotificationClaims()))
	if err != nil {
		t.Fatalf("test notification must verify: %v", err)
	}
	if !n.IsTest() || n.Data != nil {
		t.Fatalf("notification: %+v", n)
	}
}

// Production is tried first, and only a 404 falls back to sandbox.
func TestGetSubscriptionStatusesFallsBackToSandbox(t *testing.T) {
	chain := appstoretest.New(t)
	key := newKey(t)

	prodHits, sandboxHits := 0, 0
	prod := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		prodHits++
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
			t.Error("missing bearer token")
		}
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"errorCode":4040010,"errorMessage":"Original transaction id not found."}`))
	}))
	defer prod.Close()

	sandbox := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		sandboxHits++
		body, _ := json.Marshal(appstore.SubscriptionStatusResponse{
			Environment: appstore.EnvSandbox,
			BundleID:    appstoretest.DefaultBundleID,
			Data: []appstore.SubscriptionGroupStatus{{
				SubscriptionGroupIdentifier: "21653891",
				LastTransactions: []appstore.LastTransaction{{
					OriginalTransactionID: "2000000812345678",
					Status:                appstore.SubStatusActive,
					SignedTransactionInfo: chain.Sign(t, appstoretest.TxClaims(func(m jwt.MapClaims) {
						m["environment"] = appstore.EnvSandbox
					})),
				}},
			}},
		})
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
	defer sandbox.Close()

	c := clientFor(t, chain, func(cfg *appstore.Config) {
		cfg.IssuerID, cfg.KeyID, cfg.PrivateKey = "issuer", "KEYID", key
		cfg.BaseProd, cfg.BaseSandbox = prod.URL, sandbox.URL
	})

	res, err := c.GetSubscriptionStatuses(context.Background(), "2000000812345678", "")
	if err != nil {
		t.Fatalf("statuses: %v", err)
	}
	if prodHits != 1 || sandboxHits != 1 {
		t.Fatalf("prod hits %d, sandbox hits %d, want 1 and 1", prodHits, sandboxHits)
	}
	latest := res.Latest()
	if latest == nil || latest.Status != appstore.SubStatusActive {
		t.Fatalf("latest: %+v", latest)
	}
	// The signed payload is what we trust, not the JSON around it.
	if latest.Transaction == nil || latest.Transaction.ProductID != "org.mediarise.tarot.premium.yearly" {
		t.Fatalf("nested transaction: %+v", latest.Transaction)
	}

	// A known environment must not probe the other host.
	prodHits, sandboxHits = 0, 0
	if _, err := c.GetSubscriptionStatuses(context.Background(), "2000000812345678", appstore.EnvSandbox); err != nil {
		t.Fatalf("explicit sandbox: %v", err)
	}
	if prodHits != 0 || sandboxHits != 1 {
		t.Fatalf("explicit env: prod %d sandbox %d, want 0 and 1", prodHits, sandboxHits)
	}
}

// Until an app has live in-app purchases, Apple's production host rejects an
// otherwise valid developer token with 401 while sandbox accepts the same
// token. Verified against the real API: prod 401, sandbox 404. So a 401 from
// production must fall through to sandbox, or App Review's sandbox purchase
// would fail.
func TestProductionUnauthorizedFallsBackToSandbox(t *testing.T) {
	chain := appstoretest.New(t)
	key := newKey(t)

	prodHits, sandboxHits := 0, 0
	prod := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		prodHits++
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer prod.Close()
	sandbox := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		sandboxHits++
		body, _ := json.Marshal(appstore.SubscriptionStatusResponse{
			Environment: appstore.EnvSandbox,
			BundleID:    appstoretest.DefaultBundleID,
			Data: []appstore.SubscriptionGroupStatus{{
				LastTransactions: []appstore.LastTransaction{{
					OriginalTransactionID: "2000000812345678",
					Status:                appstore.SubStatusActive,
					SignedTransactionInfo: chain.Sign(t, appstoretest.TxClaims(func(m jwt.MapClaims) {
						m["environment"] = appstore.EnvSandbox
					})),
				}},
			}},
		})
		_, _ = w.Write(body)
	}))
	defer sandbox.Close()

	c := clientFor(t, chain, func(cfg *appstore.Config) {
		cfg.IssuerID, cfg.KeyID, cfg.PrivateKey = "issuer", "KEYID", key
		cfg.BaseProd, cfg.BaseSandbox = prod.URL, sandbox.URL
	})

	res, err := c.GetSubscriptionStatuses(context.Background(), "2000000812345678", "")
	if err != nil {
		t.Fatalf("a 401 from production must fall through to sandbox: %v", err)
	}
	if prodHits != 1 || sandboxHits != 1 {
		t.Fatalf("prod %d sandbox %d, want 1 and 1", prodHits, sandboxHits)
	}
	if latest := res.Latest(); latest == nil || latest.Transaction == nil {
		t.Fatalf("latest: %+v", latest)
	}
}

// A genuinely bad key must still be reported as a configuration problem, not
// hidden by the sandbox fallback.
func TestUnauthorizedInBothStaysNotConfigured(t *testing.T) {
	chain := appstoretest.New(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := clientFor(t, chain, func(cfg *appstore.Config) {
		cfg.IssuerID, cfg.KeyID, cfg.PrivateKey = "issuer", "KEYID", newKey(t)
		cfg.BaseProd, cfg.BaseSandbox = srv.URL, srv.URL
	})
	if _, err := c.GetSubscriptionStatuses(context.Background(), "2000000812345678", ""); !errors.Is(err, appstore.ErrNotConfigured) {
		t.Fatalf("err = %v, want ErrNotConfigured", err)
	}
}

// With sandbox disabled there is no fallback at all.
func TestNoFallbackWhenSandboxDisallowed(t *testing.T) {
	chain := appstoretest.New(t)
	sandboxHits := 0
	prod := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"errorCode":4040010}`))
	}))
	defer prod.Close()
	sandbox := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		sandboxHits++
	}))
	defer sandbox.Close()

	c := clientFor(t, chain, func(cfg *appstore.Config) {
		cfg.AllowSandbox = false
		cfg.IssuerID, cfg.KeyID, cfg.PrivateKey = "issuer", "KEYID", newKey(t)
		cfg.BaseProd, cfg.BaseSandbox = prod.URL, sandbox.URL
	})
	if _, err := c.GetSubscriptionStatuses(context.Background(), "2000000812345678", ""); !errors.Is(err, appstore.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
	if sandboxHits != 0 {
		t.Fatalf("sandbox was probed %d times despite AllowSandbox=false", sandboxHits)
	}
}

func TestGetSubscriptionStatusesNotFoundInBoth(t *testing.T) {
	chain := appstoretest.New(t)
	notFound := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"errorCode":4040010,"errorMessage":"not found"}`))
	}))
	defer notFound.Close()

	c := clientFor(t, chain, func(cfg *appstore.Config) {
		cfg.IssuerID, cfg.KeyID, cfg.PrivateKey = "issuer", "KEYID", newKey(t)
		cfg.BaseProd, cfg.BaseSandbox = notFound.URL, notFound.URL
	})
	if _, err := c.GetSubscriptionStatuses(context.Background(), "2000000812345678", ""); !errors.Is(err, appstore.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestServerAPIMapsErrors(t *testing.T) {
	chain := appstoretest.New(t)
	cases := []struct {
		status int
		want   error
	}{
		{http.StatusUnauthorized, appstore.ErrNotConfigured},
		{http.StatusInternalServerError, appstore.ErrProviderUnavailable},
		{http.StatusTooManyRequests, appstore.ErrProviderUnavailable},
	}
	for _, tc := range cases {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(tc.status)
			_, _ = w.Write([]byte(`{"errorCode":1,"errorMessage":"nope"}`))
		}))
		c := clientFor(t, chain, func(cfg *appstore.Config) {
			cfg.IssuerID, cfg.KeyID, cfg.PrivateKey = "issuer", "KEYID", newKey(t)
			cfg.BaseProd, cfg.BaseSandbox = srv.URL, srv.URL
		})
		_, err := c.GetSubscriptionStatuses(context.Background(), "2000000812345678", appstore.EnvProduction)
		if !errors.Is(err, tc.want) {
			t.Fatalf("http %d gave %v, want %v", tc.status, err, tc.want)
		}
		srv.Close()
	}
}

// The Server API's plain JSON is never trusted; only the signed payload inside.
func TestGetTransactionInfoRejectsUnsignedResponse(t *testing.T) {
	chain := appstoretest.New(t)
	attacker := appstoretest.New(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"signedTransactionInfo":"` + attacker.Sign(t, appstoretest.TxClaims()) + `"}`))
	}))
	defer srv.Close()

	c := clientFor(t, chain, func(cfg *appstore.Config) {
		cfg.IssuerID, cfg.KeyID, cfg.PrivateKey = "issuer", "KEYID", newKey(t)
		cfg.BaseProd, cfg.BaseSandbox = srv.URL, srv.URL
	})
	if _, err := c.GetTransactionInfo(context.Background(), "2000000812345679", appstore.EnvProduction); !errors.Is(err, appstore.ErrBadSignature) {
		t.Fatalf("err = %v, want ErrBadSignature", err)
	}
}

func TestTransactionHelpers(t *testing.T) {
	var tx appstore.Transaction
	if tx.ExpiresAt() != nil || tx.RevokedAt() != nil || tx.PurchasedAt() != nil {
		t.Fatal("zero dates must be nil, not the epoch")
	}
	if _, _, ok := tx.AmountMinor(); ok {
		t.Fatal("a missing price must report ok=false so the catalog price is used")
	}
	tx.Price, tx.Currency = 7990, "usd"
	minor, currency, ok := tx.AmountMinor()
	if !ok || minor != 799 || currency != "USD" {
		t.Fatalf("AmountMinor = %d %q %v, want 799 USD true", minor, currency, ok)
	}
	tx.Price, tx.Currency = 7990, ""
	if _, _, ok := tx.AmountMinor(); ok {
		t.Fatal("a price without a currency is unusable")
	}
	tx.RevocationDate = time.Now().UnixMilli()
	if !tx.IsRevoked() || tx.RevokedAt() == nil {
		t.Fatal("revocation not detected")
	}
	tx.InAppOwnershipType = appstore.OwnershipFamilyShared
	if !tx.IsFamilyShared() {
		t.Fatal("family sharing not detected")
	}
}

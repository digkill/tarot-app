package appstore_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/digkill/tarot-app/backend/internal/appstore"
	"github.com/digkill/tarot-app/backend/internal/appstore/appstoretest"
)

// clientFor returns a client that trusts the given synthetic chain.
func clientFor(t *testing.T, chain *appstoretest.Chain, mutate ...func(*appstore.Config)) *appstore.Client {
	t.Helper()
	cfg := appstore.Config{
		BundleID:     appstoretest.DefaultBundleID,
		AllowSandbox: true,
		RootCA:       chain.Root(),
	}
	for _, m := range mutate {
		m(&cfg)
	}
	c := appstore.New(cfg)
	if c == nil {
		t.Fatal("New returned nil")
	}
	return c
}

func TestVerifyTransactionHappyPath(t *testing.T) {
	chain := appstoretest.New(t)
	c := clientFor(t, chain)

	tx, err := c.VerifyTransaction(chain.Sign(t, appstoretest.TxClaims()))
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if tx.TransactionID != "2000000812345679" || tx.OriginalTransactionID != "2000000812345678" {
		t.Fatalf("ids: %+v", tx)
	}
	if tx.ProductID != "org.mediarise.tarot.premium.yearly" {
		t.Fatalf("productId: %q", tx.ProductID)
	}
	if tx.AppAccountToken != "8f14e45f-ea1d-4b0a-9f3c-000000000001" {
		t.Fatalf("appAccountToken: %q", tx.AppAccountToken)
	}
	if tx.ExpiresAt() == nil || !tx.ExpiresAt().After(time.Now()) {
		t.Fatalf("expiresAt: %v", tx.ExpiresAt())
	}
	minor, currency, ok := tx.AmountMinor()
	if !ok || minor != 5999 || currency != "USD" {
		t.Fatalf("AmountMinor = %d %q %v, want 5999 USD true", minor, currency, ok)
	}
}

func TestVerifyRejectsForgedSignature(t *testing.T) {
	chain := appstoretest.New(t)
	c := clientFor(t, chain)

	parts := strings.Split(chain.Sign(t, appstoretest.TxClaims()), ".")
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		t.Fatal(err)
	}
	sig[0] ^= 0xff
	parts[2] = base64.RawURLEncoding.EncodeToString(sig)

	if _, err := c.VerifyTransaction(strings.Join(parts, ".")); !errors.Is(err, appstore.ErrBadSignature) {
		t.Fatalf("err = %v, want ErrBadSignature", err)
	}
}

func TestVerifyRejectsTamperedPayload(t *testing.T) {
	chain := appstoretest.New(t)
	c := clientFor(t, chain)

	parts := strings.Split(chain.Sign(t, appstoretest.TxClaims()), ".")
	// Re-encode the payload claiming a lifetime product; the signature no
	// longer covers it.
	parts[1] = base64.RawURLEncoding.EncodeToString([]byte(
		`{"bundleId":"org.mediarise.tarot","productId":"org.mediarise.tarot.premium.lifetime"}`))

	if _, err := c.VerifyTransaction(strings.Join(parts, ".")); !errors.Is(err, appstore.ErrBadSignature) {
		t.Fatalf("err = %v, want ErrBadSignature", err)
	}
}

// The trust anchor is pinned, so a perfectly well-formed chain from someone
// else's root must be rejected. This is the check that stops the whole feature
// from becoming an unauthenticated premium grant.
func TestVerifyRejectsUnpinnedRoot(t *testing.T) {
	attacker := appstoretest.New(t)
	// Trusts the real embedded Apple root, not the attacker's.
	c := appstore.New(appstore.Config{BundleID: appstoretest.DefaultBundleID, AllowSandbox: true})
	if c == nil {
		t.Fatal("New returned nil")
	}

	if _, err := c.VerifyTransaction(attacker.Sign(t, appstoretest.TxClaims())); !errors.Is(err, appstore.ErrBadSignature) {
		t.Fatalf("err = %v, want ErrBadSignature", err)
	}
}

// A chain whose self-signed tail is not our root is refused outright, so the
// pin cannot be argued away by chain shape.
func TestVerifyRejectsSwappedRootInChain(t *testing.T) {
	real := appstoretest.New(t)
	other := appstoretest.New(t)
	c := clientFor(t, real, func(cfg *appstore.Config) { cfg.RootCA = other.Root() })

	if _, err := c.VerifyTransaction(real.Sign(t, appstoretest.TxClaims())); !errors.Is(err, appstore.ErrBadSignature) {
		t.Fatalf("err = %v, want ErrBadSignature", err)
	}
}

// Without pinning the algorithm an attacker picks it and forges a token while
// presenting a genuine-looking x5c chain.
func TestVerifyRejectsAlgConfusion(t *testing.T) {
	chain := appstoretest.New(t)
	c := clientFor(t, chain)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, appstoretest.TxClaims())
	token.Header["x5c"] = chain.X5C()
	forged, err := token.SignedString([]byte("secret"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.VerifyTransaction(forged); !errors.Is(err, appstore.ErrBadSignature) {
		t.Fatalf("err = %v, want ErrBadSignature", err)
	}

	// Sanity: the HMAC really is valid for that secret, so only the algorithm
	// pin rejected it.
	parts := strings.Split(forged, ".")
	mac := hmac.New(sha256.New, []byte("secret"))
	mac.Write([]byte(parts[0] + "." + parts[1]))
	if parts[2] != base64.RawURLEncoding.EncodeToString(mac.Sum(nil)) {
		t.Fatal("test built an invalid HMAC, so the assertion above proved nothing")
	}
}

func TestVerifyRejectsMissingMarkerOID(t *testing.T) {
	chain := appstoretest.New(t, appstoretest.WithoutMarkerOID)
	c := clientFor(t, chain)

	_, err := c.VerifyTransaction(chain.Sign(t, appstoretest.TxClaims()))
	if !errors.Is(err, appstore.ErrBadSignature) {
		t.Fatalf("err = %v, want ErrBadSignature", err)
	}
	if !strings.Contains(err.Error(), "1.2.840.113635.100.6.11.1") {
		t.Fatalf("error should name the missing marker OID: %v", err)
	}
}

func TestVerifyRejectsMissingX5C(t *testing.T) {
	chain := appstoretest.New(t)
	c := clientFor(t, chain)

	token := jwt.NewWithClaims(jwt.SigningMethodES256, appstoretest.TxClaims())
	signed, err := token.SignedString(chain.LeafKey())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.VerifyTransaction(signed); !errors.Is(err, appstore.ErrBadSignature) {
		t.Fatalf("err = %v, want ErrBadSignature", err)
	}
}

func TestVerifyRejectsWrongBundleID(t *testing.T) {
	chain := appstoretest.New(t)
	c := clientFor(t, chain)

	signed := chain.Sign(t, appstoretest.TxClaims(func(m jwt.MapClaims) { m["bundleId"] = "com.evil.app" }))
	if _, err := c.VerifyTransaction(signed); !errors.Is(err, appstore.ErrBundleMismatch) {
		t.Fatalf("err = %v, want ErrBundleMismatch", err)
	}
}

func TestVerifyRejectsSandboxWhenDisallowed(t *testing.T) {
	chain := appstoretest.New(t)
	signed := chain.Sign(t, appstoretest.TxClaims(func(m jwt.MapClaims) { m["environment"] = appstore.EnvSandbox }))

	strict := clientFor(t, chain, func(cfg *appstore.Config) { cfg.AllowSandbox = false })
	if _, err := strict.VerifyTransaction(signed); !errors.Is(err, appstore.ErrSandboxRejected) {
		t.Fatalf("err = %v, want ErrSandboxRejected", err)
	}

	// The default must accept sandbox: App Review and TestFlight both use it
	// against a production build.
	if _, err := clientFor(t, chain).VerifyTransaction(signed); err != nil {
		t.Fatalf("sandbox must be accepted by default: %v", err)
	}
}

func TestVerifyUnconfigured(t *testing.T) {
	var c *appstore.Client
	if _, err := c.VerifyTransaction("whatever"); !errors.Is(err, appstore.ErrNotConfigured) {
		t.Fatalf("nil client err = %v, want ErrNotConfigured", err)
	}
	if appstore.New(appstore.Config{BundleID: "  "}) != nil {
		t.Fatal("New must return nil without a bundle id")
	}
}

func TestVerifyEmptyPayload(t *testing.T) {
	chain := appstoretest.New(t)
	if _, err := clientFor(t, chain).VerifyTransaction("   "); !errors.Is(err, appstore.ErrBadSignature) {
		t.Fatalf("err = %v, want ErrBadSignature", err)
	}
}

// An old but genuine transaction — replayed from currentEntitlements on a cold
// start — must still verify even though its certificates have since expired.
// This is why the chain is evaluated at signedDate rather than now.
func TestVerifyAcceptsOldSignedDateWithExpiredChain(t *testing.T) {
	notBefore := time.Now().Add(-3 * 365 * 24 * time.Hour)
	notAfter := time.Now().Add(-2 * 365 * 24 * time.Hour)
	chain := appstoretest.NewWindow(t, notBefore, notAfter)
	c := clientFor(t, chain)

	signedAt := notBefore.Add(30 * 24 * time.Hour)
	signed := chain.Sign(t, appstoretest.TxClaims(func(m jwt.MapClaims) {
		m["signedDate"] = float64(signedAt.UnixMilli())
		m["purchaseDate"] = float64(signedAt.UnixMilli())
		m["expiresDate"] = float64(signedAt.Add(366 * 24 * time.Hour).UnixMilli())
	}))
	if _, err := c.VerifyTransaction(signed); err != nil {
		t.Fatalf("an old genuine transaction must still verify: %v", err)
	}

	// With no signedDate there is nothing to anchor to, so the long-expired
	// chain is correctly refused at the current time.
	stale := chain.Sign(t, appstoretest.TxClaims(func(m jwt.MapClaims) { delete(m, "signedDate") }))
	if _, err := c.VerifyTransaction(stale); !errors.Is(err, appstore.ErrBadSignature) {
		t.Fatalf("err = %v, want ErrBadSignature for an expired chain with no signedDate", err)
	}
}

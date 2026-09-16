package appstore_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/joho/godotenv"

	"github.com/digkill/tarot-app/backend/internal/appstore"
)

// TestAppleCredentialsAreAccepted talks to Apple's real App Store Server API to
// prove the issuer id, key id and .p8 in backend/.env actually work together.
// It asks about a transaction id that cannot exist, so nothing is changed:
//
//	APPLE_LIVE_CHECK=1 go test ./internal/appstore/ -run TestAppleCredentialsAreAccepted -v
//
// A 404 (ErrNotFound) is the success case — Apple authenticated us and simply
// has no such subscription. ErrNotConfigured means Apple rejected the developer
// token, which is what a wrong issuer id, key id, bundle id or key looks like.
func TestAppleCredentialsAreAccepted(t *testing.T) {
	if os.Getenv("APPLE_LIVE_CHECK") == "" {
		t.Skip("set APPLE_LIVE_CHECK=1 to call Apple for real")
	}
	env, err := godotenv.Read(filepath.Join("..", "..", ".env"))
	if err != nil {
		t.Skipf("no backend/.env: %v", err)
	}
	key := parseP8(t, env["APPLE_IAP_PRIVATE_KEY"], env["APPLE_IAP_PRIVATE_KEY_FILE"])

	c := appstore.New(appstore.Config{
		BundleID:     env["APPLE_BUNDLE_ID"],
		IssuerID:     env["APPLE_IAP_ISSUER_ID"],
		KeyID:        env["APPLE_IAP_KEY_ID"],
		PrivateKey:   key,
		AllowSandbox: true,
	})
	if c == nil || !c.ServerAPIEnabled() {
		t.Fatal("the client is not fully configured; check APPLE_* in backend/.env")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	// Sandbox is the environment that matters before release: it is what App
	// Review and TestFlight use, and until the app has live in-app purchases
	// Apple's production host rejects even a perfectly valid developer token.
	// An id of the right shape that Apple cannot have issued changes nothing.
	_, err = c.GetSubscriptionStatuses(ctx, "1000000000000001", appstore.EnvSandbox)

	switch {
	case errors.Is(err, appstore.ErrNotFound):
		t.Log("sandbox accepted the developer token (404 for an unknown subscription) — credentials are good")
	case errors.Is(err, appstore.ErrNotConfigured):
		t.Fatalf("Apple rejected the developer token: %v\n"+
			"check APPLE_IAP_ISSUER_ID, APPLE_IAP_KEY_ID, APPLE_BUNDLE_ID and that the key is an "+
			"In-App Purchase key for this app", err)
	case errors.Is(err, appstore.ErrProviderUnavailable):
		t.Fatalf("could not reach the App Store Server API: %v", err)
	case err == nil:
		t.Fatal("Apple returned a subscription for an id it cannot have issued")
	default:
		t.Fatalf("unexpected error: %v", err)
	}

	// Production is reported, not asserted: a 401 here is expected until the
	// in-app purchases are live, and the client falls back to sandbox for it.
	if _, perr := c.GetSubscriptionStatuses(ctx, "1000000000000001", appstore.EnvProduction); perr != nil {
		t.Logf("production says: %v (a 401 is normal before the IAPs go live)", perr)
	} else {
		t.Log("production accepted the token too")
	}
}

func parseP8(t *testing.T, raw, file string) *ecdsa.PrivateKey {
	t.Helper()
	if file != "" {
		b, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read key file: %v", err)
		}
		raw = string(b)
	}
	if strings.TrimSpace(raw) == "" {
		t.Skip("no Apple IAP key configured in backend/.env")
	}
	der := []byte(strings.ReplaceAll(raw, `\n`, "\n"))
	if !strings.Contains(string(der), "-----BEGIN") {
		decoded, err := base64.StdEncoding.DecodeString(strings.Join(strings.Fields(raw), ""))
		if err != nil {
			t.Fatalf("key is neither PEM nor base64: %v", err)
		}
		der = decoded
	}
	if block, _ := pem.Decode(der); block != nil {
		der = block.Bytes
	}
	parsed, err := x509.ParsePKCS8PrivateKey(der)
	if err != nil {
		t.Fatalf("parse key: %v", err)
	}
	key, ok := parsed.(*ecdsa.PrivateKey)
	if !ok {
		t.Fatalf("key is %T, want ECDSA", parsed)
	}
	return key
}

package config

import (
	"path/filepath"
	"testing"

	"github.com/joho/godotenv"
)

// TestAppleKeyInDotEnvLoads proves the value that
// scripts/install-apple-iap-key.sh writes into backend/.env is actually
// loadable by the server. Run it after installing or rotating the key:
//
//	go test ./internal/config/ -run TestAppleKeyInDotEnvLoads -v
//
// It skips when there is no .env or no key, so CI stays green.
func TestAppleKeyInDotEnvLoads(t *testing.T) {
	// Read rather than Load: Load would mutate this process's environment and
	// leak into the other tests in this package.
	env, err := godotenv.Read(filepath.Join("..", "..", ".env"))
	if err != nil {
		t.Skipf("no backend/.env: %v", err)
	}
	if env["APPLE_IAP_PRIVATE_KEY"] == "" && env["APPLE_IAP_PRIVATE_KEY_FILE"] == "" {
		t.Skip("no Apple IAP key configured in backend/.env")
	}

	key, err := parseAppleP8(env["APPLE_IAP_PRIVATE_KEY"], env["APPLE_IAP_PRIVATE_KEY_FILE"])
	if err != nil {
		t.Fatalf("the key in backend/.env does not parse: %v", err)
	}
	if key == nil {
		t.Fatal("the key parsed to nil, so the App Store Server API would stay disabled")
	}
	if name := key.Curve.Params().Name; name != "P-256" {
		t.Fatalf("curve = %s, want P-256 — Apple signs developer tokens with ES256", name)
	}

	// Without these the client reports ServerAPIEnabled() == false and silently
	// degrades to trusting the client's own signed transaction.
	for _, name := range []string{"APPLE_IAP_ISSUER_ID", "APPLE_IAP_KEY_ID", "APPLE_BUNDLE_ID"} {
		if env[name] == "" {
			t.Errorf("%s is missing from backend/.env", name)
		}
	}
	t.Logf("key ok: issuer=%s keyId=%s bundle=%s curve=P-256",
		env["APPLE_IAP_ISSUER_ID"], env["APPLE_IAP_KEY_ID"], env["APPLE_BUNDLE_ID"])
}

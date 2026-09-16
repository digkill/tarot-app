package config

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newP8(t *testing.T) (*ecdsa.PrivateKey, string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	return key, string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))
}

func TestParseAppleP8Formats(t *testing.T) {
	key, pemText := newP8(t)

	t.Run("raw PEM", func(t *testing.T) {
		got, err := parseAppleP8(pemText, "")
		if err != nil || got == nil || !got.Equal(key) {
			t.Fatalf("got %v err %v", got, err)
		}
	})

	// How the key arrives when pasted into a single-line docker env var.
	t.Run("PEM with literal newline escapes", func(t *testing.T) {
		escaped := strings.ReplaceAll(pemText, "\n", `\n`)
		got, err := parseAppleP8(escaped, "")
		if err != nil || got == nil || !got.Equal(key) {
			t.Fatalf("got %v err %v", got, err)
		}
	})

	t.Run("base64 of the whole file", func(t *testing.T) {
		got, err := parseAppleP8(base64.StdEncoding.EncodeToString([]byte(pemText)), "")
		if err != nil || got == nil || !got.Equal(key) {
			t.Fatalf("got %v err %v", got, err)
		}
	})

	t.Run("base64 of raw DER", func(t *testing.T) {
		block, _ := pem.Decode([]byte(pemText))
		got, err := parseAppleP8(base64.StdEncoding.EncodeToString(block.Bytes), "")
		if err != nil || got == nil || !got.Equal(key) {
			t.Fatalf("got %v err %v", got, err)
		}
	})

	t.Run("file path", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "AuthKey_ABCDE12345.p8")
		if err := os.WriteFile(path, []byte(pemText), 0o600); err != nil {
			t.Fatal(err)
		}
		got, err := parseAppleP8("", path)
		if err != nil || got == nil || !got.Equal(key) {
			t.Fatalf("got %v err %v", got, err)
		}
	})

	t.Run("SEC 1 EC key", func(t *testing.T) {
		der, err := x509.MarshalECPrivateKey(key)
		if err != nil {
			t.Fatal(err)
		}
		text := string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: der}))
		got, err := parseAppleP8(text, "")
		if err != nil || got == nil || !got.Equal(key) {
			t.Fatalf("got %v err %v", got, err)
		}
	})
}

// Absence disables the provider; a malformed value must fail at startup rather
// than at the first purchase.
func TestParseAppleP8BlankIsDisabledNotAnError(t *testing.T) {
	for _, raw := range []string{"", "   ", "\n"} {
		got, err := parseAppleP8(raw, "")
		if err != nil || got != nil {
			t.Fatalf("parseAppleP8(%q) = %v, %v; want nil, nil", raw, got, err)
		}
	}
}

func TestParseAppleP8RejectsGarbage(t *testing.T) {
	cases := map[string]string{
		"not base64 and not PEM": "this is clearly not a key",
		"valid base64, not a key": base64.StdEncoding.EncodeToString(
			[]byte("still not a key, but it does decode")),
		"PEM header with no body": "-----BEGIN PRIVATE KEY-----",
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := parseAppleP8(raw, ""); err == nil {
				t.Fatal("a present but unparseable key must be a startup error")
			}
		})
	}

	if _, err := parseAppleP8("", filepath.Join(t.TempDir(), "missing.p8")); err == nil {
		t.Fatal("a missing key file must be a startup error")
	}
}

// An RSA key would sign, but Apple only accepts ES256.
func TestParseAppleP8RejectsNonECDSA(t *testing.T) {
	// A PKCS#8 Ed25519 key parses fine but is the wrong type.
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		t.Fatal(err)
	}
	text := string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))
	if _, err := parseAppleP8(text, ""); err == nil {
		t.Fatal("a non-ECDSA key must be rejected")
	}
}

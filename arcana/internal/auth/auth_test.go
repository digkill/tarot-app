package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const secret = "0123456789abcdef0123456789abcdef"

func sign(t *testing.T, method jwt.SigningMethod, key any, claims jwt.RegisteredClaims) string {
	t.Helper()
	s, err := jwt.NewWithClaims(method, claims).SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestUserID(t *testing.T) {
	v, err := NewVerifier(secret)
	if err != nil {
		t.Fatal(err)
	}
	valid := jwt.RegisteredClaims{Subject: "user-1", ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute))}
	if id, err := v.UserID(sign(t, jwt.SigningMethodHS256, []byte(secret), valid)); err != nil || id != "user-1" {
		t.Fatalf("valid token: %q %v", id, err)
	}

	cases := map[string]string{
		"wrong secret": sign(t, jwt.SigningMethodHS256, []byte("another-secret-another-secret-xx"), valid),
		"expired": sign(t, jwt.SigningMethodHS256, []byte(secret), jwt.RegisteredClaims{
			Subject: "user-1", ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Minute))}),
		"no expiry":  sign(t, jwt.SigningMethodHS256, []byte(secret), jwt.RegisteredClaims{Subject: "user-1"}),
		"no subject": sign(t, jwt.SigningMethodHS256, []byte(secret), jwt.RegisteredClaims{ExpiresAt: valid.ExpiresAt}),
		"hs512":      sign(t, jwt.SigningMethodHS512, []byte(secret), valid),
		"garbage":    "not.a.token",
	}
	for name, token := range cases {
		if _, err := v.UserID(token); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	if _, err := NewVerifier("short"); err == nil {
		t.Error("short secret accepted")
	}
}

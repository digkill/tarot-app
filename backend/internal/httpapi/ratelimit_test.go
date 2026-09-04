package httpapi

import (
	"testing"
	"time"
)

func TestRateLimiterAllowsThenBlocks(t *testing.T) {
	l := newRateLimiter()
	key := "ip:1.2.3.4"
	for i := 0; i < 8; i++ {
		if !l.allow(key, 8, 15*time.Minute) {
			t.Fatalf("allowed hit %d blocked", i+1)
		}
	}
	if l.allow(key, 8, 15*time.Minute) {
		t.Fatal("9th hit should be blocked")
	}
	if !l.allow("ip:9.9.9.9", 8, 15*time.Minute) {
		t.Fatal("other IP should be independent")
	}
}

func TestIsAuthSensitive(t *testing.T) {
	if !isAuthSensitive("/api/v1/auth/login") {
		t.Fatal("login")
	}
	if isAuthSensitive("/api/v1/me") {
		t.Fatal("me should not be auth-sensitive")
	}
}

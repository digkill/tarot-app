package httpapi

import (
	"testing"
	"time"

	"github.com/digkill/tarot-app/backend/internal/usage"
)

func TestAdTokenRejectsEarlyConsume(t *testing.T) {
	store := newAdTokenStore()
	token, err := store.issue("user-1")
	if err != nil {
		t.Fatal(err)
	}
	if err = store.consume("user-1", token, usage.AdMinWatch); err != errAdWatchTooShort {
		t.Fatalf("got %v", err)
	}
}

func TestAdTokenHappyPath(t *testing.T) {
	store := newAdTokenStore()
	token, err := store.issue("user-1")
	if err != nil {
		t.Fatal(err)
	}
	store.mu.Lock()
	g := store.tokens[token]
	g.issuedAt = time.Now().Add(-usage.AdMinWatch - time.Second)
	store.tokens[token] = g
	store.mu.Unlock()
	if err = store.consume("user-1", token, usage.AdMinWatch); err != nil {
		t.Fatal(err)
	}
	if err = store.consume("user-1", token, usage.AdMinWatch); err != errAdTokenUnknown {
		t.Fatalf("reuse got %v", err)
	}
}

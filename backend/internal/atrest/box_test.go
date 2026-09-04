package atrest

import (
	"encoding/hex"
	"strings"
	"testing"
)

func TestSealOpenRoundTrip(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}
	box, err := NewBox(key)
	if err != nil {
		t.Fatal(err)
	}

	plain := "203.0.113.10"
	first, err := box.Seal(plain)
	if err != nil {
		t.Fatal(err)
	}
	second, err := box.Seal(plain)
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("expected unique ciphertext per seal")
	}
	if !strings.HasPrefix(first, "v1:") {
		t.Fatalf("missing version prefix: %s", first)
	}
	got, err := box.Open(first)
	if err != nil {
		t.Fatal(err)
	}
	if got != plain {
		t.Fatalf("got %q want %q", got, plain)
	}
}

func TestSealEmpty(t *testing.T) {
	box, err := NewBox(make([]byte, 32))
	if err != nil {
		t.Fatal(err)
	}
	got, err := box.Seal("")
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Fatalf("empty plaintext should stay empty, got %q", got)
	}
}

func TestParseKey(t *testing.T) {
	raw := strings.Repeat("ab", 32)
	key, err := ParseKey(raw)
	if err != nil {
		t.Fatal(err)
	}
	if hex.EncodeToString(key) != raw {
		t.Fatal("hex roundtrip mismatch")
	}
	if _, err = ParseKey("short"); err == nil {
		t.Fatal("expected error for short key")
	}
}

package appstore

import (
	"encoding/base64"
	"testing"
	"time"
)

// The chain is evaluated at the payload's own signedDate so a genuinely old
// transaction still verifies, but a claimed time must never run ahead of ours.
func TestChainTimeUsesSignedDate(t *testing.T) {
	now := time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)
	c := New(Config{BundleID: "x", Now: func() time.Time { return now }})

	past := now.Add(-365 * 24 * time.Hour)
	if got := c.chainTime(past); !got.Equal(past) {
		t.Fatalf("a past signedDate must be used: %v", got)
	}
	if got := c.chainTime(time.Time{}); !got.Equal(now) {
		t.Fatalf("a missing signedDate falls back to now: %v", got)
	}
	if got := c.chainTime(now.Add(48 * time.Hour)); !got.Equal(now) {
		t.Fatalf("a future signedDate must be clamped to now: %v", got)
	}
}

func TestPeekSignedDate(t *testing.T) {
	if got := peekSignedDate("not.a.jws"); !got.IsZero() {
		t.Fatalf("garbage must yield zero: %v", got)
	}
	if got := peekSignedDate("only.two"); !got.IsZero() {
		t.Fatalf("wrong segment count must yield zero: %v", got)
	}
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"signedDate":1757937600000}`))
	if got := peekSignedDate("h." + payload + ".s"); got.UnixMilli() != 1757937600000 {
		t.Fatalf("signedDate = %v", got)
	}
}

// The marker OID is the check that stops any other Apple-issued certificate
// from signing payloads, so pin its value here too.
func TestMarkerOIDValue(t *testing.T) {
	if appleMarkerOID != "1.2.840.113635.100.6.11.1" {
		t.Fatalf("marker OID = %q", appleMarkerOID)
	}
}

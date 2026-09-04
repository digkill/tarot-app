package billing

import (
	"testing"
	"time"
)

func TestExpiresAt(t *testing.T) {
	from := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	monthly, _ := Lookup(ProductMonthly)
	got := ExpiresAt(monthly, from)
	if got == nil || !got.Equal(from.AddDate(0, 0, 31)) {
		t.Fatalf("monthly: %v", got)
	}
	yearly, _ := Lookup(ProductYearly)
	got = ExpiresAt(yearly, from)
	if got == nil || !got.Equal(from.AddDate(0, 0, 366)) {
		t.Fatalf("yearly: %v", got)
	}
	life, _ := Lookup(ProductLifetime)
	if ExpiresAt(life, from) != nil {
		t.Fatal("lifetime must not expire")
	}
}

func TestParseTime(t *testing.T) {
	if ParseTime("") != nil {
		t.Fatal("empty")
	}
	rfc := ParseTime("2026-10-05T12:00:00Z")
	if rfc == nil || rfc.Year() != 2026 || rfc.Month() != 10 {
		t.Fatalf("rfc: %v", rfc)
	}
	unix := ParseTime("1788883200")
	if unix == nil {
		t.Fatal("unix")
	}
}

func TestIsStoreManaged(t *testing.T) {
	if !IsStoreManaged(ProviderRuStore) || !IsStoreManaged(ProviderDev) {
		t.Fatal("store sources")
	}
	if IsStoreManaged(ProviderAdmin) || IsStoreManaged("") {
		t.Fatal("admin/legacy must not auto-revoke from store")
	}
}

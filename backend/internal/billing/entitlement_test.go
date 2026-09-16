package billing

import "testing"

func TestCanRevokeFrom(t *testing.T) {
	if !CanRevokeFrom(ProviderRuStore, ProviderRuStore) {
		t.Fatal("rustore can revoke rustore")
	}
	if CanRevokeFrom(ProviderYooKassa, ProviderRuStore) {
		t.Fatal("rustore must not revoke yookassa")
	}
	if CanRevokeFrom(ProviderAdmin, ProviderRuStore) {
		t.Fatal("rustore must not revoke admin")
	}
	if !CanRevokeFrom("", ProviderRuStore) {
		t.Fatal("legacy empty source: rustore sync may revoke")
	}
	if CanRevokeFrom(ProviderYooKassa, "") {
		t.Fatal("empty reporter")
	}
}

func TestActivePremiumSource(t *testing.T) {
	if _, ok := ActivePremiumSource(false, ProviderYooKassa); ok {
		t.Fatal("inactive must not block")
	}
	source, ok := ActivePremiumSource(true, "")
	if !ok || source != ProviderRuStore {
		t.Fatalf("legacy rustore: %s %v", source, ok)
	}
	source, ok = ActivePremiumSource(true, ProviderYooKassa)
	if !ok || source != ProviderYooKassa {
		t.Fatalf("yookassa: %s %v", source, ok)
	}
	source, ok = ActivePremiumSource(true, ProviderCloudPayments)
	if !ok || source != ProviderCloudPayments {
		t.Fatalf("cloudpayments must not collapse to admin: %s %v", source, ok)
	}
	source, ok = ActivePremiumSource(true, ProviderApple)
	if !ok || source != ProviderApple {
		t.Fatalf("apple must not collapse to admin: %s %v", source, ok)
	}
}

func TestCloudPaymentsCannotBeSelfReported(t *testing.T) {
	// NormalizeSource feeds the client-reported purchase endpoint; a
	// CloudPayments grant must only ever come from a signed webhook.
	if got := NormalizeSource(ProviderCloudPayments); got == ProviderCloudPayments {
		t.Fatalf("client could self-report a cloudpayments purchase: %s", got)
	}
	if CanRevokeFrom(ProviderCloudPayments, ProviderRuStore) {
		t.Fatal("rustore sync must not revoke cloudpayments premium")
	}
}

func TestAppleCannotBeSelfReported(t *testing.T) {
	// Same rule as CloudPayments: an Apple entitlement may only come from a
	// server-verified signed transaction, never from the client-reported
	// purchase endpoint. Both spellings used to alias onto yookassa.
	for _, raw := range []string{"apple", "Apple", "APPLE", "ios", "iOS"} {
		if got := NormalizeSource(raw); got == ProviderApple {
			t.Fatalf("client could self-report an apple purchase via %q: %s", raw, got)
		}
		if got := NormalizeSource(raw); got == ProviderYooKassa {
			t.Fatalf("%q must no longer alias onto yookassa: %s", raw, got)
		}
	}
	if CanRevokeFrom(ProviderApple, ProviderRuStore) {
		t.Fatal("rustore sync must not revoke apple premium")
	}
	if CanRevokeFrom(ProviderApple, ProviderDev) {
		t.Fatal("dev sync must not revoke apple premium")
	}
	if !CanRevokeFrom(ProviderApple, ProviderApple) {
		t.Fatal("apple must be able to revoke its own grant")
	}
}

func TestIsAppleManaged(t *testing.T) {
	if !IsAppleManaged(" Apple ") {
		t.Fatal("apple is apple-managed")
	}
	for _, source := range []string{ProviderRuStore, ProviderYooKassa, ProviderCloudPayments, ProviderAdmin, ""} {
		if IsAppleManaged(source) {
			t.Fatalf("%q must not be apple-managed", source)
		}
	}
}

func TestKeepLifetime(t *testing.T) {
	if !KeepLifetime(ProductLifetime, ProductMonthly) {
		t.Fatal("keep lifetime")
	}
	if KeepLifetime(ProductMonthly, ProductLifetime) {
		t.Fatal("allow upgrade to lifetime")
	}
}

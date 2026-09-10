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
}

func TestKeepLifetime(t *testing.T) {
	if !KeepLifetime(ProductLifetime, ProductMonthly) {
		t.Fatal("keep lifetime")
	}
	if KeepLifetime(ProductMonthly, ProductLifetime) {
		t.Fatal("allow upgrade to lifetime")
	}
}

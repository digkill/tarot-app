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

func TestKeepLifetime(t *testing.T) {
	if !KeepLifetime(ProductLifetime, ProductMonthly) {
		t.Fatal("keep lifetime")
	}
	if KeepLifetime(ProductMonthly, ProductLifetime) {
		t.Fatal("allow upgrade to lifetime")
	}
}

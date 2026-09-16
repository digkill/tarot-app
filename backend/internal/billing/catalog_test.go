package billing

import "testing"

func TestLookupKnownProducts(t *testing.T) {
	monthly, ok := Lookup(ProductMonthly)
	if !ok || monthly.AmountKop != 59900 || monthly.Kind != KindSubscription {
		t.Fatalf("monthly: %+v ok=%v", monthly, ok)
	}
	yearly, ok := Lookup(ProductYearly)
	if !ok || yearly.AmountKop != 499000 {
		t.Fatalf("yearly: %+v", yearly)
	}
	life, ok := Lookup(ProductLifetime)
	if !ok || life.Kind != KindOneTime || life.AmountKop != 699000 {
		t.Fatalf("lifetime: %+v", life)
	}
}

func TestEveryProductHasAUSDPrice(t *testing.T) {
	for _, p := range Products {
		if p.AmountUSDCents <= 0 {
			t.Fatalf("%s has no USD price for CloudPayments", p.ID)
		}
	}
}

func TestLookupUnknown(t *testing.T) {
	if _, ok := Lookup("nope"); ok {
		t.Fatal("expected miss")
	}
}

func TestEveryProductHasAUniqueAppleSKU(t *testing.T) {
	seen := map[string]string{}
	for _, p := range Products {
		if p.AppleProductID == "" {
			t.Fatalf("%s has no Apple SKU, so an IAP purchase of it cannot be mapped back", p.ID)
		}
		if prev, dup := seen[p.AppleProductID]; dup {
			t.Fatalf("Apple SKU %q is shared by %s and %s", p.AppleProductID, prev, p.ID)
		}
		seen[p.AppleProductID] = p.ID
	}
}

func TestLookupAppleRoundTrips(t *testing.T) {
	for _, p := range Products {
		got, ok := LookupApple(p.AppleProductID)
		if !ok || got.ID != p.ID {
			t.Fatalf("LookupApple(%q) = %+v ok=%v, want %s", p.AppleProductID, got, ok, p.ID)
		}
		if sku := AppleSKU(p.ID); sku != p.AppleProductID {
			t.Fatalf("AppleSKU(%q) = %q, want %q", p.ID, sku, p.AppleProductID)
		}
	}
	if _, ok := LookupApple("org.mediarise.tarot.deck.japanese"); ok {
		t.Fatal("a deck SKU must miss the premium catalog so the deck repo is consulted")
	}
	if _, ok := LookupApple("  "); ok {
		t.Fatal("blank SKU must miss")
	}
	if sku := AppleSKU("nope"); sku != "" {
		t.Fatalf("AppleSKU of an unknown product = %q, want empty", sku)
	}
}

func TestNormalizeProviderApple(t *testing.T) {
	for _, raw := range []string{"apple", "Apple", " APPLE ", "ios", "iOS"} {
		if got := NormalizeProvider(raw); got != ProviderApple {
			t.Fatalf("NormalizeProvider(%q) = %q, want %q", raw, got, ProviderApple)
		}
	}
	if got := NormalizeProvider(ProviderYooKassa); got != ProviderYooKassa {
		t.Fatalf("yookassa must stay itself: %q", got)
	}
	if got := NormalizeProvider("stripe"); got != ProviderAdmin {
		t.Fatalf("unknown provider = %q, want %q", got, ProviderAdmin)
	}
}

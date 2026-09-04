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

func TestLookupUnknown(t *testing.T) {
	if _, ok := Lookup("nope"); ok {
		t.Fatal("expected miss")
	}
}

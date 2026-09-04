package billing

import "strings"

type Product struct {
	ID        string
	Title     string
	Kind      string
	AmountKop int
}

const (
	KindSubscription = "subscription"
	KindOneTime      = "one_time"
	KindPromo        = "promo"
	KindRestore      = "restore"

	ProviderRuStore  = "rustore"
	ProviderAdmin    = "admin"
	ProviderPromo    = "promo"
	ProviderDev      = "dev"
	ProviderYooKassa = "yookassa"

	StatusPending  = "pending"
	StatusPaid     = "paid"
	StatusRefunded = "refunded"
	StatusCanceled = "canceled"

	ProductMonthly  = "premium_monthly"
	ProductYearly   = "premium_yearly"
	ProductLifetime = "premium_lifetime"
)

var Products = []Product{
	{ID: ProductMonthly, Title: "Премиум на месяц", Kind: KindSubscription, AmountKop: 599_00},
	{ID: ProductYearly, Title: "Премиум на год", Kind: KindSubscription, AmountKop: 4_990_00},
	{ID: ProductLifetime, Title: "Премиум навсегда", Kind: KindOneTime, AmountKop: 6_990_00},
}

func Lookup(id string) (Product, bool) {
	id = strings.TrimSpace(id)
	for _, p := range Products {
		if p.ID == id {
			return p, true
		}
	}
	return Product{}, false
}

func IsPremiumProduct(id string) bool {
	_, ok := Lookup(strings.TrimSpace(id))
	return ok
}

func Title(id string) string {
	if p, ok := Lookup(id); ok {
		return p.Title
	}
	if id == "" {
		return "—"
	}
	return id
}

func NormalizeProvider(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case ProviderRuStore:
		return ProviderRuStore
	case ProviderYooKassa, "apple", "ios":
		return ProviderYooKassa
	case ProviderPromo:
		return ProviderPromo
	case ProviderDev:
		return ProviderDev
	default:
		return ProviderAdmin
	}
}

func NormalizeSource(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "restore":
		return KindRestore
	case ProviderDev:
		return ProviderDev
	case ProviderYooKassa, "apple", "ios":
		return ProviderYooKassa
	default:
		return ProviderRuStore
	}
}

package billing

import "strings"

type Product struct {
	ID        string
	Title     string
	Kind      string
	AmountKop int
	// AmountUSDCents is the price charged to foreign cards via CloudPayments.
	AmountUSDCents int
	TitleEN        string
	// AppleProductID is the App Store Connect SKU. Apple charges the storefront
	// price it holds itself, so this only maps a purchase back to a product.
	AppleProductID string
}

const (
	KindSubscription = "subscription"
	KindOneTime      = "one_time"
	KindPromo        = "promo"
	KindRestore      = "restore"

	ProviderRuStore       = "rustore"
	ProviderAdmin         = "admin"
	ProviderPromo         = "promo"
	ProviderDev           = "dev"
	ProviderYooKassa      = "yookassa"
	ProviderCloudPayments = "cloudpayments"
	ProviderApple         = "apple"

	CurrencyRUB = "RUB"
	CurrencyUSD = "USD"

	StatusPending  = "pending"
	StatusPaid     = "paid"
	StatusRefunded = "refunded"
	StatusCanceled = "canceled"

	ProductMonthly  = "premium_monthly"
	ProductYearly   = "premium_yearly"
	ProductLifetime = "premium_lifetime"
)

var Products = []Product{
	{ID: ProductMonthly, Title: "Премиум на месяц", Kind: KindSubscription, AmountKop: 599_00, AmountUSDCents: 7_99, TitleEN: "Premium, 1 month", AppleProductID: "org.mediarise.tarot.premium.monthly"},
	{ID: ProductYearly, Title: "Премиум на год", Kind: KindSubscription, AmountKop: 4_990_00, AmountUSDCents: 59_99, TitleEN: "Premium, 1 year", AppleProductID: "org.mediarise.tarot.premium.yearly"},
	{ID: ProductLifetime, Title: "Премиум навсегда", Kind: KindOneTime, AmountKop: 6_990_00, AmountUSDCents: 79_99, TitleEN: "Premium, lifetime", AppleProductID: "org.mediarise.tarot.premium.lifetime"},
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

// LookupApple resolves an App Store Connect SKU to a premium product. Deck SKUs
// are not in this catalog; callers fall back to the deck repo, the same way
// ReportPurchase already branches between premium products and shop decks.
func LookupApple(sku string) (Product, bool) {
	sku = strings.TrimSpace(sku)
	if sku == "" {
		return Product{}, false
	}
	for _, p := range Products {
		if p.AppleProductID == sku {
			return p, true
		}
	}
	return Product{}, false
}

// AppleSKU maps one of our product ids to its App Store SKU, "" when unmapped.
func AppleSKU(productID string) string {
	if p, ok := Lookup(productID); ok {
		return p.AppleProductID
	}
	return ""
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
	case ProviderYooKassa:
		return ProviderYooKassa
	case ProviderCloudPayments:
		return ProviderCloudPayments
	case ProviderApple, "ios":
		return ProviderApple
	case ProviderPromo:
		return ProviderPromo
	case ProviderDev:
		return ProviderDev
	default:
		return ProviderAdmin
	}
}

// NormalizeSource maps a client-reported purchase source. Only providers a
// client may legitimately self-report appear here: ProviderCloudPayments and
// ProviderApple are deliberately absent, because those entitlements are granted
// solely from a server-verified webhook or signed transaction. Anything
// unrecognised collapses to ProviderRuStore.
func NormalizeSource(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "restore":
		return KindRestore
	case ProviderDev:
		return ProviderDev
	case ProviderYooKassa:
		return ProviderYooKassa
	default:
		return ProviderRuStore
	}
}

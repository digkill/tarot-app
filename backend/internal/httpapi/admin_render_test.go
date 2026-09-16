package httpapi

import (
	"strings"
	"testing"
	"time"

	"github.com/digkill/tarot-app/backend/internal/appstore"
	"github.com/digkill/tarot-app/backend/internal/billing"
	"github.com/digkill/tarot-app/backend/internal/storage"
)

// Templates are parsed at init, but a wrong field name or a missing template
// function only surfaces when a page is actually rendered.
func TestAdminUserTemplateRendersAppleSubscriptions(t *testing.T) {
	expires := time.Now().Add(30 * 24 * time.Hour)
	status := appstore.SubStatusActive
	autoRenew := 1

	var out strings.Builder
	data := struct {
		adminBase
		User      *storage.User
		Readings  int
		Tx        []storage.Transaction
		Products  []billing.Product
		Decks     []storage.Deck
		AppleSubs []storage.AppleSubscription
	}{
		adminBase: adminBase{Title: "t@example.com", Nav: "users"},
		User: &storage.User{
			ID: "8f14e45f-ea1d-4b0a-9f3c-000000000001", Email: "t@example.com",
			HasPremium: true, PremiumProductID: billing.ProductYearly, PremiumSource: billing.ProviderApple,
			PremiumExpiresAt: &expires,
		},
		Products: billing.Products,
		AppleSubs: []storage.AppleSubscription{{
			OriginalTransactionID: "2000000812345678",
			ProductID:             billing.ProductYearly,
			AppleProductID:        "org.mediarise.tarot.premium.yearly",
			Environment:           appstore.EnvSandbox,
			Status:                &status,
			AutoRenewStatus:       &autoRenew,
			ExpiresAt:             &expires,
			LastNotificationType:  billing.AppleNotifDidRenew,
			UpdatedAt:             time.Now(),
		}},
	}

	if err := adminTmpl.ExecuteTemplate(&out, "user", data); err != nil {
		t.Fatalf("render user page: %v", err)
	}
	for _, want := range []string{"2000000812345678", "активна", "включено", "Sandbox", "DID_RENEW"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("rendered page is missing %q", want)
		}
	}
}

// An Apple transaction must render with the Apple-specific refund wording and
// its USD amount, not the RuStore prose or a ruble sign.
func TestAdminTransactionTemplateRendersAppleProvider(t *testing.T) {
	invoice, purchase := "2000000812345679", "2000000812345678"
	data := struct {
		adminBase
		Tx *storage.Transaction
	}{
		adminBase: adminBase{Nav: "transactions"},
		Tx: &storage.Transaction{
			ID: "3f2a", UserID: "u1", UserEmail: "t@example.com",
			ProductID: billing.ProductYearly, Kind: billing.KindSubscription,
			Status: billing.StatusPaid, Provider: billing.ProviderApple,
			ProviderInvoiceID: &invoice, ProviderPurchaseID: &purchase,
			AmountKop: 5999, Currency: billing.CurrencyUSD, Sandbox: true,
			CreatedAt: time.Now(), PaidAt: time.Now(),
		},
	}

	var out strings.Builder
	if err := adminTmpl.ExecuteTemplate(&out, "transaction", data); err != nil {
		t.Fatalf("render transaction page: %v", err)
	}
	rendered := out.String()
	if !strings.Contains(rendered, "$59.99") {
		t.Fatal("USD amount not rendered; amount_kop holds minor units of its own currency")
	}
	if !strings.Contains(rendered, "Деньги возвращает Apple") {
		t.Fatal("Apple refund wording missing")
	}
	if strings.Contains(rendered, "консоли разработчика") {
		t.Fatal("RuStore refund wording leaked onto an Apple transaction")
	}
	if !strings.Contains(rendered, purchase) {
		t.Fatal("originalTransactionId should be visible for support")
	}
}

// The provider pickers must offer every provider the CHECK constraint allows,
// otherwise an admin cannot hand-create the row.
func TestAdminProviderSelectsCoverApple(t *testing.T) {
	rendered := map[string]string{}

	var txPage strings.Builder
	if err := adminTmpl.ExecuteTemplate(&txPage, "transactions", struct {
		adminBase
		pagerView
		Tx       []storage.Transaction
		Query    string
		Status   string
		Products []billing.Product
		Decks    []storage.Deck
	}{Products: billing.Products}); err != nil {
		t.Fatalf("render transactions: %v", err)
	}
	rendered["transactions"] = txPage.String()

	var userPage strings.Builder
	if err := adminTmpl.ExecuteTemplate(&userPage, "user", struct {
		adminBase
		User      *storage.User
		Readings  int
		Tx        []storage.Transaction
		Products  []billing.Product
		Decks     []storage.Deck
		AppleSubs []storage.AppleSubscription
	}{
		User:     &storage.User{ID: "u1", Email: "t@example.com"},
		Products: billing.Products,
	}); err != nil {
		t.Fatalf("render user: %v", err)
	}
	rendered["user"] = userPage.String()

	for page, html := range rendered {
		for _, provider := range []string{billing.ProviderApple, billing.ProviderCloudPayments} {
			if !strings.Contains(html, `value="`+provider+`"`) {
				t.Fatalf("%s page has no %s option in its provider select", page, provider)
			}
		}
	}
}

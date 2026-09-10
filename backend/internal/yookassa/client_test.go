package yookassa

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFormatRUB(t *testing.T) {
	if got := FormatRUB(59900); got != "599.00" {
		t.Fatalf("got %s", got)
	}
	if got := FormatRUB(4990); got != "49.90" {
		t.Fatalf("got %s", got)
	}
}

func TestCreateRedirectPaymentIncludesReceipt(t *testing.T) {
	for _, tc := range []struct {
		name    string
		vatCode int
		wantVAT int
	}{
		{name: "default VAT", vatCode: 0, wantVAT: 1},
		{name: "configured VAT", vatCode: 2, wantVAT: 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || r.URL.Path != "/payments" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
				}
				if shop, secret, ok := r.BasicAuth(); !ok || shop != "shop" || secret != "secret" {
					t.Error("missing shop authentication")
				}
				if r.Header.Get("Idempotence-Key") != "transaction-id" {
					t.Error("missing transaction idempotence key")
				}
				var body createPaymentRequest
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				if body.Receipt == nil || body.Receipt.Customer == nil || len(body.Receipt.Items) != 1 {
					t.Error("payment must include a receipt with customer and one item")
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				if body.Receipt.Customer.Email != "buyer@example.com" {
					t.Error("receipt must contain the buyer's email")
				}
				item := body.Receipt.Items[0]
				if body.Amount != (Amount{Value: "599.00", Currency: "RUB"}) || item.Amount != body.Amount || item.Quantity != "1" {
					t.Errorf("receipt total must match payment: amount=%+v item=%+v", body.Amount, item)
				}
				if item.VatCode != tc.wantVAT || item.PaymentSubject != "service" || item.PaymentMode != "full_payment" || item.Description != "Tarot Premium" {
					t.Errorf("unexpected receipt item: %+v", item)
				}
				if !body.Capture || body.Confirmation.Type != "redirect" || body.Confirmation.ReturnURL != "https://example.com/pay/return" || body.Metadata["transaction_id"] != "transaction-id" {
					t.Errorf("unexpected payment settings: %+v", body)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"id":"payment-id","status":"pending","confirmation":{"type":"redirect","confirmation_url":"https://example.com/checkout"}}`))
			}))
			defer server.Close()
			client := New("shop", "secret", tc.vatCode)
			client.base = server.URL
			pay, err := client.CreateRedirectPayment("transaction-id", "https://example.com/pay/return", "Tarot Premium", 59900, map[string]string{"transaction_id": "transaction-id"}, "buyer@example.com")
			if err != nil {
				t.Fatal(err)
			}
			if pay.ID != "payment-id" || pay.Confirmation == nil || pay.Confirmation.ConfirmationURL != "https://example.com/checkout" {
				t.Fatalf("unexpected payment response: %+v", pay)
			}
		})
	}
}

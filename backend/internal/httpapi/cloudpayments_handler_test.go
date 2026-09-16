package httpapi

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/digkill/tarot-app/backend/internal/cloudpayments"
	"github.com/digkill/tarot-app/backend/internal/config"
)

func TestParseCloudPaymentsNotificationForm(t *testing.T) {
	body := []byte("TransactionId=42&Amount=7.99&Currency=USD&InvoiceId=tx-1&Status=Completed&Email=a%40b.c")
	got, err := parseCloudPaymentsNotification("application/x-www-form-urlencoded; charset=utf-8", body)
	if err != nil {
		t.Fatal(err)
	}
	for k, want := range map[string]string{
		"TransactionId": "42", "Amount": "7.99", "Currency": "USD",
		"InvoiceId": "tx-1", "Status": "Completed", "Email": "a@b.c",
	} {
		if got[k] != want {
			t.Fatalf("%s = %q, want %q", k, got[k], want)
		}
	}
}

func TestParseCloudPaymentsNotificationJSON(t *testing.T) {
	body := []byte(`{"TransactionId":3402523,"Amount":59.99,"Currency":"USD","InvoiceId":"tx-2","Status":"Completed","Data":{"user_id":"u"},"Name":null}`)
	got, err := parseCloudPaymentsNotification("application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	if got["TransactionId"] != "3402523" || got["Amount"] != "59.99" || got["InvoiceId"] != "tx-2" {
		t.Fatalf("got %v", got)
	}
	if _, present := got["Name"]; present {
		t.Fatal("null fields should be dropped")
	}
}

func TestCloudPaymentsPayRejectsBadSignature(t *testing.T) {
	// txns is nil: a forged notification must be rejected before any lookup.
	h := &Handler{cp: cloudpayments.New("pk_test", "secret_test")}
	body := "TransactionId=42&Amount=0.01&Currency=USD&InvoiceId=tx-1&Status=Completed"
	req := httptest.NewRequest(http.MethodPost, "/api/v1/billing/cloudpayments/pay", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Content-HMAC", base64.StdEncoding.EncodeToString([]byte("forged")))
	rec := httptest.NewRecorder()

	h.CloudPaymentsPay(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status %d, want 401", rec.Code)
	}
}

func TestCloudPaymentsPayAcksSignedNotificationWithoutInvoice(t *testing.T) {
	h := &Handler{cp: cloudpayments.New("pk_test", "secret_test")}
	body := "TransactionId=42&Amount=7.99&Currency=USD&Status=Completed"
	mac := hmac.New(sha256.New, []byte("secret_test"))
	mac.Write([]byte(body))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/billing/cloudpayments/pay", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Content-HMAC", base64.StdEncoding.EncodeToString(mac.Sum(nil)))
	rec := httptest.NewRecorder()

	h.CloudPaymentsPay(rec, req)

	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != `{"code":0}` {
		t.Fatalf("status %d body %q", rec.Code, rec.Body.String())
	}
}

func TestCloudPaymentsPayUnconfigured(t *testing.T) {
	h := &Handler{}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/billing/cloudpayments/pay", strings.NewReader("x=1"))
	rec := httptest.NewRecorder()
	h.CloudPaymentsPay(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status %d, want 503", rec.Code)
	}
}

func TestCreateCheckoutRejectsUnknownProvider(t *testing.T) {
	h := &Handler{cfg: &config.Config{}}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/billing/checkout",
		strings.NewReader(`{"productId":"premium_monthly","provider":"stripe"}`))
	rec := httptest.NewRecorder()
	h.CreateCheckout(rec, req)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status %d, want 422", rec.Code)
	}
}

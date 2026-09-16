package cloudpayments

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func testClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	c := New("pk_test", "secret_test")
	c.base = server.URL
	return c
}

func TestCreateOrder(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/orders/create" {
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
		if user, pass, ok := r.BasicAuth(); !ok || user != "pk_test" || pass != "secret_test" {
			t.Fatalf("bad basic auth %q %q %v", user, pass, ok)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["Amount"] != 7.99 || body["Currency"] != "USD" || body["InvoiceId"] != "tx-1" {
			t.Fatalf("unexpected body %v", body)
		}
		if body["RequireConfirmation"] != false || body["SendEmail"] != false {
			t.Fatalf("order must be single-stage without email: %v", body)
		}
		_, _ = w.Write([]byte(`{"Success":true,"Message":null,"Model":{"Id":"ord1","Url":"https://orders.cloudpayments.ru/d/ord1","Status":"Created"}}`))
	})
	order, err := c.CreateOrder(OrderParams{AmountMinor: 799, Currency: "USD", Description: "x", InvoiceID: "tx-1"})
	if err != nil {
		t.Fatal(err)
	}
	if order.ID != "ord1" || order.URL != "https://orders.cloudpayments.ru/d/ord1" {
		t.Fatalf("got %+v", order)
	}
}

func TestCreateOrderFailure(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"Success":false,"Message":"Currency is not allowed","Model":null}`))
	})
	if _, err := c.CreateOrder(OrderParams{AmountMinor: 799, Currency: "USD", Description: "x"}); err == nil {
		t.Fatal("expected error")
	}
}

func TestFindPayment(t *testing.T) {
	for _, tc := range []struct {
		name      string
		response  string
		wantFound bool
		wantState string
	}{
		{"not found", `{"Success":false,"Message":"Not found"}`, false, ""},
		{"completed", `{"Success":true,"Message":null,"Model":{"TransactionId":42,"Amount":7.99,"Currency":"USD","InvoiceId":"tx-1","Status":"Completed"}}`, true, "Completed"},
		{"declined", `{"Success":false,"Message":null,"Model":{"TransactionId":43,"InvoiceId":"tx-1","Status":"Declined"}}`, true, "Declined"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/payments/find" {
					t.Fatalf("unexpected path %s", r.URL.Path)
				}
				_, _ = w.Write([]byte(tc.response))
			})
			pay, found, err := c.FindPayment("tx-1")
			if err != nil {
				t.Fatal(err)
			}
			if found != tc.wantFound {
				t.Fatalf("found=%v want %v", found, tc.wantFound)
			}
			if found && pay.Status != tc.wantState {
				t.Fatalf("status=%s want %s", pay.Status, tc.wantState)
			}
		})
	}
}

func sign(secret string, message []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(message)
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func TestVerifySignature(t *testing.T) {
	c := New("pk_test", "secret_test")
	body := []byte("TransactionId=42&Amount=7.99&Email=a%40b.c&InvoiceId=tx-1")
	decoded := []byte("TransactionId=42&Amount=7.99&Email=a@b.c&InvoiceId=tx-1")
	contentSig := sign("secret_test", body)
	xContentSig := sign("secret_test", decoded)

	if !c.VerifySignature(body, "", contentSig) {
		t.Fatal("Content-HMAC over the raw body rejected")
	}
	if !c.VerifySignature(body, xContentSig, "") {
		t.Fatal("X-Content-HMAC over the decoded body rejected")
	}
	if c.VerifySignature(body, contentSig, "") {
		t.Fatal("X-Content-HMAC must be checked against the decoded body")
	}
	if c.VerifySignature(body, "bm9wZQ==", "bm9wZQ==") {
		t.Fatal("wrong signature accepted")
	}
	if c.VerifySignature([]byte("TransactionId=42&Amount=0.01"), xContentSig, contentSig) {
		t.Fatal("tampered body accepted")
	}
	if c.VerifySignature(body, "", "") {
		t.Fatal("missing signature accepted")
	}
	if New("pk_test", "other").VerifySignature(body, xContentSig, contentSig) {
		t.Fatal("signature from another secret accepted")
	}
}

func TestMinorUnits(t *testing.T) {
	if got := MinorUnits(7.99); got != 799 {
		t.Fatalf("got %d", got)
	}
	if got := MinorUnits(59.99); got != 5999 {
		t.Fatalf("got %d", got)
	}
}

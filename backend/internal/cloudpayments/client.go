package cloudpayments

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const apiBase = "https://api.cloudpayments.ru"

type Client struct {
	publicID string
	secret   string
	http     *http.Client
	base     string
}

func New(publicID, apiSecret string) *Client {
	publicID = strings.TrimSpace(publicID)
	apiSecret = strings.TrimSpace(apiSecret)
	if publicID == "" || apiSecret == "" {
		return nil
	}
	return &Client{
		publicID: publicID,
		secret:   apiSecret,
		http:     &http.Client{Timeout: 25 * time.Second},
		base:     apiBase,
	}
}

func (c *Client) Enabled() bool {
	return c != nil && c.publicID != "" && c.secret != ""
}

// Amount converts minor units (cents/kopecks) to the decimal CloudPayments
// expects.
func Amount(minor int) float64 {
	return float64(minor) / 100
}

// MinorUnits converts a CloudPayments decimal amount back to minor units.
func MinorUnits(amount float64) int {
	return int(math.Round(amount * 100))
}

type OrderParams struct {
	AmountMinor        int
	Currency           string
	Description        string
	Email              string
	InvoiceID          string
	AccountID          string
	CultureName        string
	SuccessRedirectURL string
	FailRedirectURL    string
	JSONData           map[string]string
}

type Order struct {
	ID     string `json:"Id"`
	URL    string `json:"Url"`
	Status string `json:"Status"`
}

type orderRequest struct {
	Amount              float64           `json:"Amount"`
	Currency            string            `json:"Currency,omitempty"`
	Description         string            `json:"Description"`
	Email               string            `json:"Email,omitempty"`
	RequireConfirmation bool              `json:"RequireConfirmation"`
	SendEmail           bool              `json:"SendEmail"`
	InvoiceID           string            `json:"InvoiceId,omitempty"`
	AccountID           string            `json:"AccountId,omitempty"`
	CultureName         string            `json:"CultureName,omitempty"`
	SuccessRedirectURL  string            `json:"SuccessRedirectUrl,omitempty"`
	FailRedirectURL     string            `json:"FailRedirectUrl,omitempty"`
	JSONData            map[string]string `json:"JsonData,omitempty"`
}

type envelope[T any] struct {
	Success bool    `json:"Success"`
	Message *string `json:"Message"`
	Model   *T      `json:"Model"`
}

// CreateOrder creates a hosted payment page ("счёт") and returns its URL.
// Payments are single-stage, and CloudPayments does not email the link —
// the app opens it directly.
func (c *Client) CreateOrder(p OrderParams) (*Order, error) {
	if !c.Enabled() {
		return nil, fmt.Errorf("cloudpayments is not configured")
	}
	body := orderRequest{
		Amount:             Amount(p.AmountMinor),
		Currency:           p.Currency,
		Description:        p.Description,
		Email:              p.Email,
		InvoiceID:          p.InvoiceID,
		AccountID:          p.AccountID,
		CultureName:        p.CultureName,
		SuccessRedirectURL: p.SuccessRedirectURL,
		FailRedirectURL:    p.FailRedirectURL,
		JSONData:           p.JSONData,
	}
	var out envelope[Order]
	if err := c.post("/orders/create", body, &out); err != nil {
		return nil, fmt.Errorf("cloudpayments create order: %w", err)
	}
	if !out.Success || out.Model == nil || out.Model.URL == "" {
		return nil, fmt.Errorf("cloudpayments create order: %s", message(out.Message))
	}
	return out.Model, nil
}

type Payment struct {
	TransactionID int64   `json:"TransactionId"`
	Amount        float64 `json:"Amount"`
	Currency      string  `json:"Currency"`
	InvoiceID     string  `json:"InvoiceId"`
	AccountID     string  `json:"AccountId"`
	Status        string  `json:"Status"`
	TestMode      bool    `json:"TestMode"`
}

// FindPayment looks up the latest payment for an invoice. found is false when
// CloudPayments has no payment for it yet (the user hasn't paid).
func (c *Client) FindPayment(invoiceID string) (pay *Payment, found bool, err error) {
	if !c.Enabled() {
		return nil, false, fmt.Errorf("cloudpayments is not configured")
	}
	invoiceID = strings.TrimSpace(invoiceID)
	if invoiceID == "" {
		return nil, false, fmt.Errorf("empty invoice id")
	}
	var out envelope[Payment]
	if err = c.post("/payments/find", map[string]string{"InvoiceId": invoiceID}, &out); err != nil {
		return nil, false, fmt.Errorf("cloudpayments find payment: %w", err)
	}
	if out.Model == nil {
		// Declined payments come back with Success=false but a Model; only a
		// missing Model means there is nothing to reconcile.
		return nil, false, nil
	}
	return out.Model, true, nil
}

// VerifySignature checks a notification's HMAC-SHA256 (keyed with the API
// secret). CloudPayments sends two digests: X-Content-HMAC over the
// URL-decoded body and Content-HMAC over the body as sent. Either one
// matching its own variant of the body proves the notification is genuine.
func (c *Client) VerifySignature(body []byte, xContentHMAC, contentHMAC string) bool {
	if !c.Enabled() {
		return false
	}
	if matchesHMAC(c.secret, body, contentHMAC) {
		return true
	}
	decoded, err := url.QueryUnescape(string(body))
	if err != nil {
		return false
	}
	return matchesHMAC(c.secret, []byte(decoded), xContentHMAC)
}

func matchesHMAC(secret string, message []byte, header string) bool {
	header = strings.TrimSpace(header)
	if header == "" {
		return false
	}
	got, err := base64.StdEncoding.DecodeString(header)
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(message)
	return hmac.Equal(got, mac.Sum(nil))
}

func (c *Client) post(path string, body, out any) error {
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, c.base+path, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.SetBasicAuth(c.publicID, c.secret)
	req.Header.Set("Content-Type", "application/json")
	res, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	payload, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 300 {
		return fmt.Errorf("%s: %s", res.Status, strings.TrimSpace(string(payload)))
	}
	return json.Unmarshal(payload, out)
}

func message(m *string) string {
	if m == nil || *m == "" {
		return "unknown error"
	}
	return *m
}

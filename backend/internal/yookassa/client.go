package yookassa

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const apiBase = "https://api.yookassa.ru/v3"

type Client struct {
	shopID string
	secret string
	http   *http.Client
	base   string
}

func New(shopID, secret string) *Client {
	shopID = strings.TrimSpace(shopID)
	secret = strings.TrimSpace(secret)
	if shopID == "" || secret == "" {
		return nil
	}
	return &Client{
		shopID: shopID,
		secret: secret,
		http:   &http.Client{Timeout: 25 * time.Second},
		base:   apiBase,
	}
}

func (c *Client) Enabled() bool {
	return c != nil && c.shopID != "" && c.secret != ""
}

type Amount struct {
	Value    string `json:"value"`
	Currency string `json:"currency"`
}

type Confirmation struct {
	Type            string `json:"type"`
	ReturnURL       string `json:"return_url,omitempty"`
	ConfirmationURL string `json:"confirmation_url,omitempty"`
}

type Payment struct {
	ID           string            `json:"id"`
	Status       string            `json:"status"`
	Paid         bool              `json:"paid"`
	Amount       Amount            `json:"amount"`
	Description  string            `json:"description,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	Confirmation *Confirmation     `json:"confirmation,omitempty"`
	Refundable   bool              `json:"refundable"`
}

type createPaymentRequest struct {
	Amount       Amount            `json:"amount"`
	Capture      bool              `json:"capture"`
	Confirmation Confirmation      `json:"confirmation"`
	Description  string            `json:"description"`
	Metadata     map[string]string `json:"metadata"`
}

func FormatRUB(kop int) string {
	if kop < 0 {
		kop = 0
	}
	return fmt.Sprintf("%d.%02d", kop/100, kop%100)
}

func (c *Client) CreateRedirectPayment(idempotenceKey, returnURL, description string, kop int, metadata map[string]string) (*Payment, error) {
	if !c.Enabled() {
		return nil, fmt.Errorf("yookassa is not configured")
	}
	body := createPaymentRequest{
		Amount:  Amount{Value: FormatRUB(kop), Currency: "RUB"},
		Capture: true,
		Confirmation: Confirmation{
			Type:      "redirect",
			ReturnURL: returnURL,
		},
		Description: description,
		Metadata:    metadata,
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, c.base+"/payments", bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	c.auth(req)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotence-Key", idempotenceKey)

	res, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	payload, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 300 {
		return nil, fmt.Errorf("yookassa create payment: %s: %s", res.Status, strings.TrimSpace(string(payload)))
	}
	var pay Payment
	if err = json.Unmarshal(payload, &pay); err != nil {
		return nil, err
	}
	return &pay, nil
}

func (c *Client) GetPayment(id string) (*Payment, error) {
	if !c.Enabled() {
		return nil, fmt.Errorf("yookassa is not configured")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("empty payment id")
	}
	req, err := http.NewRequest(http.MethodGet, c.base+"/payments/"+id, nil)
	if err != nil {
		return nil, err
	}
	c.auth(req)
	res, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	payload, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 300 {
		return nil, fmt.Errorf("yookassa get payment: %s: %s", res.Status, strings.TrimSpace(string(payload)))
	}
	var pay Payment
	if err = json.Unmarshal(payload, &pay); err != nil {
		return nil, err
	}
	return &pay, nil
}

func (c *Client) auth(req *http.Request) {
	token := base64.StdEncoding.EncodeToString([]byte(c.shopID + ":" + c.secret))
	req.Header.Set("Authorization", "Basic "+token)
}

type Notification struct {
	Type   string  `json:"type"`
	Event  string  `json:"event"`
	Object Payment `json:"object"`
}

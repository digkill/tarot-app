package appstore

import (
	"context"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	baseProd    = "https://api.storekit.itunes.apple.com"
	baseSandbox = "https://api.storekit-sandbox.itunes.apple.com"

	developerTokenTTL = 20 * time.Minute
	requestTimeout    = 25 * time.Second
)

type Config struct {
	// BundleID is required; without it New returns nil and the provider is off.
	BundleID string
	// IssuerID, KeyID and PrivateKey come from an App Store Connect In-App
	// Purchase key. Without them signature verification still works but the
	// App Store Server API does not.
	IssuerID   string
	KeyID      string
	PrivateKey *ecdsa.PrivateKey
	// AppAppleID, when non-zero, is checked against notification payloads.
	AppAppleID int64
	// AllowSandbox must stay true for App Review and TestFlight, which both
	// report Sandbox transactions against a production build.
	AllowSandbox bool
	// RootCA replaces the embedded Apple Root CA G3. Tests inject their own.
	RootCA *x509.Certificate

	// Test seams.
	BaseProd    string
	BaseSandbox string
	Now         func() time.Time
}

type Client struct {
	bundleID     string
	issuerID     string
	keyID        string
	key          *ecdsa.PrivateKey
	appAppleID   int64
	allowSandbox bool

	rootCA   *x509.Certificate
	rootOnce sync.Once
	rootVal  *x509.Certificate
	rootErr  error

	baseProd    string
	baseSandbox string
	now         func() time.Time
	http        *http.Client

	tokenMu  sync.Mutex
	token    string
	tokenExp time.Time
}

// New returns nil when the bundle id is missing, matching the yookassa and
// cloudpayments clients: an unconfigured provider is simply absent.
func New(cfg Config) *Client {
	bundleID := strings.TrimSpace(cfg.BundleID)
	if bundleID == "" {
		return nil
	}
	c := &Client{
		bundleID:     bundleID,
		issuerID:     strings.TrimSpace(cfg.IssuerID),
		keyID:        strings.TrimSpace(cfg.KeyID),
		key:          cfg.PrivateKey,
		appAppleID:   cfg.AppAppleID,
		allowSandbox: cfg.AllowSandbox,
		rootCA:       cfg.RootCA,
		baseProd:     strings.TrimRight(orDefault(cfg.BaseProd, baseProd), "/"),
		baseSandbox:  strings.TrimRight(orDefault(cfg.BaseSandbox, baseSandbox), "/"),
		now:          cfg.Now,
		http:         &http.Client{Timeout: requestTimeout},
	}
	if c.now == nil {
		c.now = time.Now
	}
	return c
}

// Enabled reports whether signed payloads can be verified. This is all the
// notification endpoint needs: the trust anchor is embedded, so no Apple
// credentials are involved.
func (c *Client) Enabled() bool {
	return c != nil && c.bundleID != ""
}

// ServerAPIEnabled additionally requires the In-App Purchase key, which is what
// lets us ask Apple for the authoritative current state of a subscription.
func (c *Client) ServerAPIEnabled() bool {
	return c.Enabled() && c.issuerID != "" && c.keyID != "" && c.key != nil
}

func (c *Client) BundleID() string {
	if c == nil {
		return ""
	}
	return c.bundleID
}

func (c *Client) AllowsSandbox() bool {
	return c != nil && c.allowSandbox
}

func (c *Client) root() (*x509.Certificate, error) {
	c.rootOnce.Do(func() {
		if c.rootCA != nil {
			c.rootVal = c.rootCA
			return
		}
		c.rootVal, c.rootErr = EmbeddedRoot()
	})
	return c.rootVal, c.rootErr
}

// developerToken returns a cached ES256 token for the App Store Server API.
func (c *Client) developerToken() (string, error) {
	if !c.ServerAPIEnabled() {
		return "", ErrNotConfigured
	}
	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()

	now := c.now()
	if c.token != "" && now.Before(c.tokenExp.Add(-time.Minute)) {
		return c.token, nil
	}

	exp := now.Add(developerTokenTTL)
	token := jwt.NewWithClaims(jwt.SigningMethodES256, jwt.MapClaims{
		"iss": c.issuerID,
		"iat": now.Unix(),
		"exp": exp.Unix(),
		"aud": "appstoreconnect-v1",
		"bid": c.bundleID,
	})
	token.Header["kid"] = c.keyID

	signed, err := token.SignedString(c.key)
	if err != nil {
		return "", fmt.Errorf("sign developer token: %w", err)
	}
	c.token, c.tokenExp = signed, exp
	return signed, nil
}

// SubscriptionStatusResponse is Apple's StatusResponse.
type SubscriptionStatusResponse struct {
	Environment string                    `json:"environment"`
	BundleID    string                    `json:"bundleId"`
	AppAppleID  int64                     `json:"appAppleId"`
	Data        []SubscriptionGroupStatus `json:"data"`
}

type SubscriptionGroupStatus struct {
	SubscriptionGroupIdentifier string            `json:"subscriptionGroupIdentifier"`
	LastTransactions            []LastTransaction `json:"lastTransactions"`
}

type LastTransaction struct {
	OriginalTransactionID string `json:"originalTransactionId"`
	Status                int    `json:"status"`
	SignedTransactionInfo string `json:"signedTransactionInfo"`
	SignedRenewalInfo     string `json:"signedRenewalInfo"`

	// Populated after the signed payloads above have been verified. The plain
	// JSON around them is never trusted for product id or expiry.
	Transaction *Transaction `json:"-"`
	Renewal     *RenewalInfo `json:"-"`
}

// Latest returns the most relevant transaction Apple knows about for the
// subscription, preferring an active one.
func (r *SubscriptionStatusResponse) Latest() *LastTransaction {
	var fallback *LastTransaction
	for i := range r.Data {
		for j := range r.Data[i].LastTransactions {
			lt := &r.Data[i].LastTransactions[j]
			if lt.Status == SubStatusActive || lt.Status == SubStatusGracePeriod {
				return lt
			}
			if fallback == nil {
				fallback = lt
			}
		}
	}
	return fallback
}

// GetSubscriptionStatuses asks Apple for the authoritative state of a
// subscription. An empty env tries production first and falls back to sandbox,
// so real users pay only one round trip.
func (c *Client) GetSubscriptionStatuses(ctx context.Context, originalTransactionID, env string) (*SubscriptionStatusResponse, error) {
	originalTransactionID = strings.TrimSpace(originalTransactionID)
	if originalTransactionID == "" {
		return nil, ErrNotFound
	}
	path := "/inApps/v1/subscriptions/" + originalTransactionID

	var out SubscriptionStatusResponse
	if err := c.getJSON(ctx, env, path, &out); err != nil {
		return nil, err
	}
	for i := range out.Data {
		for j := range out.Data[i].LastTransactions {
			lt := &out.Data[i].LastTransactions[j]
			if s := strings.TrimSpace(lt.SignedTransactionInfo); s != "" {
				tx, err := c.VerifyTransaction(s)
				if err != nil {
					return nil, fmt.Errorf("verify subscription transaction: %w", err)
				}
				lt.Transaction = tx
			}
			if s := strings.TrimSpace(lt.SignedRenewalInfo); s != "" {
				info, err := c.VerifyRenewalInfo(s)
				if err != nil {
					return nil, fmt.Errorf("verify renewal info: %w", err)
				}
				lt.Renewal = info
			}
		}
	}
	return &out, nil
}

// GetTransactionInfo fetches and verifies a single transaction, which is how a
// non-consumable (lifetime premium, a deck) is re-checked for revocation.
func (c *Client) GetTransactionInfo(ctx context.Context, transactionID, env string) (*Transaction, error) {
	transactionID = strings.TrimSpace(transactionID)
	if transactionID == "" {
		return nil, ErrNotFound
	}
	var body struct {
		SignedTransactionInfo string `json:"signedTransactionInfo"`
	}
	if err := c.getJSON(ctx, env, "/inApps/v1/transactions/"+transactionID, &body); err != nil {
		return nil, err
	}
	if strings.TrimSpace(body.SignedTransactionInfo) == "" {
		return nil, ErrNotFound
	}
	return c.VerifyTransaction(body.SignedTransactionInfo)
}

// RequestTestNotification asks Apple to send a TEST notification to our
// configured webhook. Operational tool: it is the only way to prove the
// notification URL and our signature verification agree, without a purchase.
func (c *Client) RequestTestNotification(ctx context.Context, env string) (string, error) {
	var body struct {
		TestNotificationToken string `json:"testNotificationToken"`
	}
	if err := c.doJSON(ctx, http.MethodPost, envBase(c, env), "/inApps/v1/notifications/test", &body); err != nil {
		return "", err
	}
	return body.TestNotificationToken, nil
}

func (c *Client) getJSON(ctx context.Context, env, path string, out any) error {
	if !c.ServerAPIEnabled() {
		return ErrNotConfigured
	}
	if strings.TrimSpace(env) != "" {
		return c.doJSON(ctx, http.MethodGet, envBase(c, env), path, out)
	}
	// Environment unknown: production first, then sandbox. Real users then cost
	// one round trip and sandbox users two.
	err := c.doJSON(ctx, http.MethodGet, c.baseProd, path, out)
	if err == nil {
		return nil
	}
	if !c.allowSandbox {
		return err
	}
	// Fall back on 404 (Apple has no such transaction in production) and also on
	// 401: until an app has live in-app purchases, the production host rejects
	// an otherwise valid developer token, while the very same token is accepted
	// by sandbox. Without this, every sandbox purchase — App Review's included —
	// would fail. A genuinely bad key still surfaces as ErrNotConfigured,
	// because sandbox rejects it too.
	if !errors.Is(err, ErrNotFound) && !errors.Is(err, ErrNotConfigured) {
		return err
	}
	return c.doJSON(ctx, http.MethodGet, c.baseSandbox, path, out)
}

func (c *Client) doJSON(ctx context.Context, method, base, path string, out any) error {
	if !c.ServerAPIEnabled() {
		return ErrNotConfigured
	}
	token, err := c.developerToken()
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, method, base+path, nil)
	if err != nil {
		return fmt.Errorf("build app store request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrProviderUnavailable, err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(io.LimitReader(res.Body, 4<<20))
	if err != nil {
		return fmt.Errorf("%w: read body: %v", ErrProviderUnavailable, err)
	}

	switch {
	case res.StatusCode == http.StatusNotFound:
		return fmt.Errorf("%w: %s", ErrNotFound, apiErrorText(body))
	case res.StatusCode == http.StatusUnauthorized:
		// Bad or expired key material; retrying will not help.
		return fmt.Errorf("%w: app store rejected the developer token: %s", ErrNotConfigured, apiErrorText(body))
	case res.StatusCode >= 400:
		return fmt.Errorf("%w: http %d: %s", ErrProviderUnavailable, res.StatusCode, apiErrorText(body))
	}

	if out == nil {
		return nil
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("%w: decode response: %v", ErrProviderUnavailable, err)
	}
	return nil
}

func apiErrorText(body []byte) string {
	var e struct {
		ErrorCode    int64  `json:"errorCode"`
		ErrorMessage string `json:"errorMessage"`
	}
	if err := json.Unmarshal(body, &e); err == nil && e.ErrorCode != 0 {
		return fmt.Sprintf("errorCode=%d %s", e.ErrorCode, e.ErrorMessage)
	}
	if len(body) > 300 {
		body = body[:300]
	}
	return string(body)
}

func envBase(c *Client, env string) string {
	if strings.EqualFold(strings.TrimSpace(env), EnvSandbox) {
		return c.baseSandbox
	}
	return c.baseProd
}

func orDefault(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

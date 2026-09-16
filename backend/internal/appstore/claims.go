package appstore

import (
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Environments as Apple reports them per transaction.
const (
	EnvProduction = "Production"
	EnvSandbox    = "Sandbox"
)

// Transaction types (the `type` claim).
const (
	TypeAutoRenewable = "Auto-Renewable Subscription"
	TypeNonConsumable = "Non-Consumable"
	TypeConsumable    = "Consumable"
	TypeNonRenewing   = "Non-Renewing Subscription"
)

// Ownership types (the `inAppOwnershipType` claim).
const (
	OwnershipPurchased    = "PURCHASED"
	OwnershipFamilyShared = "FAMILY_SHARED"
)

// Subscription statuses returned by the App Store Server API.
const (
	SubStatusActive       = 1
	SubStatusExpired      = 2
	SubStatusBillingRetry = 3
	SubStatusGracePeriod  = 4
	SubStatusRevoked      = 5
)

// stdClaimsShim satisfies jwt.Claims for payloads that carry none of the
// registered JWT claims. Apple's payloads have no exp/iat/nbf/iss/sub/aud at
// all, so embedding jwt.RegisteredClaims would suggest fields that never exist.
// Returning nil from each getter tells the validator "not present", which it
// accepts because none of them are required by default.
type stdClaimsShim struct{}

func (stdClaimsShim) GetExpirationTime() (*jwt.NumericDate, error) { return nil, nil }
func (stdClaimsShim) GetIssuedAt() (*jwt.NumericDate, error)       { return nil, nil }
func (stdClaimsShim) GetNotBefore() (*jwt.NumericDate, error)      { return nil, nil }
func (stdClaimsShim) GetIssuer() (string, error)                   { return "", nil }
func (stdClaimsShim) GetSubject() (string, error)                  { return "", nil }
func (stdClaimsShim) GetAudience() (jwt.ClaimStrings, error)       { return nil, nil }

// Transaction is Apple's JWSTransactionDecodedPayload. All date fields are
// milliseconds since the epoch.
type Transaction struct {
	TransactionID               string `json:"transactionId"`
	OriginalTransactionID       string `json:"originalTransactionId"`
	WebOrderLineItemID          string `json:"webOrderLineItemId"`
	BundleID                    string `json:"bundleId"`
	ProductID                   string `json:"productId"`
	SubscriptionGroupIdentifier string `json:"subscriptionGroupIdentifier"`
	PurchaseDate                int64  `json:"purchaseDate"`
	OriginalPurchaseDate        int64  `json:"originalPurchaseDate"`
	ExpiresDate                 int64  `json:"expiresDate"`
	Quantity                    int    `json:"quantity"`
	Type                        string `json:"type"`
	AppAccountToken             string `json:"appAccountToken"`
	InAppOwnershipType          string `json:"inAppOwnershipType"`
	SignedDate                  int64  `json:"signedDate"`
	Environment                 string `json:"environment"`
	RevocationDate              int64  `json:"revocationDate"`
	RevocationReason            *int   `json:"revocationReason"`
	OfferType                   int    `json:"offerType"`
	OfferIdentifier             string `json:"offerIdentifier"`
	OfferDiscountType           string `json:"offerDiscountType"`
	IsUpgraded                  bool   `json:"isUpgraded"`
	Storefront                  string `json:"storefront"`
	StorefrontID                string `json:"storefrontId"`
	TransactionReason           string `json:"transactionReason"`
	// Price is in milliunits of Currency: 7990 means 7.99. Absent on older
	// API versions and on some notification payloads.
	Price            int64  `json:"price"`
	Currency         string `json:"currency"`
	AppTransactionID string `json:"appTransactionId"`

	stdClaimsShim
}

// ExpiresAt is the subscription expiry, nil for non-consumables.
func (t *Transaction) ExpiresAt() *time.Time {
	return msToTime(t.ExpiresDate)
}

// RevokedAt is set when Apple has refunded or revoked the purchase.
func (t *Transaction) RevokedAt() *time.Time {
	return msToTime(t.RevocationDate)
}

func (t *Transaction) PurchasedAt() *time.Time {
	return msToTime(t.PurchaseDate)
}

func (t *Transaction) IsSandbox() bool {
	return strings.EqualFold(strings.TrimSpace(t.Environment), EnvSandbox)
}

func (t *Transaction) IsFamilyShared() bool {
	return strings.EqualFold(strings.TrimSpace(t.InAppOwnershipType), OwnershipFamilyShared)
}

func (t *Transaction) IsRevoked() bool {
	return t.RevocationDate > 0
}

// AmountMinor converts Apple's milliunit price into minor currency units, the
// form our transactions table stores. ok is false when Apple sent no price, in
// which case the caller falls back to the catalog price.
func (t *Transaction) AmountMinor() (int, string, bool) {
	currency := strings.ToUpper(strings.TrimSpace(t.Currency))
	if t.Price <= 0 || currency == "" {
		return 0, "", false
	}
	return int(t.Price / 10), currency, true
}

// RenewalInfo is Apple's JWSRenewalInfoDecodedPayload.
type RenewalInfo struct {
	OriginalTransactionID       string `json:"originalTransactionId"`
	ProductID                   string `json:"productId"`
	AutoRenewProductID          string `json:"autoRenewProductId"`
	AutoRenewStatus             int    `json:"autoRenewStatus"`
	ExpirationIntent            int    `json:"expirationIntent"`
	GracePeriodExpiresDate      int64  `json:"gracePeriodExpiresDate"`
	IsInBillingRetryPeriod      bool   `json:"isInBillingRetryPeriod"`
	RenewalDate                 int64  `json:"renewalDate"`
	RecentSubscriptionStartDate int64  `json:"recentSubscriptionStartDate"`
	Environment                 string `json:"environment"`
	SignedDate                  int64  `json:"signedDate"`
	OfferType                   int    `json:"offerType"`
	OfferIdentifier             string `json:"offerIdentifier"`
	RenewalPrice                int64  `json:"renewalPrice"`
	Currency                    string `json:"currency"`
	PriceIncreaseStatus         *int   `json:"priceIncreaseStatus"`

	stdClaimsShim
}

// GracePeriodEnd is the deadline Apple keeps serving a subscription past a
// failed renewal, nil when there is no grace period.
func (r *RenewalInfo) GracePeriodEnd() *time.Time {
	return msToTime(r.GracePeriodExpiresDate)
}

func (r *RenewalInfo) RenewsAt() *time.Time {
	return msToTime(r.RenewalDate)
}

// Notification is Apple's responseBodyV2DecodedPayload.
type Notification struct {
	NotificationType string               `json:"notificationType"`
	Subtype          string               `json:"subtype"`
	NotificationUUID string               `json:"notificationUUID"`
	Version          string               `json:"version"`
	SignedDate       int64                `json:"signedDate"`
	Data             *NotificationData    `json:"data"`
	Summary          *NotificationSummary `json:"summary"`

	// Populated by VerifyNotification after the nested JWS payloads have been
	// verified with the same chain and signature checks as the outer one.
	Transaction *Transaction `json:"-"`
	Renewal     *RenewalInfo `json:"-"`

	stdClaimsShim
}

type NotificationData struct {
	AppAppleID            int64  `json:"appAppleId"`
	BundleID              string `json:"bundleId"`
	BundleVersion         string `json:"bundleVersion"`
	Environment           string `json:"environment"`
	SignedTransactionInfo string `json:"signedTransactionInfo"`
	SignedRenewalInfo     string `json:"signedRenewalInfo"`
	Status                int    `json:"status"`
}

type NotificationSummary struct {
	RequestIdentifier      string   `json:"requestIdentifier"`
	Environment            string   `json:"environment"`
	AppAppleID             int64    `json:"appAppleId"`
	BundleID               string   `json:"bundleId"`
	ProductID              string   `json:"productId"`
	StorefrontCountryCodes []string `json:"storefrontCountryCodes"`
	FailedCount            int64    `json:"failedCount"`
	SucceededCount         int64    `json:"succeededCount"`
}

// OriginalTransactionID is the stable handle for the subscription this
// notification concerns, "" when the payload carries no transaction.
func (n *Notification) OriginalTransactionID() string {
	if n.Transaction != nil && n.Transaction.OriginalTransactionID != "" {
		return n.Transaction.OriginalTransactionID
	}
	if n.Renewal != nil {
		return n.Renewal.OriginalTransactionID
	}
	return ""
}

func (n *Notification) IsTest() bool {
	return strings.EqualFold(strings.TrimSpace(n.NotificationType), "TEST")
}

func msToTime(ms int64) *time.Time {
	if ms <= 0 {
		return nil
	}
	t := time.UnixMilli(ms).UTC()
	return &t
}

package billing

import (
	"strings"
	"time"
)

// App Store Server Notifications V2 notification types.
const (
	AppleNotifSubscribed             = "SUBSCRIBED"
	AppleNotifDidRenew               = "DID_RENEW"
	AppleNotifOfferRedeemed          = "OFFER_REDEEMED"
	AppleNotifDidChangeRenewalPref   = "DID_CHANGE_RENEWAL_PREF"
	AppleNotifDidChangeRenewalStatus = "DID_CHANGE_RENEWAL_STATUS"
	AppleNotifDidFailToRenew         = "DID_FAIL_TO_RENEW"
	AppleNotifExpired                = "EXPIRED"
	AppleNotifGracePeriodExpired     = "GRACE_PERIOD_EXPIRED"
	AppleNotifRefund                 = "REFUND"
	AppleNotifRefundDeclined         = "REFUND_DECLINED"
	AppleNotifRefundReversed         = "REFUND_REVERSED"
	AppleNotifRevoke                 = "REVOKE"
	AppleNotifRenewalExtended        = "RENEWAL_EXTENDED"
	AppleNotifRenewalExtension       = "RENEWAL_EXTENSION"
	AppleNotifConsumptionRequest     = "CONSUMPTION_REQUEST"
	AppleNotifPriceIncrease          = "PRICE_INCREASE"
	AppleNotifMetadataUpdate         = "METADATA_UPDATE"
	AppleNotifOneTimeCharge          = "ONE_TIME_CHARGE"
	AppleNotifTest                   = "TEST"
)

// App Store Server Notifications V2 subtypes.
const (
	AppleSubtypeInitialBuy      = "INITIAL_BUY"
	AppleSubtypeResubscribe     = "RESUBSCRIBE"
	AppleSubtypeGracePeriod     = "GRACE_PERIOD"
	AppleSubtypeBillingRecovery = "BILLING_RECOVERY"
	AppleSubtypeBillingRetry    = "BILLING_RETRY"
	AppleSubtypeVoluntary       = "VOLUNTARY"
	AppleSubtypeAutoRenewOff    = "AUTO_RENEW_OFF"
	AppleSubtypeAutoRenewOn     = "AUTO_RENEW_ON"
	AppleSubtypeUpgrade         = "UPGRADE"
	AppleSubtypeDowngrade       = "DOWNGRADE"
)

// AppleTxFacts is the subset of a verified Apple transaction (and its renewal
// info) that the effect table needs. It is a plain struct on purpose: it keeps
// this package free of any dependency on the Apple client, so the decision
// table below stays a pure function and is fully table-testable.
type AppleTxFacts struct {
	// ProductID is our own product id, already resolved from the Apple SKU.
	// Empty for a deck purchase or an unmapped SKU.
	ProductID string
	// ExpiresAt comes from the transaction; nil for non-consumables.
	ExpiresAt *time.Time
	// GracePeriodEnd comes from the renewal info; nil when not in a grace period.
	GracePeriodEnd *time.Time
}

// AppleEffect is what a notification should do to our own state. The caller
// still applies its own guards — CanRevokeFrom before revoking, and the
// exactly-once transaction insert — this only decides the intent.
type AppleEffect struct {
	// RecordTransaction inserts a paid transaction row for this Apple transaction.
	RecordTransaction bool
	// Kind overrides the product's own kind for the recorded row; "" means use
	// the product's kind.
	Kind string
	// Grant applies premium with ExpiresAt.
	Grant bool
	// Revoke takes premium away (subject to the caller's CanRevokeFrom check).
	Revoke bool
	// Refund marks an existing transaction row refunded.
	Refund bool
	// ExpiresAt is the expiry to grant; nil means lifetime / no expiry.
	ExpiresAt *time.Time
}

// AppleNotificationEffect maps a notification type and subtype onto our state.
// Unknown types produce a zero effect so the webhook stays forward compatible:
// it acks and logs rather than failing.
func AppleNotificationEffect(notificationType, subtype string, facts AppleTxFacts) AppleEffect {
	nt := strings.ToUpper(strings.TrimSpace(notificationType))
	st := strings.ToUpper(strings.TrimSpace(subtype))
	lifetime := strings.TrimSpace(facts.ProductID) == ProductLifetime

	switch nt {
	case AppleNotifSubscribed, AppleNotifDidRenew, AppleNotifOfferRedeemed:
		// A renewal carries a fresh transactionId, so recording it both bills
		// correctly and pushes premium_expires_at forward — which is what keeps
		// the background expiry sweep from cancelling an active subscription.
		return AppleEffect{RecordTransaction: true, Grant: true, ExpiresAt: facts.ExpiresAt}

	case AppleNotifOneTimeCharge:
		return AppleEffect{RecordTransaction: true, Grant: true, ExpiresAt: nil}

	case AppleNotifRefundReversed:
		return AppleEffect{RecordTransaction: true, Kind: KindRestore, Grant: true, ExpiresAt: facts.ExpiresAt}

	case AppleNotifDidFailToRenew:
		if st == AppleSubtypeGracePeriod {
			// Billing failed but Apple keeps serving the subscription; extend to
			// the grace deadline so we don't revoke under the user.
			return AppleEffect{Grant: true, ExpiresAt: facts.GracePeriodEnd}
		}
		// Plain billing retry: let it lapse naturally at expires_at.
		return AppleEffect{}

	case AppleNotifExpired, AppleNotifGracePeriodExpired:
		if lifetime {
			return AppleEffect{}
		}
		return AppleEffect{Revoke: true}

	case AppleNotifRefund, AppleNotifRevoke:
		// Refund() only revokes premium when no other paid premium row remains,
		// and prior renewals are paid rows, so revoke explicitly.
		return AppleEffect{Refund: true, Revoke: true}

	case AppleNotifRenewalExtended, AppleNotifRenewalExtension:
		if facts.ExpiresAt == nil {
			return AppleEffect{}
		}
		return AppleEffect{Grant: true, ExpiresAt: facts.ExpiresAt}

	case AppleNotifDidChangeRenewalPref, AppleNotifDidChangeRenewalStatus:
		// Turning auto-renew off does not end the current period, and an
		// upgrade/downgrade charge arrives separately as DID_RENEW/SUBSCRIBED.
		return AppleEffect{}

	default:
		// TEST, CONSUMPTION_REQUEST, REFUND_DECLINED, PRICE_INCREASE,
		// METADATA_UPDATE and anything Apple adds later: log and ack.
		return AppleEffect{}
	}
}

package billing

import (
	"testing"
	"time"
)

func TestAppleNotificationEffect(t *testing.T) {
	expires := time.Date(2027, 3, 1, 10, 0, 0, 0, time.UTC)
	grace := time.Date(2026, 10, 20, 10, 0, 0, 0, time.UTC)
	sub := AppleTxFacts{ProductID: ProductYearly, ExpiresAt: &expires}
	subInGrace := AppleTxFacts{ProductID: ProductMonthly, GracePeriodEnd: &grace}
	life := AppleTxFacts{ProductID: ProductLifetime}

	cases := []struct {
		name     string
		ntype    string
		subtype  string
		facts    AppleTxFacts
		want     AppleEffect
		wantExpN bool // want ExpiresAt to be nil
	}{
		{
			name:  "initial buy grants and records",
			ntype: AppleNotifSubscribed, subtype: AppleSubtypeInitialBuy, facts: sub,
			want: AppleEffect{RecordTransaction: true, Grant: true, ExpiresAt: &expires},
		},
		{
			name:  "resubscribe grants and records",
			ntype: AppleNotifSubscribed, subtype: AppleSubtypeResubscribe, facts: sub,
			want: AppleEffect{RecordTransaction: true, Grant: true, ExpiresAt: &expires},
		},
		{
			name:  "renewal pushes expiry forward",
			ntype: AppleNotifDidRenew, facts: sub,
			want: AppleEffect{RecordTransaction: true, Grant: true, ExpiresAt: &expires},
		},
		{
			name:  "renewal after billing recovery",
			ntype: AppleNotifDidRenew, subtype: AppleSubtypeBillingRecovery, facts: sub,
			want: AppleEffect{RecordTransaction: true, Grant: true, ExpiresAt: &expires},
		},
		{
			name:  "offer redeemed grants and records",
			ntype: AppleNotifOfferRedeemed, facts: sub,
			want: AppleEffect{RecordTransaction: true, Grant: true, ExpiresAt: &expires},
		},
		{
			name:  "one time charge grants without expiry",
			ntype: AppleNotifOneTimeCharge, facts: life,
			want: AppleEffect{RecordTransaction: true, Grant: true}, wantExpN: true,
		},
		{
			name:  "refund reversed re-grants as a restore row",
			ntype: AppleNotifRefundReversed, facts: sub,
			want: AppleEffect{RecordTransaction: true, Kind: KindRestore, Grant: true, ExpiresAt: &expires},
		},
		{
			name:  "grace period extends to the grace deadline",
			ntype: AppleNotifDidFailToRenew, subtype: AppleSubtypeGracePeriod, facts: subInGrace,
			want: AppleEffect{Grant: true, ExpiresAt: &grace},
		},
		{
			name:  "plain billing retry lapses naturally",
			ntype: AppleNotifDidFailToRenew, facts: sub,
			want: AppleEffect{}, wantExpN: true,
		},
		{
			name:  "expired revokes",
			ntype: AppleNotifExpired, subtype: AppleSubtypeVoluntary, facts: sub,
			want: AppleEffect{Revoke: true}, wantExpN: true,
		},
		{
			name:  "expired after billing retry revokes",
			ntype: AppleNotifExpired, subtype: AppleSubtypeBillingRetry, facts: sub,
			want: AppleEffect{Revoke: true}, wantExpN: true,
		},
		{
			name:  "grace period expired revokes",
			ntype: AppleNotifGracePeriodExpired, facts: sub,
			want: AppleEffect{Revoke: true}, wantExpN: true,
		},
		{
			name:  "expired must not revoke a lifetime purchase",
			ntype: AppleNotifExpired, facts: life,
			want: AppleEffect{}, wantExpN: true,
		},
		{
			name:  "refund refunds and revokes",
			ntype: AppleNotifRefund, facts: sub,
			want: AppleEffect{Refund: true, Revoke: true}, wantExpN: true,
		},
		{
			name:  "refund revokes lifetime too",
			ntype: AppleNotifRefund, facts: life,
			want: AppleEffect{Refund: true, Revoke: true}, wantExpN: true,
		},
		{
			name:  "revoke refunds and revokes",
			ntype: AppleNotifRevoke, facts: sub,
			want: AppleEffect{Refund: true, Revoke: true}, wantExpN: true,
		},
		{
			name:  "renewal extended refreshes expiry",
			ntype: AppleNotifRenewalExtended, facts: sub,
			want: AppleEffect{Grant: true, ExpiresAt: &expires},
		},
		{
			name:  "renewal extension without an expiry is a no-op",
			ntype: AppleNotifRenewalExtension, facts: AppleTxFacts{ProductID: ProductMonthly},
			want: AppleEffect{}, wantExpN: true,
		},
		{
			name:  "auto renew off keeps premium to the period end",
			ntype: AppleNotifDidChangeRenewalStatus, subtype: AppleSubtypeAutoRenewOff, facts: sub,
			want: AppleEffect{}, wantExpN: true,
		},
		{
			name:  "upgrade pref change waits for the charge",
			ntype: AppleNotifDidChangeRenewalPref, subtype: AppleSubtypeUpgrade, facts: sub,
			want: AppleEffect{}, wantExpN: true,
		},
		{
			name:  "test notification is a no-op",
			ntype: AppleNotifTest,
			want:  AppleEffect{}, wantExpN: true,
		},
		{
			name:  "consumption request is a no-op",
			ntype: AppleNotifConsumptionRequest, facts: sub,
			want: AppleEffect{}, wantExpN: true,
		},
		{
			name:  "refund declined is a no-op",
			ntype: AppleNotifRefundDeclined, facts: sub,
			want: AppleEffect{}, wantExpN: true,
		},
		{
			name:  "unknown future type is a no-op, not an error",
			ntype: "SOMETHING_APPLE_ADDS_IN_2028", facts: sub,
			want: AppleEffect{}, wantExpN: true,
		},
		{
			name:  "type matching is case and space insensitive",
			ntype: "  did_renew  ", facts: sub,
			want: AppleEffect{RecordTransaction: true, Grant: true, ExpiresAt: &expires},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := AppleNotificationEffect(tc.ntype, tc.subtype, tc.facts)
			if got.RecordTransaction != tc.want.RecordTransaction ||
				got.Grant != tc.want.Grant ||
				got.Revoke != tc.want.Revoke ||
				got.Refund != tc.want.Refund ||
				got.Kind != tc.want.Kind {
				t.Fatalf("got %+v, want %+v", got, tc.want)
			}
			if tc.wantExpN {
				if got.ExpiresAt != nil {
					t.Fatalf("ExpiresAt = %v, want nil", *got.ExpiresAt)
				}
				return
			}
			if got.ExpiresAt == nil {
				t.Fatalf("ExpiresAt = nil, want %v", *tc.want.ExpiresAt)
			}
			if !got.ExpiresAt.Equal(*tc.want.ExpiresAt) {
				t.Fatalf("ExpiresAt = %v, want %v", *got.ExpiresAt, *tc.want.ExpiresAt)
			}
		})
	}
}

// A grant must never be issued without knowing until when, except for the
// deliberately unbounded lifetime/one-time cases.
func TestAppleGrantAlwaysCarriesAnExpiryOrIsOneTime(t *testing.T) {
	expires := time.Now().Add(720 * time.Hour)
	for _, ntype := range []string{
		AppleNotifSubscribed, AppleNotifDidRenew, AppleNotifOfferRedeemed,
		AppleNotifRefundReversed, AppleNotifRenewalExtended,
	} {
		eff := AppleNotificationEffect(ntype, "", AppleTxFacts{ProductID: ProductMonthly, ExpiresAt: &expires})
		if !eff.Grant {
			t.Fatalf("%s should grant", ntype)
		}
		if eff.ExpiresAt == nil {
			t.Fatalf("%s granted a subscription with no expiry", ntype)
		}
	}
	eff := AppleNotificationEffect(AppleNotifOneTimeCharge, "", AppleTxFacts{ProductID: ProductLifetime})
	if !eff.Grant || eff.ExpiresAt != nil {
		t.Fatalf("one time charge should grant without expiry: %+v", eff)
	}
}

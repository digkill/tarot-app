package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/digkill/tarot-app/backend/internal/appstore"
	"github.com/digkill/tarot-app/backend/internal/billing"
	"github.com/digkill/tarot-app/backend/internal/config"
	"github.com/digkill/tarot-app/backend/internal/storage"
)

// newAppStoreClient builds the Apple client from config, resolving the optional
// root CA override. A bad override is logged and ignored rather than fatal: the
// embedded certificate is the safe default.
func newAppStoreClient(cfg *config.Config) *appstore.Client {
	c := appstore.Config{
		BundleID:     cfg.AppleBundleID,
		IssuerID:     cfg.AppleIAPIssuerID,
		KeyID:        cfg.AppleIAPKeyID,
		PrivateKey:   cfg.AppleIAPPrivateKey,
		AppAppleID:   cfg.AppleAppAppleID,
		AllowSandbox: cfg.AppleAllowSandbox,
	}
	if path := strings.TrimSpace(cfg.AppleRootCAFile); path != "" {
		root, err := appstore.LoadRootCAFile(path)
		if err != nil {
			slog.Error("apple root CA override ignored, using the embedded certificate",
				"file", path, "error", err)
		} else {
			c.RootCA = root
		}
	}
	return appstore.New(c)
}

type appleVerifyRequest struct {
	SignedTransaction     string `json:"signedTransaction"`
	OriginalTransactionID string `json:"originalTransactionId"`
	AppAccountToken       string `json:"appAccountToken"`
}

type appleNotificationRequest struct {
	SignedPayload string `json:"signedPayload"`
}

// AppleVerifyPurchase is the only way an Apple entitlement is granted from the
// client side. It doubles as restore and refresh: it is idempotent, so the iOS
// app calls it after every purchase, for every entry in currentEntitlements on
// cold start, and for everything arriving on Transaction.updates.
func (h *Handler) AppleVerifyPurchase(w http.ResponseWriter, r *http.Request) {
	if h.as == nil || !h.as.Enabled() {
		writeError(w, http.StatusServiceUnavailable, "payments_unavailable", "Apple In-App Purchase is not configured")
		return
	}
	userID := userIDFromCtx(r.Context())

	var req appleVerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}
	signed := strings.TrimSpace(req.SignedTransaction)
	originalID := strings.TrimSpace(req.OriginalTransactionID)
	if signed == "" && originalID == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "signedTransaction or originalTransactionId is required")
		return
	}

	// The signature is checked before anything else touches the database, so a
	// forged payload costs nothing. A client-supplied JWS proves provenance and
	// ownership, but it is only a snapshot: it cannot know the purchase was
	// since refunded or revoked.
	var claimed *appstore.Transaction
	if signed != "" {
		verified, err := h.as.VerifyTransaction(signed)
		if err != nil {
			writeAppleError(w, err)
			return
		}
		claimed = verified
		if originalID == "" {
			originalID = claimed.OriginalTransactionID
		}
	}

	user, err := h.users.GetByID(r.Context(), userID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			writeError(w, http.StatusUnauthorized, "unauthorized", "user not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to load user")
		return
	}

	// Ownership is settled before anything is granted.
	owner, err := h.resolveAppleOwner(r, user, claimed, originalID)
	if err != nil {
		writeAppleError(w, err)
		return
	}

	// Apple's servers are authoritative for status and expiry.
	authoritative, status, err := h.appleAuthoritativeTransaction(r, claimed, originalID)
	if err != nil {
		writeAppleError(w, err)
		return
	}
	if authoritative == nil {
		writeError(w, http.StatusUnprocessableEntity, "validation_error", "no verifiable Apple transaction")
		return
	}

	if reason := appleInactiveReason(authoritative, status); reason != "" {
		// Not an error: the client must be able to tell "you have no premium"
		// apart from "the check failed".
		h.recordAppleSubscription(r, owner, authoritative, status, "")
		h.revokeAppleEntitlement(r, owner, authoritative)
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":                    true,
			"recorded":              false,
			"hasPremium":            false,
			"reason":                reason,
			"environment":           authoritative.Environment,
			"appleTransactionId":    authoritative.TransactionID,
			"originalTransactionId": authoritative.OriginalTransactionID,
		})
		return
	}

	res := h.applyAppleTransaction(r, owner, authoritative, appleApplyOpts{
		Grant:     true,
		ExpiresAt: authoritative.ExpiresAt(),
	})
	if res.err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to record purchase")
		return
	}
	h.recordAppleSubscription(r, owner, authoritative, status, "")

	fresh, err := h.users.GetByID(r.Context(), owner)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to load user")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":                    true,
		"recorded":              res.recorded,
		"hasPremium":            fresh.HasPremium,
		"premiumSource":         fresh.PremiumSource,
		"productId":             res.productID,
		"deckSlug":              res.deckSlug,
		"expiresAt":             fresh.PremiumExpiresAt,
		"environment":           authoritative.Environment,
		"transactionId":         res.txID,
		"appleTransactionId":    authoritative.TransactionID,
		"originalTransactionId": authoritative.OriginalTransactionID,
	})
}

// AppleNotifications handles App Store Server Notifications V2. Like the other
// provider webhooks it acks anything with a valid signature — Apple retries on
// a non-2xx up to 5 times over ~3 days, and a retry of something we simply
// cannot act on would repeat forever.
func (h *Handler) AppleNotifications(w http.ResponseWriter, r *http.Request) {
	if h.as == nil || !h.as.Enabled() {
		// A 503 is correct here: Apple should retry a misconfiguration.
		writeError(w, http.StatusServiceUnavailable, "payments_unavailable", "Apple In-App Purchase is not configured")
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid notification")
		return
	}
	var req appleNotificationRequest
	if err := json.Unmarshal(body, &req); err != nil || strings.TrimSpace(req.SignedPayload) == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "signedPayload is required")
		return
	}

	note, err := h.as.VerifyNotification(req.SignedPayload)
	if err != nil {
		slog.Warn("apple notification with invalid signature", "ip", clientIP(r), "error", err)
		writeError(w, http.StatusUnauthorized, "invalid_signature", "invalid signature")
		return
	}

	// Everything past this point is a genuine Apple notification, so it is
	// always acked; failures are logged, never surfaced.
	h.processAppleNotification(r, note)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) processAppleNotification(r *http.Request, note *appstore.Notification) {
	logger := slog.With(
		"type", note.NotificationType,
		"subtype", note.Subtype,
		"uuid", note.NotificationUUID,
	)
	if note.IsTest() {
		logger.Info("apple test notification received")
		return
	}

	originalID := note.OriginalTransactionID()
	fresh, err := h.appleSubs.MarkNotification(r.Context(), note.NotificationUUID, note.NotificationType, note.Subtype, originalID)
	if err != nil {
		logger.Error("record apple notification", "error", err)
		return
	}
	if !fresh {
		logger.Info("apple notification already processed, ignoring retry")
		return
	}

	tx := note.Transaction
	if tx == nil {
		logger.Info("apple notification carries no transaction, nothing to apply")
		return
	}

	owner, err := h.appleOwnerFromTransaction(r, tx, originalID)
	if err != nil {
		// Nothing to do but record it: without an owner we cannot act, and
		// Apple must not keep retrying.
		logger.Warn("apple notification owner unresolved", "error", err, "original", originalID)
		return
	}

	product, deck := h.resolveAppleProduct(r, tx.ProductID)
	facts := billing.AppleTxFacts{ProductID: product.ID, ExpiresAt: tx.ExpiresAt()}
	if note.Renewal != nil {
		facts.GracePeriodEnd = note.Renewal.GracePeriodEnd()
	}
	effect := billing.AppleNotificationEffect(note.NotificationType, note.Subtype, facts)

	if effect.RecordTransaction || effect.Grant {
		res := h.applyAppleTransaction(r, owner, tx, appleApplyOpts{
			Record:    effect.RecordTransaction,
			Grant:     effect.Grant,
			ExpiresAt: effect.ExpiresAt,
			Kind:      effect.Kind,
			Notes:     appleNotes(tx, note),
			Product:   product,
			Deck:      deck,
		})
		if res.err != nil {
			logger.Error("apply apple notification", "error", res.err)
		}
	}
	if effect.Refund {
		h.refundAppleTransaction(r, tx, note)
	}
	if effect.Revoke {
		h.revokeAppleEntitlement(r, owner, tx)
	}

	var status *int
	if note.Data != nil && note.Data.Status != 0 {
		s := note.Data.Status
		status = &s
	}
	h.recordAppleSubscription(r, owner, tx, status, note.NotificationType+"|"+note.NotificationUUID)
	logger.Info("apple notification applied", "user", owner, "product", product.ID)
}

// ---------------------------------------------------------------------------
// ownership
// ---------------------------------------------------------------------------

var (
	errAppleFamilyShared  = errors.New("apple: family shared purchase")
	errAppleOwnerMismatch = errors.New("apple: transaction belongs to another account")
	errAppleOwnerUnknown  = errors.New("apple: no user mapped to this transaction")
)

// resolveAppleOwner settles who a client-reported transaction belongs to.
// appAccountToken is the primary mechanism — our user ids are UUIDs, which is
// exactly what that field holds — and the apple_subscriptions table is the
// backstop for purchases that carry no token.
func (h *Handler) resolveAppleOwner(r *http.Request, user *storage.User, tx *appstore.Transaction, originalID string) (string, error) {
	if tx != nil && tx.IsFamilyShared() {
		return "", errAppleFamilyShared
	}

	if tx != nil {
		if token := strings.TrimSpace(tx.AppAccountToken); token != "" {
			if !strings.EqualFold(token, user.ID) {
				slog.Warn("apple appAccountToken does not match the authenticated user",
					"token", token, "user", user.ID, "original", originalID)
				return "", errAppleOwnerMismatch
			}
			return user.ID, nil
		}
	}

	if originalID == "" {
		// Nothing to bind against; trust the authenticated session.
		return user.ID, nil
	}

	bound, err := h.appleSubs.Bind(r.Context(), storage.AppleSubscription{
		OriginalTransactionID: originalID,
		UserID:                user.ID,
		Environment:           appleEnvironment(tx),
	})
	if err != nil {
		return "", err
	}
	if !strings.EqualFold(bound.UserID, user.ID) {
		slog.Warn("apple original transaction already bound to another user",
			"original", originalID, "owner", bound.UserID, "requester", user.ID)
		return "", errAppleOwnerMismatch
	}
	return user.ID, nil
}

// appleOwnerFromTransaction resolves the owner of a notification, where there
// is no authenticated session to fall back on.
func (h *Handler) appleOwnerFromTransaction(r *http.Request, tx *appstore.Transaction, originalID string) (string, error) {
	if token := strings.TrimSpace(tx.AppAccountToken); token != "" {
		if _, err := h.users.GetByID(r.Context(), token); err == nil {
			return token, nil
		}
	}
	if originalID == "" {
		return "", errAppleOwnerUnknown
	}
	sub, err := h.appleSubs.GetByOriginalTransactionID(r.Context(), originalID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return "", errAppleOwnerUnknown
		}
		return "", err
	}
	return sub.UserID, nil
}

// ---------------------------------------------------------------------------
// applying a transaction
// ---------------------------------------------------------------------------

type appleApplyOpts struct {
	Record    bool
	Grant     bool
	ExpiresAt *time.Time
	Kind      string
	Notes     string
	Product   billing.Product
	Deck      *storage.Deck
}

type appleApplyResult struct {
	recorded  bool
	txID      string
	productID string
	deckSlug  string
	err       error
}

// applyAppleTransaction records the Apple transaction and grants what it buys.
// Idempotency rides on the existing UNIQUE (provider, provider_invoice_id)
// index: Apple's transactionId is unique per purchase and per renewal, so a
// duplicate insert means "already processed" — we refresh the entitlement and
// report success rather than erroring.
func (h *Handler) applyAppleTransaction(r *http.Request, userID string, tx *appstore.Transaction, opts appleApplyOpts) appleApplyResult {
	product, deck := opts.Product, opts.Deck
	if product.ID == "" && deck == nil {
		product, deck = h.resolveAppleProduct(r, tx.ProductID)
	}
	if product.ID == "" && deck == nil {
		slog.Error("apple purchase of an unmapped product", "sku", tx.ProductID, "tx", tx.TransactionID)
		return appleApplyResult{err: fmt.Errorf("unmapped apple sku %q", tx.ProductID)}
	}

	res := appleApplyResult{productID: product.ID}
	if deck != nil {
		res.productID = deck.ProductID()
		res.deckSlug = deck.Slug
	}

	kind := opts.Kind
	if kind == "" {
		kind = product.Kind
		if deck != nil {
			kind = billing.KindOneTime
		}
	}

	amount, currency := appleAmount(tx, product, deck)
	notes := opts.Notes
	if notes == "" {
		notes = appleNotes(tx, nil)
	}

	created, err := h.txns.Create(r.Context(), storage.CreateTransactionParams{
		UserID:             userID,
		ProductID:          res.productID,
		Kind:               kind,
		Status:             billing.StatusPaid,
		Provider:           billing.ProviderApple,
		ProviderInvoiceID:  storage.NullIfBlank(tx.TransactionID),
		ProviderPurchaseID: storage.NullIfBlank(tx.OriginalTransactionID),
		AmountKop:          amount,
		Currency:           currency,
		Sandbox:            tx.IsSandbox(),
		Notes:              truncateRunes(notes, 200),
	})
	switch {
	case err == nil:
		res.recorded = true
		res.txID = created.ID
	case errors.Is(err, storage.ErrConflict):
		// Already seen: /verify on cold start and the DID_RENEW notification
		// race routinely both arrive.
		existing, gerr := h.txns.GetByProviderInvoice(r.Context(), billing.ProviderApple, tx.TransactionID)
		if gerr != nil {
			return appleApplyResult{err: gerr}
		}
		res.txID = existing.ID
	default:
		return appleApplyResult{err: err}
	}

	if !opts.Grant {
		return res
	}
	if deck != nil {
		txID := res.txID
		if err := h.decks.Grant(r.Context(), userID, deck.ID, billing.ProviderApple, &txID); err != nil {
			slog.Error("grant deck from apple purchase", "error", err, "user", userID, "deck", deck.Slug)
		}
		return res
	}
	if err := h.grantPremium(r, userID, product, billing.ProviderApple, opts.ExpiresAt); err != nil {
		slog.Error("grant apple premium", "error", err, "user", userID)
		return appleApplyResult{err: err}
	}
	return res
}

func (h *Handler) refundAppleTransaction(r *http.Request, tx *appstore.Transaction, note *appstore.Notification) {
	existing, err := h.txns.GetByProviderInvoice(r.Context(), billing.ProviderApple, tx.TransactionID)
	if err != nil {
		slog.Warn("apple refund for an unknown transaction", "apple_tx", tx.TransactionID, "error", err)
		return
	}
	reason := "apple " + note.NotificationType
	if note.Subtype != "" {
		reason += " " + note.Subtype
	}
	if _, _, err := h.txns.Refund(r.Context(), existing.ID, "apple", reason); err != nil {
		if errors.Is(err, storage.ErrNotRefundable) {
			slog.Info("apple refund on a non-paid transaction", "tx", existing.ID)
			return
		}
		slog.Error("record apple refund", "error", err, "tx", existing.ID)
	}
}

// revokeAppleEntitlement takes premium away, but only from a grant Apple owns.
// CanRevokeFrom is what stops an Apple event touching RuStore or YooKassa
// premium held by the same account.
func (h *Handler) revokeAppleEntitlement(r *http.Request, userID string, tx *appstore.Transaction) {
	user, err := h.users.GetByID(r.Context(), userID)
	if err != nil {
		slog.Error("load user for apple revoke", "error", err, "user", userID)
		return
	}
	if !user.HasPremium {
		return
	}
	if !billing.CanRevokeFrom(user.PremiumSource, billing.ProviderApple) {
		slog.Info("apple event cannot revoke premium from another provider",
			"user", userID, "source", user.PremiumSource)
		return
	}
	// A lifetime purchase only ends on a refund or revoke, which arrive as
	// their own notifications and are handled by the effect table.
	if user.PremiumProductID == billing.ProductLifetime && !tx.IsRevoked() &&
		!strings.EqualFold(tx.ProductID, billing.AppleSKU(billing.ProductLifetime)) {
		return
	}
	if err := h.users.RevokePremium(r.Context(), userID); err != nil {
		slog.Error("revoke apple premium", "error", err, "user", userID)
	}
}

func (h *Handler) recordAppleSubscription(r *http.Request, userID string, tx *appstore.Transaction, status *int, lastNotification string) {
	if tx == nil || strings.TrimSpace(tx.OriginalTransactionID) == "" {
		return
	}
	ntype, nuuid := "", ""
	if lastNotification != "" {
		parts := strings.SplitN(lastNotification, "|", 2)
		ntype = parts[0]
		if len(parts) == 2 {
			nuuid = parts[1]
		}
	}
	product, _ := h.resolveAppleProduct(r, tx.ProductID)
	if _, err := h.appleSubs.Bind(r.Context(), storage.AppleSubscription{
		OriginalTransactionID: tx.OriginalTransactionID,
		UserID:                userID,
		ProductID:             product.ID,
		AppleProductID:        tx.ProductID,
		Environment:           appleEnvironment(tx),
		Status:                status,
		ExpiresAt:             tx.ExpiresAt(),
		LastTransactionID:     tx.TransactionID,
		LastNotificationType:  ntype,
		LastNotificationUUID:  nuuid,
	}); err != nil {
		slog.Error("record apple subscription", "error", err, "original", tx.OriginalTransactionID)
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// appleAuthoritativeTransaction re-checks the purchase against Apple's servers
// so a refunded or lapsed entitlement is never granted from a stale JWS. Without
// the In-App Purchase key it degrades to the client-supplied claims.
func (h *Handler) appleAuthoritativeTransaction(r *http.Request, claimed *appstore.Transaction, originalID string) (*appstore.Transaction, *int, error) {
	if !h.as.ServerAPIEnabled() {
		if claimed == nil {
			return nil, nil, fmt.Errorf("%w: the In-App Purchase key is required to verify by id alone", appstore.ErrNotConfigured)
		}
		slog.Warn("apple purchase accepted from its signed transaction alone; set APPLE_IAP_* to confirm with Apple",
			"apple_tx", claimed.TransactionID)
		return claimed, nil, nil
	}

	// Leave the environment unknown when the client sent only an id: the client
	// then has no signed payload to tell us, and the empty value makes the
	// Apple client probe production and then sandbox.
	env := ""
	if claimed != nil {
		env = appleEnvironment(claimed)
	}
	res, err := h.as.GetSubscriptionStatuses(r.Context(), originalID, env)
	if err == nil {
		if latest := res.Latest(); latest != nil && latest.Transaction != nil {
			status := latest.Status
			return latest.Transaction, &status, nil
		}
	}
	if err != nil && !errors.Is(err, appstore.ErrNotFound) {
		return nil, nil, err
	}

	// Non-consumables (lifetime premium, decks) are not subscriptions, so the
	// subscription endpoint has nothing for them.
	if claimed != nil {
		fetched, ferr := h.as.GetTransactionInfo(r.Context(), claimed.TransactionID, env)
		if ferr == nil {
			return fetched, nil, nil
		}
		if !errors.Is(ferr, appstore.ErrNotFound) {
			return nil, nil, ferr
		}
		slog.Warn("apple has no record of this transaction, using the signed payload",
			"apple_tx", claimed.TransactionID)
		return claimed, nil, nil
	}
	return nil, nil, appstore.ErrNotFound
}

// appleInactiveReason explains why an entitlement is not active, "" when it is.
func appleInactiveReason(tx *appstore.Transaction, status *int) string {
	if tx.IsRevoked() {
		return "revoked"
	}
	if status != nil {
		switch *status {
		case appstore.SubStatusExpired:
			return "expired"
		case appstore.SubStatusRevoked:
			return "revoked"
		case appstore.SubStatusBillingRetry:
			return "billing_retry"
		case appstore.SubStatusActive, appstore.SubStatusGracePeriod:
			return ""
		}
	}
	if expires := tx.ExpiresAt(); expires != nil && expires.Before(time.Now()) {
		return "expired"
	}
	return ""
}

// resolveAppleProduct maps an Apple SKU to a premium product or a shop deck,
// mirroring how ReportPurchase already branches between the two.
func (h *Handler) resolveAppleProduct(r *http.Request, sku string) (billing.Product, *storage.Deck) {
	if product, ok := billing.LookupApple(sku); ok {
		return product, nil
	}
	deck, err := h.decks.GetByAppleProductID(r.Context(), sku)
	if err != nil {
		if !errors.Is(err, storage.ErrNotFound) {
			slog.Error("look up deck by apple sku", "error", err, "sku", sku)
		}
		return billing.Product{}, nil
	}
	return billing.Product{}, deck
}

// appleAmount prefers Apple's own price. It is in milliunits, and amount_kop
// holds minor units of its currency — the same convention CloudPayments uses
// for USD cents. The catalog fallback is not optional: price and currency are
// absent from older payloads.
func appleAmount(tx *appstore.Transaction, product billing.Product, deck *storage.Deck) (int, string) {
	if minor, currency, ok := tx.AmountMinor(); ok {
		return minor, currency
	}
	if deck != nil {
		return deck.PriceKop, billing.CurrencyRUB
	}
	return product.AmountUSDCents, billing.CurrencyUSD
}

func appleNotes(tx *appstore.Transaction, note *appstore.Notification) string {
	parts := []string{"apple"}
	if note != nil {
		parts = append(parts, note.NotificationType)
		if note.Subtype != "" {
			parts = append(parts, note.Subtype)
		}
	} else if tx.TransactionReason != "" {
		parts = append(parts, tx.TransactionReason)
	}
	parts = append(parts, "env="+appleEnvironment(tx), "orig="+tx.OriginalTransactionID)
	if tx.WebOrderLineItemID != "" {
		parts = append(parts, "wo="+tx.WebOrderLineItemID)
	}
	if note != nil && note.NotificationUUID != "" {
		parts = append(parts, "notif="+note.NotificationUUID)
	}
	return strings.Join(parts, " ")
}

func appleEnvironment(tx *appstore.Transaction) string {
	if tx == nil || strings.TrimSpace(tx.Environment) == "" {
		return appstore.EnvProduction
	}
	return tx.Environment
}

func writeAppleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errAppleFamilyShared), errors.Is(err, appstore.ErrFamilyShared):
		writeError(w, http.StatusForbidden, "apple_family_shared",
			"family shared purchases cannot activate premium")
	case errors.Is(err, errAppleOwnerMismatch):
		writeError(w, http.StatusForbidden, "apple_account_mismatch",
			"this purchase belongs to another account")
	case errors.Is(err, errAppleOwnerUnknown):
		writeError(w, http.StatusForbidden, "apple_account_mismatch",
			"this purchase is not linked to an account")
	case errors.Is(err, appstore.ErrBundleMismatch):
		writeError(w, http.StatusUnprocessableEntity, "validation_error", "transaction is for another app")
	case errors.Is(err, appstore.ErrSandboxRejected):
		writeError(w, http.StatusUnprocessableEntity, "validation_error", "sandbox transactions are not accepted")
	case errors.Is(err, appstore.ErrBadSignature):
		writeError(w, http.StatusUnprocessableEntity, "invalid_signature", "transaction signature is invalid")
	case errors.Is(err, appstore.ErrNotConfigured):
		writeError(w, http.StatusServiceUnavailable, "payments_unavailable", "Apple In-App Purchase is not configured")
	case errors.Is(err, appstore.ErrNotFound):
		writeError(w, http.StatusUnprocessableEntity, "validation_error", "Apple has no record of this transaction")
	case errors.Is(err, appstore.ErrProviderUnavailable):
		writeError(w, http.StatusBadGateway, "payment_provider_error", "could not reach the App Store")
	default:
		slog.Error("apple verify failed", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to verify purchase")
	}
}

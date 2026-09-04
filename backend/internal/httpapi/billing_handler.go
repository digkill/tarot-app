package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/digkill/tarot-app/backend/internal/billing"
	"github.com/digkill/tarot-app/backend/internal/storage"
)

type reportPurchaseRequest struct {
	ProductID  string `json:"productId"`
	InvoiceID  string `json:"invoiceId"`
	PurchaseID string `json:"purchaseId"`
	OrderID    string `json:"orderId"`
	Source     string `json:"source"`
	Sandbox    bool   `json:"sandbox"`
	ExpiresAt  string `json:"expiresAt"`
}

type subscriptionStatusRequest struct {
	Active    bool   `json:"active"`
	ProductID string `json:"productId"`
	ExpiresAt string `json:"expiresAt"`
}

func (h *Handler) ReportPurchase(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromCtx(r.Context())
	var req reportPurchaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
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

	premium, isPremium := billing.Lookup(req.ProductID)
	var deck *storage.Deck
	if !isPremium {
		deck, err = h.decks.GetByProductID(r.Context(), req.ProductID)
		if errors.Is(err, storage.ErrNotFound) {
			writeError(w, http.StatusUnprocessableEntity, "validation_error", "unknown productId")
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to load product")
			return
		}
	}

	source := billing.NormalizeSource(req.Source)
	provider := billing.ProviderRuStore
	kind := billing.KindOneTime
	amount := 0
	productID := strings.TrimSpace(req.ProductID)
	if isPremium {
		kind = premium.Kind
		amount = premium.AmountKop
		productID = premium.ID
	} else {
		kind = billing.KindOneTime
		amount = deck.PriceKop
		productID = deck.ProductID()
	}

	storeExpiry := billing.ParseTime(req.ExpiresAt)

	switch source {
	case billing.KindRestore:
		provider = billing.ProviderRuStore
		kind = billing.KindRestore
		if strings.TrimSpace(req.InvoiceID) == "" {
			if isPremium {
				if err = h.grantPremium(r, user.ID, premium, billing.ProviderRuStore, storeExpiry); err != nil {
					writeError(w, http.StatusInternalServerError, "internal_error", "failed to grant premium")
					return
				}
				writeJSON(w, http.StatusOK, map[string]any{"ok": true, "hasPremium": true})
				return
			}
			_ = h.decks.Grant(r.Context(), user.ID, deck.ID, "restore", nil)
			writeJSON(w, http.StatusOK, map[string]any{"ok": true, "hasPremium": user.HasPremium, "deckSlug": deck.Slug})
			return
		}
	case billing.ProviderDev:
		provider = billing.ProviderDev
		amount = 0
	}

	if invoice := strings.TrimSpace(req.InvoiceID); invoice != "" {
		existing, err := h.txns.GetByProviderInvoice(r.Context(), provider, invoice)
		if err == nil {
			h.applyPurchaseEntitlement(r, user.ID, existing.ID, isPremium, premium, deck, provider, storeExpiry)
			writeJSON(w, http.StatusOK, map[string]any{
				"ok":            true,
				"transactionId": existing.ID,
				"hasPremium":    isPremium || user.HasPremium,
				"deckSlug":      deckSlug(deck),
			})
			return
		}
		if !errors.Is(err, storage.ErrNotFound) {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to load transaction")
			return
		}
	}

	tx, err := h.txns.Create(r.Context(), storage.CreateTransactionParams{
		UserID:             user.ID,
		ProductID:          productID,
		Kind:               kind,
		Status:             billing.StatusPaid,
		Provider:           provider,
		ProviderInvoiceID:  storage.NullIfBlank(req.InvoiceID),
		ProviderPurchaseID: storage.NullIfBlank(req.PurchaseID),
		AmountKop:          amount,
		Currency:           "RUB",
		Sandbox:            req.Sandbox,
		Notes:              strings.TrimSpace(req.OrderID),
	})
	if err != nil {
		if errors.Is(err, storage.ErrConflict) {
			existing, gerr := h.txns.GetByProviderInvoice(r.Context(), provider, strings.TrimSpace(req.InvoiceID))
			if gerr == nil {
				h.applyPurchaseEntitlement(r, user.ID, existing.ID, isPremium, premium, deck, provider, storeExpiry)
				writeJSON(w, http.StatusOK, map[string]any{
					"ok":            true,
					"transactionId": existing.ID,
					"hasPremium":    isPremium || user.HasPremium,
					"deckSlug":      deckSlug(deck),
				})
				return
			}
			writeError(w, http.StatusConflict, "conflict", "duplicate invoice")
			return
		}
		slog.Error("create purchase transaction", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to store purchase")
		return
	}

	h.applyPurchaseEntitlement(r, user.ID, tx.ID, isPremium, premium, deck, provider, storeExpiry)
	writeJSON(w, http.StatusCreated, map[string]any{
		"ok":            true,
		"transactionId": tx.ID,
		"hasPremium":    isPremium || user.HasPremium,
		"deckSlug":      deckSlug(deck),
	})
}

func (h *Handler) SyncSubscriptionStatus(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromCtx(r.Context())
	var req subscriptionStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
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

	if req.Active {
		product, ok := billing.Lookup(req.ProductID)
		if !ok {
			product, ok = billing.Lookup(user.PremiumProductID)
		}
		if !ok {
			writeError(w, http.StatusUnprocessableEntity, "validation_error", "unknown productId")
			return
		}
		if err = h.grantPremium(r, user.ID, product, billing.ProviderRuStore, billing.ParseTime(req.ExpiresAt)); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to refresh premium")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "hasPremium": true})
		return
	}

	if user.PremiumProductID == billing.ProductLifetime {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "hasPremium": true})
		return
	}
	if !billing.IsStoreManaged(user.PremiumSource) {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "hasPremium": user.HasPremium})
		return
	}
	if err = h.users.RevokePremium(r.Context(), user.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to revoke premium")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "hasPremium": false})
}

func deckSlug(d *storage.Deck) string {
	if d == nil {
		return ""
	}
	return d.Slug
}

func (h *Handler) applyPurchaseEntitlement(
	r *http.Request,
	userID, txID string,
	isPremium bool,
	premium billing.Product,
	deck *storage.Deck,
	source string,
	storeExpiry *time.Time,
) {
	if isPremium {
		if err := h.grantPremium(r, userID, premium, source, storeExpiry); err != nil {
			slog.Error("grant premium after purchase", "error", err)
		}
		return
	}
	if deck == nil || deck.IsFree {
		return
	}
	tid := txID
	if err := h.decks.Grant(r.Context(), userID, deck.ID, source, &tid); err != nil {
		slog.Error("grant deck after purchase", "error", err)
	}
}

func (h *Handler) grantPremium(r *http.Request, userID string, product billing.Product, source string, storeExpiry *time.Time) error {
	if product.ID == "" {
		return h.users.GrantPremium(r.Context(), userID, storage.PremiumGrant{Source: source})
	}
	return h.users.GrantPremium(r.Context(), userID, storage.PremiumGrant{
		ProductID: product.ID,
		Source:    source,
		ExpiresAt: billing.ChooseExpiry(product, storeExpiry, time.Now()),
	})
}

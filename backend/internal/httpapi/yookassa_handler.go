package httpapi

import (
	"encoding/json"
	"errors"
	"html"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/digkill/tarot-app/backend/internal/billing"
	"github.com/digkill/tarot-app/backend/internal/storage"
	"github.com/digkill/tarot-app/backend/internal/yookassa"
)

type checkoutRequest struct {
	ProductID string `json:"productId"`
	// Provider is "yookassa" (Russian cards, RUB; the default) or
	// "cloudpayments" (foreign cards, USD).
	Provider string `json:"provider"`
}

func (h *Handler) CreateCheckout(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromCtx(r.Context())
	var req checkoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}
	provider := strings.ToLower(strings.TrimSpace(req.Provider))
	if provider == "" {
		provider = billing.ProviderYooKassa
	}
	if provider != billing.ProviderYooKassa && provider != billing.ProviderCloudPayments {
		writeError(w, http.StatusUnprocessableEntity, "validation_error", "unknown provider")
		return
	}
	product, ok := billing.Lookup(req.ProductID)
	if !ok {
		writeError(w, http.StatusUnprocessableEntity, "validation_error", "unknown productId")
		return
	}
	user, err := h.users.GetByID(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "user not found")
		return
	}
	if source, blocked := billing.ActivePremiumSource(user.HasPremium, user.PremiumSource); blocked {
		writeJSON(w, http.StatusConflict, map[string]any{
			"error": map[string]any{
				"code":          "premium_already_active",
				"message":       "premium is already active for this account",
				"premiumSource": source,
			},
		})
		return
	}

	if provider == billing.ProviderCloudPayments {
		h.startCloudPaymentsCheckout(w, r, user, product)
		return
	}
	h.startYooKassaCheckout(w, r, user, product)
}

func (h *Handler) startYooKassaCheckout(w http.ResponseWriter, r *http.Request, user *storage.User, product billing.Product) {
	if h.yk == nil || !h.yk.Enabled() {
		writeError(w, http.StatusServiceUnavailable, "payments_unavailable", "YooKassa is not configured")
		return
	}

	tx, err := h.txns.Create(r.Context(), storage.CreateTransactionParams{
		UserID:    user.ID,
		ProductID: product.ID,
		Kind:      product.Kind,
		Status:    billing.StatusPending,
		Provider:  billing.ProviderYooKassa,
		AmountKop: product.AmountKop,
		Currency:  "RUB",
		Notes:     "ios-web-gateway",
	})
	if err != nil {
		slog.Error("create yookassa transaction", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to start checkout")
		return
	}

	returnURL := h.cfg.PublicBaseURL + "/pay/return?tx=" + tx.ID
	pay, err := h.yk.CreateRedirectPayment(
		tx.ID,
		returnURL,
		"Tarot Premium — "+product.Title,
		product.AmountKop,
		map[string]string{
			"user_id":        user.ID,
			"product_id":     product.ID,
			"transaction_id": tx.ID,
		},
		user.Email,
	)
	if err != nil {
		_ = h.txns.SetStatus(r.Context(), tx.ID, billing.StatusCanceled)
		slog.Error("yookassa create payment", "error", err)
		writeError(w, http.StatusBadGateway, "payment_provider_error", "failed to start payment")
		return
	}
	confirmURL := ""
	if pay.Confirmation != nil {
		confirmURL = pay.Confirmation.ConfirmationURL
	}
	if confirmURL == "" {
		_ = h.txns.SetStatus(r.Context(), tx.ID, billing.StatusCanceled)
		writeError(w, http.StatusBadGateway, "payment_provider_error", "payment has no confirmation url")
		return
	}
	if err = h.txns.SetProviderRefs(r.Context(), tx.ID, pay.ID, pay.ID); err != nil {
		slog.Error("store yookassa payment id", "error", err)
	}
	if err = h.checkout.Upsert(r.Context(), storage.CheckoutSession{
		TransactionID:     tx.ID,
		ConfirmationURL:   confirmURL,
		ProviderPaymentID: pay.ID,
	}); err != nil {
		slog.Error("store checkout session", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to start checkout")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"transactionId":   tx.ID,
		"paymentId":       pay.ID,
		"checkoutUrl":     h.cfg.PublicBaseURL + "/pay/go?tx=" + tx.ID,
		"confirmationUrl": confirmURL,
		"returnUrl":       returnURL,
		"productId":       product.ID,
		"provider":        billing.ProviderYooKassa,
		"currency":        billing.CurrencyRUB,
		"amountMinor":     product.AmountKop,
	})
}

func (h *Handler) GetCheckout(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromCtx(r.Context())
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	tx, err := h.txns.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "checkout not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to load checkout")
		return
	}
	if tx.UserID != userID {
		writeError(w, http.StatusNotFound, "not_found", "checkout not found")
		return
	}
	user, err := h.users.GetByID(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to load user")
		return
	}
	if tx.Status == billing.StatusPending {
		reconciled := false
		switch tx.Provider {
		case billing.ProviderYooKassa:
			if tx.ProviderInvoiceID != nil && h.yk != nil && h.yk.Enabled() {
				if pay, gerr := h.yk.GetPayment(*tx.ProviderInvoiceID); gerr == nil {
					h.applyYooKassaPayment(r, tx, pay)
					reconciled = true
				}
			}
		case billing.ProviderCloudPayments:
			h.reconcileCloudPayments(r, tx)
			reconciled = true
			// No billing.ProviderApple case on purpose: Apple never produces a
			// pending transaction and has no checkout session, so there is
			// nothing to reconcile. Note that a provider left out of this
			// switch silently never reconciles at all.
		}
		if reconciled {
			if refreshed, rerr := h.txns.GetByID(r.Context(), tx.ID); rerr == nil {
				tx = refreshed
			}
			if refreshedUser, uerr := h.users.GetByID(r.Context(), userID); uerr == nil {
				user = refreshedUser
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"transactionId": tx.ID,
		"status":        tx.Status,
		"productId":     tx.ProductID,
		"hasPremium":    user.HasPremium,
		"premiumSource": user.PremiumSource,
	})
}

func (h *Handler) PayGo(w http.ResponseWriter, r *http.Request) {
	txID := strings.TrimSpace(r.URL.Query().Get("tx"))
	if txID == "" {
		http.Error(w, "missing tx", http.StatusBadRequest)
		return
	}
	sess, err := h.checkout.GetByTransaction(r.Context(), txID)
	if err != nil {
		http.Error(w, "checkout not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(payGoHTML(html.EscapeString(sess.ConfirmationURL))))
}

func (h *Handler) PayReturn(w http.ResponseWriter, r *http.Request) {
	txID := strings.TrimSpace(r.URL.Query().Get("tx"))
	deep := "mediarisetarot://billing/complete"
	if txID != "" {
		deep += "?tx=" + txID
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(payReturnHTML(html.EscapeString(deep))))
}

func (h *Handler) YooKassaWebhook(w http.ResponseWriter, r *http.Request) {
	if h.yk == nil || !h.yk.Enabled() {
		writeError(w, http.StatusServiceUnavailable, "payments_unavailable", "YooKassa is not configured")
		return
	}
	var note yookassa.Notification
	if err := json.NewDecoder(r.Body).Decode(&note); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid webhook")
		return
	}
	paymentID := strings.TrimSpace(note.Object.ID)
	if paymentID == "" {
		w.WriteHeader(http.StatusOK)
		return
	}
	pay, err := h.yk.GetPayment(paymentID)
	if err != nil {
		slog.Error("yookassa webhook fetch", "error", err, "payment", paymentID)
		writeError(w, http.StatusBadGateway, "payment_provider_error", "failed to verify payment")
		return
	}
	tx, err := h.txns.GetByProviderInvoice(r.Context(), billing.ProviderYooKassa, pay.ID)
	if err != nil {
		if txID := strings.TrimSpace(pay.Metadata["transaction_id"]); txID != "" {
			tx, err = h.txns.GetByID(r.Context(), txID)
		}
	}
	if err != nil {
		slog.Error("yookassa webhook tx missing", "payment", pay.ID, "error", err)
		w.WriteHeader(http.StatusOK)
		return
	}
	h.applyYooKassaPayment(r, tx, pay)
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) applyYooKassaPayment(r *http.Request, tx *storage.Transaction, pay *yookassa.Payment) {
	if tx == nil || pay == nil {
		return
	}
	switch pay.Status {
	case "succeeded":
		if tx.Status != billing.StatusPaid {
			_ = h.txns.SetStatus(r.Context(), tx.ID, billing.StatusPaid)
		}
		product, ok := billing.Lookup(tx.ProductID)
		if !ok {
			product, ok = billing.Lookup(pay.Metadata["product_id"])
		}
		if ok {
			if err := h.grantPremium(r, tx.UserID, product, billing.ProviderYooKassa, nil); err != nil {
				slog.Error("grant yookassa premium", "error", err, "user", tx.UserID)
			}
		}
	case "canceled":
		if tx.Status == billing.StatusPending {
			_ = h.txns.SetStatus(r.Context(), tx.ID, billing.StatusCanceled)
		}
	}
}

func payGoHTML(url string) string {
	return `<!doctype html><html lang="ru"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Tarot — оплата</title>
<style>body{font-family:-apple-system,sans-serif;background:#0B1220;color:#f7f4ea;display:flex;min-height:100vh;align-items:center;justify-content:center;margin:0}a{color:#6c5ce7}</style></head>
<body><p>Переходим к оплате… / Redirecting to payment…</p><script>location.replace("` + url + `");</script>
<noscript><p><a href="` + url + `">Продолжить оплату</a></p></noscript></body></html>`
}

func payReturnHTML(deep string) string {
	return `<!doctype html><html lang="ru"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Tarot — готово</title>
<style>body{font-family:-apple-system,sans-serif;background:#0B1220;color:#f7f4ea;display:flex;min-height:100vh;align-items:center;justify-content:center;margin:0;padding:24px;text-align:center}a{color:#6c5ce7}</style></head>
<body><div><p>Оплата обработана. Можно вернуться в приложение.</p><p><a href="` + deep + `">Открыть Tarot</a></p></div>
<script>setTimeout(function(){location.replace("` + deep + `");},400);</script></body></html>`
}

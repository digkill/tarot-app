package httpapi

import (
	"encoding/json"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/digkill/tarot-app/backend/internal/billing"
	"github.com/digkill/tarot-app/backend/internal/cloudpayments"
	"github.com/digkill/tarot-app/backend/internal/storage"
)

const cloudPaymentsCompleted = "Completed"

func (h *Handler) startCloudPaymentsCheckout(w http.ResponseWriter, r *http.Request, user *storage.User, product billing.Product) {
	if h.cp == nil || !h.cp.Enabled() {
		writeError(w, http.StatusServiceUnavailable, "payments_unavailable", "CloudPayments is not configured")
		return
	}

	tx, err := h.txns.Create(r.Context(), storage.CreateTransactionParams{
		UserID:    user.ID,
		ProductID: product.ID,
		Kind:      product.Kind,
		Status:    billing.StatusPending,
		Provider:  billing.ProviderCloudPayments,
		// amount_kop holds minor units of Currency, so USD cents here.
		AmountKop: product.AmountUSDCents,
		Currency:  billing.CurrencyUSD,
		Notes:     "ios-web-gateway-foreign",
	})
	if err != nil {
		slog.Error("create cloudpayments transaction", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to start checkout")
		return
	}

	returnURL := h.cfg.PublicBaseURL + "/pay/return?tx=" + tx.ID
	order, err := h.cp.CreateOrder(cloudpayments.OrderParams{
		AmountMinor:        product.AmountUSDCents,
		Currency:           billing.CurrencyUSD,
		Description:        "Tarot " + product.TitleEN,
		Email:              user.Email,
		InvoiceID:          tx.ID,
		AccountID:          user.ID,
		CultureName:        "en-US",
		SuccessRedirectURL: returnURL,
		FailRedirectURL:    returnURL,
		JSONData: map[string]string{
			"user_id":        user.ID,
			"product_id":     product.ID,
			"transaction_id": tx.ID,
		},
	})
	if err != nil {
		_ = h.txns.SetStatus(r.Context(), tx.ID, billing.StatusCanceled)
		slog.Error("cloudpayments create order", "error", err)
		writeError(w, http.StatusBadGateway, "payment_provider_error", "failed to start payment")
		return
	}
	if err = h.txns.SetProviderRefs(r.Context(), tx.ID, order.ID, ""); err != nil {
		slog.Error("store cloudpayments order id", "error", err)
	}
	if err = h.checkout.Upsert(r.Context(), storage.CheckoutSession{
		TransactionID:     tx.ID,
		ConfirmationURL:   order.URL,
		ProviderPaymentID: order.ID,
	}); err != nil {
		slog.Error("store checkout session", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to start checkout")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"transactionId":   tx.ID,
		"paymentId":       order.ID,
		"checkoutUrl":     h.cfg.PublicBaseURL + "/pay/go?tx=" + tx.ID,
		"confirmationUrl": order.URL,
		"returnUrl":       returnURL,
		"productId":       product.ID,
		"provider":        billing.ProviderCloudPayments,
		"currency":        billing.CurrencyUSD,
		"amountMinor":     product.AmountUSDCents,
	})
}

// reconcileCloudPayments covers a missed or delayed Pay notification: the app
// polls GetCheckout after the browser closes, and we ask CloudPayments
// directly whether the invoice was paid.
func (h *Handler) reconcileCloudPayments(r *http.Request, tx *storage.Transaction) {
	if tx.Status != billing.StatusPending || h.cp == nil || !h.cp.Enabled() {
		return
	}
	pay, found, err := h.cp.FindPayment(tx.ID)
	if err != nil {
		slog.Warn("cloudpayments find payment", "error", err, "tx", tx.ID)
		return
	}
	if !found {
		return
	}
	h.applyCloudPaymentsPayment(r, tx, cloudPaymentsResult{
		Status:        pay.Status,
		AmountMinor:   cloudpayments.MinorUnits(pay.Amount),
		Currency:      pay.Currency,
		TransactionID: strconv.FormatInt(pay.TransactionID, 10),
	})
}

type cloudPaymentsResult struct {
	Status        string
	AmountMinor   int
	Currency      string
	TransactionID string
}

func (h *Handler) applyCloudPaymentsPayment(r *http.Request, tx *storage.Transaction, res cloudPaymentsResult) {
	if tx.Provider != billing.ProviderCloudPayments || res.Status != cloudPaymentsCompleted {
		return
	}
	// The order amount is fixed server-side, but never grant on a payment
	// that doesn't match what this transaction was created for.
	if res.AmountMinor != tx.AmountKop || !strings.EqualFold(res.Currency, tx.Currency) {
		slog.Error("cloudpayments amount mismatch",
			"tx", tx.ID, "want", tx.AmountKop, "wantCurrency", tx.Currency,
			"got", res.AmountMinor, "gotCurrency", res.Currency)
		return
	}
	transitioned, err := h.txns.MarkPaidFromPending(r.Context(), tx.ID, res.TransactionID)
	if err != nil {
		slog.Error("mark cloudpayments transaction paid", "error", err, "tx", tx.ID)
		return
	}
	if !transitioned {
		return
	}
	product, ok := billing.Lookup(tx.ProductID)
	if !ok {
		slog.Error("cloudpayments paid for unknown product", "tx", tx.ID, "product", tx.ProductID)
		return
	}
	if err = h.grantPremium(r, tx.UserID, product, billing.ProviderCloudPayments, nil); err != nil {
		slog.Error("grant cloudpayments premium", "error", err, "user", tx.UserID)
	}
}

// CloudPaymentsPay handles the "Pay" notification. It must answer {"code":0}
// for any genuine notification — otherwise CloudPayments keeps retrying — and
// only a valid HMAC signature is trusted.
func (h *Handler) CloudPaymentsPay(w http.ResponseWriter, r *http.Request) {
	if h.cp == nil || !h.cp.Enabled() {
		writeError(w, http.StatusServiceUnavailable, "payments_unavailable", "CloudPayments is not configured")
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid notification")
		return
	}
	if !h.cp.VerifySignature(body, r.Header.Get("X-Content-HMAC"), r.Header.Get("Content-HMAC")) {
		slog.Warn("cloudpayments notification with invalid signature", "ip", clientIP(r))
		writeError(w, http.StatusUnauthorized, "invalid_signature", "invalid signature")
		return
	}

	fields, err := parseCloudPaymentsNotification(r.Header.Get("Content-Type"), body)
	if err != nil {
		slog.Error("parse cloudpayments notification", "error", err)
		writeCloudPaymentsAck(w)
		return
	}
	invoiceID := strings.TrimSpace(fields["InvoiceId"])
	if invoiceID == "" {
		writeCloudPaymentsAck(w)
		return
	}
	tx, err := h.txns.GetByID(r.Context(), invoiceID)
	if err != nil {
		slog.Error("cloudpayments notification for unknown transaction", "invoice", invoiceID, "error", err)
		writeCloudPaymentsAck(w)
		return
	}
	amount, err := strconv.ParseFloat(strings.TrimSpace(fields["Amount"]), 64)
	if err != nil {
		slog.Error("cloudpayments notification amount", "invoice", invoiceID, "amount", fields["Amount"])
		writeCloudPaymentsAck(w)
		return
	}
	h.applyCloudPaymentsPayment(r, tx, cloudPaymentsResult{
		Status:        strings.TrimSpace(fields["Status"]),
		AmountMinor:   cloudpayments.MinorUnits(amount),
		Currency:      strings.TrimSpace(fields["Currency"]),
		TransactionID: strings.TrimSpace(fields["TransactionId"]),
	})
	writeCloudPaymentsAck(w)
}

func writeCloudPaymentsAck(w http.ResponseWriter) {
	writeJSON(w, http.StatusOK, map[string]int{"code": 0})
}

// parseCloudPaymentsNotification accepts both notification formats the
// cabinet can be configured with: form-urlencoded (default) and JSON.
func parseCloudPaymentsNotification(contentType string, body []byte) (map[string]string, error) {
	mediaType, _, _ := mime.ParseMediaType(contentType)
	if mediaType == "application/json" {
		var raw map[string]any
		if err := json.Unmarshal(body, &raw); err != nil {
			return nil, err
		}
		out := make(map[string]string, len(raw))
		for k, v := range raw {
			switch val := v.(type) {
			case string:
				out[k] = val
			case float64:
				out[k] = strconv.FormatFloat(val, 'f', -1, 64)
			case nil:
			default:
				encoded, _ := json.Marshal(val)
				out[k] = string(encoded)
			}
		}
		return out, nil
	}
	values, err := url.ParseQuery(string(body))
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(values))
	for k := range values {
		out[k] = values.Get(k)
	}
	return out, nil
}

package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/digkill/tarot-app/backend/internal/storage"
)

func (h *Handler) RedeemPremiumCode(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromCtx(r.Context())
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized", "sign in to redeem a code")
		return
	}
	if !h.limiter.allow("premium-code-user:"+userID, 5, 15*time.Minute) ||
		!h.limiter.allow("premium-code-ip:"+clientIP(r), 15, 15*time.Minute) {
		writeRateLimit(w, 15*time.Minute, "too many code attempts")
		return
	}
	var req struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}
	code, valid := storage.NormalizePremiumCode(req.Code)
	if !valid {
		writeError(w, http.StatusUnprocessableEntity, "invalid_code", "enter a six-character code")
		return
	}
	user, err := h.users.RedeemPremiumCode(r.Context(), userID, code)
	switch {
	case errors.Is(err, storage.ErrPremiumCodeUnavailable):
		writeError(w, http.StatusConflict, "code_unavailable", "code is invalid or already used")
	case errors.Is(err, storage.ErrAlreadyLifetime):
		writeError(w, http.StatusConflict, "already_lifetime", "lifetime access is already active")
	case errors.Is(err, storage.ErrNotFound):
		writeError(w, http.StatusUnauthorized, "unauthorized", "account not found")
	case err != nil:
		writeError(w, http.StatusInternalServerError, "internal_error", "could not activate access")
	default:
		writeJSON(w, http.StatusOK, map[string]any{"user": toUserView(user)})
	}
}

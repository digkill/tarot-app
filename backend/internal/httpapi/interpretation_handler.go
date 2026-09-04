package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/digkill/tarot-app/backend/internal/llm"
	"github.com/digkill/tarot-app/backend/internal/storage"
	"github.com/digkill/tarot-app/backend/internal/usage"
)

type interpretRequest struct {
	SpreadID          string          `json:"spreadId"`
	SpreadName        string          `json:"spreadName"`
	SpreadDescription string          `json:"spreadDescription"`
	Language          string          `json:"language"`
	Cards             []llm.CardEntry `json:"cards"`
}

type interpretResponse struct {
	Summary   string                `json:"summary"`
	Positions []llm.InsightPosition `json:"positions"`
}

func (h *Handler) Interpret(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromCtx(r.Context())

	user, err := h.users.GetByID(r.Context(), userID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			writeError(w, http.StatusUnauthorized, "unauthorized", "user not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to load user")
		return
	}

	if !user.HasPremium {
		writeError(w, http.StatusPaymentRequired, "premium_required", "AI interpretations require a premium subscription")
		return
	}

	var req interpretRequest
	if err = json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}

	if req.SpreadID == "" || len(req.Cards) == 0 {
		writeError(w, http.StatusUnprocessableEntity, "validation_error", "spreadId and cards are required")
		return
	}

	if req.Language == "" {
		req.Language = "en"
	}
	if req.SpreadName == "" {
		req.SpreadName = req.SpreadID
	}

	if h.llmClient == nil {
		writeError(w, http.StatusServiceUnavailable, "llm_unavailable", "AI service is not configured")
		return
	}

	loc := requestLocation(r)
	day := usage.CalendarDay(time.Now(), loc)
	if _, err = h.usage.IncrementInterpretation(
		r.Context(),
		user.ID,
		day,
		usage.PremiumInterpretations,
		time.Now().Add(-usage.InterpretGap),
	); err != nil {
		if errors.Is(err, storage.ErrQuotaExceeded) {
			writeError(w, http.StatusTooManyRequests, "quota_exceeded", "daily interpretation quota reached")
			return
		}
		if errors.Is(err, storage.ErrActionCooldown) {
			writeRateLimit(w, usage.InterpretGap, "please wait before another interpretation")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to update usage")
		return
	}

	insight, err := h.llmClient.Interpret(r.Context(), llm.InterpretRequest{
		SpreadID:          req.SpreadID,
		SpreadName:        req.SpreadName,
		SpreadDescription: req.SpreadDescription,
		Language:          req.Language,
		Cards:             req.Cards,
	})
	if err != nil {
		slog.Error("interpretation failed", "err", err)
		writeError(w, http.StatusBadGateway, "llm_error", "failed to get AI interpretation")
		return
	}

	writeJSON(w, http.StatusOK, interpretResponse{
		Summary:   insight.Summary,
		Positions: insight.Positions,
	})
}

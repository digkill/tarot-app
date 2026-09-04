package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/digkill/tarot-app/backend/internal/storage"
)

type readingView struct {
	ID          string                `json:"id"`
	SpreadID    string                `json:"spreadId"`
	DeckID      string                `json:"deckId"`
	Items       []storage.ReadingItem `json:"items"`
	SummaryText string                `json:"summaryText"`
	Notes       string                `json:"notes"`
	Favorite    bool                  `json:"favorite"`
	CreatedAt   time.Time             `json:"createdAt"`
}

func toReadingView(rd *storage.Reading) readingView {
	items := rd.Items
	if items == nil {
		items = []storage.ReadingItem{}
	}
	return readingView{
		ID:          rd.ID,
		SpreadID:    rd.SpreadID,
		DeckID:      rd.DeckID,
		Items:       items,
		SummaryText: rd.SummaryText,
		Notes:       rd.Notes,
		Favorite:    rd.Favorite,
		CreatedAt:   rd.CreatedAt,
	}
}

type createReadingRequest struct {
	SpreadID    string                `json:"spreadId"`
	DeckID      string                `json:"deckId"`
	Items       []storage.ReadingItem `json:"items"`
	SummaryText string                `json:"summaryText"`
	Notes       string                `json:"notes"`
}

func (h *Handler) CreateReading(w http.ResponseWriter, r *http.Request) {
	var req createReadingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}

	if req.SpreadID == "" || req.DeckID == "" {
		writeError(w, http.StatusUnprocessableEntity, "validation_error", "spreadId and deckId are required")
		return
	}

	userID := userIDFromCtx(r.Context())
	rd, err := h.readings.Create(r.Context(), storage.CreateReadingParams{
		UserID:      userID,
		SpreadID:    req.SpreadID,
		DeckID:      req.DeckID,
		Items:       req.Items,
		SummaryText: req.SummaryText,
		Notes:       req.Notes,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to save reading")
		return
	}

	writeJSON(w, http.StatusCreated, toReadingView(rd))
}

func (h *Handler) ListReadings(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromCtx(r.Context())

	limit := 20
	offset := 0

	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}

	list, err := h.readings.List(r.Context(), userID, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to list readings")
		return
	}

	views := make([]readingView, len(list))
	for i := range list {
		views[i] = toReadingView(&list[i])
	}
	writeJSON(w, http.StatusOK, map[string]any{"readings": views})
}

func (h *Handler) GetReading(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromCtx(r.Context())
	id := chi.URLParam(r, "id")

	rd, err := h.readings.GetByID(r.Context(), id, userID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "reading not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to get reading")
		return
	}

	writeJSON(w, http.StatusOK, toReadingView(rd))
}

type patchReadingRequest struct {
	Notes    *string `json:"notes"`
	Favorite *bool   `json:"favorite"`
}

func (h *Handler) PatchReading(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromCtx(r.Context())
	id := chi.URLParam(r, "id")

	var req patchReadingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}

	rd, err := h.readings.Update(r.Context(), id, userID, storage.UpdateReadingParams{
		Notes:    req.Notes,
		Favorite: req.Favorite,
	})
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "reading not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to update reading")
		return
	}

	writeJSON(w, http.StatusOK, toReadingView(rd))
}

func (h *Handler) DeleteReading(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromCtx(r.Context())
	id := chi.URLParam(r, "id")

	if err := h.readings.Delete(r.Context(), id, userID); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "reading not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to delete reading")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

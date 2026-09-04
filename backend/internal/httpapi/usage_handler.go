package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/digkill/tarot-app/backend/internal/storage"
	"github.com/digkill/tarot-app/backend/internal/usage"
)

type quotaBucket struct {
	Used      int `json:"used"`
	Limit     int `json:"limit"`
	Remaining int `json:"remaining"`
}

type usageView struct {
	Date            string      `json:"date"`
	Timezone        string      `json:"timezone"`
	HasPremium      bool        `json:"hasPremium"`
	DailyCards      quotaBucket `json:"dailyCards"`
	AdUnlocks       quotaBucket `json:"adUnlocks"`
	Interpretations quotaBucket `json:"interpretations"`
}

type consumeDailyCardRequest struct {
	AdToken string `json:"adToken"`
}

func (h *Handler) GetUsage(w http.ResponseWriter, r *http.Request) {
	user, loc, day, ok := h.loadUsageContext(w, r)
	if !ok {
		return
	}
	row, err := h.usage.Get(r.Context(), user.ID, day)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to load usage")
		return
	}
	writeJSON(w, http.StatusOK, toUsageView(user.HasPremium, loc, day, row))
}

func (h *Handler) StartAdSession(w http.ResponseWriter, r *http.Request) {
	user, loc, day, ok := h.loadUsageContext(w, r)
	if !ok {
		return
	}
	if user.HasPremium {
		writeError(w, http.StatusConflict, "premium_active", "premium accounts do not need ad unlocks")
		return
	}
	row, err := h.usage.Get(r.Context(), user.ID, day)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to load usage")
		return
	}
	if usage.Remaining(row.AdUnlocks, usage.FreeAdUnlocks) <= 0 ||
		usage.Remaining(row.DailyCards, usage.FreeDailyCards+usage.FreeAdUnlocks) <= 0 {
		writeError(w, http.StatusTooManyRequests, "quota_exceeded", "daily ad unlock limit reached")
		return
	}
	if row.LastAdAt != nil && time.Since(*row.LastAdAt) < usage.AdCooldown {
		writeRateLimit(w, usage.AdCooldown-time.Since(*row.LastAdAt), "please wait before another video")
		return
	}
	token, err := h.ads.issue(user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to start ad session")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"adToken":         token,
		"minWatchSeconds": int(usage.AdMinWatch.Seconds()),
		"date":            day.In(loc).Format("2006-01-02"),
	})
}

func (h *Handler) ConsumeDailyCard(w http.ResponseWriter, r *http.Request) {
	user, loc, day, ok := h.loadUsageContext(w, r)
	if !ok {
		return
	}

	var req consumeDailyCardRequest
	if r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", "invalid request body")
			return
		}
	}

	var (
		row storage.DailyUsage
		err error
	)
	switch {
	case user.HasPremium:
		row, err = h.usage.IncrementDailyCards(r.Context(), user.ID, day, usage.PremiumDailyCards)
	case req.AdToken != "":
		if consumeErr := h.ads.consume(user.ID, req.AdToken, usage.AdMinWatch); consumeErr != nil {
			h.writeAdTokenError(w, consumeErr)
			return
		}
		row, err = h.usage.IncrementAdCard(
			r.Context(),
			user.ID,
			day,
			usage.FreeDailyCards+usage.FreeAdUnlocks,
			usage.FreeAdUnlocks,
			time.Now().Add(-usage.AdCooldown),
		)
	default:
		row, err = h.usage.IncrementDailyCards(r.Context(), user.ID, day, usage.FreeDailyCards)
	}
	if err != nil {
		h.writeUsageError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toUsageView(user.HasPremium, loc, day, row))
}

func (h *Handler) loadUsageContext(w http.ResponseWriter, r *http.Request) (*storage.User, *time.Location, time.Time, bool) {
	userID := userIDFromCtx(r.Context())
	user, err := h.users.GetByID(r.Context(), userID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			writeError(w, http.StatusUnauthorized, "unauthorized", "user not found")
			return nil, nil, time.Time{}, false
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to load user")
		return nil, nil, time.Time{}, false
	}
	loc := requestLocation(r)
	day := usage.CalendarDay(time.Now(), loc)
	return user, loc, day, true
}

func toUsageView(hasPremium bool, loc *time.Location, day time.Time, row storage.DailyUsage) usageView {
	cardLimit := usage.DailyCardLimit(hasPremium)
	if !hasPremium {
		cardLimit = usage.FreeDailyCards + usage.FreeAdUnlocks
	}
	adLimit := usage.AdUnlockLimit(hasPremium)
	interpLimit := usage.InterpretationLimit(hasPremium)
	tz := usage.DefaultTimezone
	if loc != nil {
		tz = loc.String()
	}
	return usageView{
		Date:       day.In(loc).Format("2006-01-02"),
		Timezone:   tz,
		HasPremium: hasPremium,
		DailyCards: quotaBucket{
			Used:      row.DailyCards,
			Limit:     cardLimit,
			Remaining: usage.Remaining(row.DailyCards, cardLimit),
		},
		AdUnlocks: quotaBucket{
			Used:      row.AdUnlocks,
			Limit:     adLimit,
			Remaining: usage.Remaining(row.AdUnlocks, adLimit),
		},
		Interpretations: quotaBucket{
			Used:      row.Interpretations,
			Limit:     interpLimit,
			Remaining: usage.Remaining(row.Interpretations, interpLimit),
		},
	}
}

func (h *Handler) writeUsageError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, storage.ErrQuotaExceeded):
		writeError(w, http.StatusTooManyRequests, "quota_exceeded", "daily limit reached")
	case errors.Is(err, storage.ErrActionCooldown):
		writeRateLimit(w, usage.AdCooldown, "please wait before repeating this action")
	default:
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to update usage")
	}
}

func (h *Handler) writeAdTokenError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errAdWatchTooShort):
		writeError(w, http.StatusUnprocessableEntity, "ad_watch_incomplete", "watch the video until the end")
	case errors.Is(err, errAdTokenExpired), errors.Is(err, errAdTokenUnknown), errors.Is(err, errAdTokenMismatch):
		writeError(w, http.StatusUnprocessableEntity, "ad_token_invalid", "video session expired, try again")
	default:
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to validate video session")
	}
}

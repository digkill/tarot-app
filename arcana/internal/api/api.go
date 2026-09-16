// Package api serves the plain HTTP endpoints: the card and hero catalog the
// clients render from, and a player's match history and replays.
package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/digkill/tarot-app/arcana/internal/auth"
	"github.com/digkill/tarot-app/arcana/internal/cards"
	"github.com/digkill/tarot-app/arcana/internal/game"
	"github.com/digkill/tarot-app/arcana/internal/heroes"
	"github.com/digkill/tarot-app/arcana/internal/match"
	"github.com/digkill/tarot-app/arcana/internal/protocol"
	"github.com/digkill/tarot-app/arcana/internal/storage"
)

type API struct {
	Verifier *auth.Verifier
	Store    storage.Store
	Match    match.Config
}

func (a *API) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/arcana/catalog", a.catalog)
	mux.HandleFunc("GET /api/v1/arcana/matches", a.authed(a.listMatches))
	mux.HandleFunc("GET /api/v1/arcana/matches/{id}", a.authed(a.getMatch))
}

type catalogResponse struct {
	ProtocolVersion int          `json:"protocol_version"`
	RulesVersion    int          `json:"rules_version"`
	Rules           rulesView    `json:"rules"`
	Heroes          []heroes.Def `json:"heroes"`
	Cards           []cards.Def  `json:"cards"`
	Emotes          []string     `json:"emotes"`
}

type rulesView struct {
	DeckSize        int `json:"deck_size"`
	StartHand       int `json:"start_hand"`
	HandLimit       int `json:"hand_limit"`
	ManaCap         int `json:"mana_cap"`
	MulliganMax     int `json:"mulligan_max"`
	TurnSeconds     int `json:"turn_seconds"`
	MulliganSeconds int `json:"mulligan_seconds"`
	ReconnectSecs   int `json:"reconnect_seconds"`
}

func (a *API) catalog(w http.ResponseWriter, r *http.Request) {
	rules := game.DefaultRules()
	w.Header().Set("Cache-Control", "public, max-age=300")
	writeJSON(w, http.StatusOK, catalogResponse{
		ProtocolVersion: protocol.Version, RulesVersion: game.RulesVersion,
		Rules: rulesView{
			DeckSize: rules.DeckSize, StartHand: rules.StartHand, HandLimit: rules.HandLimit,
			ManaCap: rules.ManaCap, MulliganMax: rules.MulliganMax,
			TurnSeconds: int(a.Match.TurnTime / time.Second), MulliganSeconds: int(a.Match.MulliganTime / time.Second),
			ReconnectSecs: int(a.Match.ReconnectWindow / time.Second),
		},
		Heroes: heroes.All(), Cards: cards.All(), Emotes: protocol.Emotes,
	})
}

type matchSummary struct {
	ID           string      `json:"id"`
	OpponentID   string      `json:"opponent_id,omitempty"`
	Hero         string      `json:"hero"`
	OpponentHero string      `json:"opponent_hero"`
	Result       string      `json:"result"` // win, loss, none
	Reason       game.Reason `json:"reason"`
	Turns        int         `json:"turns"`
	StartedAt    time.Time   `json:"started_at"`
	FinishedAt   time.Time   `json:"finished_at"`
	DurationMS   int64       `json:"duration_ms"`
	RatingDelta  *int        `json:"rating_delta,omitempty"`
}

func summarize(m storage.MatchRecord, userID string) matchSummary {
	s := matchSummary{
		ID: m.ID, Reason: m.Reason, Turns: m.Turns, StartedAt: m.StartedAt, FinishedAt: m.FinishedAt,
		DurationMS: m.Duration().Milliseconds(), RatingDelta: m.RatingDelta, Result: "none",
		Hero: m.Player1Hero, OpponentHero: m.Player2Hero, OpponentID: m.Player2ID,
	}
	if m.Player2ID == userID {
		s.Hero, s.OpponentHero, s.OpponentID = m.Player2Hero, m.Player1Hero, m.Player1ID
	}
	switch m.WinnerID {
	case "":
	case userID:
		s.Result = "win"
	default:
		s.Result = "loss"
	}
	return s
}

func (a *API) listMatches(w http.ResponseWriter, r *http.Request, userID string) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	records, err := a.Store.ListMatches(r.Context(), userID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", "could not load matches")
		return
	}
	out := make([]matchSummary, 0, len(records))
	for _, m := range records {
		out = append(out, summarize(m, userID))
	}
	writeJSON(w, http.StatusOK, map[string]any{"matches": out})
}

// getMatch returns a summary and the replay as this player saw it: the
// opponent's hand and hidden plays stay hidden after the match too.
func (a *API) getMatch(w http.ResponseWriter, r *http.Request, userID string) {
	m, err := a.Store.GetMatch(r.Context(), r.PathValue("id"))
	if errors.Is(err, storage.ErrNotFound) || (err == nil && m.Player1ID != userID && m.Player2ID != userID) {
		writeError(w, http.StatusNotFound, "not_found", "match not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", "could not load the match")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"match":  summarize(m, userID),
		"replay": game.RedactLog(m.Replay, userID),
	})
}

func (a *API) authed(next func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		userID, err := a.Verifier.UserID(strings.TrimSpace(token))
		if !ok || err != nil {
			writeError(w, http.StatusUnauthorized, "unauthorized", "sign in required")
			return
		}
		next(w, r, userID)
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}

// Prefix is where the game routes live on the shared domain.
const Prefix = "/api/v1/arcana"

// AcceptStrippedPrefix lets the server work behind a proxy that routes
// /api/v1/arcana/* to it and strips that prefix (as Coolify does for a
// domain with a path) as well as behind one that keeps it.
func AcceptStrippedPrefix(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/healthz" && !strings.HasPrefix(r.URL.Path, Prefix+"/") {
			r2 := r.Clone(r.Context())
			r2.URL.Path = Prefix + "/" + strings.TrimPrefix(r.URL.Path, "/")
			r2.URL.RawPath = ""
			next.ServeHTTP(w, r2)
			return
		}
		next.ServeHTTP(w, r)
	})
}

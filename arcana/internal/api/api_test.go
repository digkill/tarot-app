package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/digkill/tarot-app/arcana/internal/auth"
	"github.com/digkill/tarot-app/arcana/internal/game"
	"github.com/digkill/tarot-app/arcana/internal/match"
	"github.com/digkill/tarot-app/arcana/internal/storage"
)

const secret = "0123456789abcdef0123456789abcdef"

func setup(t *testing.T) (*http.ServeMux, *storage.Memory) {
	t.Helper()
	v, _ := auth.NewVerifier(secret)
	store := storage.NewMemory()
	mux := http.NewServeMux()
	(&API{Verifier: v, Store: store, Match: match.DefaultConfig()}).Register(mux)
	return mux, store
}

func get(t *testing.T, mux *http.ServeMux, path, user string) (int, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if user != "" {
		tok, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
			Subject: user, ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))}).SignedString([]byte(secret))
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	return rec.Code, body
}

func TestCatalogIsPublic(t *testing.T) {
	mux, _ := setup(t)
	code, body := get(t, mux, "/api/v1/arcana/catalog", "")
	if code != 200 || len(body["heroes"].([]any)) != 4 || len(body["cards"].([]any)) < 20 {
		t.Fatalf("%d %v", code, body)
	}
}

func TestHistoryAndReplayArePrivate(t *testing.T) {
	mux, store := setup(t)
	g, _ := game.New(1, game.PlayerSetup{ID: "a", Hero: "death"}, game.PlayerSetup{ID: "b", Hero: "strength"})
	_ = g.Mulligan("a", []string{g.Players[0].Hand[0].UID})
	_ = g.Surrender("b")
	now := time.Now()
	_ = store.SaveMatch(context.Background(), storage.MatchRecord{
		ID: "m1", Player1ID: "a", Player2ID: "b", Player1Hero: "death", Player2Hero: "strength",
		WinnerID: "a", Reason: game.ReasonSurrender, StartedAt: now.Add(-time.Minute), FinishedAt: now, Replay: g.Log(),
	})

	if code, _ := get(t, mux, "/api/v1/arcana/matches", ""); code != 401 {
		t.Fatal("history needs a session")
	}
	_, body := get(t, mux, "/api/v1/arcana/matches", "b")
	m := body["matches"].([]any)[0].(map[string]any)
	if m["result"] != "loss" || m["hero"] != "strength" || m["opponent_hero"] != "death" {
		t.Fatalf("%v", m)
	}
	if code, _ := get(t, mux, "/api/v1/arcana/matches/m1", "mallory"); code != 404 {
		t.Fatal("strangers cannot read a replay")
	}
	_, body = get(t, mux, "/api/v1/arcana/matches/m1", "b")
	first := body["replay"].([]any)[0].(map[string]any)
	if first["action"] != "mulligan" || first["card_uids"] != nil {
		t.Fatalf("opponent mulligan leaked: %v", first)
	}
	for _, ev := range first["events"].([]any) {
		if ev.(map[string]any)["card_id"] != nil {
			t.Fatal("opponent draw leaked")
		}
	}
}

func TestRoutesWorkWithOrWithoutThePrefix(t *testing.T) {
	mux, _ := setup(t)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) })
	h := AcceptStrippedPrefix(mux)
	for _, path := range []string{"/api/v1/arcana/catalog", "/catalog", "/healthz"} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != 200 {
			t.Errorf("%s: %d", path, rec.Code)
		}
	}
}

// A player without an account row (a test bot, or a deleted account) is
// stored as NULL, which must not turn the other player's loss into "no
// result".
func TestResultSurvivesAPlayerWithoutAnAccount(t *testing.T) {
	mux, store := setup(t)
	now := time.Now()
	if err := store.SaveMatch(context.Background(), storage.MatchRecord{
		ID: "m2", Player1ID: "", Player2ID: "b", Player1Hero: "death", Player2Hero: "strength",
		WinnerID: "", WinnerSeat: 1, Reason: game.ReasonSurrender,
		StartedAt: now.Add(-time.Minute), FinishedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	_, body := get(t, mux, "/api/v1/arcana/matches", "b")
	m := body["matches"].([]any)[0].(map[string]any)
	if m["result"] != "loss" {
		t.Fatalf("result = %v, want loss", m["result"])
	}
}

func TestResultFromTheRecord(t *testing.T) {
	record := storage.MatchRecord{Player1ID: "a", Player2ID: "b"}
	if got := record.Result("a"); got != "none" {
		t.Fatalf("no winner: %s", got)
	}
	record.WinnerSeat = 2
	if record.Result("a") != "loss" || record.Result("b") != "win" || record.Result("c") != "none" {
		t.Fatal("by seat")
	}
	// Older rows have only the id.
	old := storage.MatchRecord{Player1ID: "a", Player2ID: "b", WinnerID: "a"}
	if old.Result("a") != "win" || old.Result("b") != "loss" {
		t.Fatal("by id")
	}
}

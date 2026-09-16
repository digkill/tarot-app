package ws

import (
	"context"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/golang-jwt/jwt/v5"

	"github.com/digkill/tarot-app/arcana/internal/auth"
	"github.com/digkill/tarot-app/arcana/internal/bot"
	"github.com/digkill/tarot-app/arcana/internal/game"
	"github.com/digkill/tarot-app/arcana/internal/protocol"
	"github.com/digkill/tarot-app/arcana/internal/storage"
)

const secret = "0123456789abcdef0123456789abcdef"

func token(t *testing.T, user string) string {
	t.Helper()
	s, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject: user, ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}).SignedString([]byte(secret))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func newServer(t *testing.T) (*Server, *storage.Memory, string) {
	t.Helper()
	v, _ := auth.NewVerifier(secret)
	store := storage.NewMemory()
	cfg := DefaultConfig()
	cfg.Match.TurnTime = 5 * time.Second
	cfg.Match.ReconnectWindow = 5 * time.Second
	cfg.Match.EmoteCooldown = 0
	srv := NewServer(cfg, v, store)
	hs := httptest.NewServer(srv)
	t.Cleanup(hs.Close)
	return srv, store, "ws" + strings.TrimPrefix(hs.URL, "http")
}

// Two real clients find each other and play a whole match over WebSocket;
// the result and a replayable log end up in storage.
func TestTwoClientsPlayAFullMatch(t *testing.T) {
	testFullMatch(t, 0)
}

// One client loses its connection mid-match and resumes it.
func TestClientReconnectsMidMatch(t *testing.T) {
	testFullMatch(t, 6)
}

func testFullMatch(t *testing.T, dropAfter uint64) {
	srv, store, url := newServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	bots := []*bot.Bot{
		{URL: url, Token: token(t, "11111111-1111-4111-8111-111111111111"), Hero: "death", DropAfterSeq: dropAfter},
		{URL: url, Token: token(t, "22222222-2222-4222-8222-222222222222"), Hero: "strength"},
	}
	results := make([]bot.Result, 2)
	errs := make([]error, 2)
	var wg sync.WaitGroup
	for i, b := range bots {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[i], errs[i] = b.Play(ctx)
		}()
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Fatalf("bot %d: %v", i, err)
		}
	}
	if results[0].MatchID == "" || results[0].MatchID != results[1].MatchID {
		t.Fatalf("different matches: %+v", results)
	}
	outcomes := results[0].Outcome + "/" + results[1].Outcome
	if outcomes != "win/loss" && outcomes != "loss/win" {
		t.Fatalf("outcomes %s (%s)", outcomes, results[0].Reason)
	}
	if results[0].Reason != game.ReasonNormal {
		t.Fatalf("reason %s", results[0].Reason)
	}

	var rec storage.MatchRecord
	deadline := time.Now().Add(3 * time.Second)
	for {
		var err error
		if rec, err = store.GetMatch(ctx, results[0].MatchID); err == nil || time.Now().After(deadline) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if rec.ID == "" || rec.WinnerID == "" || len(rec.Replay) == 0 || rec.ProtocolVersion != protocol.Version {
		t.Fatalf("stored %+v", rec)
	}
	replayed, err := game.Replay(game.DefaultRules(), rec.Seed,
		game.PlayerSetup{ID: rec.Player1ID, Hero: rec.Player1Hero},
		game.PlayerSetup{ID: rec.Player2ID, Hero: rec.Player2Hero}, rec.Replay)
	if err != nil {
		t.Fatal(err)
	}
	if replayed.WinnerID() != rec.WinnerID {
		t.Fatal("replay winner differs")
	}
	if srv.Matches().Len() != 0 {
		t.Fatal("finished match still registered")
	}
}

func dial(t *testing.T, ctx context.Context, url string) *websocket.Conn {
	t.Helper()
	c, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.CloseNow() })
	return c
}

func read(t *testing.T, ctx context.Context, c *websocket.Conn) map[string]any {
	t.Helper()
	var m map[string]any
	if err := wsjson.Read(ctx, c, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestHelloIsRequiredAndVerified(t *testing.T) {
	_, _, url := newServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	c := dial(t, ctx, url)
	_ = wsjson.Write(ctx, c, map[string]any{"type": "hello", "protocol": 1, "token": "forged"})
	if err := wsjson.Read(ctx, c, &map[string]any{}); websocket.CloseStatus(err) != websocket.StatusPolicyViolation {
		t.Fatalf("forged token: %v", err)
	}

	c = dial(t, ctx, url)
	_ = wsjson.Write(ctx, c, map[string]any{"type": "queue.join"})
	if err := wsjson.Read(ctx, c, &map[string]any{}); websocket.CloseStatus(err) != websocket.StatusPolicyViolation {
		t.Fatalf("no hello: %v", err)
	}

	c = dial(t, ctx, url)
	_ = wsjson.Write(ctx, c, map[string]any{"type": "hello", "protocol": 99, "token": token(t, "u1")})
	if err := wsjson.Read(ctx, c, &map[string]any{}); websocket.CloseStatus(err) != websocket.StatusPolicyViolation {
		t.Fatalf("old protocol: %v", err)
	}
}

func TestQueueAndStrangerActions(t *testing.T) {
	_, _, url := newServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	c := dial(t, ctx, url)
	_ = wsjson.Write(ctx, c, map[string]any{"type": "hello", "protocol": 1, "token": token(t, "u1")})
	if m := read(t, ctx, c); m["type"] != "hello" {
		t.Fatal(m)
	}
	_ = wsjson.Write(ctx, c, map[string]any{"type": "queue.join", "hero": "nobody", "ref": "x"})
	if m := read(t, ctx, c); m["type"] != "error" || m["payload"].(map[string]any)["code"] != "invalid_hero" {
		t.Fatal(m)
	}
	_ = wsjson.Write(ctx, c, map[string]any{"type": "queue.join", "hero": "death"})
	if m := read(t, ctx, c); m["type"] != "queue.waiting" {
		t.Fatal(m)
	}
	_ = wsjson.Write(ctx, c, map[string]any{"type": "card.play", "match_id": "someone-elses", "card_uid": "p1c1"})
	if m := read(t, ctx, c); m["payload"].(map[string]any)["code"] != "match_not_found" {
		t.Fatal(m)
	}
	_ = wsjson.Write(ctx, c, map[string]any{"type": "queue.leave"})
	if m := read(t, ctx, c); m["type"] != "queue.left" {
		t.Fatal(m)
	}
	if err := c.Write(ctx, websocket.MessageText, []byte("{not json")); err != nil {
		t.Fatal(err)
	}
	if m := read(t, ctx, c); m["payload"].(map[string]any)["code"] != "bad_request" {
		t.Fatal(m)
	}
}

func TestAPlayerOutsideTheMatchCannotAct(t *testing.T) {
	srv, _, url := newServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	join := func(user string) *websocket.Conn {
		c := dial(t, ctx, url)
		_ = wsjson.Write(ctx, c, map[string]any{"type": "hello", "protocol": 1, "token": token(t, user)})
		read(t, ctx, c)
		_ = wsjson.Write(ctx, c, map[string]any{"type": "queue.join", "hero": "death"})
		return c
	}
	join("a")
	join("b")
	var matchID string
	for range 100 {
		if all := srv.Matches().All(); len(all) == 1 {
			matchID = all[0].ID
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if matchID == "" {
		t.Fatal("no match")
	}
	mallory := dial(t, ctx, url)
	_ = wsjson.Write(ctx, mallory, map[string]any{"type": "hello", "protocol": 1, "token": token(t, "mallory")})
	read(t, ctx, mallory)
	_ = wsjson.Write(ctx, mallory, map[string]any{"type": "turn.end", "match_id": matchID})
	if m := read(t, ctx, mallory); m["payload"].(map[string]any)["code"] != "not_in_match" {
		t.Fatal(m)
	}
}

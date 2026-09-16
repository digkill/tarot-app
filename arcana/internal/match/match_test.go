package match

import (
	"sync"
	"testing"
	"time"

	"github.com/digkill/tarot-app/arcana/internal/game"
	"github.com/digkill/tarot-app/arcana/internal/protocol"
)

type recorder struct {
	mu   sync.Mutex
	msgs []protocol.ServerMessage
}

func (r *recorder) Send(m protocol.ServerMessage) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.msgs = append(r.msgs, m)
}

func (r *recorder) all() []protocol.ServerMessage {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]protocol.ServerMessage(nil), r.msgs...)
}

// waitFor polls until a frame matching pred arrives.
func (r *recorder) waitFor(t *testing.T, pred func(protocol.ServerMessage) bool) protocol.ServerMessage {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		for _, m := range r.all() {
			if pred(m) {
				return m
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out; got %d frames", len(r.all()))
	return protocol.ServerMessage{}
}

func ofType(typ string) func(protocol.ServerMessage) bool {
	return func(m protocol.ServerMessage) bool { return m.Type == typ }
}

func lastState(t *testing.T, r *recorder) protocol.StatePayload {
	t.Helper()
	r.waitFor(t, ofType(protocol.MatchState))
	var last protocol.StatePayload
	for _, m := range r.all() {
		if m.Type == protocol.MatchState {
			last = m.Payload.(protocol.StatePayload)
		}
	}
	return last
}

func testConfig() Config {
	return Config{TurnTime: time.Hour, MulliganTime: time.Hour, ReconnectWindow: time.Hour, EmoteCooldown: time.Hour}
}

func start(t *testing.T, cfg Config) (*Match, *recorder, *recorder, chan Result) {
	t.Helper()
	a, b := &recorder{}, &recorder{}
	results := make(chan Result, 1)
	m, err := Start("m1", 11, game.PlayerSetup{ID: "alice", Hero: "death"}, game.PlayerSetup{ID: "bob", Hero: "strength"},
		[2]Outbox{a, b}, cfg, func(r Result) { results <- r })
	if err != nil {
		t.Fatal(err)
	}
	return m, a, b, results
}

func waitResult(t *testing.T, results chan Result) Result {
	t.Helper()
	select {
	case r := <-results:
		return r
	case <-time.After(3 * time.Second):
		t.Fatal("match did not finish")
		return Result{}
	}
}

func TestStartSendsStartedAndState(t *testing.T) {
	_, a, b, _ := start(t, testConfig())
	started := a.waitFor(t, ofType(protocol.MatchStarted)).Payload.(protocol.StartedPayload)
	if started.You.ID != "alice" || started.Opponent.Hero != "strength" {
		t.Fatalf("%+v", started)
	}
	st := lastState(t, b)
	if st.State.Phase != game.PhaseMulligan || st.State.You.ID != "bob" || !st.OpponentConnected {
		t.Fatalf("%+v", st.State)
	}
}

func TestInvalidActionsAnswerWithErrorAndRef(t *testing.T) {
	m, a, _, _ := start(t, testConfig())
	lastState(t, a)
	m.Submit("alice", protocol.ClientMessage{Type: protocol.TurnEnd, Ref: "r1"})
	e := a.waitFor(t, ofType(protocol.Error)).Payload.(protocol.ErrorPayload)
	if e.Code != "wrong_phase" || e.Ref != "r1" {
		t.Fatalf("%+v", e)
	}
	if m.Submit("mallory", protocol.ClientMessage{Type: protocol.TurnEnd}) {
		t.Fatal("a stranger cannot submit")
	}
}

func TestMulliganTimeoutAndTurnTimeoutAdvanceTheGame(t *testing.T) {
	cfg := testConfig()
	cfg.MulliganTime = 30 * time.Millisecond
	cfg.TurnTime = 30 * time.Millisecond
	_, a, _, results := start(t, cfg)
	// Nobody acts: the mulligan is skipped, then every turn times out until
	// the timeout streak loses the match.
	r := waitResult(t, results)
	if r.Reason != game.ReasonTurnTimeout || r.WinnerID == "" {
		t.Fatalf("%+v", r)
	}
	fin := a.waitFor(t, ofType(protocol.MatchFinished)).Payload.(protocol.FinishedPayload)
	if fin.Reason != game.ReasonTurnTimeout {
		t.Fatal("finished frame")
	}
	var prev uint64
	for _, msg := range a.all() {
		if msg.Seq < prev {
			t.Fatalf("seq went backwards: %d after %d", msg.Seq, prev)
		}
		prev = msg.Seq
	}
}

func TestDisconnectedPlayerForfeitsAfterTheWindow(t *testing.T) {
	cfg := testConfig()
	cfg.ReconnectWindow = 40 * time.Millisecond
	m, a, b, results := start(t, cfg)
	lastState(t, a)
	m.Detach("alice", a)
	b.waitFor(t, ofType(protocol.OpponentDisconnected))
	r := waitResult(t, results)
	if r.Reason != game.ReasonDisconnectTimeout || r.WinnerID != "bob" {
		t.Fatalf("%+v", r)
	}
	<-m.Done()
	if m.Submit("bob", protocol.ClientMessage{Type: protocol.TurnEnd}) {
		t.Fatal("finished match accepted a command")
	}
}

func TestReconnectInTimeKeepsTheMatch(t *testing.T) {
	cfg := testConfig()
	cfg.ReconnectWindow = 80 * time.Millisecond
	m, a, b, results := start(t, cfg)
	lastState(t, a)
	m.Detach("alice", a)
	b.waitFor(t, ofType(protocol.OpponentDisconnected))

	// A stale connection detaching later must not knock the new one off.
	fresh := &recorder{}
	if !m.Attach("alice", fresh) {
		t.Fatal("attach")
	}
	m.Detach("alice", a)
	st := lastState(t, fresh)
	if st.State.You.ID != "alice" || len(st.State.You.Hand) == 0 {
		t.Fatal("resume must send the full state")
	}
	b.waitFor(t, ofType(protocol.OpponentReconnected))

	select {
	case r := <-results:
		t.Fatalf("match ended: %+v", r)
	case <-time.After(150 * time.Millisecond):
	}
}

func TestSurrenderFinishesAndReportsResults(t *testing.T) {
	m, a, b, results := start(t, testConfig())
	lastState(t, a)
	m.Submit("bob", protocol.ClientMessage{Type: protocol.Surrender})
	r := waitResult(t, results)
	if r.WinnerID != "alice" || r.Reason != game.ReasonSurrender || len(r.Log) != 1 || r.Seed != 11 {
		t.Fatalf("%+v", r)
	}
	if a.waitFor(t, ofType(protocol.MatchFinished)).Payload.(protocol.FinishedPayload).Result != "win" ||
		b.waitFor(t, ofType(protocol.MatchFinished)).Payload.(protocol.FinishedPayload).Result != "loss" {
		t.Fatal("results per player")
	}
}

func TestEmotesAreRelayedAndRateLimited(t *testing.T) {
	m, a, b, _ := start(t, testConfig())
	lastState(t, a)
	m.Submit("alice", protocol.ClientMessage{Type: protocol.PlayerEmote, Emote: "wow"})
	got := b.waitFor(t, ofType(protocol.PlayerEmote)).Payload.(protocol.EmotePayload)
	if got.Player != "alice" || got.Emote != "wow" {
		t.Fatal("relay")
	}
	m.Submit("alice", protocol.ClientMessage{Type: protocol.PlayerEmote, Emote: "wow"})
	if a.waitFor(t, ofType(protocol.Error)).Payload.(protocol.ErrorPayload).Code != "rate_limited" {
		t.Fatal("rate limit")
	}
	m.Submit("bob", protocol.ClientMessage{Type: protocol.PlayerEmote, Emote: "you are bad"})
	if b.waitFor(t, ofType(protocol.Error)).Payload.(protocol.ErrorPayload).Code != "invalid_emote" {
		t.Fatal("free text is not allowed")
	}
}

func TestRegistry(t *testing.T) {
	r := NewRegistry()
	m := &Match{ID: "x", Players: [2]game.PlayerSetup{{ID: "a"}, {ID: "b"}}}
	r.Add(m)
	if r.Get("x") != m || r.ForUser("b") != m || r.Len() != 1 {
		t.Fatal("add")
	}
	r.Remove("x")
	if r.Get("x") != nil || r.ForUser("a") != nil {
		t.Fatal("remove")
	}
}

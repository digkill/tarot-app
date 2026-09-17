// Package match runs live matches.
//
// Each match is an actor: one goroutine owns the game state and executes
// commands from a channel, one at a time, whichever player or timer they come
// from. Nothing else touches the game, so the engine needs no locks.
//
//	player A connection ─┐
//	                     ├─► commands ─► match goroutine ─► game engine
//	player B connection ─┘                      │
//	turn / reconnect timers ─────────────────────┘
package match

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/digkill/tarot-app/arcana/internal/game"
	"github.com/digkill/tarot-app/arcana/internal/protocol"
)

// Outbox delivers frames to one connection. Send must not block.
type Outbox interface {
	Send(protocol.ServerMessage)
}

type Config struct {
	TurnTime        time.Duration
	MulliganTime    time.Duration
	ReconnectWindow time.Duration
	// Minimum gap between emotes from one player.
	EmoteCooldown time.Duration
	Now           func() time.Time
}

func DefaultConfig() Config {
	return Config{
		TurnTime: 40 * time.Second, MulliganTime: 30 * time.Second,
		ReconnectWindow: 45 * time.Second, EmoteCooldown: 2 * time.Second, Now: time.Now,
	}
}

// Result is what is kept of a finished match.
type Result struct {
	ID         string
	Seed       uint64
	Rules      game.Rules
	Players    [2]game.PlayerSetup
	Decks      [2]string
	WinnerID   string
	Reason     game.Reason
	StartedAt  time.Time
	FinishedAt time.Time
	Turns      int
	Log        []game.Entry
}

type Match struct {
	ID      string
	Players [2]game.PlayerSetup
	// Art decks, per seat. Cosmetic: the engine never sees them.
	Decks [2]string

	cmds chan func()
	done chan struct{}

	// Owned by the match goroutine.
	cfg          Config
	g            *game.Game
	outs         [2]Outbox
	reconnectBy  [2]time.Time
	reconnectGen [2]uint64
	deadline     time.Time
	deadlineTurn int
	deadlinePh   game.Phase
	timerGen     uint64
	lastEmote    [2]time.Time
	startedAt    time.Time
	finished     bool
	onFinish     func(Result)
}

// New deals the game. The match does nothing until Run, so it can be
// registered before any player hears about it.
func New(id string, seed uint64, a, b game.PlayerSetup, outs [2]Outbox, cfg Config, onFinish func(Result)) (*Match, error) {
	g, err := game.New(seed, a, b)
	if err != nil {
		return nil, err
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &Match{
		ID: id, Players: [2]game.PlayerSetup{a, b},
		cmds: make(chan func(), 64), done: make(chan struct{}),
		cfg: cfg, g: g, outs: outs, onFinish: onFinish,
	}, nil
}

// Start is New followed by Run.
func Start(id string, seed uint64, a, b game.PlayerSetup, outs [2]Outbox, cfg Config, onFinish func(Result)) (*Match, error) {
	m, err := New(id, seed, a, b, outs, cfg, onFinish)
	if err != nil {
		return nil, err
	}
	m.Run()
	return m, nil
}

// Run starts the match goroutine and tells both players the match began.
// onFinish is called once, from that goroutine, with the final result.
func (m *Match) Run() {
	m.startedAt = m.cfg.Now()
	go m.run()
	m.post(func() {
		for i, out := range m.outs {
			if out == nil {
				m.startReconnectTimer(i)
				continue
			}
			out.Send(protocol.ServerMessage{Type: protocol.MatchStarted, MatchID: m.ID, Payload: protocol.StartedPayload{
				You:      protocol.PlayerInfo{ID: m.Players[i].ID, Hero: m.Players[i].Hero, Deck: m.Decks[i]},
				Opponent: protocol.PlayerInfo{ID: m.Players[1-i].ID, Hero: m.Players[1-i].Hero, Deck: m.Decks[1-i]},
			}})
		}
		m.afterChange()
	})
}

// Abort ends the match with no winner, e.g. when the server shuts down.
func (m *Match) Abort() {
	m.post(func() {
		if m.g.Abort() == nil {
			m.afterChange()
		}
	})
}

// Done is closed when the match has finished.
func (m *Match) Done() <-chan struct{} { return m.done }

// Has reports whether a user plays in this match.
func (m *Match) Has(userID string) bool { return m.seat(userID) >= 0 }

func (m *Match) seat(userID string) int {
	for i, p := range m.Players {
		if p.ID == userID {
			return i
		}
	}
	return -1
}

// post runs f in the match goroutine. False once the match is over.
func (m *Match) post(f func()) bool {
	select {
	case <-m.done:
		return false
	default:
	}
	select {
	case m.cmds <- f:
		return true
	case <-m.done:
		return false
	}
}

func (m *Match) run() {
	defer close(m.done)
	for f := range m.cmds {
		m.safely(f)
		if m.finished {
			return
		}
	}
}

// safely turns a panic in the engine into an aborted match instead of a
// crashed server.
func (m *Match) safely(f func()) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("arcana: match panic", "match", m.ID, "panic", fmt.Sprint(r))
			if !m.finished {
				_ = m.g.Abort()
				m.finish()
			}
		}
	}()
	f()
}

// Attach connects (or reconnects) a player and sends the full state.
func (m *Match) Attach(userID string, out Outbox) bool {
	i := m.seat(userID)
	if i < 0 {
		return false
	}
	return m.post(func() {
		wasAway := m.outs[i] == nil
		m.outs[i] = out
		m.reconnectGen[i]++
		m.reconnectBy[i] = time.Time{}
		m.sendState(i)
		if wasAway {
			m.sendTo(1-i, protocol.OpponentReconnected, nil)
		}
	})
}

// Detach marks a player's connection as lost, if it is still the current
// one, and gives them the reconnect window before they forfeit.
func (m *Match) Detach(userID string, out Outbox) {
	i := m.seat(userID)
	if i < 0 {
		return
	}
	m.post(func() {
		if m.outs[i] != out {
			return // an older connection; the player is already back
		}
		m.outs[i] = nil
		m.startReconnectTimer(i)
	})
}

func (m *Match) startReconnectTimer(i int) {
	m.reconnectGen[i]++
	gen := m.reconnectGen[i]
	m.reconnectBy[i] = m.cfg.Now().Add(m.cfg.ReconnectWindow)
	m.sendTo(1-i, protocol.OpponentDisconnected, protocol.DisconnectedPayload{ReconnectBy: m.reconnectBy[i].UnixMilli()})
	time.AfterFunc(m.cfg.ReconnectWindow, func() {
		m.post(func() {
			if m.reconnectGen[i] != gen || m.outs[i] != nil {
				return
			}
			_ = m.g.Forfeit(m.Players[i].ID, game.ReasonDisconnectTimeout)
			m.afterChange()
		})
	})
}

// Submit hands a player's intent to the match. False when the match is over.
func (m *Match) Submit(userID string, msg protocol.ClientMessage) bool {
	i := m.seat(userID)
	if i < 0 {
		return false
	}
	return m.post(func() { m.handle(i, msg) })
}

func (m *Match) handle(i int, msg protocol.ClientMessage) {
	id := m.Players[i].ID
	target := game.Target{}
	if msg.Target != nil {
		target = *msg.Target
	}
	var err error
	switch msg.Type {
	case protocol.Mulligan:
		err = m.g.Mulligan(id, msg.CardUIDs)
	case protocol.CardPlay:
		err = m.g.PlayCard(id, msg.CardUID, target)
	case protocol.HeroAbility:
		err = m.g.UseAbility(id, target)
	case protocol.HeroUltimate:
		err = m.g.UseUltimate(id, target)
	case protocol.TurnEnd:
		err = m.g.EndTurn(id)
	case protocol.Surrender:
		err = m.g.Surrender(id)
	case protocol.PlayerEmote:
		m.emote(i, msg)
		return
	default:
		err = &game.Error{Code: "unknown_type", Message: msg.Type}
	}
	if err != nil {
		code, text := "invalid_action", err.Error()
		if ge, ok := err.(*game.Error); ok {
			code, text = ge.Code, ge.Message
		}
		m.sendError(i, code, text, msg.Ref)
		return
	}
	m.afterChange()
}

func (m *Match) emote(i int, msg protocol.ClientMessage) {
	if !protocol.ValidEmote(msg.Emote) {
		m.sendError(i, "invalid_emote", "unknown emote", msg.Ref)
		return
	}
	now := m.cfg.Now()
	if now.Sub(m.lastEmote[i]) < m.cfg.EmoteCooldown {
		m.sendError(i, "rate_limited", "too many emotes", msg.Ref)
		return
	}
	m.lastEmote[i] = now
	for j := range m.outs {
		m.sendTo(j, protocol.PlayerEmote, protocol.EmotePayload{Player: m.Players[i].ID, Emote: msg.Emote})
	}
}

// afterChange runs after every accepted change: timers follow the phase and
// turn, everyone gets the new state, and a finished game ends the match.
func (m *Match) afterChange() {
	if m.g.Phase == game.PhaseFinished {
		m.finish()
		return
	}
	if m.g.Phase != m.deadlinePh || m.g.Turn != m.deadlineTurn {
		m.deadlinePh, m.deadlineTurn = m.g.Phase, m.g.Turn
		d := m.cfg.TurnTime
		if m.g.Phase == game.PhaseMulligan {
			d = m.cfg.MulliganTime
		}
		m.deadline = m.cfg.Now().Add(d)
		m.timerGen++
		gen := m.timerGen
		time.AfterFunc(d, func() {
			m.post(func() {
				if m.timerGen != gen {
					return
				}
				if m.g.Phase == game.PhaseMulligan {
					_ = m.g.SkipMulligans()
				} else {
					_ = m.g.TimeoutTurn()
				}
				m.afterChange()
			})
		})
	}
	for i := range m.outs {
		m.sendState(i)
	}
}

func (m *Match) finish() {
	m.finished = true
	m.timerGen++
	now := m.cfg.Now()
	for i := range m.outs {
		result := "none"
		switch m.g.WinnerID() {
		case "":
		case m.Players[i].ID:
			result = "win"
		default:
			result = "loss"
		}
		m.sendTo(i, protocol.MatchFinished, protocol.FinishedPayload{
			Winner: m.g.WinnerID(), Reason: m.g.Reason, Result: result, State: m.g.ViewFor(m.Players[i].ID),
		})
	}
	if m.onFinish != nil {
		m.onFinish(Result{
			ID: m.ID, Seed: m.g.Seed, Rules: m.g.Rules, Players: m.Players, Decks: m.Decks,
			WinnerID: m.g.WinnerID(), Reason: m.g.Reason, StartedAt: m.startedAt, FinishedAt: now,
			Turns: m.g.Turn, Log: m.g.Log(),
		})
	}
}

func (m *Match) sendState(i int) {
	now := m.cfg.Now()
	m.sendTo(i, protocol.MatchState, protocol.StatePayload{
		State: m.g.ViewFor(m.Players[i].ID), ServerTime: now.UnixMilli(),
		DeadlineAt: m.deadline.UnixMilli(), OpponentConnected: m.outs[1-i] != nil,
		YourDeck: m.Decks[i], OpponentDeck: m.Decks[1-i],
	})
}

func (m *Match) sendTo(i int, typ string, payload any) {
	if out := m.outs[i]; out != nil {
		out.Send(protocol.ServerMessage{Type: typ, Seq: m.g.Seq, MatchID: m.ID, Payload: payload})
	}
}

func (m *Match) sendError(i int, code, message, ref string) {
	m.sendTo(i, protocol.Error, protocol.ErrorPayload{Code: code, Message: message, Ref: ref})
}

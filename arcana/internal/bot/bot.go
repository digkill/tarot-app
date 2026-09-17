// Package bot is a protocol-level player: it connects like a real client,
// joins the queue and plays legal moves read from the state it receives. It
// exists for end-to-end tests and for trying the mobile client against
// someone.
package bot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"

	"github.com/digkill/tarot-app/arcana/internal/cards"
	"github.com/digkill/tarot-app/arcana/internal/game"
	"github.com/digkill/tarot-app/arcana/internal/protocol"
)

type frame struct {
	Type    string          `json:"type"`
	Seq     uint64          `json:"seq"`
	MatchID string          `json:"match_id"`
	Payload json.RawMessage `json:"payload"`
}

type Result struct {
	MatchID string
	Outcome string // win, loss, none
	Reason  game.Reason
	Seq     uint64
}

type Bot struct {
	URL   string
	Token string
	Hero  string
	// The art deck to be seen with, as a shop deck slug.
	Deck string
	// DropAfterSeq, when set, closes the connection once the match reaches
	// that sequence number and reconnects with match.resume — to exercise
	// reconnection.
	DropAfterSeq uint64
	// Pause before each move, so the bot stays within the server's
	// per-connection rate limit like a person would. Default 60ms.
	Think time.Duration
	Log   func(format string, args ...any)

	rng     *rand.Rand
	matchID string
	// Last sequence number acted on; -1 before any action.
	acted   int64
	last    game.View
	dropped bool
}

func (b *Bot) logf(format string, args ...any) {
	if b.Log != nil {
		b.Log(format, args...)
	}
}

// Play joins the queue and plays one match to the end.
func (b *Bot) Play(ctx context.Context) (Result, error) {
	b.rng = rand.New(rand.NewPCG(uint64(len(b.Token)), 7))
	b.acted = -1
	for {
		res, reconnect, err := b.session(ctx, b.matchID == "")
		if err != nil || !reconnect {
			return res, err
		}
		b.logf("reconnecting to %s", b.matchID)
	}
}

func (b *Bot) session(ctx context.Context, join bool) (Result, bool, error) {
	conn, _, err := websocket.Dial(ctx, b.URL, nil)
	if err != nil {
		return Result{}, false, err
	}
	defer conn.CloseNow()
	conn.SetReadLimit(1 << 20)

	send := func(m protocol.ClientMessage) error { return wsjson.Write(ctx, conn, m) }
	if err := send(protocol.ClientMessage{Type: protocol.Hello, Protocol: protocol.Version, Token: b.Token}); err != nil {
		return Result{}, false, err
	}
	for {
		var f frame
		if err := wsjson.Read(ctx, conn, &f); err != nil {
			return Result{}, false, fmt.Errorf("read: %w", err)
		}
		switch f.Type {
		case protocol.Hello:
			var h protocol.HelloPayload
			_ = json.Unmarshal(f.Payload, &h)
			switch {
			case join:
				err = send(protocol.ClientMessage{Type: protocol.QueueJoin, Hero: b.Hero, Deck: b.Deck})
			case h.ActiveMatch != "":
				err = send(protocol.ClientMessage{Type: protocol.MatchResume, MatchID: h.ActiveMatch})
			default:
				return Result{}, false, errors.New("nothing to resume")
			}
		case protocol.MatchStarted:
			b.matchID = f.MatchID
			var started protocol.StartedPayload
			_ = json.Unmarshal(f.Payload, &started)
			b.logf("match %s started: you %s/%q vs %s/%q", f.MatchID,
				started.You.Hero, started.You.Deck, started.Opponent.Hero, started.Opponent.Deck)
		case protocol.MatchState:
			var st protocol.StatePayload
			if err := json.Unmarshal(f.Payload, &st); err != nil {
				return Result{}, false, err
			}
			b.matchID = f.MatchID
			b.last = st.State
			if b.DropAfterSeq > 0 && !b.dropped && st.State.Seq >= b.DropAfterSeq {
				b.dropped = true
				b.logf("dropping connection at seq %d", st.State.Seq)
				return Result{}, true, nil
			}
			if msg, ok := b.decide(st.State); ok {
				think := b.Think
				if think == 0 {
					think = 60 * time.Millisecond
				}
				time.Sleep(think)
				msg.MatchID = f.MatchID
				err = send(msg)
			}
		case protocol.Error:
			var e protocol.ErrorPayload
			_ = json.Unmarshal(f.Payload, &e)
			b.logf("error %s: %s", e.Code, e.Message)
			if b.matchID != "" && b.last.Phase == game.PhasePlaying && b.last.ActivePlayer == b.last.You.ID &&
				e.Code != "not_your_turn" && e.Code != "rate_limited" {
				// A rejected move on our own turn: ending the turn keeps the match moving.
				err = send(protocol.ClientMessage{Type: protocol.TurnEnd, MatchID: b.matchID})
			}
		case protocol.MatchFinished:
			var fin protocol.FinishedPayload
			if err := json.Unmarshal(f.Payload, &fin); err != nil {
				return Result{}, false, err
			}
			conn.Close(websocket.StatusNormalClosure, "")
			return Result{MatchID: f.MatchID, Outcome: fin.Result, Reason: fin.Reason, Seq: f.Seq}, false, nil
		}
		if err != nil {
			return Result{}, false, err
		}
	}
}

// decide picks one action for a state, at most once per sequence number.
func (b *Bot) decide(v game.View) (protocol.ClientMessage, bool) {
	if int64(v.Seq) <= b.acted {
		return protocol.ClientMessage{}, false
	}
	switch {
	case v.Phase == game.PhaseMulligan && !v.You.Mulliganed:
		var swap []string
		for _, c := range v.You.Hand {
			if c.Cost >= 4 && len(swap) < 3 {
				swap = append(swap, c.UID)
			}
		}
		b.acted = int64(v.Seq)
		return protocol.ClientMessage{Type: protocol.Mulligan, CardUIDs: swap}, true
	case v.Phase != game.PhasePlaying || v.ActivePlayer != v.You.ID:
		return protocol.ClientMessage{}, false
	}
	b.acted = int64(v.Seq)

	if v.You.UltimateUsable {
		return protocol.ClientMessage{Type: protocol.HeroUltimate, Target: target(v, v.You.UltimateTargets, "")}, true
	}
	playable := []game.CardView{}
	for _, c := range v.You.Hand {
		if c.Playable {
			playable = append(playable, c)
		}
	}
	if len(playable) > 0 {
		c := playable[b.rng.IntN(len(playable))]
		return protocol.ClientMessage{Type: protocol.CardPlay, CardUID: c.UID, Target: target(v, c.Targets, c.UID)}, true
	}
	if v.You.AbilityUsable {
		return protocol.ClientMessage{Type: protocol.HeroAbility, Target: target(v, v.You.AbilityTargets, "")}, true
	}
	return protocol.ClientMessage{Type: protocol.TurnEnd}, true
}

func target(v game.View, kinds []cards.Target, self string) *game.Target {
	if len(kinds) == 0 || kinds[0] == "none" {
		return nil
	}
	t := &game.Target{Kind: kinds[0]}
	switch kinds[0] {
	case cards.TargetHandCard:
		for _, c := range v.You.Hand {
			if c.UID != self {
				t.CardUID = c.UID
				break
			}
		}
	case cards.TargetDiscardCard:
		if len(v.You.Discard) > 0 {
			t.CardUID = v.You.Discard[len(v.You.Discard)-1].UID
		}
	}
	return t
}

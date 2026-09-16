package fixturegen

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/digkill/tarot-app/arcana/internal/cards"
	"github.com/digkill/tarot-app/arcana/internal/game"
	"github.com/digkill/tarot-app/arcana/internal/protocol"
)

func TestWriteFixture(t *testing.T) {
	out := os.Getenv("FIXTURE_OUT")
	if out == "" {
		t.Skip("FIXTURE_OUT not set")
	}
	g, _ := game.New(5, game.PlayerSetup{ID: "me", Hero: "the_moon"}, game.PlayerSetup{ID: "them", Hero: "death"})
	_ = g.SkipMulligans()
	for g.ActivePlayerID() != "me" {
		_ = g.EndTurn(g.ActivePlayerID())
	}
	v := g.ViewFor("me")
	for _, c := range v.You.Hand {
		if c.Playable {
			target := game.Target{}
			if len(c.Targets) > 0 && c.Targets[0] != "none" {
				target.Kind = c.Targets[0]
			}
			if c.Targets[0] == cards.TargetDiscardCard || c.Targets[0] == cards.TargetHandCard {
				continue
			}
			_ = g.PlayCard("me", c.UID, target)
			break
		}
	}
	msg := protocol.ServerMessage{Type: protocol.MatchState, Seq: g.Seq, MatchID: "m-1", Payload: protocol.StatePayload{
		State: g.ViewFor("me"), ServerTime: 1_800_000_000_000, DeadlineAt: 1_800_000_030_000, OpponentConnected: true,
	}}
	data, _ := json.MarshalIndent(msg, "", "  ")
	if err := os.WriteFile(out, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

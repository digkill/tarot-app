// Package heroes declares the playable heroes as data. A hero's passive is
// named by id and implemented once in internal/game/passives.go; its ability
// and ultimate reuse the same operations as cards.
package heroes

import (
	"sort"

	"github.com/digkill/tarot-app/arcana/internal/cards"
)

// Power is a hero ability or ultimate.
type Power struct {
	// Ability: mana spent. Ultimate: charge needed (one charge per card
	// played), or souls when ChargeWithSouls is set.
	Cost            int        `json:"cost"`
	ChargeWithSouls bool       `json:"charge_with_souls,omitempty"`
	Side            cards.Side `json:"side"`
}

type Def struct {
	ID       string `json:"id"`
	HP       int    `json:"hp"`
	Passive  string `json:"passive"`
	Ability  Power  `json:"ability"`
	Ultimate Power  `json:"ultimate"`
}

var all = []Def{
	{
		ID: "the_magician", HP: 28, Passive: "magician_cheap_first",
		Ability: Power{Cost: 2, Side: cards.Side{Ops: []cards.Op{{Kind: "draw", Who: cards.WhoSelf, Amount: 1}}}},
		// Copies a chosen card in hand.
		Ultimate: Power{Cost: 5, Side: cards.Side{Targets: []cards.Target{cards.TargetHandCard},
			Ops: []cards.Op{{Kind: "copy_card"}}}},
	},
	{
		ID: "strength", HP: 34, Passive: "strength_might",
		Ability: Power{Cost: 2, Side: cards.Side{Ops: []cards.Op{{Kind: "armor", Who: cards.WhoSelf, Amount: 3}}}},
		// The next damage dealt is doubled.
		Ultimate: Power{Cost: 6, Side: cards.Side{Ops: []cards.Op{
			{Kind: "status", Who: cards.WhoSelf, Status: "fury", Amount: 2},
		}}},
	},
	{
		ID: "the_moon", HP: 27, Passive: "moon_shroud",
		Ability: Power{Cost: 1, Side: cards.Side{Targets: []cards.Target{cards.TargetEnemy}, Ops: []cards.Op{
			{Kind: "status", Status: "weakness", Amount: 2, Turns: 1},
		}}},
		// Hides the next three actions entirely and strikes.
		Ultimate: Power{Cost: 5, Side: cards.Side{Targets: []cards.Target{cards.TargetEnemy}, Ops: []cards.Op{
			{Kind: "status", Who: cards.WhoSelf, Status: "veil", Amount: 3},
			{Kind: "damage", Amount: 4, Element: cards.Magic},
		}}},
	},
	{
		ID: "death", HP: 30, Passive: "death_souls",
		Ability: Power{Cost: 2, Side: cards.Side{Targets: []cards.Target{cards.TargetEnemy}, Ops: []cards.Op{
			{Kind: "reap", Amount: 1},
		}}},
		// Spends every soul: damage per soul and the last discarded card back.
		Ultimate: Power{Cost: 3, ChargeWithSouls: true, Side: cards.Side{Targets: []cards.Target{cards.TargetEnemy}, Ops: []cards.Op{
			{Kind: "soul_harvest", Amount: 2},
		}}},
	},
}

var byID = func() map[string]Def {
	m := make(map[string]Def, len(all))
	for _, h := range all {
		m[h.ID] = h
	}
	return m
}()

func ByID(id string) (Def, bool) {
	h, ok := byID[id]
	return h, ok
}

func All() []Def {
	out := append([]Def(nil), all...)
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

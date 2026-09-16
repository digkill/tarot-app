// Package cards declares every Arcana Clash card as data.
//
// A card is a cost plus two sides — upright and reversed — and each side is a
// list of operations (`Op`) interpreted by the game engine's op registry. Adding
// a card means adding an entry here; adding a new *kind* of effect means
// registering one handler in internal/game/ops.go. Nothing in the transport or
// the UI knows about individual cards.
//
// Card ids are the image basenames the mobile clients already ship
// (`the_tower`, `ace_of_swords`), so names and art come from the existing
// tarot data in every language.
package cards

import "sort"

type Suit string

const (
	Swords    Suit = "swords"
	Cups      Suit = "cups"
	Wands     Suit = "wands"
	Pentacles Suit = "pentacles"
	Major     Suit = "major"
)

// Target is what a player picks when playing a card or using a hero power.
type Target string

const (
	TargetEnemy Target = "enemy"
	TargetSelf  Target = "self"
	// A card in the player's own hand (e.g. the Magician's ultimate).
	TargetHandCard Target = "hand_card"
	// A card in the player's own discard pile (e.g. Judgement).
	TargetDiscardCard Target = "discard_card"
)

// Element decides which modifiers apply to damage: physical damage is raised
// by Strength and triggers Riposte; magic is not.
type Element string

const (
	Physical Element = "physical"
	Magic    Element = "magic"
)

// Who an operation affects, relative to the player who played the card and
// the target they chose.
type Who string

const (
	WhoTarget   Who = "" // the chosen player target (the default)
	WhoSelf     Who = "self"
	WhoOpponent Who = "opponent"
	WhoBoth     Who = "both"
)

// Op is one step of an effect. Only the fields an op kind reads are set.
type Op struct {
	Kind    string  `json:"kind"`
	Who     Who     `json:"who,omitempty"`
	Amount  int     `json:"amount,omitempty"`
	Bonus   int     `json:"bonus,omitempty"`
	Turns   int     `json:"turns,omitempty"`
	Chance  int     `json:"chance,omitempty"` // percent, for critical hits
	Status  string  `json:"status,omitempty"`
	Element Element `json:"element,omitempty"`
	Pierce  bool    `json:"pierce,omitempty"`
}

// Side is one orientation of a card.
type Side struct {
	// Allowed targets, the first being the default. Empty means the card
	// needs no target.
	Targets []Target `json:"targets,omitempty"`
	Ops     []Op     `json:"ops"`
}

type Def struct {
	ID       string `json:"id"`
	Suit     Suit   `json:"suit"`
	Cost     int    `json:"cost"`
	Upright  Side   `json:"upright"`
	Reversed Side   `json:"reversed"`
}

func (d Def) IsMajor() bool { return d.Suit == Major }

// SideFor returns the side in play for a card's orientation.
func (d Def) SideFor(reversed bool) Side {
	if reversed {
		return d.Reversed
	}
	return d.Upright
}

// Op constructors keep the table below readable.

func dmg(n int, el Element) Op { return Op{Kind: "damage", Amount: n, Element: el} }
func crit(n, chance int) Op    { return Op{Kind: "damage", Amount: n, Element: Physical, Chance: chance} }
func pierce(n int) Op          { return Op{Kind: "damage", Amount: n, Element: Physical, Pierce: true} }
func heal(n int) Op            { return Op{Kind: "heal", Who: WhoSelf, Amount: n} }
func armor(n int) Op           { return Op{Kind: "armor", Who: WhoSelf, Amount: n} }
func draw(n int) Op            { return Op{Kind: "draw", Who: WhoSelf, Amount: n} }
func mana(n int) Op            { return Op{Kind: "mana", Who: WhoSelf, Amount: n} }
func drain(n int) Op           { return Op{Kind: "lifesteal", Amount: n, Element: Magic} }
func cleanse(who Who) Op       { return Op{Kind: "cleanse", Who: who} }

func status(who Who, name string, amount, turns int) Op {
	return Op{Kind: "status", Who: who, Status: name, Amount: amount, Turns: turns}
}

var (
	enemy   = []Target{TargetEnemy}
	self    = []Target{TargetSelf}
	anyone  = []Target{TargetEnemy, TargetSelf}
	discard = []Target{TargetDiscardCard}
)

// all is the rules-version-1 card pool.
var all = []Def{
	// Swords — physical damage, crits, piercing, counters.
	{ID: "ace_of_swords", Suit: Swords, Cost: 1,
		Upright:  Side{Targets: enemy, Ops: []Op{dmg(3, Physical)}},
		Reversed: Side{Targets: self, Ops: []Op{status(WhoSelf, "riposte", 3, 0)}}},
	{ID: "two_of_swords", Suit: Swords, Cost: 2,
		Upright:  Side{Targets: enemy, Ops: []Op{crit(4, 25)}},
		Reversed: Side{Targets: self, Ops: []Op{armor(3), status(WhoSelf, "riposte", 2, 0)}}},
	{ID: "five_of_swords", Suit: Swords, Cost: 2,
		Upright:  Side{Targets: enemy, Ops: []Op{pierce(3)}},
		Reversed: Side{Targets: enemy, Ops: []Op{status(WhoTarget, "vulnerable", 2, 2)}}},
	{ID: "seven_of_swords", Suit: Swords, Cost: 3,
		Upright:  Side{Targets: enemy, Ops: []Op{crit(5, 30)}},
		Reversed: Side{Targets: enemy, Ops: []Op{{Kind: "discard_random", Amount: 1}, dmg(1, Physical)}}},
	{ID: "knight_of_swords", Suit: Swords, Cost: 4,
		Upright:  Side{Targets: enemy, Ops: []Op{pierce(6)}},
		Reversed: Side{Targets: enemy, Ops: []Op{crit(5, 50), status(WhoSelf, "vulnerable", 2, 1)}}},
	{ID: "king_of_swords", Suit: Swords, Cost: 5,
		Upright:  Side{Targets: enemy, Ops: []Op{dmg(9, Physical)}},
		Reversed: Side{Targets: self, Ops: []Op{status(WhoSelf, "strength", 2, 0), armor(2)}}},

	// Cups — healing, regeneration, lifesteal, cleansing.
	{ID: "ace_of_cups", Suit: Cups, Cost: 1,
		Upright:  Side{Targets: self, Ops: []Op{heal(4)}},
		Reversed: Side{Targets: self, Ops: []Op{cleanse(WhoSelf), heal(1)}}},
	{ID: "two_of_cups", Suit: Cups, Cost: 2,
		Upright:  Side{Targets: self, Ops: []Op{heal(2), status(WhoSelf, "regen", 2, 2)}},
		Reversed: Side{Targets: enemy, Ops: []Op{drain(3)}}},
	{ID: "three_of_cups", Suit: Cups, Cost: 2,
		Upright:  Side{Targets: self, Ops: []Op{cleanse(WhoSelf), draw(1)}},
		Reversed: Side{Targets: self, Ops: []Op{status(WhoSelf, "regen", 3, 2)}}},
	{ID: "six_of_cups", Suit: Cups, Cost: 3,
		Upright:  Side{Targets: enemy, Ops: []Op{drain(4)}},
		Reversed: Side{Targets: self, Ops: []Op{heal(6)}}},
	{ID: "queen_of_cups", Suit: Cups, Cost: 4,
		Upright:  Side{Targets: self, Ops: []Op{heal(7), cleanse(WhoSelf)}},
		Reversed: Side{Targets: enemy, Ops: []Op{drain(6)}}},

	// Wands — magic damage, damage over time, empowering the next card.
	{ID: "ace_of_wands", Suit: Wands, Cost: 1,
		Upright:  Side{Targets: enemy, Ops: []Op{dmg(2, Magic), status(WhoSelf, "empower", 2, 0)}},
		Reversed: Side{Targets: enemy, Ops: []Op{status(WhoTarget, "burn", 2, 2)}}},
	{ID: "three_of_wands", Suit: Wands, Cost: 2,
		Upright:  Side{Targets: enemy, Ops: []Op{status(WhoTarget, "burn", 2, 3)}},
		Reversed: Side{Targets: enemy, Ops: []Op{dmg(4, Magic)}}},
	{ID: "five_of_wands", Suit: Wands, Cost: 3,
		Upright:  Side{Targets: enemy, Ops: []Op{{Kind: "combo_damage", Amount: 3, Bonus: 3, Status: "burn", Element: Magic}}},
		Reversed: Side{Targets: enemy, Ops: []Op{status(WhoTarget, "burn", 3, 2)}}},
	{ID: "eight_of_wands", Suit: Wands, Cost: 2,
		Upright:  Side{Targets: enemy, Ops: []Op{dmg(3, Magic), draw(1)}},
		Reversed: Side{Targets: self, Ops: []Op{status(WhoSelf, "empower", 3, 0)}}},
	{ID: "king_of_wands", Suit: Wands, Cost: 5,
		Upright:  Side{Targets: enemy, Ops: []Op{dmg(6, Magic), status(WhoTarget, "burn", 2, 2)}},
		Reversed: Side{Targets: self, Ops: []Op{status(WhoSelf, "empower", 4, 0), draw(2)}}},

	// Pentacles — armor, wards, extra energy.
	{ID: "ace_of_pentacles", Suit: Pentacles, Cost: 1,
		Upright:  Side{Targets: self, Ops: []Op{armor(4)}},
		Reversed: Side{Targets: self, Ops: []Op{status(WhoSelf, "surge", 2, 0)}}},
	{ID: "two_of_pentacles", Suit: Pentacles, Cost: 1,
		Upright:  Side{Targets: self, Ops: []Op{mana(1), draw(1)}},
		Reversed: Side{Targets: self, Ops: []Op{armor(2), mana(1)}}},
	{ID: "four_of_pentacles", Suit: Pentacles, Cost: 2,
		Upright:  Side{Targets: self, Ops: []Op{armor(6)}},
		Reversed: Side{Targets: self, Ops: []Op{status(WhoSelf, "ward", 2, 1)}}},
	{ID: "seven_of_pentacles", Suit: Pentacles, Cost: 3,
		Upright:  Side{Targets: self, Ops: []Op{armor(4), status(WhoSelf, "regen", 1, 3)}},
		Reversed: Side{Targets: self, Ops: []Op{status(WhoSelf, "surge", 3, 0), armor(2)}}},
	{ID: "king_of_pentacles", Suit: Pentacles, Cost: 4,
		Upright:  Side{Targets: self, Ops: []Op{armor(8), status(WhoSelf, "ward", 1, 1)}},
		Reversed: Side{Targets: self, Ops: []Op{armor(5), mana(2)}}},

	// Major Arcana — rare, each with its own mechanic.
	{ID: "the_fool", Suit: Major, Cost: 0,
		Upright:  Side{Ops: []Op{{Kind: "redraw_hand"}}},
		Reversed: Side{Ops: []Op{{Kind: "redraw_hand", Bonus: 1}, status(WhoSelf, "exhausted", 1, 0)}}},
	{ID: "the_magician", Suit: Major, Cost: 2,
		Upright:  Side{Ops: []Op{{Kind: "repeat_last", Who: WhoSelf}}},
		Reversed: Side{Ops: []Op{{Kind: "repeat_last", Who: WhoOpponent}}}},
	{ID: "the_tower", Suit: Major, Cost: 4,
		Upright:  Side{Targets: enemy, Ops: []Op{dmg(10, Magic), {Kind: "damage", Who: WhoSelf, Amount: 3, Element: Magic, Pierce: true}}},
		Reversed: Side{Targets: self, Ops: []Op{armor(12), status(WhoSelf, "exhausted", 2, 0)}}},
	{ID: "death", Suit: Major, Cost: 4,
		Upright:  Side{Ops: []Op{{Kind: "purge", Who: WhoBoth, Amount: 2}}},
		Reversed: Side{Targets: anyone, Ops: []Op{{Kind: "purge", Amount: 2, Bonus: 1}}}},
	{ID: "the_moon", Suit: Major, Cost: 2,
		Upright:  Side{Ops: []Op{status(WhoSelf, "veil", 2, 0), draw(1)}},
		Reversed: Side{Targets: enemy, Ops: []Op{status(WhoTarget, "confused", 1, 0), dmg(2, Magic)}}},
	{ID: "wheel_of_fortune", Suit: Major, Cost: 3,
		Upright:  Side{Ops: []Op{{Kind: "wheel"}}},
		Reversed: Side{Ops: []Op{{Kind: "discount_hand", Amount: 1}}}},
	{ID: "the_lovers", Suit: Major, Cost: 2,
		Upright:  Side{Ops: []Op{status(WhoSelf, "bond", 2, 0)}},
		Reversed: Side{Targets: self, Ops: []Op{heal(3), armor(3)}}},
	{ID: "judgement", Suit: Major, Cost: 3,
		Upright:  Side{Targets: discard, Ops: []Op{{Kind: "recall"}}},
		Reversed: Side{Ops: []Op{{Kind: "recycle"}, draw(1)}}},
	{ID: "the_world", Suit: Major, Cost: 6,
		Upright: Side{Targets: enemy, Ops: []Op{
			dmg(4, Physical), heal(4), status(WhoTarget, "burn", 2, 2), armor(4),
		}},
		Reversed: Side{Ops: []Op{mana(3), draw(2)}}},
}

var byID = func() map[string]Def {
	m := make(map[string]Def, len(all))
	for _, d := range all {
		m[d.ID] = d
	}
	return m
}()

func ByID(id string) (Def, bool) {
	d, ok := byID[id]
	return d, ok
}

// All returns the pool sorted by id, so iteration — and therefore any random
// pick from it — is identical on every run.
func All() []Def {
	out := append([]Def(nil), all...)
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// Minors and Majors split the pool for deck building.
func Minors() []Def { return filter(func(d Def) bool { return !d.IsMajor() }) }
func Majors() []Def { return filter(Def.IsMajor) }

func filter(keep func(Def) bool) []Def {
	var out []Def
	for _, d := range All() {
		if keep(d) {
			out = append(out, d)
		}
	}
	return out
}

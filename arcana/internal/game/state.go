// Package game is the Arcana Clash rules engine.
//
// It knows nothing about networks, clocks or databases: a match is created
// from a seed and two player setups, and advanced only by method calls
// (`Mulligan`, `PlayCard`, `EndTurn`, …). Every random decision goes through
// the game's own seeded generator, so replaying the recorded actions against
// the same seed rebuilds the exact same state (see Replay).
//
// A Game is not safe for concurrent use. The server gives each match a single
// goroutine that owns it.
package game

import (
	"fmt"
	"math/rand/v2"

	"github.com/digkill/tarot-app/arcana/internal/cards"
	"github.com/digkill/tarot-app/arcana/internal/heroes"
)

// RulesVersion changes whenever card numbers or rules change, so a stored
// replay is only ever re-run with the rules it was played under.
const RulesVersion = 1

type Rules struct {
	Version        int
	DeckSize       int
	Majors         int
	MaxCopies      int
	StartHand      int
	SecondBonus    int // extra starting cards for the player going second
	HandLimit      int
	BaseMana       int // max mana on a turn is BaseMana + turns taken, capped
	ManaCap        int
	ReversedChance int // percent
	MulliganMax    int
	// A player who lets the timer run out this many turns in a row loses.
	TimeoutStreak int
}

func DefaultRules() Rules {
	return Rules{
		Version: RulesVersion, DeckSize: 16, Majors: 2, MaxCopies: 2,
		StartHand: 4, SecondBonus: 1, HandLimit: 8, BaseMana: 2, ManaCap: 7,
		ReversedChance: 25, MulliganMax: 3, TimeoutStreak: 3,
	}
}

type Phase string

const (
	PhaseMulligan Phase = "mulligan"
	PhasePlaying  Phase = "playing"
	PhaseFinished Phase = "finished"
)

type Reason string

const (
	ReasonNormal            Reason = "normal"
	ReasonSurrender         Reason = "surrender"
	ReasonDisconnectTimeout Reason = "disconnect_timeout"
	ReasonTurnTimeout       Reason = "turn_timeout"
	ReasonServerError       Reason = "server_error"
)

// Target is the intent a client sends with an action.
type Target struct {
	Kind    cards.Target `json:"kind,omitempty"`
	CardUID string       `json:"card_uid,omitempty"`
}

type CardInstance struct {
	UID      string
	DefID    string
	Reversed bool
	// Cost change that lasts until the end of the owner's turn.
	TempCostMod int
	// Played under a veil: the opponent never learns what it was.
	Hidden bool
}

type Status struct {
	Name   string
	Amount int
	// Owner-independent duration: decremented at the start of each turn of
	// the player who applied it, removed at zero. Zero means "until used".
	Turns  int
	Source int
	Tag    string
}

type Player struct {
	ID      string
	Hero    heroes.Def
	index   int
	HP      int
	MaxHP   int
	Armor   int
	Mana    int
	MaxMana int

	Deck    []*CardInstance // index 0 is the top
	Hand    []*CardInstance
	Discard []*CardInstance

	Statuses []*Status

	Charge int // ultimate charge: one per card played
	Souls  int

	AbilityUsed   bool
	Mulliganed    bool
	TurnsTaken    int
	CardsThisTurn int
	Fatigue       int
	TimeoutStreak int
	PassiveStacks int

	lastPlayed *playedCard
	nextUID    int
}

type playedCard struct {
	def      cards.Def
	reversed bool
	target   Target
}

type PlayerSetup struct {
	ID   string
	Hero string
}

type Game struct {
	Rules   Rules
	Seed    uint64
	Seq     uint64
	Phase   Phase
	Turn    int
	Active  int
	Players [2]*Player
	Winner  int // player index, -1 while playing or with no winner
	Reason  Reason

	setup [2]PlayerSetup
	rng   *rand.Rand
	log   []Entry
}

// New deals a match: two random decks and opening hands from the seed. The
// game starts in the mulligan phase.
func New(seed uint64, a, b PlayerSetup) (*Game, error) {
	return NewWithRules(DefaultRules(), seed, a, b)
}

func NewWithRules(rules Rules, seed uint64, a, b PlayerSetup) (*Game, error) {
	if a.ID == "" || b.ID == "" || a.ID == b.ID {
		return nil, fmt.Errorf("game: two distinct players are required")
	}
	g := &Game{
		Rules: rules, Seed: seed, Phase: PhaseMulligan, Winner: -1,
		setup: [2]PlayerSetup{a, b},
		rng:   rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15)),
	}
	for i, s := range g.setup {
		hero, ok := heroes.ByID(s.Hero)
		if !ok {
			return nil, fmt.Errorf("game: unknown hero %q", s.Hero)
		}
		g.Players[i] = &Player{ID: s.ID, Hero: hero, index: i, HP: hero.HP, MaxHP: hero.HP}
	}
	for _, p := range g.Players {
		p.Deck = g.buildDeck(p)
	}
	g.Active = g.rng.IntN(2)
	for _, p := range g.Players {
		n := rules.StartHand
		if p.index != g.Active {
			n += rules.SecondBonus
		}
		for range n {
			g.drawCard(p, nil)
		}
	}
	return g, nil
}

// buildDeck picks distinct Major Arcana and random minors (at most MaxCopies
// of each), orients each card, and shuffles.
func (g *Game) buildDeck(p *Player) []*CardInstance {
	var defs []cards.Def
	majors := cards.Majors()
	g.rng.Shuffle(len(majors), func(i, j int) { majors[i], majors[j] = majors[j], majors[i] })
	defs = append(defs, majors[:min(g.Rules.Majors, len(majors))]...)

	minors := cards.Minors()
	copies := map[string]int{}
	for len(defs) < g.Rules.DeckSize {
		d := minors[g.rng.IntN(len(minors))]
		if copies[d.ID] >= g.Rules.MaxCopies {
			continue
		}
		copies[d.ID]++
		defs = append(defs, d)
	}
	deck := make([]*CardInstance, 0, len(defs))
	for _, d := range defs {
		deck = append(deck, g.newInstance(p, d.ID, g.rng.IntN(100) < g.Rules.ReversedChance))
	}
	g.rng.Shuffle(len(deck), func(i, j int) { deck[i], deck[j] = deck[j], deck[i] })
	return deck
}

func (g *Game) newInstance(p *Player, defID string, reversed bool) *CardInstance {
	p.nextUID++
	return &CardInstance{UID: fmt.Sprintf("p%dc%d", p.index+1, p.nextUID), DefID: defID, Reversed: reversed}
}

func (g *Game) player(id string) (*Player, error) {
	for _, p := range g.Players {
		if p.ID == id {
			return p, nil
		}
	}
	return nil, ErrUnknownPlayer
}

func (g *Game) opponent(p *Player) *Player { return g.Players[1-p.index] }

// PlayerIndex reports a player's seat, or -1.
func (g *Game) PlayerIndex(id string) int {
	for i, p := range g.Players {
		if p.ID == id {
			return i
		}
	}
	return -1
}

// ActivePlayerID is whose turn it is (meaningful in the playing phase).
func (g *Game) ActivePlayerID() string { return g.Players[g.Active].ID }

// WinnerID is empty until someone has won.
func (g *Game) WinnerID() string {
	if g.Winner < 0 {
		return ""
	}
	return g.Players[g.Winner].ID
}

// Log returns the full, unredacted action log — for storage and replay only,
// never for clients.
func (g *Game) Log() []Entry { return append([]Entry(nil), g.log...) }

func def(c *CardInstance) cards.Def {
	d, _ := cards.ByID(c.DefID)
	return d
}

func findCard(pile []*CardInstance, uid string) (int, *CardInstance) {
	for i, c := range pile {
		if c.UID == uid {
			return i, c
		}
	}
	return -1, nil
}

func removeAt(pile []*CardInstance, i int) []*CardInstance {
	return append(pile[:i:i], pile[i+1:]...)
}

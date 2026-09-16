package game

import "github.com/digkill/tarot-app/arcana/internal/cards"

// View is the match as one player may see it. It never contains the
// opponent's hand, either deck's order, hidden statuses or the generator.
type View struct {
	RulesVersion int          `json:"rules_version"`
	Seq          uint64       `json:"seq"`
	Phase        Phase        `json:"phase"`
	Turn         int          `json:"turn"`
	ActivePlayer string       `json:"active_player"`
	You          PlayerView   `json:"you"`
	Opponent     OpponentView `json:"opponent"`
	Log          []Entry      `json:"log"`
	Winner       string       `json:"winner,omitempty"`
	Reason       Reason       `json:"reason,omitempty"`
}

type CardView struct {
	UID      string `json:"uid"`
	CardID   string `json:"card_id,omitempty"`
	Reversed bool   `json:"reversed,omitempty"`
	Cost     int    `json:"cost"`
	BaseCost int    `json:"base_cost"`
	Hidden   bool   `json:"hidden,omitempty"`
	// Hand cards only: whether playing it now would be accepted, and with
	// which target kinds.
	Playable bool           `json:"playable,omitempty"`
	Targets  []cards.Target `json:"targets,omitempty"`
}

type StatusView struct {
	Name   string `json:"name"`
	Amount int    `json:"amount"`
	Turns  int    `json:"turns,omitempty"`
}

type HeroView struct {
	ID            string `json:"id"`
	HP            int    `json:"hp"`
	MaxHP         int    `json:"max_hp"`
	Armor         int    `json:"armor"`
	Charge        int    `json:"charge"`
	UltimateCost  int    `json:"ultimate_cost"`
	Souls         int    `json:"souls"`
	UltimateReady bool   `json:"ultimate_ready"`
}

type PlayerView struct {
	ID          string       `json:"id"`
	Hero        HeroView     `json:"hero"`
	Mana        int          `json:"mana"`
	MaxMana     int          `json:"max_mana"`
	Hand        []CardView   `json:"hand"`
	DeckCount   int          `json:"deck_count"`
	Discard     []CardView   `json:"discard"`
	Statuses    []StatusView `json:"statuses"`
	AbilityCost int          `json:"ability_cost"`
	// Whether the ability or ultimate would be accepted now, and their targets.
	AbilityUsable   bool           `json:"ability_usable"`
	AbilityTargets  []cards.Target `json:"ability_targets,omitempty"`
	UltimateUsable  bool           `json:"ultimate_usable"`
	UltimateTargets []cards.Target `json:"ultimate_targets,omitempty"`
	Mulliganed      bool           `json:"mulliganed"`
	TimeoutStreak   int            `json:"timeout_streak"`
}

type OpponentView struct {
	ID         string       `json:"id"`
	Hero       HeroView     `json:"hero"`
	Mana       int          `json:"mana"`
	MaxMana    int          `json:"max_mana"`
	HandCount  int          `json:"hand_count"`
	DeckCount  int          `json:"deck_count"`
	Discard    []CardView   `json:"discard"`
	Statuses   []StatusView `json:"statuses"`
	Mulliganed bool         `json:"mulliganed"`
}

// ViewLogSize is how many recent log entries a view carries.
const ViewLogSize = 20

// ViewFor builds a player's view. An unknown id gets a zero View.
func (g *Game) ViewFor(playerID string) View {
	idx := g.PlayerIndex(playerID)
	if idx < 0 {
		return View{}
	}
	me, them := g.Players[idx], g.Players[1-idx]
	v := View{
		RulesVersion: g.Rules.Version, Seq: g.Seq, Phase: g.Phase, Turn: g.Turn,
		ActivePlayer: g.Players[g.Active].ID, Winner: g.WinnerID(), Reason: g.Reason,
	}
	myTurn := g.Phase == PhasePlaying && g.Active == idx

	v.You = PlayerView{
		ID: me.ID, Hero: heroView(me), Mana: me.Mana, MaxMana: me.MaxMana,
		DeckCount: len(me.Deck), Discard: g.pileView(me.Discard, true),
		Statuses: statusViews(me, true), AbilityCost: me.Hero.Ability.Cost,
		Mulliganed: me.Mulliganed, TimeoutStreak: me.TimeoutStreak,
		Hand: make([]CardView, 0, len(me.Hand)),
	}
	for _, c := range me.Hand {
		cv := CardView{UID: c.UID, CardID: c.DefID, Reversed: c.Reversed, Cost: g.cost(me, c), BaseCost: def(c).Cost}
		if myTurn && cv.Cost <= me.Mana {
			cv.Targets = g.validTargetKinds(me, def(c).SideFor(c.Reversed), c)
			cv.Playable = len(cv.Targets) > 0
		}
		v.You.Hand = append(v.You.Hand, cv)
	}
	if myTurn {
		if !me.AbilityUsed && me.Hero.Ability.Cost <= me.Mana {
			v.You.AbilityTargets = g.validTargetKinds(me, me.Hero.Ability.Side, nil)
			v.You.AbilityUsable = len(v.You.AbilityTargets) > 0
		}
		if ultimateReady(me) {
			v.You.UltimateTargets = g.validTargetKinds(me, me.Hero.Ultimate.Side, nil)
			v.You.UltimateUsable = len(v.You.UltimateTargets) > 0
		}
	}

	v.Opponent = OpponentView{
		ID: them.ID, Hero: heroView(them), Mana: them.Mana, MaxMana: them.MaxMana,
		HandCount: len(them.Hand), DeckCount: len(them.Deck),
		Discard: g.pileView(them.Discard, false), Statuses: statusViews(them, false),
		Mulliganed: them.Mulliganed,
	}

	start := max(len(g.log)-ViewLogSize, 0)
	v.Log = make([]Entry, 0, len(g.log)-start)
	for _, e := range g.log[start:] {
		v.Log = append(v.Log, e.redacted(me.ID))
	}
	return v
}

// EntriesSince returns the redacted entries after seq, for incremental updates.
func (g *Game) EntriesSince(playerID string, seq uint64) []Entry {
	var out []Entry
	for _, e := range g.log {
		if e.Seq > seq {
			out = append(out, e.redacted(playerID))
		}
	}
	return out
}

// validTargetKinds lists the target kinds that would pass validation. A
// "none" target is reported as an empty-kind entry being valid: the list then
// contains the single pseudo-kind "none".
func (g *Game) validTargetKinds(p *Player, side cards.Side, self *CardInstance) []cards.Target {
	if len(side.Targets) == 0 {
		if _, err := g.validateTarget(p, side, Target{}, self); err == nil {
			return []cards.Target{"none"}
		}
		return nil
	}
	var out []cards.Target
	for _, kind := range side.Targets {
		t := Target{Kind: kind}
		switch kind {
		case cards.TargetHandCard:
			for _, c := range p.Hand {
				if c != self {
					t.CardUID = c.UID
					break
				}
			}
		case cards.TargetDiscardCard:
			if len(p.Discard) > 0 {
				t.CardUID = p.Discard[0].UID
			}
		}
		if _, err := g.validateTarget(p, side, t, self); err == nil {
			out = append(out, kind)
		}
	}
	return out
}

func heroView(p *Player) HeroView {
	return HeroView{
		ID: p.Hero.ID, HP: p.HP, MaxHP: p.MaxHP, Armor: p.Armor, Charge: p.Charge,
		UltimateCost: p.Hero.Ultimate.Cost, Souls: p.Souls, UltimateReady: ultimateReady(p),
	}
}

func (g *Game) pileView(pile []*CardInstance, owner bool) []CardView {
	out := make([]CardView, 0, len(pile))
	for _, c := range pile {
		if c.Hidden && !owner {
			out = append(out, CardView{UID: c.UID, Hidden: true})
			continue
		}
		out = append(out, CardView{UID: c.UID, CardID: c.DefID, Reversed: c.Reversed, Cost: def(c).Cost, BaseCost: def(c).Cost})
	}
	return out
}

func statusViews(p *Player, owner bool) []StatusView {
	out := make([]StatusView, 0, len(p.Statuses))
	for _, s := range p.Statuses {
		if statusDefs[s.Name].Hidden && !owner {
			continue
		}
		out = append(out, StatusView{Name: s.Name, Amount: s.Amount, Turns: s.Turns})
	}
	return out
}

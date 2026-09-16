package game

import (
	"github.com/digkill/tarot-app/arcana/internal/cards"
)

// effect is the context one card, ability or ultimate resolves in.
type effect struct {
	g      *Game
	e      *Entry
	src    *Player
	target Target
	// Empower bonus, added to the first amount-based op.
	bonus int
	// Souls spent by an ultimate paid with souls.
	souls int
	// Guards against a repeat repeating a repeat.
	depth int
}

func (fx *effect) amount(base int) int {
	n := base + fx.bonus
	fx.bonus = 0
	return n
}

// players resolves who an op applies to.
func (fx *effect) players(who cards.Who) []*Player {
	enemy := fx.g.opponent(fx.src)
	switch who {
	case cards.WhoSelf:
		return []*Player{fx.src}
	case cards.WhoOpponent:
		return []*Player{enemy}
	case cards.WhoBoth:
		return []*Player{fx.src, enemy}
	default:
		if fx.target.Kind == cards.TargetSelf {
			return []*Player{fx.src}
		}
		return []*Player{enemy}
	}
}

func (fx *effect) one(who cards.Who) *Player { return fx.players(who)[0] }

type opHandler struct {
	run func(fx *effect, op cards.Op)
	// Checked before anything is paid; an error makes the action invalid.
	require func(g *Game, p *Player, op cards.Op, t Target, self *CardInstance) error
	// Whether an empower bonus applies to this op.
	usesAmount bool
}

// ops is the effect registry: every op kind a card or hero may use.
var ops map[string]opHandler

func init() {
	ops = map[string]opHandler{
		"damage": {usesAmount: true, run: func(fx *effect, op cards.Op) {
			for _, p := range fx.players(op.Who) {
				fx.g.dealDamage(fx.e, fx.src, p, fx.amount(op.Amount), op.Element, op.Pierce, op.Chance, fx)
			}
		}},
		"combo_damage": {usesAmount: true, run: func(fx *effect, op cards.Op) {
			p := fx.one(op.Who)
			n := fx.amount(op.Amount)
			if p.status(op.Status) != nil {
				n += op.Bonus
			}
			fx.g.dealDamage(fx.e, fx.src, p, n, op.Element, op.Pierce, op.Chance, fx)
		}},
		"lifesteal": {usesAmount: true, run: func(fx *effect, op cards.Op) {
			dealt := fx.g.dealDamage(fx.e, fx.src, fx.one(op.Who), fx.amount(op.Amount), op.Element, op.Pierce, 0, fx)
			fx.g.heal(fx.e, fx.src, dealt)
		}},
		"heal": {usesAmount: true, run: func(fx *effect, op cards.Op) {
			for _, p := range fx.players(op.Who) {
				fx.g.heal(fx.e, p, fx.amount(op.Amount))
			}
		}},
		"armor": {usesAmount: true, run: func(fx *effect, op cards.Op) {
			for _, p := range fx.players(op.Who) {
				n := fx.amount(op.Amount)
				p.Armor += n
				fx.e.add(Event{Kind: "armor", Player: p.ID, Amount: n})
			}
		}},
		"status": {run: func(fx *effect, op cards.Op) {
			for _, p := range fx.players(op.Who) {
				fx.g.addStatus(fx.e, fx.src, p, op.Status, op.Amount, op.Turns)
			}
		}},
		"cleanse": {run: func(fx *effect, op cards.Op) {
			for _, p := range fx.players(op.Who) {
				fx.g.strip(fx.e, p, func(s *Status) bool { return statusDefs[s.Name].Negative })
			}
		}},
		"draw": {run: func(fx *effect, op cards.Op) {
			for _, p := range fx.players(op.Who) {
				for range op.Amount {
					fx.g.drawCard(p, fx.e)
				}
			}
		}},
		"mana": {run: func(fx *effect, op cards.Op) {
			for _, p := range fx.players(op.Who) {
				p.Mana += op.Amount
				fx.e.add(Event{Kind: "mana", Player: p.ID, Amount: op.Amount})
			}
		}},
		"discard_random": {run: func(fx *effect, op cards.Op) {
			p := fx.one(op.Who)
			for range op.Amount {
				if len(p.Hand) == 0 {
					return
				}
				i := fx.g.rng.IntN(len(p.Hand))
				c := p.Hand[i]
				p.Hand = removeAt(p.Hand, i)
				p.Discard = append(p.Discard, c)
				fx.e.add(Event{Kind: "destroyed", Player: p.ID, CardID: c.DefID, CardUID: c.UID})
				fx.g.notifyRemoved(fx.e, 1)
			}
		}},
		// The Fool: discard the hand and draw as many (plus Bonus).
		"redraw_hand": {run: func(fx *effect, op cards.Op) {
			p := fx.src
			n := len(p.Hand)
			p.Discard = append(p.Discard, p.Hand...)
			p.Hand = nil
			fx.e.add(Event{Kind: "hand_discarded", Player: p.ID, Amount: n})
			for range n + op.Bonus {
				fx.g.drawCard(p, fx.e)
			}
		}},
		// The Magician: resolve the last card played again.
		"repeat_last": {
			require: func(g *Game, p *Player, op cards.Op, t Target, self *CardInstance) error {
				source := p
				if op.Who == cards.WhoOpponent {
					source = g.opponent(p)
				}
				if source.lastPlayed == nil {
					return ErrNothingToRepeat
				}
				return nil
			},
			run: func(fx *effect, op cards.Op) {
				source := fx.src
				if op.Who == cards.WhoOpponent {
					source = fx.g.opponent(fx.src)
				}
				last := source.lastPlayed
				if last == nil || fx.depth > 0 {
					return
				}
				side := last.def.SideFor(last.reversed)
				fx.e.add(Event{Kind: "repeat", Player: fx.src.ID, CardID: last.def.ID})
				sub := &effect{g: fx.g, e: fx.e, src: fx.src, target: defaultTarget(side), depth: fx.depth + 1}
				if last.target.Kind != "" && allows(side, last.target.Kind) {
					sub.target = last.target
				}
				sub.run(side)
			},
		},
		// Death: wipe statuses; damage (or, reversed, heal) per status removed.
		"purge": {run: func(fx *effect, op cards.Op) {
			removed := 0
			for _, p := range fx.players(op.Who) {
				removed += fx.g.strip(fx.e, p, func(*Status) bool { return true })
			}
			if removed == 0 {
				return
			}
			if op.Bonus > 0 {
				fx.g.heal(fx.e, fx.src, op.Amount*removed)
			} else {
				fx.g.dealDamage(fx.e, fx.src, fx.g.opponent(fx.src), op.Amount*removed, cards.Magic, false, 0, fx)
			}
		}},
		// Wheel of Fortune: a new random hand whose costs shift by -1..+1 this turn.
		"wheel": {run: func(fx *effect, op cards.Op) {
			p := fx.src
			n := len(p.Hand)
			p.Discard = append(p.Discard, p.Hand...)
			p.Hand = nil
			fx.e.add(Event{Kind: "hand_discarded", Player: p.ID, Amount: n})
			minors := cards.Minors()
			for range n {
				d := minors[fx.g.rng.IntN(len(minors))]
				c := fx.g.newInstance(p, d.ID, fx.g.rng.IntN(100) < fx.g.Rules.ReversedChance)
				c.TempCostMod = fx.g.rng.IntN(3) - 1
				p.Hand = append(p.Hand, c)
				fx.e.add(Event{Kind: "card_added", Player: p.ID, CardID: c.DefID, CardUID: c.UID, private: true})
			}
		}},
		"discount_hand": {run: func(fx *effect, op cards.Op) {
			for _, c := range fx.src.Hand {
				c.TempCostMod -= op.Amount
			}
			fx.e.add(Event{Kind: "hand_discounted", Player: fx.src.ID, Amount: op.Amount})
		}},
		// Judgement: a chosen card from the discard pile back to hand.
		"recall": {
			require: func(g *Game, p *Player, op cards.Op, t Target, self *CardInstance) error {
				if t.Kind != cards.TargetDiscardCard {
					return ErrInvalidTarget
				}
				if _, c := findCard(p.Discard, t.CardUID); c == nil {
					return ErrInvalidTarget
				}
				if len(p.Hand) > g.Rules.HandLimit-1 { // the played card leaves first
					return ErrHandFull
				}
				return nil
			},
			run: func(fx *effect, op cards.Op) {
				p := fx.src
				i, c := findCard(p.Discard, fx.target.CardUID)
				if c == nil {
					return
				}
				p.Discard = removeAt(p.Discard, i)
				c.Hidden, c.TempCostMod = false, 0
				p.Hand = append(p.Hand, c)
				fx.e.add(Event{Kind: "recalled", Player: p.ID, CardID: c.DefID, CardUID: c.UID})
			},
		},
		"recycle": {run: func(fx *effect, op cards.Op) {
			p := fx.src
			p.Deck = append(p.Deck, p.Discard...)
			n := len(p.Discard)
			p.Discard = nil
			fx.g.rng.Shuffle(len(p.Deck), func(i, j int) { p.Deck[i], p.Deck[j] = p.Deck[j], p.Deck[i] })
			fx.e.add(Event{Kind: "recycled", Player: p.ID, Amount: n})
		}},
		// The Magician's ultimate: a copy of a card in hand.
		"copy_card": {
			require: func(g *Game, p *Player, op cards.Op, t Target, self *CardInstance) error {
				if t.Kind != cards.TargetHandCard {
					return ErrInvalidTarget
				}
				if _, c := findCard(p.Hand, t.CardUID); c == nil || c == self {
					return ErrInvalidTarget
				}
				if len(p.Hand) >= g.Rules.HandLimit {
					return ErrHandFull
				}
				return nil
			},
			run: func(fx *effect, op cards.Op) {
				p := fx.src
				_, c := findCard(p.Hand, fx.target.CardUID)
				if c == nil {
					return
				}
				dup := fx.g.newInstance(p, c.DefID, c.Reversed)
				p.Hand = append(p.Hand, dup)
				fx.e.add(Event{Kind: "card_added", Player: p.ID, CardID: dup.DefID, CardUID: dup.UID, private: true})
			},
		},
		// Death's ability: take one helpful status from the target, or strike.
		"reap": {run: func(fx *effect, op cards.Op) {
			p := fx.one(op.Who)
			var positive []*Status
			for _, s := range p.Statuses {
				if !statusDefs[s.Name].Negative {
					positive = append(positive, s)
				}
			}
			if len(positive) == 0 {
				fx.g.dealDamage(fx.e, fx.src, p, op.Amount, cards.Magic, false, 0, fx)
				return
			}
			victim := positive[fx.g.rng.IntN(len(positive))]
			fx.g.strip(fx.e, p, func(s *Status) bool { return s == victim })
		}},
		// Death's ultimate: damage per soul spent, and the last discarded card back.
		"soul_harvest": {run: func(fx *effect, op cards.Op) {
			fx.g.dealDamage(fx.e, fx.src, fx.one(op.Who), op.Amount*fx.souls, cards.Magic, false, 0, fx)
			p := fx.src
			if n := len(p.Discard); n > 0 && len(p.Hand) < fx.g.Rules.HandLimit {
				c := p.Discard[n-1]
				p.Discard = p.Discard[:n-1]
				c.Hidden, c.TempCostMod = false, 0
				p.Hand = append(p.Hand, c)
				fx.e.add(Event{Kind: "recalled", Player: p.ID, CardID: c.DefID, CardUID: c.UID})
			}
		}},
	}
}

func (fx *effect) run(side cards.Side) {
	for _, op := range side.Ops {
		if h, ok := ops[op.Kind]; ok {
			h.run(fx, op)
		}
	}
}

func allows(side cards.Side, kind cards.Target) bool {
	for _, t := range side.Targets {
		if t == kind {
			return true
		}
	}
	return false
}

func defaultTarget(side cards.Side) Target {
	if len(side.Targets) == 0 {
		return Target{}
	}
	switch side.Targets[0] {
	case cards.TargetEnemy, cards.TargetSelf:
		return Target{Kind: side.Targets[0]}
	}
	return Target{}
}

func usesAmount(side cards.Side) bool {
	for _, op := range side.Ops {
		if ops[op.Kind].usesAmount {
			return true
		}
	}
	return false
}

// validateTarget checks a target against a side and every op's requirement.
// An empty target is filled with the default when the side has a single
// player target.
func (g *Game) validateTarget(p *Player, side cards.Side, t Target, self *CardInstance) (Target, error) {
	if t.Kind == "" {
		t = defaultTarget(side)
	}
	if len(side.Targets) == 0 {
		if t.Kind != "" || t.CardUID != "" {
			return t, ErrInvalidTarget
		}
	} else {
		if !allows(side, t.Kind) {
			return t, ErrInvalidTarget
		}
		if (t.Kind == cards.TargetEnemy || t.Kind == cards.TargetSelf) && t.CardUID != "" {
			return t, ErrInvalidTarget
		}
	}
	for _, op := range side.Ops {
		h, ok := ops[op.Kind]
		if !ok {
			return t, &Error{"unknown_effect", op.Kind}
		}
		if h.require != nil {
			if err := h.require(g, p, op, t, self); err != nil {
				return t, err
			}
		}
	}
	return t, nil
}

// dealDamage applies every modifier and returns the damage done to armor and
// HP together.
func (g *Game) dealDamage(e *Entry, src, dst *Player, amount int, el cards.Element, pierce bool, critChance int, fx *effect) int {
	if g.Phase == PhaseFinished {
		return 0
	}
	n := amount
	hostile := src != dst
	if hostile && el == cards.Physical {
		n += src.statusAmount("strength")
	}
	if hostile {
		n -= src.statusAmount("weakness")
	}
	crit := false
	if critChance > 0 && g.rng.IntN(100) < critChance {
		n *= 2
		crit = true
	}
	if hostile && src.status("fury") != nil && n > 0 {
		n *= src.statusAmount("fury")
		g.removeStatus(e, src, "fury")
	}
	n += dst.statusAmount("vulnerable")
	n -= dst.statusAmount("ward")
	n = max(n, 0)

	absorbed := 0
	if !pierce {
		absorbed = min(dst.Armor, n)
		dst.Armor -= absorbed
	}
	hpLoss := min(n-absorbed, dst.HP)
	dst.HP -= n - absorbed
	e.add(Event{Kind: "damage", Player: dst.ID, Amount: n, Absorbed: absorbed, Crit: crit, Element: string(el)})

	if hpLoss > 0 {
		if hook := dst.passive().onHPLoss; hook != nil {
			hook(g, e, dst, hpLoss)
		}
	}
	if hostile && el == cards.Physical && n > 0 && dst.status("riposte") != nil {
		counter := dst.statusAmount("riposte")
		g.removeStatus(e, dst, "riposte")
		e.add(Event{Kind: "riposte", Player: dst.ID, Amount: counter})
		g.dealDamage(e, dst, src, counter, cards.Magic, false, 0, nil)
	}
	return n
}

func (g *Game) heal(e *Entry, p *Player, amount int) {
	if amount <= 0 || g.Phase == PhaseFinished {
		return
	}
	n := min(amount, p.MaxHP-p.HP)
	p.HP += n
	e.add(Event{Kind: "heal", Player: p.ID, Amount: n})
}

// drawCard takes the top card; an empty deck costs fatigue damage and a full
// hand destroys the card. e is nil only while dealing opening hands.
func (g *Game) drawCard(p *Player, e *Entry) {
	if len(p.Deck) == 0 {
		p.Fatigue++
		if e != nil {
			e.add(Event{Kind: "fatigue", Player: p.ID, Amount: p.Fatigue})
			g.dealDamage(e, p, p, p.Fatigue, cards.Magic, true, 0, nil)
		}
		return
	}
	c := p.Deck[0]
	p.Deck = p.Deck[1:]
	if len(p.Hand) >= g.Rules.HandLimit {
		p.Discard = append(p.Discard, c)
		if e != nil {
			e.add(Event{Kind: "destroyed", Player: p.ID, CardID: c.DefID, CardUID: c.UID})
			g.notifyRemoved(e, 1)
		}
		return
	}
	p.Hand = append(p.Hand, c)
	if e != nil {
		e.add(Event{Kind: "draw", Player: p.ID, CardID: c.DefID, CardUID: c.UID, private: true})
	}
}

package game

import "github.com/digkill/tarot-app/arcana/internal/cards"

// statusDef describes a status effect. Behaviour that reads a status (damage
// modifiers, mana at turn start) looks it up by name; this table only holds
// what is generic.
type statusDef struct {
	Negative bool
	// Hidden from the opponent's view.
	Hidden bool
	// Runs at the start of the owner's turn.
	Tick func(g *Game, owner *Player, s *Status, e *Entry)
}

var statusDefs map[string]statusDef

func init() {
	statusDefs = map[string]statusDef{
		"burn": {Negative: true, Tick: func(g *Game, owner *Player, s *Status, e *Entry) {
			g.dealDamage(e, g.Players[s.Source], owner, s.Amount, cards.Magic, true, 0, nil)
		}},
		"regen": {Tick: func(g *Game, owner *Player, s *Status, e *Entry) {
			g.heal(e, owner, s.Amount)
		}},
		"strength":   {},
		"weakness":   {Negative: true},
		"vulnerable": {Negative: true},
		"ward":       {},
		"riposte":    {},
		"empower":    {},
		"fury":       {},
		"surge":      {},
		"exhausted":  {Negative: true},
		"confused":   {Negative: true},
		"veil":       {Hidden: true},
		"bond":       {Hidden: true},
	}
}

func (p *Player) status(name string) *Status {
	for _, s := range p.Statuses {
		if s.Name == name {
			return s
		}
	}
	return nil
}

func (p *Player) statusAmount(name string) int {
	if s := p.status(name); s != nil {
		return s.Amount
	}
	return 0
}

// addStatus stacks with an existing status of the same name: amounts add up,
// the longer duration wins.
func (g *Game) addStatus(e *Entry, source, owner *Player, name string, amount, turns int) {
	if s := owner.status(name); s != nil {
		s.Amount += amount
		s.Turns = max(s.Turns, turns)
		s.Source = source.index
	} else {
		owner.Statuses = append(owner.Statuses, &Status{Name: name, Amount: amount, Turns: turns, Source: source.index})
	}
	e.add(Event{Kind: "status_added", Player: owner.ID, Status: name, Amount: amount, private: statusDefs[name].Hidden})
}

// removeStatus drops a status that was used up; it does not count as removal
// by an effect.
func (g *Game) removeStatus(e *Entry, owner *Player, name string) {
	for i, s := range owner.Statuses {
		if s.Name == name {
			owner.Statuses = append(owner.Statuses[:i:i], owner.Statuses[i+1:]...)
			e.add(Event{Kind: "status_removed", Player: owner.ID, Status: name, private: statusDefs[name].Hidden})
			return
		}
	}
}

// consume uses one unit of a status and removes it when none is left.
func (g *Game) consume(e *Entry, owner *Player, name string) {
	s := owner.status(name)
	if s == nil {
		return
	}
	s.Amount--
	if s.Amount <= 0 {
		g.removeStatus(e, owner, name)
	}
}

// strip removes, by an effect, the statuses that match and reports how
// many went, for Death's souls.
func (g *Game) strip(e *Entry, owner *Player, match func(*Status) bool) int {
	var kept []*Status
	removed := 0
	for _, s := range owner.Statuses {
		if match(s) {
			removed++
			e.add(Event{Kind: "status_removed", Player: owner.ID, Status: s.Name, private: statusDefs[s.Name].Hidden})
		} else {
			kept = append(kept, s)
		}
	}
	owner.Statuses = kept
	if removed > 0 {
		g.notifyRemoved(e, removed)
	}
	return removed
}

// tickStatuses runs at the start of a player's turn: the owner's damage and
// healing over time first, then every status that player applied (on either
// side) loses a turn.
func (g *Game) tickStatuses(e *Entry, p *Player) {
	for _, s := range append([]*Status(nil), p.Statuses...) {
		if tick := statusDefs[s.Name].Tick; tick != nil && s.Turns > 0 {
			tick(g, p, s, e)
		}
	}
	expired := 0
	for _, owner := range g.Players {
		var kept []*Status
		for _, s := range owner.Statuses {
			if s.Source == p.index && s.Turns > 0 {
				s.Turns--
				if s.Turns == 0 {
					expired++
					e.add(Event{Kind: "status_removed", Player: owner.ID, Status: s.Name, private: statusDefs[s.Name].Hidden})
					continue
				}
			}
			kept = append(kept, s)
		}
		owner.Statuses = kept
	}
	if expired > 0 {
		g.notifyRemoved(e, expired)
	}
}

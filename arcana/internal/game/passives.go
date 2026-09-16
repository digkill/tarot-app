package game

// passive is a hero's always-on rule. Each hook is optional; the engine calls
// it at a fixed point. Adding a hero passive means adding one entry here.
type passive struct {
	// Change to a card's cost before it is played.
	costDelta func(g *Game, p *Player, c *CardInstance, cost int) int
	// After the hero lost HP.
	onHPLoss func(g *Game, e *Entry, p *Player, amount int)
	// Whether the next card play is hidden from the opponent.
	hidesCard func(g *Game, p *Player) bool
	// Statuses were removed or cards destroyed, anywhere on the table.
	onRemoved func(g *Game, e *Entry, p *Player, count int)
}

const mightCap = 5

var passives map[string]passive

func init() {
	passives = map[string]passive{
		// The Magician: the first card of the turn costing 2 or less is 1 cheaper.
		"magician_cheap_first": {
			costDelta: func(g *Game, p *Player, c *CardInstance, cost int) int {
				if p.CardsThisTurn == 0 && cost > 0 && cost <= 2 {
					return -1
				}
				return 0
			},
		},
		// Strength: every hit that gets through armor adds a point of strength,
		// up to five.
		"strength_might": {
			onHPLoss: func(g *Game, e *Entry, p *Player, amount int) {
				if p.PassiveStacks >= mightCap {
					return
				}
				p.PassiveStacks++
				g.addStatus(e, p, p, "strength", 1, 0)
			},
		},
		// The Moon: the first card of each turn is played face down.
		"moon_shroud": {
			hidesCard: func(g *Game, p *Player) bool { return p.CardsThisTurn == 0 },
		},
		// Death: a soul for every removed effect and destroyed card.
		"death_souls": {
			onRemoved: func(g *Game, e *Entry, p *Player, count int) {
				p.Souls += count
				e.add(Event{Kind: "souls", Player: p.ID, Amount: count})
			},
		},
	}
}

func (p *Player) passive() passive { return passives[p.Hero.Passive] }

func (g *Game) notifyRemoved(e *Entry, count int) {
	for _, p := range g.Players {
		if hook := p.passive().onRemoved; hook != nil {
			hook(g, e, p, count)
		}
	}
}

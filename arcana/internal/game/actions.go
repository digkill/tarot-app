package game

import (
	"github.com/digkill/tarot-app/arcana/internal/cards"
)

// Actions recorded in the log. Replay maps each back to a method call.
const (
	ActMulligan        = "mulligan"
	ActMulliganTimeout = "mulligan.timeout"
	ActPlayCard        = "card.play"
	ActAbility         = "hero.ability"
	ActUltimate        = "hero.ultimate"
	ActEndTurn         = "turn.end"
	ActTurnTimeout     = "turn.timeout"
	ActSurrender       = "surrender"
	ActForfeit         = "forfeit"
	ActAbort           = "abort"
)

// begin validates the common preconditions of a player action and opens a
// log entry for it.
func (g *Game) begin(playerID string, phase Phase, needTurn bool) (*Player, error) {
	p, err := g.player(playerID)
	if err != nil {
		return nil, err
	}
	if g.Phase == PhaseFinished {
		return nil, ErrMatchFinished
	}
	if g.Phase != phase {
		return nil, ErrWrongPhase
	}
	if needTurn && g.Active != p.index {
		return nil, ErrNotYourTurn
	}
	return p, nil
}

func (g *Game) commit(e *Entry) {
	g.Seq++
	e.Seq = g.Seq
	g.log = append(g.log, *e)
}

// Mulligan returns up to MulliganMax chosen cards to the deck and draws as
// many new ones. An empty list keeps the hand. Each player gets one; the
// first turn starts when both have decided.
func (g *Game) Mulligan(playerID string, cardUIDs []string) error {
	p, err := g.begin(playerID, PhaseMulligan, false)
	if err != nil {
		return err
	}
	if p.Mulliganed {
		return ErrAlreadyMulliganed
	}
	if len(cardUIDs) > g.Rules.MulliganMax {
		return ErrTooManyCards
	}
	seen := map[string]bool{}
	for _, uid := range cardUIDs {
		if _, c := findCard(p.Hand, uid); c == nil || seen[uid] {
			return ErrCardNotInHand
		}
		seen[uid] = true
	}

	e := &Entry{Player: p.ID, Action: ActMulligan, CardUIDs: append([]string(nil), cardUIDs...), private: true}
	var returned []*CardInstance
	for _, uid := range cardUIDs {
		i, c := findCard(p.Hand, uid)
		p.Hand = removeAt(p.Hand, i)
		returned = append(returned, c)
	}
	// Draw the replacements before shuffling the old cards back, so a card
	// is never swapped for itself.
	for range returned {
		g.drawCard(p, e)
	}
	p.Deck = append(p.Deck, returned...)
	g.rng.Shuffle(len(p.Deck), func(i, j int) { p.Deck[i], p.Deck[j] = p.Deck[j], p.Deck[i] })
	e.add(Event{Kind: "mulligan", Player: p.ID, Amount: len(returned)})
	p.Mulliganed = true

	g.maybeStart(e)
	g.commit(e)
	return nil
}

// SkipMulligans keeps the hands of everyone who has not decided — the
// server calls it when the mulligan timer runs out.
func (g *Game) SkipMulligans() error {
	if g.Phase != PhaseMulligan {
		return ErrWrongPhase
	}
	e := &Entry{Action: ActMulliganTimeout}
	for _, p := range g.Players {
		p.Mulliganed = true
	}
	g.maybeStart(e)
	g.commit(e)
	return nil
}

func (g *Game) maybeStart(e *Entry) {
	if !g.Players[0].Mulliganed || !g.Players[1].Mulliganed {
		return
	}
	g.Phase = PhasePlaying
	g.startTurn(e)
}

func (g *Game) startTurn(e *Entry) {
	p := g.Players[g.Active]
	g.Turn++
	p.TurnsTaken++
	p.AbilityUsed = false
	p.CardsThisTurn = 0
	p.MaxMana = min(g.Rules.BaseMana+p.TurnsTaken, g.Rules.ManaCap)
	p.Mana = p.MaxMana
	if s := p.status("surge"); s != nil {
		p.Mana += s.Amount
		g.removeStatus(e, p, "surge")
	}
	if s := p.status("exhausted"); s != nil {
		p.Mana = max(p.Mana-s.Amount, 0)
		g.removeStatus(e, p, "exhausted")
	}
	e.add(Event{Kind: "turn_start", Player: p.ID, Amount: g.Turn})
	g.tickStatuses(e, p)
	if g.checkDeath(e, p) {
		return
	}
	g.drawCard(p, e)
	g.checkDeath(e, p)
}

// cost is what a card costs this player right now.
func (g *Game) cost(p *Player, c *CardInstance) int {
	n := def(c).Cost + c.TempCostMod + p.statusAmount("confused")
	if delta := p.passive().costDelta; delta != nil {
		n += delta(g, p, c, n)
	}
	return max(n, 0)
}

// PlayCard plays a card from the player's hand. The server checks, in order:
// the player is in the match, the match is on, it is their turn, the card is
// in their hand, they can pay for it, and the target is allowed.
func (g *Game) PlayCard(playerID, cardUID string, target Target) error {
	p, err := g.begin(playerID, PhasePlaying, true)
	if err != nil {
		return err
	}
	i, c := findCard(p.Hand, cardUID)
	if c == nil {
		return ErrCardNotInHand
	}
	cost := g.cost(p, c)
	if cost > p.Mana {
		return ErrNotEnoughMana
	}
	d := def(c)
	side := d.SideFor(c.Reversed)
	target, err = g.validateTarget(p, side, target, c)
	if err != nil {
		return err
	}

	hidden := false
	if hide := p.passive().hidesCard; hide != nil && hide(g, p) {
		hidden = true
	}
	e := &Entry{Player: p.ID, Action: ActPlayCard, CardUID: c.UID, CardID: c.DefID, Reversed: c.Reversed, Target: &target}
	if !hidden && p.status("veil") != nil {
		hidden = true
		g.consume(e, p, "veil")
	}
	e.Hidden = hidden

	p.Mana -= cost
	p.Hand = removeAt(p.Hand, i)
	p.TimeoutStreak = 0
	g.removeStatus(e, p, "confused")

	fx := &effect{g: g, e: e, src: p, target: target}
	if usesAmount(side) && p.status("empower") != nil {
		fx.bonus = p.statusAmount("empower")
		g.removeStatus(e, p, "empower")
	}
	repeat := g.bondRepeats(e, p, d)

	fx.run(side)
	if repeat && g.Phase != PhaseFinished {
		e.add(Event{Kind: "bond", Player: p.ID})
		(&effect{g: g, e: e, src: p, target: target, depth: 1}).run(side)
	}

	c.TempCostMod = 0
	c.Hidden = hidden
	p.Discard = append(p.Discard, c)
	p.CardsThisTurn++
	p.Charge++
	if !isRepeatOrCardTargeted(side) {
		p.lastPlayed = &playedCard{def: d, reversed: c.Reversed, target: target}
	}
	g.checkDeath(e, p)
	g.commit(e)
	return nil
}

// bondRepeats applies The Lovers: the first card after it sets a suit, and a
// second card of that suit resolves twice.
func (g *Game) bondRepeats(e *Entry, p *Player, d cards.Def) bool {
	s := p.status("bond")
	if s == nil {
		return false
	}
	if s.Tag == "" {
		s.Tag = string(d.Suit)
		return false
	}
	match := s.Tag == string(d.Suit)
	g.removeStatus(e, p, "bond")
	return match
}

func isRepeatOrCardTargeted(side cards.Side) bool {
	for _, op := range side.Ops {
		if op.Kind == "repeat_last" {
			return true
		}
	}
	for _, t := range side.Targets {
		if t == cards.TargetHandCard || t == cards.TargetDiscardCard {
			return true
		}
	}
	return false
}

// UseAbility spends mana on the hero's ability, once per turn.
func (g *Game) UseAbility(playerID string, target Target) error {
	p, err := g.begin(playerID, PhasePlaying, true)
	if err != nil {
		return err
	}
	if p.AbilityUsed {
		return ErrAbilityUsed
	}
	power := p.Hero.Ability
	if power.Cost > p.Mana {
		return ErrNotEnoughMana
	}
	target, err = g.validateTarget(p, power.Side, target, nil)
	if err != nil {
		return err
	}
	e := &Entry{Player: p.ID, Action: ActAbility, Target: &target}
	if p.status("veil") != nil {
		e.Hidden = true
		g.consume(e, p, "veil")
	}
	p.Mana -= power.Cost
	p.AbilityUsed = true
	p.TimeoutStreak = 0
	(&effect{g: g, e: e, src: p, target: target}).run(power.Side)
	g.checkDeath(e, p)
	g.commit(e)
	return nil
}

// UseUltimate fires the hero's ultimate once it is charged.
func (g *Game) UseUltimate(playerID string, target Target) error {
	p, err := g.begin(playerID, PhasePlaying, true)
	if err != nil {
		return err
	}
	power := p.Hero.Ultimate
	if !ultimateReady(p) {
		return ErrUltimateNotReady
	}
	target, err = g.validateTarget(p, power.Side, target, nil)
	if err != nil {
		return err
	}
	e := &Entry{Player: p.ID, Action: ActUltimate, Target: &target}
	if p.status("veil") != nil {
		e.Hidden = true
		g.consume(e, p, "veil")
	}
	fx := &effect{g: g, e: e, src: p, target: target}
	if power.ChargeWithSouls {
		fx.souls = p.Souls
		p.Souls = 0
	} else {
		p.Charge = 0
	}
	p.TimeoutStreak = 0
	fx.run(power.Side)
	g.checkDeath(e, p)
	g.commit(e)
	return nil
}

func ultimateReady(p *Player) bool {
	if p.Hero.Ultimate.ChargeWithSouls {
		return p.Souls >= p.Hero.Ultimate.Cost
	}
	return p.Charge >= p.Hero.Ultimate.Cost
}

// EndTurn passes the turn to the opponent.
func (g *Game) EndTurn(playerID string) error {
	p, err := g.begin(playerID, PhasePlaying, true)
	if err != nil {
		return err
	}
	p.TimeoutStreak = 0
	e := &Entry{Player: p.ID, Action: ActEndTurn}
	g.passTurn(e, p)
	g.commit(e)
	return nil
}

// TimeoutTurn ends the active player's turn because their timer ran out.
// Too many in a row lose the match.
func (g *Game) TimeoutTurn() error {
	if g.Phase != PhasePlaying {
		return ErrWrongPhase
	}
	p := g.Players[g.Active]
	p.TimeoutStreak++
	e := &Entry{Player: p.ID, Action: ActTurnTimeout}
	if p.TimeoutStreak >= g.Rules.TimeoutStreak {
		g.finish(e, g.opponent(p).index, ReasonTurnTimeout)
	} else {
		g.passTurn(e, p)
	}
	g.commit(e)
	return nil
}

func (g *Game) passTurn(e *Entry, p *Player) {
	for _, c := range p.Hand {
		c.TempCostMod = 0
	}
	g.Active = g.opponent(p).index
	g.startTurn(e)
}

// Surrender concedes the match.
func (g *Game) Surrender(playerID string) error {
	return g.Forfeit(playerID, ReasonSurrender)
}

// Forfeit ends the match against a player: surrender, or the server giving up
// on a player who did not reconnect in time.
func (g *Game) Forfeit(playerID string, reason Reason) error {
	p, err := g.player(playerID)
	if err != nil {
		return err
	}
	if g.Phase == PhaseFinished {
		return ErrMatchFinished
	}
	action := ActForfeit
	if reason == ReasonSurrender {
		action = ActSurrender
	}
	e := &Entry{Player: p.ID, Action: action, Reason: reason}
	g.finish(e, g.opponent(p).index, reason)
	g.commit(e)
	return nil
}

// Abort ends the match with no winner, e.g. after a server fault.
func (g *Game) Abort() error {
	if g.Phase == PhaseFinished {
		return ErrMatchFinished
	}
	e := &Entry{Action: ActAbort, Reason: ReasonServerError}
	g.finish(e, -1, ReasonServerError)
	g.commit(e)
	return nil
}

// checkDeath ends the match when a hero has fallen. If both fell to the same
// action, the player who took it loses.
func (g *Game) checkDeath(e *Entry, actor *Player) bool {
	if g.Phase == PhaseFinished {
		return true
	}
	enemy := g.opponent(actor)
	switch {
	case actor.HP <= 0:
		g.finish(e, enemy.index, ReasonNormal)
	case enemy.HP <= 0:
		g.finish(e, actor.index, ReasonNormal)
	default:
		return false
	}
	return true
}

func (g *Game) finish(e *Entry, winner int, reason Reason) {
	g.Phase = PhaseFinished
	g.Winner = winner
	g.Reason = reason
	ev := Event{Kind: "finished", Reason: string(reason)}
	if winner >= 0 {
		ev.Player = g.Players[winner].ID
	}
	e.add(ev)
}

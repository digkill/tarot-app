package game

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/digkill/tarot-app/arcana/internal/cards"
)

const (
	alice = "alice"
	bob   = "bob"
)

var (
	enemyT = Target{Kind: cards.TargetEnemy}
	selfT  = Target{Kind: cards.TargetSelf}
	none   = Target{}
)

// newPlaying returns a game past the mulligan with alice to act, both hands
// emptied and plenty of mana, so each test sets up exactly what it needs.
func newPlaying(t *testing.T, heroA, heroB string) (*Game, *Player, *Player) {
	t.Helper()
	g, err := New(42, PlayerSetup{ID: alice, Hero: heroA}, PlayerSetup{ID: bob, Hero: heroB})
	if err != nil {
		t.Fatal(err)
	}
	if err := g.SkipMulligans(); err != nil {
		t.Fatal(err)
	}
	a, b := g.Players[0], g.Players[1]
	g.Active = 0
	for _, p := range g.Players {
		p.Hand = nil
		p.Mana, p.MaxMana = 10, 10
		p.CardsThisTurn = 0
		p.Statuses = nil
		p.Armor = 0
		p.Charge = 0
		p.HP = p.MaxHP
	}
	return g, a, b
}

func give(g *Game, p *Player, id string, reversed bool) *CardInstance {
	if _, ok := cards.ByID(id); !ok {
		panic("unknown card " + id)
	}
	c := g.newInstance(p, id, reversed)
	p.Hand = append(p.Hand, c)
	return c
}

func mustPlay(t *testing.T, g *Game, player string, c *CardInstance, target Target) {
	t.Helper()
	if err := g.PlayCard(player, c.UID, target); err != nil {
		t.Fatalf("play %s: %v", c.DefID, err)
	}
}

func wantErr(t *testing.T, err, want error) {
	t.Helper()
	if !errors.Is(err, want) {
		t.Fatalf("error = %v, want %v", err, want)
	}
}

// MARK: - Setup

func TestNewDealsRandomDecksDeterministically(t *testing.T) {
	a, _ := New(7, PlayerSetup{alice, "strength"}, PlayerSetup{bob, "death"})
	b, _ := New(7, PlayerSetup{alice, "strength"}, PlayerSetup{bob, "death"})
	if !reflect.DeepEqual(a.ViewFor(alice), b.ViewFor(alice)) {
		t.Fatal("same seed must deal the same game")
	}
	c, _ := New(8, PlayerSetup{alice, "strength"}, PlayerSetup{bob, "death"})
	if reflect.DeepEqual(a.ViewFor(alice).You.Hand, c.ViewFor(alice).You.Hand) {
		t.Fatal("a different seed should deal differently")
	}

	rules := DefaultRules()
	for _, p := range a.Players {
		total := len(p.Deck) + len(p.Hand)
		if total != rules.DeckSize {
			t.Fatalf("deck size %d", total)
		}
		majors, copies := 0, map[string]int{}
		for _, c := range append(append([]*CardInstance{}, p.Deck...), p.Hand...) {
			if def(c).IsMajor() {
				majors++
			}
			copies[c.DefID]++
			if copies[c.DefID] > rules.MaxCopies {
				t.Fatalf("%s appears %d times", c.DefID, copies[c.DefID])
			}
		}
		if majors != rules.Majors {
			t.Fatalf("majors = %d", majors)
		}
	}
	first, second := a.Players[a.Active], a.Players[1-a.Active]
	if len(first.Hand) != rules.StartHand || len(second.Hand) != rules.StartHand+rules.SecondBonus {
		t.Fatalf("hands %d/%d", len(first.Hand), len(second.Hand))
	}
}

func TestNewRejectsBadSetup(t *testing.T) {
	if _, err := New(1, PlayerSetup{alice, "strength"}, PlayerSetup{alice, "death"}); err == nil {
		t.Fatal("same player twice")
	}
	if _, err := New(1, PlayerSetup{alice, "nobody"}, PlayerSetup{bob, "death"}); err == nil {
		t.Fatal("unknown hero")
	}
}

func TestEveryCardAndHeroUsesRegisteredEffects(t *testing.T) {
	for _, d := range cards.All() {
		for _, side := range []cards.Side{d.Upright, d.Reversed} {
			for _, op := range side.Ops {
				if _, ok := ops[op.Kind]; !ok {
					t.Errorf("%s: unknown op %q", d.ID, op.Kind)
				}
				if op.Kind == "status" {
					if _, ok := statusDefs[op.Status]; !ok {
						t.Errorf("%s: unknown status %q", d.ID, op.Status)
					}
				}
			}
		}
	}
	g, _, _ := newPlaying(t, "the_magician", "strength")
	for _, p := range g.Players {
		if _, ok := passives[p.Hero.Passive]; !ok {
			t.Errorf("%s: unknown passive", p.Hero.ID)
		}
	}
	if n := len(cards.All()); n < 20 || n > 30 {
		t.Errorf("MVP pool has %d cards", n)
	}
}

// MARK: - Mulligan

func TestMulliganReplacesChosenCardsOnce(t *testing.T) {
	g, _ := New(3, PlayerSetup{alice, "strength"}, PlayerSetup{bob, "death"})
	a := g.Players[0]
	before := len(a.Hand)
	chosen := []string{a.Hand[0].UID, a.Hand[1].UID}
	if err := g.Mulligan(alice, chosen); err != nil {
		t.Fatal(err)
	}
	if len(a.Hand) != before {
		t.Fatalf("hand size changed to %d", len(a.Hand))
	}
	for _, uid := range chosen {
		if _, c := findCard(a.Hand, uid); c != nil {
			t.Fatalf("%s was not replaced", uid)
		}
	}
	wantErr(t, g.Mulligan(alice, nil), ErrAlreadyMulliganed)
	if g.Phase != PhaseMulligan {
		t.Fatal("the match starts only after both players decide")
	}
	if err := g.Mulligan(bob, nil); err != nil {
		t.Fatal(err)
	}
	if g.Phase != PhasePlaying || g.Turn != 1 {
		t.Fatalf("phase %s turn %d", g.Phase, g.Turn)
	}
}

func TestMulliganValidation(t *testing.T) {
	g, _ := New(3, PlayerSetup{alice, "strength"}, PlayerSetup{bob, "death"})
	a := g.Players[0]
	uids := []string{a.Hand[0].UID, a.Hand[1].UID, a.Hand[2].UID, a.Hand[3].UID}
	wantErr(t, g.Mulligan(alice, uids), ErrTooManyCards)
	wantErr(t, g.Mulligan(alice, []string{g.Players[1].Hand[0].UID}), ErrCardNotInHand)
	wantErr(t, g.Mulligan(alice, []string{uids[0], uids[0]}), ErrCardNotInHand)
	wantErr(t, g.Mulligan("mallory", nil), ErrUnknownPlayer)
	wantErr(t, g.EndTurn(alice), ErrWrongPhase)
}

// MARK: - Turns and mana

func TestTurnSwitchingRefillsManaAndDraws(t *testing.T) {
	g, a, b := newPlaying(t, "strength", "death")
	bHand, bDeck, turn := len(b.Hand), len(b.Deck), g.Turn
	b.TurnsTaken = 2
	if err := g.EndTurn(alice); err != nil {
		t.Fatal(err)
	}
	if g.ActivePlayerID() != bob || g.Turn != turn+1 {
		t.Fatal("turn did not pass")
	}
	if b.MaxMana != 5 || b.Mana != 5 {
		t.Fatalf("mana %d/%d", b.Mana, b.MaxMana)
	}
	if len(b.Hand) != bHand+1 || len(b.Deck) != bDeck-1 {
		t.Fatal("no card drawn")
	}
	wantErr(t, g.EndTurn(alice), ErrNotYourTurn)
	c := give(g, a, "ace_of_swords", false)
	wantErr(t, g.PlayCard(alice, c.UID, enemyT), ErrNotYourTurn)

	b.TurnsTaken = 20
	if err := g.EndTurn(bob); err != nil {
		t.Fatal(err)
	}
	_ = g.EndTurn(alice)
	if b.MaxMana != DefaultRules().ManaCap {
		t.Fatalf("mana cap not applied: %d", b.MaxMana)
	}
}

func TestCardCostAndMana(t *testing.T) {
	g, a, b := newPlaying(t, "strength", "death")
	king := give(g, a, "king_of_swords", false)
	a.Mana = 4
	wantErr(t, g.PlayCard(alice, king.UID, enemyT), ErrNotEnoughMana)
	if len(a.Hand) != 1 || b.HP != b.MaxHP {
		t.Fatal("a rejected play must change nothing")
	}
	a.Mana = 5
	mustPlay(t, g, alice, king, enemyT)
	if a.Mana != 0 || b.HP != b.MaxHP-9 {
		t.Fatalf("mana %d hp %d", a.Mana, b.HP)
	}
}

func TestCardCannotBePlayedTwiceOrByTheOtherPlayer(t *testing.T) {
	g, a, b := newPlaying(t, "strength", "death")
	c := give(g, a, "ace_of_swords", false)
	mustPlay(t, g, alice, c, enemyT)
	wantErr(t, g.PlayCard(alice, c.UID, enemyT), ErrCardNotInHand)

	theirs := give(g, b, "ace_of_swords", false)
	wantErr(t, g.PlayCard(alice, theirs.UID, enemyT), ErrCardNotInHand)
	wantErr(t, g.PlayCard("mallory", theirs.UID, enemyT), ErrUnknownPlayer)
	if len(a.Discard) != 1 || a.Discard[0] != c {
		t.Fatal("played card goes to the discard pile")
	}
}

func TestInvalidTargetsAreRejected(t *testing.T) {
	g, a, b := newPlaying(t, "strength", "death")
	sword := give(g, a, "ace_of_swords", false)
	wantErr(t, g.PlayCard(alice, sword.UID, selfT), ErrInvalidTarget)
	wantErr(t, g.PlayCard(alice, sword.UID, Target{Kind: "hero_of_nowhere"}), ErrInvalidTarget)
	wantErr(t, g.PlayCard(alice, sword.UID, Target{Kind: cards.TargetEnemy, CardUID: "x"}), ErrInvalidTarget)

	fool := give(g, a, "the_fool", false)
	wantErr(t, g.PlayCard(alice, fool.UID, enemyT), ErrInvalidTarget)

	judgement := give(g, a, "judgement", false)
	wantErr(t, g.PlayCard(alice, judgement.UID, Target{Kind: cards.TargetDiscardCard, CardUID: "nope"}), ErrInvalidTarget)

	// An omitted target falls back to the card's only player target.
	mustPlay(t, g, alice, sword, none)
	if b.HP != b.MaxHP-3 {
		t.Fatal("default target not used")
	}
}

// MARK: - Damage, armor, healing

func TestArmorAbsorbsAndPierceIgnoresIt(t *testing.T) {
	g, a, b := newPlaying(t, "strength", "death")
	b.Armor = 2
	mustPlay(t, g, alice, give(g, a, "ace_of_swords", false), enemyT)
	if b.Armor != 0 || b.HP != b.MaxHP-1 {
		t.Fatalf("armor %d hp %d", b.Armor, b.HP)
	}
	b.Armor = 10
	mustPlay(t, g, alice, give(g, a, "five_of_swords", false), enemyT)
	if b.Armor != 10 || b.HP != b.MaxHP-4 {
		t.Fatalf("pierce: armor %d hp %d", b.Armor, b.HP)
	}
}

func TestHealingIsCappedAtMaxHP(t *testing.T) {
	g, a, _ := newPlaying(t, "strength", "death")
	a.HP = a.MaxHP - 2
	mustPlay(t, g, alice, give(g, a, "ace_of_cups", false), selfT)
	if a.HP != a.MaxHP {
		t.Fatalf("hp %d", a.HP)
	}
}

func TestLifestealHealsByDamageDealt(t *testing.T) {
	g, a, b := newPlaying(t, "strength", "death")
	a.HP = 10
	mustPlay(t, g, alice, give(g, a, "six_of_cups", false), enemyT)
	if b.HP != b.MaxHP-4 || a.HP != 14 {
		t.Fatalf("a %d b %d", a.HP, b.HP)
	}
}

func TestCriticalHitsComeFromTheSeededGenerator(t *testing.T) {
	crits := 0
	for seed := uint64(0); seed < 200; seed++ {
		g, err := New(seed, PlayerSetup{alice, "death"}, PlayerSetup{bob, "death"})
		if err != nil {
			t.Fatal(err)
		}
		_ = g.SkipMulligans()
		g.Active = 0
		a, b := g.Players[0], g.Players[1]
		a.Mana = 10
		mustPlay(t, g, alice, give(g, a, "knight_of_swords", true), enemyT) // 5 dmg, 50% crit
		switch b.MaxHP - b.HP {
		case 10:
			crits++
		case 5:
		default:
			t.Fatalf("unexpected damage %d", b.MaxHP-b.HP)
		}
	}
	if crits < 60 || crits > 140 {
		t.Fatalf("crit rate looks wrong: %d/200", crits)
	}
}

// MARK: - Status effects

func TestBurnTicksOnTheVictimsTurnsAndExpires(t *testing.T) {
	g, a, b := newPlaying(t, "death", "strength")
	mustPlay(t, g, alice, give(g, a, "ace_of_wands", true), enemyT) // burn 2 for 2 turns
	hp := b.HP
	_ = g.EndTurn(alice) // bob's turn: tick
	if b.HP != hp-2 {
		t.Fatalf("first tick: %d", hp-b.HP)
	}
	_ = g.EndTurn(bob) // alice's turn: duration 2 -> 1
	_ = g.EndTurn(alice)
	if b.HP != hp-4 {
		t.Fatalf("second tick: %d", hp-b.HP)
	}
	_ = g.EndTurn(bob) // 1 -> 0, removed
	if b.status("burn") != nil {
		t.Fatal("burn should have expired")
	}
	_ = g.EndTurn(alice)
	if b.HP != hp-4 {
		t.Fatal("no tick after expiry")
	}
}

func TestWardAndVulnerableModifyDamage(t *testing.T) {
	g, a, b := newPlaying(t, "death", "death")
	b.Statuses = []*Status{{Name: "ward", Amount: 2, Turns: 1, Source: 1}}
	mustPlay(t, g, alice, give(g, a, "ace_of_swords", false), enemyT)
	if b.HP != b.MaxHP-1 {
		t.Fatalf("ward: %d", b.MaxHP-b.HP)
	}
	b.HP, b.Statuses = b.MaxHP, nil
	mustPlay(t, g, alice, give(g, a, "five_of_swords", true), enemyT) // vulnerable 2
	mustPlay(t, g, alice, give(g, a, "ace_of_swords", false), enemyT)
	if b.HP != b.MaxHP-5 {
		t.Fatalf("vulnerable: %d", b.MaxHP-b.HP)
	}
	_ = g.EndTurn(alice)
	_ = g.EndTurn(bob)
	if b.status("vulnerable") == nil {
		t.Fatal("vulnerable lasts into the attacker's next turn")
	}
}

func TestEmpowerBoostsOnlyTheNextCard(t *testing.T) {
	g, a, b := newPlaying(t, "death", "death")
	mustPlay(t, g, alice, give(g, a, "ace_of_wands", false), enemyT)  // 2 dmg, empower 2
	mustPlay(t, g, alice, give(g, a, "ace_of_swords", false), enemyT) // 3+2
	mustPlay(t, g, alice, give(g, a, "ace_of_swords", false), enemyT) // 3
	if got := b.MaxHP - b.HP; got != 2+5+3 {
		t.Fatalf("damage %d", got)
	}
}

func TestRiposteCountersAPhysicalHitOnce(t *testing.T) {
	g, a, b := newPlaying(t, "death", "death")
	b.Statuses = []*Status{{Name: "riposte", Amount: 3, Source: 1}}
	mustPlay(t, g, alice, give(g, a, "ace_of_swords", false), enemyT)
	if a.HP != a.MaxHP-3 || b.status("riposte") != nil {
		t.Fatalf("riposte: alice %d", a.MaxHP-a.HP)
	}
	mustPlay(t, g, alice, give(g, a, "ace_of_swords", false), enemyT)
	if a.HP != a.MaxHP-3 {
		t.Fatal("riposte fired twice")
	}
}

func TestCleanseRemovesOnlyNegativeStatuses(t *testing.T) {
	g, a, _ := newPlaying(t, "strength", "strength")
	a.Statuses = []*Status{{Name: "burn", Amount: 2, Turns: 2, Source: 1}, {Name: "ward", Amount: 1, Turns: 1}}
	mustPlay(t, g, alice, give(g, a, "ace_of_cups", true), selfT)
	if a.status("burn") != nil || a.status("ward") == nil {
		t.Fatal("cleanse removed the wrong statuses")
	}
}

func TestExhaustedAndSurgeChangeNextTurnMana(t *testing.T) {
	g, a, b := newPlaying(t, "death", "death")
	mustPlay(t, g, alice, give(g, a, "the_tower", true), selfT) // armor 12, exhausted 2
	if a.Armor != 12 {
		t.Fatal("armor")
	}
	_ = g.EndTurn(alice)
	mustPlay(t, g, bob, give(g, b, "ace_of_pentacles", true), selfT) // surge 2
	_ = g.EndTurn(bob)
	if a.Mana != a.MaxMana-2 {
		t.Fatalf("exhausted: %d/%d", a.Mana, a.MaxMana)
	}
	_ = g.EndTurn(alice)
	if b.Mana != b.MaxMana+2 {
		t.Fatalf("surge: %d/%d", b.Mana, b.MaxMana)
	}
}

// MARK: - Major Arcana

func TestTowerHitsBothPlayers(t *testing.T) {
	g, a, b := newPlaying(t, "death", "death")
	a.Armor = 5
	mustPlay(t, g, alice, give(g, a, "the_tower", false), enemyT)
	if b.HP != b.MaxHP-10 || a.HP != a.MaxHP-3 || a.Armor != 5 {
		t.Fatalf("a %d/%d b %d", a.HP, a.Armor, b.HP)
	}
}

func TestFoolRedrawsTheHandForFree(t *testing.T) {
	g, a, _ := newPlaying(t, "death", "death")
	fool := give(g, a, "the_fool", false)
	old := []*CardInstance{give(g, a, "ace_of_cups", false), give(g, a, "ace_of_cups", false)}
	a.Mana = 0
	mustPlay(t, g, alice, fool, none)
	if len(a.Hand) != 2 {
		t.Fatalf("hand %d", len(a.Hand))
	}
	for _, c := range old {
		if _, found := findCard(a.Hand, c.UID); found != nil {
			t.Fatal("old card still in hand")
		}
	}
}

func TestMagicianRepeatsTheLastCard(t *testing.T) {
	g, a, b := newPlaying(t, "death", "death")
	magician := give(g, a, "the_magician", false)
	wantErr(t, g.PlayCard(alice, magician.UID, none), ErrNothingToRepeat)
	mustPlay(t, g, alice, give(g, a, "three_of_wands", true), enemyT) // 4 magic
	mustPlay(t, g, alice, magician, none)
	if b.HP != b.MaxHP-8 {
		t.Fatalf("damage %d", b.MaxHP-b.HP)
	}
	// Reversed: repeats the opponent's last card, aimed back at them.
	_ = g.EndTurn(alice)
	mustPlay(t, g, bob, give(g, b, "the_magician", true), none)
	if a.HP != a.MaxHP-4 {
		t.Fatalf("reversed magician: %d", a.MaxHP-a.HP)
	}
}

func TestDeathPurgesAndStrikesPerEffect(t *testing.T) {
	g, a, b := newPlaying(t, "the_moon", "the_moon")
	a.Statuses = []*Status{{Name: "burn", Amount: 1, Turns: 2, Source: 1}}
	b.Statuses = []*Status{{Name: "ward", Amount: 3, Turns: 1, Source: 1}, {Name: "regen", Amount: 1, Turns: 2, Source: 1}}
	mustPlay(t, g, alice, give(g, a, "death", false), none)
	if len(a.Statuses) != 0 || len(b.Statuses) != 0 {
		t.Fatal("statuses remain")
	}
	if b.HP != b.MaxHP-6 {
		t.Fatalf("damage %d", b.MaxHP-b.HP)
	}
}

func TestWheelReplacesTheHandWithTemporaryCosts(t *testing.T) {
	g, a, _ := newPlaying(t, "strength", "strength")
	wheel := give(g, a, "wheel_of_fortune", false)
	give(g, a, "king_of_swords", false)
	give(g, a, "king_of_swords", false)
	mustPlay(t, g, alice, wheel, none)
	if len(a.Hand) != 2 {
		t.Fatalf("hand %d", len(a.Hand))
	}
	for _, c := range a.Hand {
		if c.TempCostMod < -1 || c.TempCostMod > 1 || def(c).IsMajor() {
			t.Fatalf("bad wheel card %+v", c)
		}
	}
	_ = g.EndTurn(alice)
	for _, c := range a.Hand {
		if c.TempCostMod != 0 {
			t.Fatal("wheel costs last only this turn")
		}
	}
}

func TestLoversRepeatsASecondCardOfTheSameSuit(t *testing.T) {
	g, a, b := newPlaying(t, "death", "death")
	mustPlay(t, g, alice, give(g, a, "the_lovers", false), none)
	mustPlay(t, g, alice, give(g, a, "ace_of_swords", false), enemyT)
	mustPlay(t, g, alice, give(g, a, "ace_of_swords", false), enemyT) // resolves twice
	if got := b.MaxHP - b.HP; got != 9 {
		t.Fatalf("damage %d", got)
	}
	mustPlay(t, g, alice, give(g, a, "ace_of_swords", false), enemyT)
	if got := b.MaxHP - b.HP; got != 12 {
		t.Fatal("the bond is used up")
	}
}

func TestJudgementReturnsAChosenDiscard(t *testing.T) {
	g, a, _ := newPlaying(t, "death", "death")
	sword := give(g, a, "ace_of_swords", false)
	mustPlay(t, g, alice, sword, enemyT)
	j := give(g, a, "judgement", false)
	mustPlay(t, g, alice, j, Target{Kind: cards.TargetDiscardCard, CardUID: sword.UID})
	if _, c := findCard(a.Hand, sword.UID); c == nil {
		t.Fatal("card not recalled")
	}
}

func TestTheWorldCombinesAllSuits(t *testing.T) {
	g, a, b := newPlaying(t, "death", "death")
	a.HP = 20
	mustPlay(t, g, alice, give(g, a, "the_world", false), enemyT)
	if b.HP != b.MaxHP-4 || a.HP != 24 || a.Armor != 4 || b.status("burn") == nil {
		t.Fatalf("a %d/%d b %d", a.HP, a.Armor, b.HP)
	}
}

func TestMoonCardHidesTheNextPlayFromTheOpponent(t *testing.T) {
	g, a, _ := newPlaying(t, "death", "death")
	mustPlay(t, g, alice, give(g, a, "the_moon", false), none)
	secret := give(g, a, "king_of_swords", false)
	mustPlay(t, g, alice, secret, enemyT)

	theirs := g.ViewFor(bob)
	last := theirs.Log[len(theirs.Log)-1]
	if !last.Hidden || last.CardID != "" || last.CardUID != "" {
		t.Fatalf("bob sees %+v", last)
	}
	for _, s := range theirs.Opponent.Statuses {
		if s.Name == "veil" {
			t.Fatal("veil is hidden from the opponent")
		}
	}
	if d := theirs.Opponent.Discard[len(theirs.Opponent.Discard)-1]; d.CardID != "" || !d.Hidden {
		t.Fatal("the veiled card must stay face down in the discard pile")
	}
	mine := g.ViewFor(alice)
	if mine.Log[len(mine.Log)-1].CardID != "king_of_swords" {
		t.Fatal("the player still sees their own card")
	}
}

// MARK: - Heroes

func TestMagicianPassiveDiscountsFirstCheapCard(t *testing.T) {
	g, a, _ := newPlaying(t, "the_magician", "death")
	first := give(g, a, "five_of_swords", false)
	second := give(g, a, "five_of_swords", false)
	if g.cost(a, first) != 1 {
		t.Fatalf("cost %d", g.cost(a, first))
	}
	a.Mana = 3
	mustPlay(t, g, alice, first, enemyT)
	if g.cost(a, second) != 2 || a.Mana != 2 {
		t.Fatal("only the first cheap card is discounted")
	}
}

func TestMagicianUltimateCopiesACard(t *testing.T) {
	g, a, _ := newPlaying(t, "the_magician", "death")
	c := give(g, a, "king_of_wands", false)
	wantErr(t, g.UseUltimate(alice, Target{Kind: cards.TargetHandCard, CardUID: c.UID}), ErrUltimateNotReady)
	a.Charge = 5
	wantErr(t, g.UseUltimate(alice, enemyT), ErrInvalidTarget)
	if err := g.UseUltimate(alice, Target{Kind: cards.TargetHandCard, CardUID: c.UID}); err != nil {
		t.Fatal(err)
	}
	if len(a.Hand) != 2 || a.Hand[1].DefID != "king_of_wands" || a.Charge != 0 {
		t.Fatal("no copy")
	}
}

func TestStrengthGrowsWhenHurtAndUltimateDoublesAHit(t *testing.T) {
	g, a, b := newPlaying(t, "death", "strength")
	mustPlay(t, g, alice, give(g, a, "ace_of_swords", false), enemyT)
	mustPlay(t, g, alice, give(g, a, "ace_of_swords", false), enemyT)
	if b.statusAmount("strength") != 2 {
		t.Fatalf("might %d", b.statusAmount("strength"))
	}
	_ = g.EndTurn(alice)
	b.Charge = 6
	if err := g.UseUltimate(bob, none); err != nil {
		t.Fatal(err)
	}
	b.Mana = 10
	mustPlay(t, g, bob, give(g, b, "ace_of_swords", false), enemyT) // (3+2)*2
	if got := a.MaxHP - a.HP; got != 10 {
		t.Fatalf("damage %d", got)
	}
}

func TestAbilityOncePerTurnAndCostsMana(t *testing.T) {
	g, a, _ := newPlaying(t, "strength", "death")
	a.Mana = 2
	if err := g.UseAbility(alice, none); err != nil {
		t.Fatal(err)
	}
	if a.Armor != 3 || a.Mana != 0 {
		t.Fatal("ability effect")
	}
	a.Mana = 5
	wantErr(t, g.UseAbility(alice, none), ErrAbilityUsed)
	_ = g.EndTurn(alice)
	_ = g.EndTurn(bob)
	a.Mana = 1
	wantErr(t, g.UseAbility(alice, none), ErrNotEnoughMana)
}

func TestMoonPassiveHidesTheFirstCardEachTurn(t *testing.T) {
	g, a, _ := newPlaying(t, "the_moon", "death")
	mustPlay(t, g, alice, give(g, a, "ace_of_swords", false), enemyT)
	mustPlay(t, g, alice, give(g, a, "ace_of_swords", false), enemyT)
	log := g.ViewFor(bob).Log
	if !log[len(log)-2].Hidden || log[len(log)-1].Hidden {
		t.Fatal("only the first card of the turn is hidden")
	}
}

func TestDeathCollectsSoulsAndHarvestsThem(t *testing.T) {
	g, a, b := newPlaying(t, "death", "strength")
	b.Statuses = []*Status{{Name: "ward", Amount: 1, Turns: 1, Source: 1}, {Name: "regen", Amount: 1, Turns: 1, Source: 1}, {Name: "strength", Amount: 1, Source: 1}}
	mustPlay(t, g, alice, give(g, a, "death", false), none)
	if a.Souls != 3 {
		t.Fatalf("souls %d", a.Souls)
	}
	hp := b.HP
	sword := give(g, a, "ace_of_swords", false)
	mustPlay(t, g, alice, sword, enemyT)
	hp = b.HP
	if err := g.UseUltimate(alice, enemyT); err != nil {
		t.Fatal(err)
	}
	if hp-b.HP != 6 || a.Souls != 0 {
		t.Fatalf("harvest %d souls %d", hp-b.HP, a.Souls)
	}
	if _, c := findCard(a.Hand, sword.UID); c == nil {
		t.Fatal("last discarded card returns")
	}
}

// MARK: - Deck, hand limit, fatigue

func TestHandLimitDestroysAndEmptyDeckCausesFatigue(t *testing.T) {
	g, a, b := newPlaying(t, "death", "strength")
	for range DefaultRules().HandLimit {
		give(g, b, "ace_of_cups", false)
	}
	top := b.Deck[0]
	_ = g.EndTurn(alice)
	if len(b.Hand) != DefaultRules().HandLimit || b.Discard[len(b.Discard)-1] != top {
		t.Fatal("overdraw should destroy the card")
	}
	if a.Souls != 1 {
		t.Fatal("death gets a soul for a destroyed card")
	}
	_ = g.EndTurn(bob)
	a.Deck = nil
	hp := a.HP
	_ = g.EndTurn(alice)
	_ = g.EndTurn(bob)
	if a.HP != hp-1 || a.Fatigue != 1 {
		t.Fatalf("fatigue hp %d", hp-a.HP)
	}
}

// MARK: - Ending

func TestHeroDeathEndsTheMatchAndBlocksFurtherActions(t *testing.T) {
	g, a, b := newPlaying(t, "death", "strength")
	b.HP = 3
	mustPlay(t, g, alice, give(g, a, "ace_of_swords", false), enemyT)
	if g.Phase != PhaseFinished || g.WinnerID() != alice || g.Reason != ReasonNormal {
		t.Fatalf("phase %s winner %q", g.Phase, g.WinnerID())
	}
	wantErr(t, g.EndTurn(alice), ErrMatchFinished)
	wantErr(t, g.PlayCard(alice, give(g, a, "ace_of_swords", false).UID, enemyT), ErrMatchFinished)
	wantErr(t, g.Surrender(bob), ErrMatchFinished)
	wantErr(t, g.TimeoutTurn(), ErrWrongPhase)
}

func TestTowerSuicideLosesForTheActor(t *testing.T) {
	g, a, b := newPlaying(t, "death", "strength")
	a.HP, b.HP = 2, 5
	mustPlay(t, g, alice, give(g, a, "the_tower", false), enemyT)
	if g.WinnerID() != bob {
		t.Fatal("the player who brought both down loses")
	}
}

func TestSurrenderAndForfeit(t *testing.T) {
	g, _, _ := newPlaying(t, "death", "strength")
	if err := g.Surrender(bob); err != nil {
		t.Fatal(err)
	}
	if g.WinnerID() != alice || g.Reason != ReasonSurrender {
		t.Fatal("surrender")
	}
	g, _, _ = newPlaying(t, "death", "strength")
	_ = g.Forfeit(alice, ReasonDisconnectTimeout)
	if g.WinnerID() != bob || g.Reason != ReasonDisconnectTimeout {
		t.Fatal("forfeit")
	}
	g, _, _ = newPlaying(t, "death", "strength")
	_ = g.Abort()
	if g.WinnerID() != "" || g.Reason != ReasonServerError {
		t.Fatal("abort has no winner")
	}
}

func TestTimeoutsPassTheTurnThenForfeit(t *testing.T) {
	g, _, _ := newPlaying(t, "death", "strength")
	for i := range 2 {
		if err := g.TimeoutTurn(); err != nil {
			t.Fatal(err)
		}
		if g.Phase == PhaseFinished {
			t.Fatalf("finished after %d timeouts", i+1)
		}
	}
	// Alice timed out, then Bob; now Alice times out again, twice more.
	_ = g.TimeoutTurn()
	_ = g.TimeoutTurn()
	_ = g.TimeoutTurn()
	if g.Phase != PhaseFinished || g.Reason != ReasonTurnTimeout || g.WinnerID() != bob {
		t.Fatalf("phase %s reason %s winner %s", g.Phase, g.Reason, g.WinnerID())
	}
}

func TestActingResetsTheTimeoutStreak(t *testing.T) {
	g, _, _ := newPlaying(t, "death", "strength")
	_ = g.TimeoutTurn() // alice 1
	_ = g.TimeoutTurn() // bob 1
	_ = g.EndTurn(alice)
	if g.Players[0].TimeoutStreak != 0 {
		t.Fatal("streak not reset")
	}
}

// MARK: - Visibility and sequence

func TestViewHidesTheOpponentsHandAndDeck(t *testing.T) {
	g, _ := New(5, PlayerSetup{alice, "death"}, PlayerSetup{bob, "strength"})
	_ = g.Mulligan(alice, []string{g.Players[0].Hand[0].UID})
	v := g.ViewFor(bob)
	raw, _ := json.Marshal(v)
	for _, c := range append(append([]*CardInstance{}, g.Players[0].Hand...), g.Players[0].Deck...) {
		if strings.Contains(string(raw), `"`+c.UID+`"`) {
			t.Fatalf("bob's view leaks alice's card %s", c.UID)
		}
	}
	if v.Opponent.HandCount != len(g.Players[0].Hand) {
		t.Fatal("hand count")
	}
	if len(v.Log) != 1 || v.Log[0].CardUIDs != nil {
		t.Fatal("mulligan choices are private")
	}
}

func TestSeqIncreasesOnlyOnAcceptedActions(t *testing.T) {
	g, a, _ := newPlaying(t, "death", "strength")
	seq := g.Seq
	_ = g.EndTurn(bob)
	c := give(g, a, "king_of_swords", false)
	a.Mana = 0
	_ = g.PlayCard(alice, c.UID, enemyT)
	if g.Seq != seq {
		t.Fatal("rejected actions must not advance seq")
	}
	_ = g.EndTurn(alice)
	if g.Seq != seq+1 || g.ViewFor(alice).Seq != g.Seq {
		t.Fatal("seq")
	}
}

// MARK: - Replay

func TestReplayRebuildsTheSameMatch(t *testing.T) {
	g := playRandomMatch(t, 99)
	a, b := g.Setup()
	raw, err := json.Marshal(g.Log())
	if err != nil {
		t.Fatal(err)
	}
	var log []Entry
	if err := json.Unmarshal(raw, &log); err != nil {
		t.Fatal(err)
	}
	r, err := Replay(g.Rules, g.Seed, a, b, log)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(r.ViewFor(alice), g.ViewFor(alice)) || !reflect.DeepEqual(r.ViewFor(bob), g.ViewFor(bob)) {
		t.Fatal("replay diverged")
	}
	if r.WinnerID() != g.WinnerID() {
		t.Fatal("winner differs")
	}
}

func TestRandomMatchesAlwaysFinish(t *testing.T) {
	for seed := uint64(1); seed <= 300; seed++ {
		g := playRandomMatch(t, seed)
		if g.Phase != PhaseFinished || g.WinnerID() == "" {
			t.Fatalf("seed %d: phase %s", seed, g.Phase)
		}
	}
}

// playRandomMatch plays legal moves chosen from the views, like a naive bot.
func playRandomMatch(t *testing.T, seed uint64) *Game {
	t.Helper()
	heroIDs := []string{"the_magician", "strength", "the_moon", "death"}
	g, err := New(seed, PlayerSetup{alice, heroIDs[seed%4]}, PlayerSetup{bob, heroIDs[(seed/4)%4]})
	if err != nil {
		t.Fatal(err)
	}
	_ = g.Mulligan(alice, []string{g.Players[0].Hand[0].UID})
	_ = g.Mulligan(bob, nil)
	for steps := 0; g.Phase != PhaseFinished; steps++ {
		if steps > 5000 {
			t.Fatalf("seed %d: match never ended", seed)
		}
		id := g.ActivePlayerID()
		v := g.ViewFor(id)
		acted := false
		for _, c := range v.You.Hand {
			if !c.Playable {
				continue
			}
			target := pickTarget(v, c.Targets[0], c.UID)
			if err := g.PlayCard(id, c.UID, target); err != nil {
				t.Fatalf("seed %d: view said playable but %s failed: %v", seed, c.CardID, err)
			}
			acted = true
			break
		}
		if acted {
			continue
		}
		if v.You.UltimateUsable {
			if err := g.UseUltimate(id, pickTarget(v, v.You.UltimateTargets[0], "")); err != nil {
				t.Fatalf("seed %d: ultimate: %v", seed, err)
			}
			continue
		}
		if v.You.AbilityUsable && steps%3 == 0 {
			if err := g.UseAbility(id, pickTarget(v, v.You.AbilityTargets[0], "")); err != nil {
				t.Fatalf("seed %d: ability: %v", seed, err)
			}
			continue
		}
		if err := g.EndTurn(id); err != nil {
			t.Fatal(err)
		}
	}
	return g
}

func pickTarget(v View, kind cards.Target, self string) Target {
	switch kind {
	case "none":
		return Target{}
	case cards.TargetHandCard:
		for _, c := range v.You.Hand {
			if c.UID != self {
				return Target{Kind: kind, CardUID: c.UID}
			}
		}
	case cards.TargetDiscardCard:
		return Target{Kind: kind, CardUID: v.You.Discard[0].UID}
	}
	return Target{Kind: kind}
}

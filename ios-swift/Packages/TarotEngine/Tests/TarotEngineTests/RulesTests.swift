import Testing
@testable import TarotEngine

private func s(_ suit: Suit, _ rank: Int) -> Card { .suited(suit, rank) }
private func t(_ value: Int) -> Card { .trump(value) }

@Suite("Deck")
struct DeckTests {
    @Test("78 distinct cards worth 91 points, with three Oudlers")
    func composition() {
        let deck = Card.fullDeck
        #expect(deck.count == 78)
        #expect(Set(deck).count == 78)
        let trumps = deck.filter(\.isTrump)
        #expect(trumps.count == 21)
        #expect(deck.filter { $0.suit != nil }.count == 56)
        #expect(deck.reduce(0) { $0 + $1.halfPoints } == 182)
        let oudlers = deck.filter(\.isOudler)
        #expect(oudlers == [t(1), t(21), .excuse])
        let kings = deck.filter(\.isKing)
        #expect(kings.count == 4)
    }

    @Test("Honour values: King 4½, Queen 3½, Knight 2½, Jack 1½, others ½")
    func values() {
        #expect(s(.hearts, Card.king).halfPoints == 9)
        #expect(s(.hearts, Card.queen).halfPoints == 7)
        #expect(s(.hearts, Card.knight).halfPoints == 5)
        #expect(s(.hearts, Card.jack).halfPoints == 3)
        #expect(s(.hearts, 10).halfPoints == 1)
        #expect(t(20).halfPoints == 1)
        #expect(t(21).halfPoints == 9)
        #expect(Card.excuse.halfPoints == 9)
    }
}

@Suite("Which cards may be played")
struct LegalCardTests {
    @Test("Leading: anything")
    func leading() {
        let hand = [s(.hearts, 3), t(5), .excuse]
        #expect(Rules.legalCards(hand: hand, trick: []) == hand)
    }

    @Test("Must follow suit, but need not play higher")
    func followSuit() {
        let hand = [s(.hearts, 2), s(.hearts, 13), s(.clubs, 14), t(7)]
        let legal = Rules.legalCards(hand: hand, trick: [s(.hearts, 10)])
        #expect(Set(legal) == [s(.hearts, 2), s(.hearts, 13)])
    }

    @Test("Void in the led suit: must trump")
    func mustTrump() {
        let hand = [s(.clubs, 14), t(3), t(15)]
        let legal = Rules.legalCards(hand: hand, trick: [s(.hearts, 10)])
        #expect(Set(legal) == [t(3), t(15)])
    }

    @Test("Void, and a trump already played: must overtrump")
    func mustOvertrump() {
        let hand = [s(.clubs, 14), t(3), t(15)]
        let legal = Rules.legalCards(hand: hand, trick: [s(.hearts, 10), t(8)])
        #expect(legal == [t(15)])
    }

    @Test("Void and unable to overtrump: must still undertrump")
    func mustUndertrump() {
        let hand = [s(.clubs, 14), t(3), t(5)]
        let legal = Rules.legalCards(hand: hand, trick: [s(.hearts, 10), t(8)])
        #expect(Set(legal) == [t(3), t(5)])
    }

    @Test("Neither the suit nor a trump: any card")
    func discard() {
        let hand = [s(.clubs, 14), s(.spades, 2)]
        #expect(Rules.legalCards(hand: hand, trick: [s(.hearts, 10)]) == hand)
    }

    @Test("Trumps led: must go above the highest trump, even a partner's")
    func trumpLead() {
        let hand = [t(4), t(12), t(19), s(.hearts, 1)]
        #expect(Set(Rules.legalCards(hand: hand, trick: [t(10)])) == [t(12), t(19)])
        #expect(Set(Rules.legalCards(hand: hand, trick: [t(10), t(20)])) == [t(4), t(12), t(19)])
    }

    @Test("The Excuse can always be played")
    func excuseAlwaysLegal() {
        let hand = [s(.hearts, 2), .excuse]
        #expect(Set(Rules.legalCards(hand: hand, trick: [s(.hearts, 10)])) == [s(.hearts, 2), .excuse])
        #expect(Set(Rules.legalCards(hand: [t(3), .excuse], trick: [t(10)])) == [t(3), .excuse])
    }

    @Test("After an Excuse lead, the next card sets the suit")
    func excuseLead() {
        let hand = [s(.hearts, 2), s(.clubs, 5), t(9)]
        #expect(Rules.legalCards(hand: hand, trick: [.excuse]) == hand)
        #expect(Rules.legalCards(hand: hand, trick: [.excuse, s(.clubs, 11)]) == [s(.clubs, 5)])
    }
}

@Suite("Who wins the trick")
struct TrickWinnerTests {
    @Test("Highest card of the led suit")
    func ledSuit() {
        #expect(Rules.winningIndex(of: [s(.hearts, 3), s(.hearts, 13), s(.clubs, 14), s(.hearts, 10)]) == 1)
    }

    @Test("Any trump beats the suit; the highest trump wins")
    func trumps() {
        #expect(Rules.winningIndex(of: [s(.hearts, 14), t(2), s(.hearts, 13)]) == 1)
        #expect(Rules.winningIndex(of: [s(.hearts, 14), t(2), t(19), t(7)]) == 2)
    }

    @Test("The Excuse never wins an ordinary trick, even when led")
    func excuseNeverWins() {
        #expect(Rules.winningIndex(of: [.excuse, s(.spades, 2), s(.spades, 9), s(.hearts, 14)]) == 2)
        #expect(Rules.winningIndex(of: [s(.spades, 2), .excuse, s(.spades, 1)]) == 0)
    }

    @Test("Led to the last trick of a Chelem, the Excuse wins")
    func excuseChelemLead() {
        #expect(Rules.winningIndex(of: [.excuse, t(21), s(.hearts, 14)], excuseLeadWins: true) == 0)
    }
}

@Suite("The Écart")
struct EcartTests {
    private let handWithChien: [Card] = [
        s(.hearts, 2), s(.hearts, 3), s(.hearts, 4), s(.hearts, 5), s(.hearts, 6), s(.hearts, 7),
        s(.clubs, Card.king), t(1), t(21), .excuse, t(8), t(9),
    ]

    @Test("Six plain cards are fine")
    func plainIsFine() {
        let ecart = Array(handWithChien.prefix(6))
        #expect(Rules.ecartProblem(ecart, hand: handWithChien, size: 6) == nil)
    }

    @Test("Kings and Oudlers can never be discarded")
    func kingsAndOudlers() {
        var withKing = Array(handWithChien.prefix(5))
        withKing.append(s(.clubs, Card.king))
        #expect(Rules.ecartProblem(withKing, hand: handWithChien, size: 6) != nil)

        var withPetit = Array(handWithChien.prefix(5))
        withPetit.append(t(1))
        #expect(Rules.ecartProblem(withPetit, hand: handWithChien, size: 6) != nil)
    }

    @Test("Trumps only when nothing else can go")
    func trumpsOnlyWhenForced() {
        var withTrump = Array(handWithChien.prefix(5))
        withTrump.append(t(8))
        #expect(Rules.ecartProblem(withTrump, hand: handWithChien, size: 6) != nil)

        // Only four plain cards available: two trumps are then allowed.
        let tight: [Card] = [s(.hearts, 2), s(.hearts, 3), s(.hearts, 4), s(.hearts, 5),
                             s(.clubs, Card.king), t(1), t(21), .excuse, t(8), t(9)]
        let forced = [s(.hearts, 2), s(.hearts, 3), s(.hearts, 4), s(.hearts, 5), t(8), t(9)]
        #expect(Rules.ecartProblem(forced, hand: tight, size: 6) == nil)
    }

    @Test("Wrong size, duplicates and cards not held are rejected")
    func malformed() {
        #expect(Rules.ecartProblem(Array(handWithChien.prefix(5)), hand: handWithChien, size: 6) != nil)
        let duplicated = Array(handWithChien.prefix(5)) + [s(.hearts, 2)]
        #expect(Rules.ecartProblem(duplicated, hand: handWithChien, size: 6) != nil)
        let foreign = Array(handWithChien.prefix(5)) + [s(.spades, 2)]
        #expect(Rules.ecartProblem(foreign, hand: handWithChien, size: 6) != nil)
    }

    @Test("The cheapest écart is always legal")
    func cheapestIsLegal() {
        var generator = SeededGenerator(seed: 7)
        for _ in 0..<500 {
            var deck = Card.fullDeck
            deck.shuffle(using: &generator)
            let hand = Array(deck.prefix(24))
            let ecart = Rules.cheapestEcart(hand: hand, size: 6)
            #expect(Rules.ecartProblem(ecart, hand: hand, size: 6) == nil, "hand \(hand)")
        }
    }
}

@Suite("Petit sec")
struct PetitSecTests {
    @Test("The Petit as the only trump, without the Excuse, voids the deal")
    func petitSec() {
        #expect(Rules.isPetitSec([t(1), s(.hearts, 2)]))
        #expect(!Rules.isPetitSec([t(1), .excuse]))
        #expect(!Rules.isPetitSec([t(1), t(2)]))
        #expect(!Rules.isPetitSec([s(.hearts, 2)]))
    }
}

import Testing
@testable import TarotEngine

private func s(_ suit: Suit, _ rank: Int) -> Card { .suited(suit, rank) }
private func t(_ value: Int) -> Card { .trump(value) }

/// All 56 suit cards, in a fixed order.
private let suitCards: [Card] = Suit.allCases.flatMap { suit in (1...14).map { s(suit, $0) } }

@Suite("Bidding")
struct BiddingTests {
    private func freshRound(dealer: Int = 0, seed: UInt64 = 1) -> Round {
        var generator = SeededGenerator(seed: seed)
        var round = Round.deal(table: .four, dealer: dealer, using: &generator)
        // Skip the rare petit-sec deal so bidding is always open.
        var extra: UInt64 = 1000
        while round.phase != .bidding {
            var retry = SeededGenerator(seed: seed + extra)
            round = Round.deal(table: .four, dealer: dealer, using: &retry)
            extra += 1
        }
        return round
    }

    @Test("The player after the dealer speaks first, and each speaks once")
    func order() throws {
        var round = freshRound(dealer: 2)
        #expect(round.turn == 3)
        #expect(throws: RoundError.notYourTurn) { try round.bid(nil, by: 0) }
        try round.bid(nil, by: 3)
        try round.bid(.petite, by: 0)
        try round.bid(nil, by: 1)
        #expect(round.phase == .bidding)
        try round.bid(nil, by: 2)
        #expect(round.taker == 0)
        #expect(round.contract == .petite)
        #expect(round.phase == .discarding)
    }

    @Test("A bid must beat the highest bid so far")
    func mustOutbid() throws {
        var round = freshRound()
        try round.bid(.garde, by: 1)
        #expect(throws: RoundError.bidTooLow) { try round.bid(.garde, by: 2) }
        #expect(throws: RoundError.bidTooLow) { try round.bid(.petite, by: 2) }
        try round.bid(.gardeSans, by: 2)
        #expect(round.highestBid == .gardeSans)
    }

    @Test("Everyone passing voids the deal")
    func allPass() throws {
        var round = freshRound()
        for seat in [1, 2, 3, 0] {
            try round.bid(nil, by: seat)
        }
        #expect(round.phase == .redeal(.allPassed))
        #expect(round.taker == nil)
    }

    @Test("On a Prise or Garde the taker takes the Chien and must discard")
    func takesChien() throws {
        var round = freshRound()
        try round.bid(.garde, by: 1)
        for seat in [2, 3, 0] { try round.bid(nil, by: seat) }
        #expect(round.hands[1].count == 24)
        #expect(throws: RoundError.wrongPhase) { try round.play(round.hands[1][0], by: 1) }

        let ecart = Rules.cheapestEcart(hand: round.hands[1], size: 6)
        try round.discard(ecart)
        #expect(round.hands[1].count == 18)
        #expect(round.phase == .playing)
        #expect(round.turn == 1)
    }

    @Test("On a Garde Sans or Garde Contre the Chien is not touched")
    func chienUntouched() throws {
        for contract in [Contract.gardeSans, .gardeContre] {
            var round = freshRound()
            try round.bid(contract, by: 1)
            for seat in [2, 3, 0] { try round.bid(nil, by: seat) }
            #expect(round.phase == .playing)
            #expect(round.hands[1].count == 18)
        }
    }

    @Test("An invalid écart is refused and nothing changes")
    func invalidEcart() throws {
        var round = freshRound()
        try round.bid(.petite, by: 1)
        for seat in [2, 3, 0] { try round.bid(nil, by: seat) }
        let before = round.hands[1]
        #expect(throws: RoundError.self) { try round.discard(Array(before.prefix(5))) }
        #expect(round.hands[1] == before)
        #expect(round.phase == .discarding)
    }
}

@Suite("A deal voided by the Petit sec")
struct PetitSecDealTests {
    @Test("The deal starts voided")
    func voided() {
        // Seat 2 holds the Petit as its only trump and no Excuse.
        let trumps = (2...21).map(t)
        let seat2 = [t(1)] + Array(suitCards[0..<17])
        let rest = Array(suitCards[17...]) + trumps + [.excuse]
        let hands = [Array(rest[0..<18]), Array(rest[18..<36]), seat2, Array(rest[36..<54])]
        let chien = Array(rest[54..<60])
        let round = Round(table: .four, dealer: 0, hands: hands, chien: chien)
        #expect(round.phase == .redeal(.petitSec(player: 2)))
    }
}

/// Fully scripted deals, so the Excuse and Chelem rules — the subtlest part of
/// the scoring — are checked end to end rather than piece by piece.
@Suite("Scripted deals")
struct ScriptedDealTests {
    /// The taker (seat 1) plays its highest trump; everyone else plays their
    /// first legal card, except that `holdExcuse` keeps the defence's Excuse
    /// until it can no longer be avoided.
    private func playOut(_ round: inout Round, holdExcuse: Bool, playExcuseEarly: Bool = false) throws {
        while round.phase == .playing {
            let seat = round.turn
            let legal = round.legalCards
            let card: Card
            if seat == round.taker {
                card = legal.filter(\.isTrump).max { lhs, rhs in
                    if case let (.trump(a), .trump(b)) = (lhs, rhs) { return a < b }
                    return false
                } ?? legal[0]
            } else if playExcuseEarly, legal.contains(.excuse), !round.hands[seat].contains(where: \.isTrump) {
                card = .excuse
            } else if holdExcuse {
                card = legal.first { $0 != .excuse } ?? legal[0]
            } else {
                card = legal[0]
            }
            try round.play(card, by: seat)
        }
    }

    @Test("Announced Chelem closed by leading the Excuse: every point to the taker")
    func announcedChelemWithExcuseLead() throws {
        let taker = (5...21).map(t) + [.excuse]
        let seat0 = (1...4).map(t) + Array(suitCards[0..<14])
        let seat2 = Array(suitCards[14..<32])
        let seat3 = Array(suitCards[32..<50])
        let chien = Array(suitCards[50..<56])
        var round = Round(table: .four, dealer: 0, hands: [seat0, taker, seat2, seat3], chien: chien)

        try round.bid(.gardeSans, by: 1)
        for seat in [2, 3, 0] { try round.bid(nil, by: seat) }
        try round.announceChelem(by: 1)
        #expect(round.turn == 1)

        // While the Chelem is alive the taker may not play the Excuse early.
        #expect(!round.legalCards.contains(.excuse))

        try playOut(&round, holdExcuse: true)

        #expect(round.phase == .finished)
        #expect(round.tricks.count == 18)
        #expect(round.tricks.last?.plays.first?.card == .excuse)
        #expect(round.tricks.allSatisfy { $0.winner == 1 })
        #expect(round.chelem == .taker)

        let tally = try #require(round.tally)
        #expect(tally.takerHalfPoints == 182)
        #expect(tally.takerOudlers == 3)

        let score = try #require(round.score)
        // 91 − 36 = 55; (25 + 55) × 4 = 320; + 400 announced Chelem.
        #expect(score.perDefender == -720)
        #expect(round.scoresBySeat == [-720, 2160, -720, -720])
    }

    @Test("A defender playing the Excuse to the last trick loses it to the winner")
    func excuseChangesSidesOnLastTrick() throws {
        let taker = (4...21).map(t)
        let seat0 = (1...3).map(t) + [.excuse] + Array(suitCards[0..<14])
        let seat2 = Array(suitCards[14..<32])
        let seat3 = Array(suitCards[32..<50])
        let chien = Array(suitCards[50..<56])
        var round = Round(table: .four, dealer: 0, hands: [seat0, taker, seat2, seat3], chien: chien)

        try round.bid(.gardeSans, by: 1)
        for seat in [2, 3, 0] { try round.bid(nil, by: seat) }
        try playOut(&round, holdExcuse: true)

        #expect(round.tricks.last?.plays.contains { $0.card == .excuse } == true)
        #expect(round.chelem == .taker)
        let tally = try #require(round.tally)
        #expect(tally.takerHalfPoints == 182)
        #expect(tally.takerOudlers == 3)

        // Unannounced Chelem: (25 + 55) × 4 + 200.
        #expect(round.score?.perDefender == -520)
    }

    /// FFT: in a Chelem made by a taker who does not hold the Excuse, the
    /// Excuse played normally stays with the defence and counts 4 points —
    /// matching the rulebook's "87 points with 2 Oudlers" example.
    @Test("The Excuse played early stays with the defence, worth 4 points")
    func excuseKeptByDefence() throws {
        let taker = (4...21).map(t)
        let seat0 = (1...3).map(t) + [.excuse] + Array(suitCards[0..<14])
        let seat2 = Array(suitCards[14..<32])
        let seat3 = Array(suitCards[32..<50])
        let chien = Array(suitCards[50..<56])
        var round = Round(table: .four, dealer: 0, hands: [seat0, taker, seat2, seat3], chien: chien)

        try round.bid(.gardeSans, by: 1)
        for seat in [2, 3, 0] { try round.bid(nil, by: seat) }
        try playOut(&round, holdExcuse: true, playExcuseEarly: true)

        #expect(round.chelem == .taker)
        let tally = try #require(round.tally)
        #expect(tally.takerHalfPoints == 87 * 2)
        #expect(tally.defenseHalfPoints == 4 * 2)
        #expect(tally.takerOudlers == 2)
        // 87 − 41 = 46; (25 + 46) × 4 = 284; + 200 unannounced Chelem.
        #expect(round.score?.perDefender == -484)
    }

    @Test("Petit au Bout counts for the side taking the last trick")
    func petitAuBout() throws {
        // A defender holding trumps must spend them on trump leads, so it can
        // never nurse the Petit to the end. Instead the taker holds trumps
        // 1–18 and nobody else holds a trump at all (19, 20, 21 and the Excuse
        // sit in the Chien of a Garde Sans): the taker leads trumps from the
        // top, wins every trick, and leads the Petit last.
        let taker = (1...18).map(t)
        let chien = [t(19), t(20), t(21), .excuse, suitCards[0], suitCards[1]]
        let seat1 = Array(suitCards[2..<20])
        let seat2 = Array(suitCards[20..<38])
        let seat3 = Array(suitCards[38..<56])
        // Dealer 3, so seat 0 — the taker — speaks and leads first.
        var round = Round(table: .four, dealer: 3, hands: [taker, seat1, seat2, seat3], chien: chien)

        try round.bid(.gardeSans, by: 0)
        for seat in [1, 2, 3] { try round.bid(nil, by: seat) }
        #expect(round.turn == 0)

        while round.phase == .playing {
            let seat = round.turn
            let legal = round.legalCards
            let card = seat == 0
                ? legal.max { lhs, rhs in
                    if case let (.trump(a), .trump(b)) = (lhs, rhs) { return a < b }
                    return false
                }!
                : legal[0]
            try round.play(card, by: seat)
        }

        #expect(round.tricks.last?.plays.first?.card == t(1))
        #expect(round.tricks.allSatisfy { $0.winner == 0 })
        #expect(round.petitAuBout == .taker)
        #expect(round.chelem == .taker)

        let tally = try #require(round.tally)
        #expect(tally.takerHalfPoints == 182)
        // The 21 and the Excuse count for the taker from the Chien of a Garde Sans.
        #expect(tally.takerOudlers == 3)
        // (25 + 55) × 4 = 320, + 10 × 4 Petit au Bout, + 200 unannounced Chelem.
        #expect(round.score?.perDefender == -560)
    }
}

/// Thousands of random legal deals. Whatever the cards, a finished deal must
/// account for every card and point, and the table's scores must net to zero.
@Suite("Random deals keep their invariants")
struct SimulationTests {
    private func simulate(table: TableSize, seed: UInt64) throws -> Round? {
        var generator = SeededGenerator(seed: seed)
        var round = Round.deal(table: table, dealer: Int(seed % UInt64(table.playerCount)), using: &generator)
        guard round.phase == .bidding else { return nil }

        while round.phase == .bidding {
            let highest = round.highestBid
            let higher = Contract.allCases.filter { candidate in highest.map { candidate > $0 } ?? true }
            let contract: Contract? = (!higher.isEmpty && Int.random(in: 0..<3, using: &generator) == 0)
                ? higher.randomElement(using: &generator)
                : nil
            try round.bid(contract, by: round.turn)
        }
        if case .redeal = round.phase { return nil }

        if round.phase == .discarding, let taker = round.taker {
            try round.discard(Rules.cheapestEcart(hand: round.hands[taker], size: table.chienSize))
        }

        while round.phase == .playing {
            let legal = round.legalCards
            #expect(!legal.isEmpty)
            try round.play(legal.randomElement(using: &generator)!, by: round.turn)
        }
        return round
    }

    @Test("Every card and point accounted for, scores net to zero", arguments: [TableSize.four, .three])
    func invariants(table: TableSize) throws {
        var played = 0
        for seed in UInt64(0)..<1500 {
            guard let round = try simulate(table: table, seed: seed) else { continue }
            played += 1

            #expect(round.phase == .finished)
            #expect(round.tricks.count == table.handSize)
            #expect(round.tricks.allSatisfy { $0.plays.count == table.playerCount })
            let allHandsEmpty = round.hands.allSatisfy(\.isEmpty)
            #expect(allHandsEmpty)

            // Every card appears exactly once across the tricks and the Chien/Écart.
            let aside = round.contract!.takerTakesChien ? round.ecart : round.chien
            let seen = round.tricks.flatMap { $0.plays.map(\.card) } + aside
            #expect(seen.count == 78)
            #expect(Set(seen).count == 78)

            let tally = try #require(round.tally)
            #expect(tally.takerHalfPoints + tally.defenseHalfPoints == 182, "seed \(seed)")
            #expect((0...3).contains(tally.takerOudlers))

            let scores = try #require(round.scoresBySeat)
            #expect(scores.reduce(0, +) == 0, "seed \(seed)")
        }
        // Sanity: most random deals get a taker.
        #expect(played > 300)
    }

    @Test("The same seed always deals the same cards")
    func reproducible() {
        var a = SeededGenerator(seed: 42)
        var b = SeededGenerator(seed: 42)
        let first = Round.deal(table: .four, dealer: 0, using: &a)
        let second = Round.deal(table: .four, dealer: 0, using: &b)
        #expect(first.hands == second.hands)
        #expect(first.chien == second.chien)
    }
}

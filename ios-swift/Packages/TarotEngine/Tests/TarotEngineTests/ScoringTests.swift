import Testing
@testable import TarotEngine

/// The worked examples printed in the FFT *Règlement officiel* (2012), section
/// "La marque en donnes libres". If any of these fail, the engine scores
/// differently from a real French Tarot table.
@Suite("FFT worked scoring examples")
struct FFTExampleTests {
    @Test("Garde, simple Poignée, Petit au Bout, 49 points with two Oudlers → 106")
    func gardeWithPoigneeAndPetitAuBout() {
        let result = Scoring.score(ScoreInput(
            table: .four, contract: .garde,
            takerHalfPoints: 49 * 2, takerOudlers: 2,
            poignees: [.simple], petitAuBout: .taker
        ))
        // 25 + 8 = 33 × 2 = 66, + 20 Poignée, + 10 × 2 Petit au Bout = 106.
        #expect(result.takerWon)
        #expect(result.margin == 8)
        #expect(result.perDefender == -106)
        #expect(result.taker == 318)
    }

    @Test("Garde Sans won by 4, Petit au Bout taken by the defence → 76")
    func gardeSansLosingPetitAuBout() {
        let result = Scoring.score(ScoreInput(
            table: .four, contract: .gardeSans,
            takerHalfPoints: 45 * 2, takerOudlers: 2,
            petitAuBout: .defense
        ))
        // (25 + 4) × 4 = 116, − 10 × 4 = 76.
        #expect(result.perDefender == -76)
        #expect(result.taker == 228)
    }

    @Test("Prise failed by 7 despite a Poignée and the Petit au Bout → defenders +42")
    func failedPriseWithBonuses() {
        let result = Scoring.score(ScoreInput(
            table: .four, contract: .petite,
            takerHalfPoints: 34 * 2, takerOudlers: 2,
            poignees: [.simple], petitAuBout: .taker
        ))
        // 25 + 7 + 20 (the Poignée goes to the winning side) − 10 = 42.
        #expect(!result.takerWon)
        #expect(result.margin == 7)
        #expect(result.perDefender == 42)
        #expect(result.taker == -126)
    }

    @Test("Garde won by 11 while the defence showed a Poignée → 92")
    func defencePoigneeGoesToWinningTaker() {
        let result = Scoring.score(ScoreInput(
            table: .four, contract: .garde,
            takerHalfPoints: 52 * 2, takerOudlers: 2,
            poignees: [.simple]
        ))
        // (25 + 11) × 2 = 72, + 20 Poignée paid by the defence = 92.
        #expect(result.perDefender == -92)
        #expect(result.taker == 276)
    }

    @Test("Announced Chelem made on a Garde, 87 points with two Oudlers → 582")
    func announcedChelem() {
        let result = Scoring.score(ScoreInput(
            table: .four, contract: .garde,
            takerHalfPoints: 87 * 2, takerOudlers: 2,
            poignees: [.simple], petitAuBout: .taker,
            chelemAnnounced: true, chelem: .taker
        ))
        // (46 + 25) × 2 = 142, + 20 + 20 + 400 = 582.
        #expect(result.margin == 46)
        #expect(result.perDefender == -582)
        #expect(result.taker == 1746)
    }
}

@Suite("Scoring rules")
struct ScoringRuleTests {
    @Test("Points needed by number of Oudlers")
    func pointsNeeded() {
        #expect(Rules.pointsNeeded(oudlers: 0) == 56)
        #expect(Rules.pointsNeeded(oudlers: 1) == 51)
        #expect(Rules.pointsNeeded(oudlers: 2) == 41)
        #expect(Rules.pointsNeeded(oudlers: 3) == 36)
    }

    @Test("Exactly on target (juste fait) wins with a margin of zero")
    func justeFait() {
        let result = Scoring.score(ScoreInput(
            table: .four, contract: .garde, takerHalfPoints: 41 * 2, takerOudlers: 2
        ))
        #expect(result.takerWon)
        #expect(result.margin == 0)
        #expect(result.perDefender == -50)
    }

    /// FFT, 3-player rules: "40½ with 2 Oudlers: the ½ goes to the defence,
    /// the contract fails by ONE point; 41½: the ½ goes to the taker, who
    /// wins by ONE point."
    @Test("A stray half point goes to the winning side")
    func halfPointRule() {
        let short = Scoring.score(ScoreInput(
            table: .three, contract: .petite, takerHalfPoints: 81, takerOudlers: 2
        ))
        #expect(!short.takerWon)
        #expect(short.margin == 1)

        let over = Scoring.score(ScoreInput(
            table: .three, contract: .petite, takerHalfPoints: 83, takerOudlers: 2
        ))
        #expect(over.takerWon)
        #expect(over.margin == 1)
    }

    @Test("Contract multipliers are 1, 2, 4 and 6")
    func multipliers() {
        #expect(Contract.allCases.map(\.multiplier) == [1, 2, 4, 6])
        #expect(Contract.petite < .garde && .garde < .gardeSans && .gardeSans < .gardeContre)
    }

    @Test("At three players the taker scores twice the per-defender amount")
    func threePlayerFactor() {
        let result = Scoring.score(ScoreInput(
            table: .three, contract: .petite, takerHalfPoints: 51 * 2, takerOudlers: 2
        ))
        #expect(result.perDefender == -35)
        #expect(result.taker == 70)
        #expect(result.taker + 2 * result.perDefender == 0)
    }

    @Test("Chelem bonuses are not multiplied")
    func chelemBonuses() {
        let base = ScoreInput(table: .four, contract: .gardeContre, takerHalfPoints: 60 * 2, takerOudlers: 2)
        let plain = Scoring.score(base).perDefender

        var unannounced = base
        unannounced.chelem = .taker
        #expect(Scoring.score(unannounced).perDefender == plain - 200)

        var failed = base
        failed.chelemAnnounced = true
        #expect(Scoring.score(failed).perDefender == plain + 200)

        var inflicted = base
        inflicted.takerHalfPoints = 0
        inflicted.takerOudlers = 0
        inflicted.chelem = .defense
        let lost = Scoring.score(inflicted)
        let withoutChelem = Scoring.score(ScoreInput(
            table: .four, contract: .gardeContre, takerHalfPoints: 0, takerOudlers: 0
        ))
        #expect(lost.perDefender == withoutChelem.perDefender + 200)
    }
}

@Suite("Poignée")
struct PoigneeTests {
    private let tenTrumps: [Card] = (2...11).map { .trump($0) }

    @Test("Exactly 10, 13 or 15 trumps at four players")
    func sizes() throws {
        let hand = (1...15).map { Card.trump($0) } + [.suited(.hearts, 2), .suited(.hearts, 3), .suited(.hearts, 4)]
        #expect(try Poignee.validate(shown: Array(hand.prefix(10)), hand: hand, table: .four).get() == .simple)
        #expect(try Poignee.validate(shown: Array(hand.prefix(13)), hand: hand, table: .four).get() == .double)
        #expect(try Poignee.validate(shown: Array(hand.prefix(15)), hand: hand, table: .four).get() == .triple)
        // Holding 11 trumps, the player must hide one: showing 11 is not a Poignée.
        #expect(throws: PoigneeError.wrongSize(11)) {
            try Poignee.validate(shown: Array(hand.prefix(11)), hand: hand, table: .four).get()
        }
    }

    @Test("At three players the sizes are 13, 15 and 18")
    func threePlayerSizes() {
        #expect(TableSize.three.poigneeSizes == [13, 15, 18])
    }

    @Test("The Excuse may complete a Poignée only when no trump is hidden")
    func excuseSubstitution() throws {
        let nineTrumps: [Card] = (2...10).map { .trump($0) }
        let hand = nineTrumps + [.excuse] + (1...8).map { Card.suited(.clubs, $0) }
        #expect(try Poignee.validate(shown: nineTrumps + [.excuse], hand: hand, table: .four).get() == .simple)

        // Ten trumps held: showing nine plus the Excuse hides a trump.
        let tenHand = tenTrumps + [.excuse] + (1...7).map { Card.suited(.clubs, $0) }
        #expect(throws: PoigneeError.excuseWhileHidingTrumps) {
            try Poignee.validate(shown: Array(tenTrumps.prefix(9)) + [.excuse], hand: tenHand, table: .four).get()
        }
    }

    @Test("Only trumps from the player's own hand")
    func onlyOwnTrumps() {
        let hand = tenTrumps + (1...8).map { Card.suited(.spades, $0) }
        #expect(throws: PoigneeError.notATrump(.suited(.spades, 1))) {
            try Poignee.validate(shown: Array(tenTrumps.prefix(9)) + [.suited(.spades, 1)], hand: hand, table: .four).get()
        }
        #expect(throws: PoigneeError.notInHand(.trump(21))) {
            try Poignee.validate(shown: Array(tenTrumps.prefix(9)) + [.trump(21)], hand: hand, table: .four).get()
        }
    }
}

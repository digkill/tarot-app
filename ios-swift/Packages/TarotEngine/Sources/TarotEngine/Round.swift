/// A deterministic random number generator (SplitMix64).
///
/// Deals must be reproducible: for tests, for replaying a reported bug, and
/// for giving every player the same "deal of the day".
public struct SeededGenerator: RandomNumberGenerator, Sendable {
    private var state: UInt64

    public init(seed: UInt64) {
        state = seed
    }

    public mutating func next() -> UInt64 {
        state &+= 0x9E37_79B9_7F4A_7C15
        var z = state
        z = (z ^ (z >> 30)) &* 0xBF58_476D_1CE4_E5B9
        z = (z ^ (z >> 27)) &* 0x94D0_49BB_1331_11EB
        return z ^ (z >> 31)
    }
}

public struct Play: Sendable, Equatable, Codable {
    public var player: Int
    public var card: Card
}

public struct CompletedTrick: Sendable, Equatable, Codable {
    public var plays: [Play]
    public var winner: Int
}

public enum RoundError: Error, Equatable, Sendable {
    case wrongPhase
    case notYourTurn
    case bidTooLow
    case notTheTaker
    case cardNotInHand(Card)
    case illegalCard(Card)
    case invalidEcart(String)
    case invalidPoignee(PoigneeError)
    case poigneeAlreadyDeclared
    case tooLateToAnnounce
}

/// One deal of French Tarot, from bidding to the final score.
///
/// Seats are numbered 0..<playerCount in playing order: the player after seat
/// `i` is seat `i + 1`. (At a real table play goes counter-clockwise; the
/// numbering hides that.)
public struct Round: Sendable {
    public enum Phase: Sendable, Equatable {
        case bidding
        /// The taker holds the Chien and must build the Écart.
        case discarding
        case playing
        case finished
        /// The deal is void and must be redealt.
        case redeal(RedealReason)
    }

    public enum RedealReason: Sendable, Equatable {
        case allPassed
        case petitSec(player: Int)
    }

    public let table: TableSize
    public let dealer: Int
    public private(set) var phase: Phase
    public private(set) var hands: [[Card]]
    public private(set) var chien: [Card]

    /// Bids in speaking order; `nil` is a pass.
    public private(set) var bids: [(player: Int, contract: Contract?)] = []
    public private(set) var taker: Int?
    public private(set) var contract: Contract?
    public private(set) var ecart: [Card] = []
    /// Trumps the taker was forced to discard, which must be shown.
    public private(set) var shownEcartTrumps: [Card] = []

    public private(set) var chelemAnnounced = false
    public private(set) var poignees: [Int: Poignee] = [:]

    public private(set) var currentTrick: [Play] = []
    public private(set) var tricks: [CompletedTrick] = []
    /// Whose turn it is: to bid during bidding, to play during play.
    public private(set) var turn: Int

    // MARK: - Setting up

    /// Shuffles and deals.
    ///
    /// The physical dealing ritual — packets of three, Chien cards one at a
    /// time, never the first or last card — exists to stop a dealer from
    /// steering cards. After a uniform shuffle every split of the deck is
    /// already equally likely, so a digital deal only needs to partition the
    /// shuffled deck.
    public static func deal(
        table: TableSize,
        dealer: Int,
        using generator: inout some RandomNumberGenerator
    ) -> Round {
        var deck = Card.fullDeck
        deck.shuffle(using: &generator)
        let chien = Array(deck.prefix(table.chienSize))
        var rest = deck.dropFirst(table.chienSize)
        var hands: [[Card]] = []
        for _ in 0..<table.playerCount {
            hands.append(Array(rest.prefix(table.handSize)))
            rest = rest.dropFirst(table.handSize)
        }
        return Round(table: table, dealer: dealer, hands: hands, chien: chien)
    }

    /// Builds a deal from known hands — for tests and replays.
    public init(table: TableSize, dealer: Int, hands: [[Card]], chien: [Card]) {
        precondition(hands.count == table.playerCount, "one hand per player")
        precondition(hands.allSatisfy { $0.count == table.handSize }, "every hand must be full")
        precondition(chien.count == table.chienSize, "the Chien must be full")
        precondition(
            Set(hands.joined()).union(chien).count == Card.fullDeck.count,
            "hands and Chien must be exactly the 78 cards"
        )
        self.table = table
        self.dealer = dealer
        self.hands = hands
        self.chien = chien
        self.turn = (dealer + 1) % table.playerCount

        if let petitSec = hands.firstIndex(where: Rules.isPetitSec) {
            phase = .redeal(.petitSec(player: petitSec))
        } else {
            phase = .bidding
        }
    }

    public var highestBid: Contract? {
        bids.compactMap(\.contract).max()
    }

    public func camp(of player: Int) -> Camp {
        player == taker ? .taker : .defense
    }

    // MARK: - Bidding

    /// Bids `contract`, or passes with `nil`. Each player speaks exactly once,
    /// starting after the dealer, and a bid must beat the highest so far.
    public mutating func bid(_ contract: Contract?, by player: Int) throws(RoundError) {
        guard phase == .bidding else { throw .wrongPhase }
        guard player == turn else { throw .notYourTurn }
        if let contract, let highest = highestBid, contract <= highest {
            throw .bidTooLow
        }

        bids.append((player: player, contract: contract))
        turn = (turn + 1) % table.playerCount

        guard bids.count == table.playerCount else { return }

        guard let winning = bids.last(where: { $0.contract != nil }),
              let won = winning.contract else {
            phase = .redeal(.allPassed)
            return
        }
        taker = winning.player
        self.contract = won
        if won.takerTakesChien {
            hands[winning.player].append(contentsOf: chien)
            phase = .discarding
        } else {
            phase = .playing
            turn = (dealer + 1) % table.playerCount
        }
    }

    // MARK: - Chien and Écart

    public mutating func discard(_ cards: [Card]) throws(RoundError) {
        guard phase == .discarding, let taker else { throw .wrongPhase }
        if let problem = Rules.ecartProblem(cards, hand: hands[taker], size: table.chienSize) {
            throw .invalidEcart(problem)
        }
        for card in cards {
            hands[taker].removeAll { $0 == card }
        }
        ecart = cards
        shownEcartTrumps = cards.filter(\.isTrump)
        phase = .playing
        turn = (dealer + 1) % table.playerCount
    }

    // MARK: - Announcements

    /// The taker announces a Chelem before the first card. The announcer then
    /// leads, whoever dealt.
    public mutating func announceChelem(by player: Int) throws(RoundError) {
        guard phase == .playing else { throw .wrongPhase }
        guard player == taker else { throw .notTheTaker }
        guard tricks.isEmpty, currentTrick.isEmpty else { throw .tooLateToAnnounce }
        chelemAnnounced = true
        turn = player
    }

    /// Shows a Poignée. Allowed once, just before the player's first card.
    public mutating func declarePoignee(_ shown: [Card], by player: Int) throws(RoundError) {
        guard phase == .playing else { throw .wrongPhase }
        guard poignees[player] == nil else { throw .poigneeAlreadyDeclared }
        guard !hasPlayed(player) else { throw .tooLateToAnnounce }
        switch Poignee.validate(shown: shown, hand: hands[player], table: table) {
        case let .success(kind):
            poignees[player] = kind
        case let .failure(error):
            throw .invalidPoignee(error)
        }
    }

    private func hasPlayed(_ player: Int) -> Bool {
        currentTrick.contains { $0.player == player }
            || tricks.contains { trick in trick.plays.contains { $0.player == player } }
    }

    // MARK: - Play

    /// Cards the player whose turn it is may play right now.
    public var legalCards: [Card] {
        guard phase == .playing else { return [] }
        let hand = hands[turn]
        var legal = Rules.legalCards(hand: hand, trick: currentTrick.map(\.card))
        // A taker who announced a Chelem must keep the Excuse for the very
        // last trick, while the Chelem is still alive.
        if chelemAnnounced, turn == taker, hand.count > 1, legal.contains(.excuse),
           tricks.allSatisfy({ $0.winner == taker }) {
            legal.removeAll { $0 == .excuse }
        }
        return legal
    }

    public mutating func play(_ card: Card, by player: Int) throws(RoundError) {
        guard phase == .playing else { throw .wrongPhase }
        guard player == turn else { throw .notYourTurn }
        guard hands[player].contains(card) else { throw .cardNotInHand(card) }
        guard legalCards.contains(card) else { throw .illegalCard(card) }

        hands[player].removeAll { $0 == card }
        currentTrick.append(Play(player: player, card: card))
        turn = (turn + 1) % table.playerCount

        guard currentTrick.count == table.playerCount else { return }

        let isLast = hands.allSatisfy(\.isEmpty)
        let leader = currentTrick[0].player
        let leaderCampWonAll = tricks.allSatisfy { camp(of: $0.winner) == camp(of: leader) }
        let excuseLeadWins = isLast && currentTrick[0].card == .excuse && leaderCampWonAll

        let index = Rules.winningIndex(of: currentTrick.map(\.card), excuseLeadWins: excuseLeadWins)
        let winner = currentTrick[index].player
        tricks.append(CompletedTrick(plays: currentTrick, winner: winner))
        currentTrick = []
        turn = winner

        if isLast {
            phase = .finished
        }
    }

    // MARK: - Result

    /// The taker's pile once the deal is over.
    public struct Tally: Sendable, Equatable {
        public var takerHalfPoints: Int
        public var defenseHalfPoints: Int
        public var takerOudlers: Int
    }

    public var tally: Tally? {
        guard phase == .finished, let contract else { return nil }

        var half: [Camp: Int] = [.taker: 0, .defense: 0]
        var oudlers: [Camp: Int] = [.taker: 0, .defense: 0]
        func add(_ card: Card, to side: Camp, halfPoints: Int? = nil) {
            half[side, default: 0] += halfPoints ?? card.halfPoints
            if card.isOudler { oudlers[side, default: 0] += 1 }
        }

        // The Chien or the Écart.
        switch contract {
        case .petite, .garde:
            ecart.forEach { add($0, to: .taker) }
        case .gardeSans:
            chien.forEach { add($0, to: .taker) }
        case .gardeContre:
            chien.forEach { add($0, to: .defense) }
        }

        for (number, trick) in tricks.enumerated() {
            let isLast = number == tricks.count - 1
            let winnerSide = camp(of: trick.winner)
            let excuseWonIt = trick.plays[0].card == .excuse && trick.winner == trick.plays[0].player

            for play in trick.plays {
                guard play.card == .excuse else {
                    add(play.card, to: winnerSide)
                    continue
                }
                let owner = camp(of: play.player)
                if excuseWonIt || owner == winnerSide {
                    add(.excuse, to: owner)
                } else if isLast {
                    // Played to the last trick outside a Chelem, the Excuse
                    // changes sides.
                    add(.excuse, to: winnerSide)
                } else {
                    // The Excuse stays with its side, which hands the trick's
                    // winners a worthless card (½ point) in exchange. Net
                    // effect: owner +4, winners +½. This also covers a Chelem
                    // by a taker who does not hold the Excuse, where the FFT
                    // counts it as 4 points for the defence.
                    add(.excuse, to: owner, halfPoints: Card.excuse.halfPoints - 1)
                    half[winnerSide, default: 0] += 1
                }
            }
        }

        return Tally(
            takerHalfPoints: half[.taker, default: 0],
            defenseHalfPoints: half[.defense, default: 0],
            takerOudlers: oudlers[.taker, default: 0]
        )
    }

    /// The side that won every trick, if any.
    public var chelem: Camp? {
        guard phase == .finished, let first = tricks.first else { return nil }
        let side = camp(of: first.winner)
        return tricks.allSatisfy { camp(of: $0.winner) == side } ? side : nil
    }

    /// The side that won the last trick with the Petit in it.
    ///
    /// When a Chelem ends with the Excuse led to the last trick, the Petit
    /// counts as "au bout" on the trick before.
    public var petitAuBout: Camp? {
        guard phase == .finished, let last = tricks.last else { return nil }
        if last.plays.contains(where: { $0.card == .trump(1) }) {
            return camp(of: last.winner)
        }
        if chelem != nil, last.plays[0].card == .excuse, tricks.count >= 2 {
            let penultimate = tricks[tricks.count - 2]
            if penultimate.plays.contains(where: { $0.card == .trump(1) }) {
                return camp(of: penultimate.winner)
            }
        }
        return nil
    }

    public var score: ScoreResult? {
        guard let tally, let contract else { return nil }
        return Scoring.score(ScoreInput(
            table: table,
            contract: contract,
            takerHalfPoints: tally.takerHalfPoints,
            takerOudlers: tally.takerOudlers,
            poignees: Array(poignees.values),
            petitAuBout: petitAuBout,
            chelemAnnounced: chelemAnnounced,
            chelem: chelem
        ))
    }

    /// Each seat's score for the deal, summing to zero.
    public var scoresBySeat: [Int]? {
        guard let score, let taker else { return nil }
        return (0..<table.playerCount).map { $0 == taker ? score.taker : score.perDefender }
    }
}

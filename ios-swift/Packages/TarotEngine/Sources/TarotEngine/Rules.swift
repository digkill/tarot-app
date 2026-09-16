/// Table sizes supported by the engine.
///
/// Five players is deliberately left out for now: the FFT notes that 5-player
/// tournaments are not officially recognised, and it adds the called-king
/// partnership, which changes the whole flow rather than a few constants.
public enum TableSize: Int, Sendable, Codable, CaseIterable {
    case three = 3
    case four = 4

    public var playerCount: Int { rawValue }

    /// 3 players: 24 cards each. 4 players: 18 each. The Chien is 6 either way.
    public var handSize: Int {
        switch self {
        case .three: return 24
        case .four: return 18
        }
    }

    public var chienSize: Int { 6 }

    /// Exact trump counts for a simple, double and triple Poignée.
    public var poigneeSizes: [Int] {
        switch self {
        case .three: return [13, 15, 18]
        case .four: return [10, 13, 15]
        }
    }

    /// The taker is scored against every defender, so the taker's total is the
    /// per-defender amount times the number of defenders — which keeps the
    /// table's scores summing to zero.
    public var takerScoreFactor: Int { playerCount - 1 }
}

/// Contracts in ascending order.
public enum Contract: Int, CaseIterable, Comparable, Sendable, Codable {
    /// Prise, also called Petite.
    case petite = 1
    case garde
    /// Garde sans le Chien: the Chien stays face down and counts for the taker.
    case gardeSans
    /// Garde contre le Chien: the Chien stays face down and counts for the defence.
    case gardeContre

    public var multiplier: Int {
        switch self {
        case .petite: return 1
        case .garde: return 2
        case .gardeSans: return 4
        case .gardeContre: return 6
        }
    }

    /// On a Prise or a Garde the taker reveals the Chien, takes it into hand
    /// and discards the same number of cards (the Écart).
    public var takerTakesChien: Bool {
        self == .petite || self == .garde
    }

    public static func < (lhs: Contract, rhs: Contract) -> Bool {
        lhs.rawValue < rhs.rawValue
    }
}

public enum Camp: Sendable, Codable, Equatable {
    case taker
    case defense
}

public enum Rules {
    /// Points the taker needs, by the number of Oudlers in the taker's pile.
    public static func pointsNeeded(oudlers: Int) -> Int {
        switch oudlers {
        case 0: return 56
        case 1: return 51
        case 2: return 41
        default: return 36
        }
    }

    /// A player holding the Petit as their only trump and without the Excuse
    /// must show it, and the deal is cancelled.
    public static func isPetitSec(_ hand: [Card]) -> Bool {
        let trumps = hand.filter(\.isTrump)
        return trumps == [.trump(1)] && !hand.contains(.excuse)
    }

    /// The cards a player may legally play, given what is already on the table
    /// in the current trick (in playing order).
    ///
    /// FFT, "Le jeu de la carte":
    /// - When trumps are led, a player must play above the highest trump
    ///   already played, even if a partner played it; with no higher trump,
    ///   any trump.
    /// - When a suit is led, a player must follow suit, but need not go higher.
    /// - With no card of the led suit, a player must trump; if a trump is
    ///   already on the table, must overtrump, or undertrump if unable.
    /// - With neither the suit nor a trump, any card.
    /// - If the Excuse is led, the next card played sets the suit.
    /// - The Excuse can always be played.
    public static func legalCards(hand: [Card], trick: [Card]) -> [Card] {
        guard let led = trick.first(where: { $0 != .excuse }) else {
            // Leading, or following an Excuse lead: anything goes.
            return hand
        }

        let hasExcuse = hand.contains(.excuse)
        func withExcuse(_ cards: [Card]) -> [Card] {
            hasExcuse ? cards + [.excuse] : cards
        }

        let trumpsInHand = hand.filter(\.isTrump)
        let highestTrumpOnTable = trick.compactMap { card -> Int? in
            if case let .trump(value) = card { return value }
            return nil
        }.max()

        /// Trumps the player is obliged to choose from.
        func forcedTrumps() -> [Card] {
            if let highest = highestTrumpOnTable {
                let higher = trumpsInHand.filter { card in
                    if case let .trump(value) = card { return value > highest }
                    return false
                }
                if !higher.isEmpty { return higher }
            }
            return trumpsInHand
        }

        switch led {
        case let .suited(ledSuit, _):
            let following = hand.filter { $0.suit == ledSuit }
            if !following.isEmpty {
                return withExcuse(following)
            }
            if !trumpsInHand.isEmpty {
                return withExcuse(forcedTrumps())
            }
            return hand
        case .trump:
            if !trumpsInHand.isEmpty {
                return withExcuse(forcedTrumps())
            }
            return hand
        case .excuse:
            // Unreachable: `led` skips the Excuse.
            return hand
        }
    }

    /// Index, within `trick`, of the card that wins it.
    ///
    /// The Excuse never wins a trick, with a single exception handled by the
    /// caller: led to the last trick by a side that has won every trick so far
    /// (a Chelem), it takes that trick. Pass `excuseLeadWins` for that case.
    public static func winningIndex(of trick: [Card], excuseLeadWins: Bool = false) -> Int {
        precondition(!trick.isEmpty, "an empty trick has no winner")
        if excuseLeadWins, trick.first == .excuse {
            return 0
        }
        guard let ledIndex = trick.firstIndex(where: { $0 != .excuse }) else {
            return 0
        }
        let ledSuit = trick[ledIndex].suit

        var best = ledIndex
        for index in trick.indices where index > ledIndex && trick[index] != .excuse {
            if beats(trick[index], trick[best], ledSuit: ledSuit) {
                best = index
            }
        }
        return best
    }

    private static func beats(_ challenger: Card, _ current: Card, ledSuit: Suit?) -> Bool {
        switch (challenger, current) {
        case let (.trump(a), .trump(b)):
            return a > b
        case (.trump, _):
            return true
        case (_, .trump):
            return false
        case let (.suited(suit, a), .suited(currentSuit, b)):
            // Only the led suit can win; the current best is always either a
            // trump or of the led suit.
            return suit == ledSuit && currentSuit == ledSuit && a > b
        default:
            return false
        }
    }

    /// Whether `ecart` is an acceptable discard for a taker holding `hand`
    /// (the hand *including* the Chien).
    ///
    /// FFT, "Le Chien et l'Écart": no King and no Oudler may be discarded;
    /// trumps only when it is unavoidable, and they must be shown to the
    /// defence.
    public static func ecartProblem(_ ecart: [Card], hand: [Card], size: Int) -> String? {
        guard ecart.count == size else {
            return "the écart must be exactly \(size) cards"
        }
        guard Set(ecart).count == ecart.count else {
            return "the écart contains the same card twice"
        }
        var remaining = hand
        for card in ecart {
            guard let index = remaining.firstIndex(of: card) else {
                return "\(card) is not in the taker's hand"
            }
            remaining.remove(at: index)
        }
        if let king = ecart.first(where: \.isKing) {
            return "a King cannot be discarded (\(king))"
        }
        if let oudler = ecart.first(where: \.isOudler) {
            return "an Oudler cannot be discarded (\(oudler))"
        }
        let trumpsDiscarded = ecart.filter(\.isTrump).count
        if trumpsDiscarded > 0 {
            let otherDiscardable = hand.filter { !$0.isTrump && !$0.isKing && !$0.isOudler }.count
            if trumpsDiscarded > max(0, size - otherDiscardable) {
                return "trumps may only be discarded when there is nothing else to discard"
            }
        }
        return nil
    }

    /// A legal écart that gives away as few points as possible: plain suit
    /// cards first, cheapest first, and trumps — the lowest ones — only if
    /// the hand leaves no choice. Good enough for AI opponents and tests; a
    /// strong player also discards to create voids.
    public static func cheapestEcart(hand: [Card], size: Int) -> [Card] {
        let plain = hand
            .filter { !$0.isTrump && !$0.isKing && !$0.isOudler }
            .sorted { lhs, rhs in
                if lhs.halfPoints != rhs.halfPoints { return lhs.halfPoints < rhs.halfPoints }
                return lhs.description < rhs.description
            }
        var ecart = Array(plain.prefix(size))
        if ecart.count < size {
            let trumps = hand
                .filter { $0.isTrump && !$0.isOudler }
                .sorted { lhs, rhs in
                    if case let (.trump(a), .trump(b)) = (lhs, rhs) { return a < b }
                    return false
                }
            ecart.append(contentsOf: trumps.prefix(size - ecart.count))
        }
        return ecart
    }
}

/// A declared handful of trumps.
public enum Poignee: Int, Sendable, Codable, CaseIterable {
    case simple
    case double
    case triple

    /// FFT: 20, 30 and 40 points, the same whatever the contract — not
    /// multiplied — and won by the side that wins the deal.
    public var bonus: Int {
        switch self {
        case .simple: return 20
        case .double: return 30
        case .triple: return 40
        }
    }
}

public enum PoigneeError: Error, Equatable, Sendable {
    case wrongSize(Int)
    case notInHand(Card)
    case notATrump(Card)
    /// The Excuse may stand in for a trump only when it reveals that the
    /// player has no other trump.
    case excuseWhileHidingTrumps
}

extension Poignee {
    /// Validates the cards a player shows as a Poignée.
    ///
    /// FFT, "La Poignée": it must contain exactly 10, 13 or 15 trumps (at four
    /// players); a player with 11, 12, 14… must hide the extras. The Excuse
    /// can replace a missing trump, but showing it means the player has no
    /// other trump.
    public static func validate(shown: [Card], hand: [Card], table: TableSize) -> Result<Poignee, PoigneeError> {
        guard let index = table.poigneeSizes.firstIndex(of: shown.count),
              let kind = Poignee(rawValue: index) else {
            return .failure(.wrongSize(shown.count))
        }
        for card in shown {
            guard hand.contains(card) else { return .failure(.notInHand(card)) }
            guard card.isTrump || card == .excuse else { return .failure(.notATrump(card)) }
        }
        if shown.contains(.excuse) {
            let shownTrumps = shown.filter(\.isTrump).count
            let heldTrumps = hand.filter(\.isTrump).count
            if shownTrumps != heldTrumps {
                return .failure(.excuseWhileHidingTrumps)
            }
        }
        return .success(kind)
    }
}

/// Everything the score of one deal depends on.
public struct ScoreInput: Sendable, Equatable {
    public var table: TableSize
    public var contract: Contract
    /// The taker's pile, in half-points (182 is the whole deck).
    public var takerHalfPoints: Int
    /// Oudlers in the taker's pile.
    public var takerOudlers: Int
    /// Every Poignée declared at the table, by either side.
    public var poignees: [Poignee]
    /// The side that won the last trick while it contained the Petit.
    public var petitAuBout: Camp?
    public var chelemAnnounced: Bool
    /// The side that won every trick, if any.
    public var chelem: Camp?

    public init(
        table: TableSize,
        contract: Contract,
        takerHalfPoints: Int,
        takerOudlers: Int,
        poignees: [Poignee] = [],
        petitAuBout: Camp? = nil,
        chelemAnnounced: Bool = false,
        chelem: Camp? = nil
    ) {
        self.table = table
        self.contract = contract
        self.takerHalfPoints = takerHalfPoints
        self.takerOudlers = takerOudlers
        self.poignees = poignees
        self.petitAuBout = petitAuBout
        self.chelemAnnounced = chelemAnnounced
        self.chelem = chelem
    }
}

public struct ScoreResult: Sendable, Equatable {
    public var takerWon: Bool
    /// Points needed for the contract.
    public var needed: Int
    /// How far above or below the target the taker finished, in whole points
    /// after the half-point rule.
    public var margin: Int
    /// The signed amount each defender pays (negative) or receives (positive).
    public var perDefender: Int
    /// The taker's score for the deal.
    public var taker: Int
}

public enum Scoring {
    /// Scores one deal.
    ///
    /// FFT, "Le calcul des scores": a contract is worth 25 points; add the
    /// margin above or below the target; multiply by the contract (×1, ×2,
    /// ×4, ×6); then add the Poignée, Petit au Bout and Chelem bonuses. Each
    /// defender scores the opposite of that total and the taker scores it once
    /// per defender, so the table always sums to zero.
    public static func score(_ input: ScoreInput) -> ScoreResult {
        let needed = Rules.pointsNeeded(oudlers: input.takerOudlers)
        let difference = input.takerHalfPoints - needed * 2
        let takerWon = difference >= 0 // "juste fait" — exactly on target — wins.

        // Half-point rule: a stray half point goes to the winning side, so the
        // margin always rounds away from zero. 40½ with two Oudlers (target 41)
        // fails by 1; 41½ wins by 1.
        let margin = (abs(difference) + 1) / 2

        let multiplier = input.contract.multiplier
        let sign = takerWon ? 1 : -1

        var total = (25 + margin) * multiplier * sign
        total += sign * input.poignees.reduce(0) { $0 + $1.bonus }

        switch input.petitAuBout {
        case .taker: total += 10 * multiplier
        case .defense: total -= 10 * multiplier
        case nil: break
        }

        // Chelem bonuses are not multiplied. The combination of a failed
        // announced Chelem *and* a Chelem made by the defence is not spelled
        // out by the FFT; both penalties are applied here.
        if input.chelem == .taker {
            total += input.chelemAnnounced ? 400 : 200
        } else if input.chelemAnnounced {
            total -= 200
        }
        if input.chelem == .defense {
            total -= 200
        }

        return ScoreResult(
            takerWon: takerWon,
            needed: needed,
            margin: margin,
            perDefender: -total,
            taker: total * input.table.takerScoreFactor
        )
    }
}

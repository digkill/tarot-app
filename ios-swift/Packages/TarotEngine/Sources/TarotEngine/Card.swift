/// The four suits. Their order only matters for display.
public enum Suit: Int, CaseIterable, Sendable, Codable, Hashable {
    case spades, hearts, diamonds, clubs
}

/// One of the 78 cards of a Tarot deck.
///
/// Rules source: Fédération Française de Tarot, *Règlement officiel du jeu de
/// Tarot*, version of 1 July 2012, section "Les cartes".
public enum Card: Hashable, Sendable, Codable {
    /// A suit card. Rank 1–10 are the pips (1 is the ace, the weakest), then
    /// the honours: 11 Jack (Valet), 12 Knight (Cavalier), 13 Queen (Dame),
    /// 14 King (Roi).
    case suited(Suit, Int)
    /// A trump (Atout), 1–21. The 1 is called the Petit.
    case trump(Int)
    /// The Excuse — the Fool. Neither a trump nor a suit card.
    case excuse

    public static let jack = 11
    public static let knight = 12
    public static let queen = 13
    public static let king = 14

    /// The full 78-card deck: 56 suit cards, 21 trumps and the Excuse.
    public static let fullDeck: [Card] = {
        var deck: [Card] = []
        for suit in Suit.allCases {
            for rank in 1...king {
                deck.append(.suited(suit, rank))
            }
        }
        for value in 1...21 {
            deck.append(.trump(value))
        }
        deck.append(.excuse)
        return deck
    }()

    public var isTrump: Bool {
        if case .trump = self { return true }
        return false
    }

    public var isKing: Bool {
        if case .suited(_, Card.king) = self { return true }
        return false
    }

    /// The three Oudlers (Bouts): the Petit, the 21 and the Excuse. How many of
    /// them the taker ends up with decides how many points the contract needs.
    public var isOudler: Bool {
        switch self {
        case .trump(1), .trump(21), .excuse: return true
        default: return false
        }
    }

    public var suit: Suit? {
        if case let .suited(suit, _) = self { return suit }
        return nil
    }

    /// Card value in half-points, so all arithmetic stays exact.
    ///
    /// Official values: Oudler 4.5, King 4.5, Queen 3.5, Knight 2.5, Jack 1.5,
    /// any other card 0.5. The deck totals 91 points (182 half-points). Every
    /// card is worth an odd number of half-points, which is why a pile with an
    /// even number of cards always totals a whole number of points.
    public var halfPoints: Int {
        switch self {
        case .excuse, .trump(1), .trump(21):
            return 9
        case .trump:
            return 1
        case let .suited(_, rank):
            switch rank {
            case Card.king: return 9
            case Card.queen: return 7
            case Card.knight: return 5
            case Card.jack: return 3
            default: return 1
            }
        }
    }
}

extension Card: CustomStringConvertible {
    public var description: String {
        switch self {
        case .excuse:
            return "Excuse"
        case let .trump(value):
            return "T\(value)"
        case let .suited(suit, rank):
            let symbol = ["♠", "♥", "♦", "♣"][suit.rawValue]
            let name: String
            switch rank {
            case Card.king: name = "K"
            case Card.queen: name = "Q"
            case Card.knight: name = "C"
            case Card.jack: name = "J"
            default: name = String(rank)
            }
            return name + symbol
        }
    }
}

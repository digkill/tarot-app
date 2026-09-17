import Foundation

// Mirrors of the Arcana Clash server's JSON (snake_case on the wire, decoded
// with `convertFromSnakeCase`). The client only ever renders these values; it
// computes no game numbers of its own.

enum ArcanaTargetKind: String, Codable, Sendable, Hashable {
    case enemy, `self`, handCard = "hand_card", discardCard = "discard_card", none
}

struct ArcanaTarget: Codable, Sendable, Hashable {
    var kind: ArcanaTargetKind
    var cardUid: String?
}

enum ArcanaPhase: String, Codable, Sendable {
    case mulligan, playing, finished
}

struct ArcanaCardView: Codable, Sendable, Hashable, Identifiable {
    var uid: String
    var cardId: String?
    var reversed: Bool?
    var cost: Int
    var baseCost: Int
    var hidden: Bool?
    var playable: Bool?
    var targets: [ArcanaTargetKind]?

    var id: String { uid }
    var isReversed: Bool { reversed ?? false }
    var isPlayable: Bool { playable ?? false }
}

struct ArcanaStatusView: Codable, Sendable, Hashable {
    var name: String
    var amount: Int
    var turns: Int?
}

struct ArcanaHeroView: Codable, Sendable, Hashable {
    var id: String
    var hp: Int
    var maxHp: Int
    var armor: Int
    var charge: Int
    var ultimateCost: Int
    var souls: Int
    var ultimateReady: Bool
}

struct ArcanaPlayerView: Codable, Sendable, Hashable {
    var id: String
    var hero: ArcanaHeroView
    var mana: Int
    var maxMana: Int
    var hand: [ArcanaCardView]
    var deckCount: Int
    var discard: [ArcanaCardView]
    var statuses: [ArcanaStatusView]
    var abilityCost: Int
    var abilityUsable: Bool
    var abilityTargets: [ArcanaTargetKind]?
    var ultimateUsable: Bool
    var ultimateTargets: [ArcanaTargetKind]?
    var mulliganed: Bool
    var timeoutStreak: Int
}

struct ArcanaOpponentView: Codable, Sendable, Hashable {
    var id: String
    var hero: ArcanaHeroView
    var mana: Int
    var maxMana: Int
    var handCount: Int
    var deckCount: Int
    var discard: [ArcanaCardView]
    var statuses: [ArcanaStatusView]
    var mulliganed: Bool
}

struct ArcanaEvent: Codable, Sendable, Hashable {
    var kind: String
    var player: String?
    var amount: Int?
    var absorbed: Int?
    var crit: Bool?
    var element: String?
    var status: String?
    var cardId: String?
    var cardUid: String?
    var reason: String?
}

struct ArcanaLogEntry: Codable, Sendable, Hashable, Identifiable {
    var seq: UInt64
    var player: String?
    var action: String
    var cardUid: String?
    var cardId: String?
    var reversed: Bool?
    var hidden: Bool?
    var events: [ArcanaEvent]?

    var id: UInt64 { seq }
}

struct ArcanaGameView: Codable, Sendable, Hashable {
    var rulesVersion: Int
    var seq: UInt64
    var phase: ArcanaPhase
    var turn: Int
    var activePlayer: String
    var you: ArcanaPlayerView
    var opponent: ArcanaOpponentView
    var log: [ArcanaLogEntry]
    var winner: String?
    var reason: String?

    var isMyTurn: Bool { phase == .playing && activePlayer == you.id }
}

// MARK: - Frames

struct ArcanaHelloPayload: Decodable, Sendable {
    var userId: String
    var protocol_: Int
    var rulesVersion: Int
    var activeMatch: String?
    var emotes: [String]

    enum CodingKeys: String, CodingKey {
        case userId, protocol_ = "protocol", rulesVersion, activeMatch, emotes
    }
}

struct ArcanaErrorPayload: Decodable, Sendable, Equatable {
    var code: String
    var message: String
    var ref: String?
}

struct ArcanaPlayerInfo: Decodable, Sendable, Equatable {
    var id: String
    var hero: String
    /// The shop deck whose art this player is seen with; nil is the classic.
    var deck: String?
}

struct ArcanaStartedPayload: Decodable, Sendable, Equatable {
    var you: ArcanaPlayerInfo
    var opponent: ArcanaPlayerInfo
}

struct ArcanaStatePayload: Decodable, Sendable {
    var state: ArcanaGameView
    var serverTime: Int64
    var deadlineAt: Int64
    var opponentConnected: Bool
    var yourDeck: String?
    var opponentDeck: String?
}

struct ArcanaEmotePayload: Decodable, Sendable, Equatable {
    var player: String
    var emote: String
}

struct ArcanaFinishedPayload: Decodable, Sendable {
    var winner: String?
    var reason: String
    var result: String
    var state: ArcanaGameView
}

/// A decoded server frame.
enum ArcanaServerEvent: Sendable {
    case hello(ArcanaHelloPayload)
    case queueWaiting
    case queueLeft
    case matchStarted(matchId: String, ArcanaStartedPayload)
    case state(matchId: String, ArcanaStatePayload)
    case finished(matchId: String, ArcanaFinishedPayload)
    case emote(ArcanaEmotePayload)
    case opponentDisconnected
    case opponentReconnected
    case error(ArcanaErrorPayload)
    case unknown(String)

    private struct Header: Decodable {
        var type: String
        var seq: UInt64?
        var matchId: String?
    }

    private struct Envelope<P: Decodable>: Decodable {
        var payload: P
    }

    static func decode(_ data: Data) throws -> ArcanaServerEvent {
        let decoder = ArcanaCoding.decoder
        let header = try decoder.decode(Header.self, from: data)
        func payload<P: Decodable>(_: P.Type) throws -> P {
            try decoder.decode(Envelope<P>.self, from: data).payload
        }
        let matchId = header.matchId ?? ""
        switch header.type {
        case "hello": return .hello(try payload(ArcanaHelloPayload.self))
        case "queue.waiting": return .queueWaiting
        case "queue.left": return .queueLeft
        case "match.started": return .matchStarted(matchId: matchId, try payload(ArcanaStartedPayload.self))
        case "match.state": return .state(matchId: matchId, try payload(ArcanaStatePayload.self))
        case "match.finished": return .finished(matchId: matchId, try payload(ArcanaFinishedPayload.self))
        case "player.emote": return .emote(try payload(ArcanaEmotePayload.self))
        case "opponent.disconnected": return .opponentDisconnected
        case "opponent.reconnected": return .opponentReconnected
        case "error": return .error(try payload(ArcanaErrorPayload.self))
        default: return .unknown(header.type)
        }
    }
}

/// A client intent. Only ids and targets — never numbers.
struct ArcanaClientMessage: Encodable, Sendable, Equatable {
    var type: String
    var ref: String?
    var `protocol`: Int?
    var token: String?
    var matchId: String?
    var hero: String?
    var deck: String?
    var cardUid: String?
    var cardUids: [String]?
    var target: ArcanaTarget?
    var emote: String?
}

enum ArcanaCoding {
    static let protocolVersion = 1

    static var decoder: JSONDecoder {
        let d = JSONDecoder()
        d.keyDecodingStrategy = .convertFromSnakeCase
        d.dateDecodingStrategy = .custom(JSONCoding.decodeDate)
        return d
    }

    static var encoder: JSONEncoder {
        let e = JSONEncoder()
        e.keyEncodingStrategy = .convertToSnakeCase
        return e
    }
}

// MARK: - Catalog and history (HTTP)

struct ArcanaOp: Codable, Sendable, Hashable {
    var kind: String
    var who: String?
    var amount: Int?
    var bonus: Int?
    var turns: Int?
    var chance: Int?
    var status: String?
    var element: String?
    var pierce: Bool?
}

struct ArcanaSide: Codable, Sendable, Hashable {
    var targets: [ArcanaTargetKind]?
    var ops: [ArcanaOp]
}

struct ArcanaCardDef: Codable, Sendable, Hashable, Identifiable {
    var id: String
    var suit: String
    var cost: Int
    var upright: ArcanaSide
    var reversed: ArcanaSide

    func side(reversed isReversed: Bool) -> ArcanaSide { isReversed ? reversed : upright }
}

struct ArcanaPower: Codable, Sendable, Hashable {
    var cost: Int
    var chargeWithSouls: Bool?
    var side: ArcanaSide
}

struct ArcanaHeroDef: Codable, Sendable, Hashable, Identifiable {
    var id: String
    var hp: Int
    var passive: String
    var ability: ArcanaPower
    var ultimate: ArcanaPower

    /// The tarot card this hero is drawn from (its name and portrait).
    var cardId: String { id == "strength" ? "the_strength" : id }
}

struct ArcanaRules: Codable, Sendable, Hashable {
    var deckSize: Int
    var startHand: Int
    var handLimit: Int
    var manaCap: Int
    var mulliganMax: Int
    var turnSeconds: Int
    var mulliganSeconds: Int
    var reconnectSeconds: Int
}

struct ArcanaCatalog: Codable, Sendable, Hashable {
    var protocolVersion: Int
    var rulesVersion: Int
    var rules: ArcanaRules
    var heroes: [ArcanaHeroDef]
    var cards: [ArcanaCardDef]
    var emotes: [String]

    func card(_ id: String?) -> ArcanaCardDef? { cards.first { $0.id == id } }
    func hero(_ id: String?) -> ArcanaHeroDef? { heroes.first { $0.id == id } }
}

struct ArcanaMatchSummary: Codable, Sendable, Hashable, Identifiable {
    var id: String
    var opponentId: String?
    var hero: String
    var opponentHero: String
    var deck: String?
    var opponentDeck: String?
    var result: String
    var reason: String
    var turns: Int
    var startedAt: Date
    var finishedAt: Date
    var durationMs: Int64
}

import Foundation

struct InterpretationRequest: Encodable, Sendable {
    struct Card: Encodable, Sendable {
        var positionIndex: Int
        var positionTitle: String
        var positionDescription: String
        var cardName: String
        var uprightMeaning: String
        var reversedMeaning: String
        /// The bundled card data has no keywords; the server accepts empty lists.
        var uprightKeywords: [String] = []
        var reversedKeywords: [String] = []
        var isReversed: Bool
    }

    var spreadId: String
    var spreadName: String
    var spreadDescription: String
    var language: String
    var cards: [Card]
}

struct InterpretationResponse: Decodable, Sendable {
    struct Position: Decodable, Sendable {
        var positionIndex: Int
        var positionTitle: String
        var cardName: String
        var orientation: String
        var meaning: String
    }

    var summary: String
    var positions: [Position]
}

/// The premium AI reading.
///
/// Errors worth distinguishing: `premium_required` (402), `quota_exceeded`
/// (429, daily limit), `rate_limited` (429, too soon after the last one),
/// `llm_unavailable` (503) and `llm_error` (502).
struct InterpretationAPI: Sendable {
    let client: APIClient

    func interpret(_ request: InterpretationRequest) async throws -> InterpretationResponse {
        try await client.send(.post, "/api/v1/interpretations", body: request, auth: .required)
    }
}

extension InterpretationRequest {
    init(spread: Spread, entries: [DrawnCard], localizer l: Localizer) {
        self.init(
            spreadId: spread.id,
            spreadName: l.t(spread.nameKey),
            spreadDescription: l.t(spread.descriptionKey),
            language: l.language.rawValue,
            cards: entries.map { entry in
                Card(
                    positionIndex: entry.position.index,
                    positionTitle: l.t(entry.position.titleKey),
                    positionDescription: l.t(entry.position.descriptionKey),
                    cardName: entry.card.name,
                    uprightMeaning: entry.card.upright,
                    reversedMeaning: entry.card.reversed,
                    isReversed: entry.isReversed
                )
            }
        )
    }
}

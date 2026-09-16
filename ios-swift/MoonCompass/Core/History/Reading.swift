import Foundation

struct ReadingItem: Codable, Hashable, Sendable {
    var positionIndex: Int
    /// A `TarotCard.id`, e.g. `the_fool`.
    var cardId: String
    var isReversed: Bool
}

struct AIInsight: Codable, Hashable, Sendable {
    struct Position: Codable, Hashable, Sendable {
        var positionIndex: Int
        var positionTitle: String
        var cardName: String
        var orientation: String
        var meaning: String
    }

    var summary: String
    var positions: [Position]
    var language: Language
    var generatedAt: Date
}

/// A saved reading in the journal.
struct Reading: Codable, Identifiable, Hashable, Sendable {
    enum Kind: String, Codable, Sendable {
        /// The first one-card draw of the day.
        case daily
        /// Another one-card draw the same day.
        case bonus
    }

    var id: String
    var spreadId: String
    var deckId: String
    var drawnAt: Date
    var items: [ReadingItem]
    var summaryText: String
    var notes = ""
    var favorite = false
    var aiInsights: AIInsight?
    var kind: Kind?

    init(id: String = UUID().uuidString.lowercased(), spreadId: String, deckId: String,
         drawnAt: Date = Date(), items: [ReadingItem], summaryText: String,
         notes: String = "", favorite: Bool = false, aiInsights: AIInsight? = nil, kind: Kind? = nil) {
        self.id = id
        self.spreadId = spreadId
        self.deckId = deckId
        self.drawnAt = drawnAt
        self.items = items
        self.summaryText = summaryText
        self.notes = notes
        self.favorite = favorite
        self.aiInsights = aiInsights
        self.kind = kind
    }
}

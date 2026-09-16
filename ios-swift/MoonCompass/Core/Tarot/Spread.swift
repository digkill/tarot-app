import Foundation

enum SpreadCategory: String, Codable, CaseIterable, Sendable {
    case basic, love, career, year, weekly, spiritual, custom

    var titleKey: String { "spread.category.\(rawValue)" }
}

struct SpreadPosition: Hashable, Sendable {
    let index: Int
    let titleKey: String
    let descriptionKey: String
    /// Centre of the card as a fraction of the table's width and height.
    let x: Double
    let y: Double
    /// Degrees, e.g. the crossing card of the Celtic Cross.
    var rotation: Double = 0
}

struct Spread: Identifiable, Hashable, Sendable {
    let id: String
    let nameKey: String
    let descriptionKey: String
    let category: SpreadCategory
    let positions: [SpreadPosition]
    var premium = false

    var cardCount: Int { positions.count }

    static let oneCardId = "one-card"

    var isOneCard: Bool { id == Self.oneCardId }

    func position(index: Int) -> SpreadPosition? {
        positions.first { $0.index == index }
    }

    static func find(_ id: String) -> Spread? {
        all.first { $0.id == id }
    }

    /// The "quick spreads" row on the home screen.
    static let basicIds = ["one-card", "three-card", "celtic-cross"]

    /// Ported from `data/spreads.ts`.
    static let all: [Spread] = [
        Spread(
            id: "one-card", nameKey: "spread.oneCard.name", descriptionKey: "spread.oneCard.description",
            category: .basic,
            positions: [SpreadPosition(index: 1, titleKey: "position.focus", descriptionKey: "position.focus.description", x: 0.5, y: 0.5)]
        ),
        Spread(
            id: "three-card", nameKey: "spread.threeCard.name", descriptionKey: "spread.threeCard.description",
            category: .basic,
            positions: [
                SpreadPosition(index: 1, titleKey: "position.past", descriptionKey: "position.past.description", x: 0.18, y: 0.5),
                SpreadPosition(index: 2, titleKey: "position.present", descriptionKey: "position.present.description", x: 0.5, y: 0.5),
                SpreadPosition(index: 3, titleKey: "position.future", descriptionKey: "position.future.description", x: 0.82, y: 0.5),
            ]
        ),
        Spread(
            id: "celtic-cross", nameKey: "spread.celticCross.name", descriptionKey: "spread.celticCross.description",
            category: .spiritual,
            positions: [
                SpreadPosition(index: 1, titleKey: "celtic.present", descriptionKey: "celtic.present.description", x: 0.32, y: 0.5),
                SpreadPosition(index: 2, titleKey: "celtic.challenge", descriptionKey: "celtic.challenge.description", x: 0.38, y: 0.44, rotation: 90),
                SpreadPosition(index: 3, titleKey: "celtic.past", descriptionKey: "celtic.past.description", x: 0.16, y: 0.5),
                SpreadPosition(index: 4, titleKey: "celtic.future", descriptionKey: "celtic.future.description", x: 0.5, y: 0.5),
                SpreadPosition(index: 5, titleKey: "celtic.conscious", descriptionKey: "celtic.conscious.description", x: 0.32, y: 0.22),
                SpreadPosition(index: 6, titleKey: "celtic.subconscious", descriptionKey: "celtic.subconscious.description", x: 0.32, y: 0.78),
                SpreadPosition(index: 7, titleKey: "celtic.advice", descriptionKey: "celtic.advice.description", x: 0.72, y: 0.88),
                SpreadPosition(index: 8, titleKey: "celtic.external", descriptionKey: "celtic.external.description", x: 0.72, y: 0.63),
                SpreadPosition(index: 9, titleKey: "celtic.hopes", descriptionKey: "celtic.hopes.description", x: 0.72, y: 0.38),
                SpreadPosition(index: 10, titleKey: "celtic.outcome", descriptionKey: "celtic.outcome.description", x: 0.72, y: 0.13),
            ]
        ),
        Spread(
            id: "love-relationship", nameKey: "spread.love.name", descriptionKey: "spread.love.description",
            category: .love,
            positions: [
                SpreadPosition(index: 1, titleKey: "love.you", descriptionKey: "love.you.description", x: 0.18, y: 0.25),
                SpreadPosition(index: 2, titleKey: "love.partner", descriptionKey: "love.partner.description", x: 0.82, y: 0.25),
                SpreadPosition(index: 3, titleKey: "love.connection", descriptionKey: "love.connection.description", x: 0.5, y: 0.25),
                SpreadPosition(index: 4, titleKey: "love.strengths", descriptionKey: "love.strengths.description", x: 0.28, y: 0.62),
                SpreadPosition(index: 5, titleKey: "love.challenges", descriptionKey: "love.challenges.description", x: 0.72, y: 0.62),
                SpreadPosition(index: 6, titleKey: "love.advice", descriptionKey: "love.advice.description", x: 0.38, y: 0.88),
                SpreadPosition(index: 7, titleKey: "love.outcome", descriptionKey: "love.outcome.description", x: 0.62, y: 0.88),
            ],
            premium: true
        ),
        Spread(
            id: "horseshoe", nameKey: "spread.horseshoe.name", descriptionKey: "spread.horseshoe.description",
            category: .career,
            positions: [
                SpreadPosition(index: 1, titleKey: "horseshoe.past", descriptionKey: "horseshoe.past.description", x: 0.12, y: 0.72),
                SpreadPosition(index: 2, titleKey: "horseshoe.present", descriptionKey: "horseshoe.present.description", x: 0.28, y: 0.5),
                SpreadPosition(index: 3, titleKey: "horseshoe.hidden", descriptionKey: "horseshoe.hidden.description", x: 0.42, y: 0.32),
                SpreadPosition(index: 4, titleKey: "horseshoe.guidance", descriptionKey: "horseshoe.guidance.description", x: 0.58, y: 0.22),
                SpreadPosition(index: 5, titleKey: "horseshoe.block", descriptionKey: "horseshoe.block.description", x: 0.72, y: 0.32),
                SpreadPosition(index: 6, titleKey: "horseshoe.newInfluence", descriptionKey: "horseshoe.newInfluence.description", x: 0.86, y: 0.5),
                SpreadPosition(index: 7, titleKey: "horseshoe.outcome", descriptionKey: "horseshoe.outcome.description", x: 0.72, y: 0.75),
            ],
            premium: true
        ),
        Spread(
            id: "weekly", nameKey: "spread.weekly.name", descriptionKey: "spread.weekly.description",
            category: .weekly,
            positions: (0..<7).map { day in
                SpreadPosition(index: day + 1, titleKey: "weekly.day\(day + 1)",
                               descriptionKey: "weekly.day\(day + 1).description",
                               x: 0.08 + Double(day) * 0.14, y: 0.5)
            },
            premium: true
        ),
        Spread(
            id: "year-wheel", nameKey: "spread.yearWheel.name", descriptionKey: "spread.yearWheel.description",
            category: .year,
            positions: (0..<12).map { month in
                let angle = Double(month) / 12 * .pi * 2
                return SpreadPosition(index: month + 1, titleKey: "year.month\(month + 1)",
                                      descriptionKey: "year.month\(month + 1).description",
                                      x: 0.5 + cos(angle) * 0.35, y: 0.5 + sin(angle) * 0.35,
                                      rotation: angle * 180 / .pi)
            },
            premium: true
        ),
    ]
}

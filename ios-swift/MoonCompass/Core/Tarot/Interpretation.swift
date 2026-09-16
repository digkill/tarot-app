import Foundation

/// A card placed in a spread position.
struct DrawnCard: Hashable, Sendable {
    let card: TarotCard
    let position: SpreadPosition
    let isReversed: Bool
}

/// The built-in, offline reading summary — the same text the React Native
/// client's `generateInterpretation` produces.
enum Interpretation {
    /// "Past: what shaped you — New beginnings, spontaneity…"
    ///
    /// Unlike the React Native text, each narrative ends with a full stop:
    /// card meanings have none, so consecutive positions used to run together
    /// ("…authority, truth Present: …").
    static func narrative(for entry: DrawnCard, localizer l: Localizer) -> String {
        var meaning = entry.card.meaning(reversed: entry.isReversed)
        if let last = meaning.last, !".!?…。".contains(last) {
            meaning += "."
        }
        return "\(l.t(entry.position.titleKey)): \(l.t(entry.position.descriptionKey)) — \(meaning)"
    }

    /// An intro, the first three positions' narratives and a closing line.
    static func summary(spread: Spread, entries: [DrawnCard], localizer l: Localizer) -> String {
        let intro = l.t("interpretation.summaryIntro", ["spread": l.t(spread.nameKey), "count": entries.count])
        let highlights = entries.prefix(3).map { narrative(for: $0, localizer: l) }.joined(separator: " ")
        let closing = l.t("interpretation.summaryClosing")
        return [intro, highlights, closing]
            .filter { !$0.isEmpty }
            .joined(separator: " ")
            .trimmingCharacters(in: .whitespaces)
    }

    /// Resolves a saved reading's cards against the spread and the catalog.
    /// Cards that can no longer be resolved are skipped rather than failing
    /// the whole reading.
    static func entries(for reading: Reading, spread: Spread, catalog: CardCatalog) -> [DrawnCard] {
        reading.items.compactMap { item in
            guard let position = spread.position(index: item.positionIndex),
                  let card = catalog.card(id: item.cardId) else { return nil }
            return DrawnCard(card: card, position: position, isReversed: item.isReversed)
        }
    }

    /// What to show for a reading in lists: the AI summary when there is one,
    /// otherwise the offline summary regenerated in the current language.
    static func displaySummary(for reading: Reading, catalog: CardCatalog, localizer l: Localizer) -> String {
        if let ai = reading.aiInsights?.summary, !ai.isEmpty {
            return ai
        }
        guard let spread = Spread.find(reading.spreadId) else { return reading.summaryText }
        let entries = entries(for: reading, spread: spread, catalog: catalog)
        guard !entries.isEmpty else { return reading.summaryText }
        return summary(spread: spread, entries: entries, localizer: l)
    }
}

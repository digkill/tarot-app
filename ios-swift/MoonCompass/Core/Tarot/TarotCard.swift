import Foundation

enum Arcana: String, Codable, Sendable {
    case major
    case minor
}

enum TarotSuit: String, Codable, CaseIterable, Sendable {
    case wands
    case cups
    case swords
    case pentacles
}

/// One card's text in the current language.
struct TarotCard: Identifiable, Hashable, Sendable {
    /// Stable across languages: the art's file name without extension, e.g.
    /// `the_fool`. The React Native client derived ids from the *translated*
    /// name, so a reading saved in Russian could not be resolved after a switch
    /// to English. This is also the key the server uses for remote deck art.
    let id: String
    let name: String
    let upright: String
    let reversed: String
    let arcana: Arcana
    let suit: TarotSuit?
    /// 0–21 for the Major Arcana; 1–14 within a suit (ace = 1, king = 14).
    let number: Int
    /// The bundled art file, e.g. `the_fool.jpeg`.
    let imageFile: String

    func meaning(reversed isReversed: Bool) -> String {
        isReversed ? reversed : upright
    }
}

/// The 78 cards in one language, loaded from `tarot_<lang>.json`.
///
/// Arcana, suit and number come from each card's position: the files list the
/// 22 Major Arcana, then Wands, Cups, Swords and Pentacles from ace to king,
/// in the same order in every language. The React Native client guessed the
/// arcana from English names only, so in Russian and Thai every card counted
/// as Minor and the arcana filter showed nothing.
struct CardCatalog: Sendable {
    let cards: [TarotCard]
    private let byId: [String: TarotCard]

    init(cards: [TarotCard]) {
        self.cards = cards
        byId = Dictionary(uniqueKeysWithValues: cards.map { ($0.id, $0) })
    }

    func card(id: String) -> TarotCard? {
        byId[id]
    }

    private struct RawCard: Decodable {
        let name: String
        let upright: String
        let reversed: String
        let image: String
    }

    /// Chinese has no card translations yet and uses English, as it did in
    /// the React Native client.
    static func dataLanguage(for language: Language) -> Language {
        language == .zh ? .en : language
    }

    static func load(_ language: Language, bundle: Bundle = .main) -> CardCatalog {
        let source = dataLanguage(for: language)
        guard let url = bundle.url(forResource: "tarot_\(source.rawValue)", withExtension: "json"),
              let data = try? Data(contentsOf: url),
              let raw = try? JSONDecoder().decode([RawCard].self, from: data) else {
            return CardCatalog(cards: [])
        }
        return CardCatalog(cards: raw.enumerated().map { index, card in
            let (arcana, suit, number) = classify(index: index)
            return TarotCard(
                id: (card.image as NSString).deletingPathExtension,
                name: card.name,
                upright: card.upright,
                reversed: card.reversed,
                arcana: arcana,
                suit: suit,
                number: number,
                imageFile: card.image
            )
        })
    }

    static func classify(index: Int) -> (Arcana, TarotSuit?, Int) {
        if index < 22 {
            return (.major, nil, index)
        }
        let minorIndex = index - 22
        let suit = TarotSuit.allCases[min(minorIndex / 14, TarotSuit.allCases.count - 1)]
        return (.minor, suit, minorIndex % 14 + 1)
    }
}

import Foundation
import Observation

/// A deck the user can see in the shop: the bundled classic Rider–Waite deck,
/// or one from the server.
struct DeckOption: Identifiable, Hashable, Sendable {
    let slug: String
    let titles: [String: String]
    let descriptions: [String: String]
    let colors: AppColors
    let priceKop: Int
    let originalPriceKop: Int?
    let appleProductId: String?
    let isFree: Bool
    /// The classic deck ships inside the app; its art is never downloaded.
    let isBundled: Bool
    /// The server says this account owns it.
    let ownedOnServer: Bool
    let coverURL: URL?
    let backURL: URL?
    /// Remote art by card key (`the_fool`). May cover fewer than 78 cards.
    let cardURLs: [String: URL]

    var id: String { slug }

    static let classicSlug = "rws"

    static let classic = DeckOption(
        slug: classicSlug, titles: [:], descriptions: [:], colors: .classic,
        priceKop: 0, originalPriceKop: nil, appleProductId: nil,
        isFree: true, isBundled: true, ownedOnServer: true,
        coverURL: nil, backURL: nil, cardURLs: [:]
    )

    init(slug: String, titles: [String: String], descriptions: [String: String], colors: AppColors,
         priceKop: Int, originalPriceKop: Int?, appleProductId: String?, isFree: Bool, isBundled: Bool,
         ownedOnServer: Bool, coverURL: URL?, backURL: URL?, cardURLs: [String: URL]) {
        self.slug = slug
        self.titles = titles
        self.descriptions = descriptions
        self.colors = colors
        self.priceKop = priceKop
        self.originalPriceKop = originalPriceKop
        self.appleProductId = appleProductId
        self.isFree = isFree
        self.isBundled = isBundled
        self.ownedOnServer = ownedOnServer
        self.coverURL = coverURL
        self.backURL = backURL
        self.cardURLs = cardURLs
    }

    init(_ deck: ShopDeck, config: APIConfig) {
        self.init(
            slug: deck.slug,
            titles: deck.titles,
            descriptions: deck.descriptions,
            colors: AppColors(payload: deck.theme),
            priceKop: deck.priceKop,
            originalPriceKop: deck.originalPriceKop,
            appleProductId: deck.appleProductId?.trimmingCharacters(in: .whitespaces).nilIfEmpty,
            isFree: deck.isFree,
            isBundled: false,
            ownedOnServer: deck.owned,
            coverURL: deck.coverURL(in: config),
            backURL: deck.hasBack ? deck.backURL(in: config) : nil,
            cardURLs: deck.cards.compactMapValues { config.mediaURL($0) }
        )
    }

    /// A translated title: the app's language, then English, then Russian,
    /// then the slug — the same fallback as the React Native `titleOf`.
    func title(in language: Language, localizer: Localizer) -> String {
        if isBundled { return localizer.t("deck.classicName") }
        return Self.localized(titles, language) ?? slug
    }

    func description(in language: Language, localizer: Localizer) -> String {
        if isBundled { return localizer.t("deck.classicHint") }
        return Self.localized(descriptions, language) ?? ""
    }

    private static func localized(_ values: [String: String], _ language: Language) -> String? {
        for key in [language.rawValue, "en", "ru"] {
            if let value = values[key]?.trimmingCharacters(in: .whitespacesAndNewlines), !value.isEmpty {
                return value
            }
        }
        return values.values.first { !$0.isEmpty }
    }
}

/// The deck shop: which decks exist, which the account owns, and where each
/// card's art comes from.
@MainActor
@Observable
final class DeckStore {
    private(set) var remoteDecks: [DeckOption] = []
    private(set) var serverOwnedSlugs: Set<String> = []
    private(set) var loading = false

    @ObservationIgnored private let shop: ShopAPI
    @ObservationIgnored private let config: APIConfig

    init(shop: ShopAPI, config: APIConfig) {
        self.shop = shop
        self.config = config
    }

    /// Fetches the catalog and, when signed in, what the account owns. Either
    /// can fail independently; the classic deck is always available.
    func refresh(signedIn: Bool) async {
        loading = true
        defer { loading = false }
        if let decks = try? await shop.decks() {
            remoteDecks = decks.map { DeckOption($0, config: config) }
        }
        if signedIn, let slugs = try? await shop.ownedDeckSlugs() {
            serverOwnedSlugs = Set(slugs)
        }
    }

    /// Replaces the server state directly — for tests and previews.
    func setRemote(_ decks: [DeckOption], owned: Set<String>) {
        remoteDecks = decks
        serverOwnedSlugs = owned
    }

    func isOwned(_ deck: DeckOption, locallyOwned: [String] = []) -> Bool {
        deck.isBundled || deck.isFree || deck.ownedOnServer
            || serverOwnedSlugs.contains(deck.slug) || locallyOwned.contains(deck.slug)
    }

    /// The decks to show on iOS: the classic deck first, then every free or
    /// owned deck, and paid decks only when they can be bought through the App
    /// Store. The server marks those with an Apple SKU and asks clients not to
    /// offer the rest — selling them any other way would break guideline 3.1.1.
    func shopDecks(locallyOwned: [String] = []) -> [DeckOption] {
        [.classic] + remoteDecks.filter { deck in
            isOwned(deck, locallyOwned: locallyOwned) || deck.appleProductId != nil
        }
    }

    func deck(slug: String) -> DeckOption? {
        slug == DeckOption.classicSlug ? .classic : remoteDecks.first { $0.slug == slug }
    }

    /// The deck actually in use: the selected one if it is still owned,
    /// otherwise the classic deck.
    func activeDeck(selectedSlug: String, locallyOwned: [String] = []) -> DeckOption {
        guard let deck = deck(slug: selectedSlug), isOwned(deck, locallyOwned: locallyOwned) else {
            return .classic
        }
        return deck
    }

    /// Where a card's face comes from in a given deck, falling back to the
    /// bundled classic art when the deck has no picture for that card.
    func faceSource(for card: TarotCard, deckSlug: String) -> CardArtSource {
        if let deck = deck(slug: deckSlug), !deck.isBundled, let url = deck.cardURLs[card.id] {
            return .remote(url, fallbackFile: card.imageFile)
        }
        return .bundled(card.imageFile)
    }

    func backSource(deckSlug: String) -> CardArtSource {
        if let deck = deck(slug: deckSlug), !deck.isBundled, let url = deck.backURL {
            return .remote(url, fallbackFile: CardArt.backFile)
        }
        return .bundled(CardArt.backFile)
    }

    /// The deck's cover, or the classic Fool for the bundled deck.
    func coverSource(_ deck: DeckOption) -> CardArtSource {
        if !deck.isBundled, let url = deck.coverURL {
            return .remote(url, fallbackFile: "the_fool.jpeg")
        }
        return .bundled("the_fool.jpeg")
    }
}

extension String {
    var nilIfEmpty: String? { isEmpty ? nil : self }
}

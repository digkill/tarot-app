import Foundation

/// A deck as listed by `GET /shop/decks` (`shopDeckView` on the server).
///
/// Media fields are server-relative paths; resolve them with
/// `APIConfig.mediaURL(_:)` or the helpers below. `cards` may cover fewer than
/// 78 keys — a missing card falls back to the classic art on the client.
struct ShopDeck: Decodable, Sendable, Identifiable {
    var slug: String
    var titles: [String: String]
    var descriptions: [String: String]
    /// Reuses the existing wire type from AppColors so a deck can be turned into
    /// a palette with `AppColors(payload:)`.
    var theme: ThemePayload
    var priceKop: Int
    /// `omitempty`: absent unless the deck is on sale.
    var originalPriceKop: Int?
    var productId: String
    /// `omitempty`: absent until the deck is configured in App Store Connect.
    var appleProductId: String?
    var isFree: Bool
    var owned: Bool
    var cardCount: Int
    var hasBack: Bool
    var coverUrl: String
    var backUrl: String
    var cards: [String: String]

    var id: String { slug }

    func coverURL(in config: APIConfig) -> URL? { config.mediaURL(coverUrl) }
    func backURL(in config: APIConfig) -> URL? { config.mediaURL(backUrl) }
    func cardURL(_ cardKey: String, in config: APIConfig) -> URL? { config.mediaURL(cards[cardKey]) }
}

struct ShopDecksResponse: Decodable, Sendable {
    var decks: [ShopDeck]
}

struct OwnedDecksResponse: Decodable, Sendable, Equatable {
    var slugs: [String]
}

import XCTest

@testable import MoonCompass

@MainActor
final class DeckStoreTests: XCTestCase {
    private let config = APIConfig(baseURL: URL(string: "https://tarot.example")!)

    private func store() -> DeckStore {
        DeckStore(shop: ShopAPI(client: APIClient(config: config, tokenStore: InMemoryTokenStore())), config: config)
    }

    private func deck(_ slug: String, free: Bool = false, owned: Bool = false, appleSKU: String? = nil,
                      cards: [String: URL] = [:], back: URL? = nil) -> DeckOption {
        DeckOption(slug: slug, titles: ["en": "Deck \(slug)", "ru": "Колода \(slug)"], descriptions: [:],
                   colors: AppColors(payload: ThemePayload(bg: "#112233")), priceKop: 49_900,
                   originalPriceKop: nil, appleProductId: appleSKU, isFree: free, isBundled: false,
                   ownedOnServer: owned, coverURL: nil, backURL: back, cardURLs: cards)
    }

    func testClassicDeckIsAlwaysFirstAndOwned() {
        let decks = store()
        XCTAssertEqual(decks.shopDecks().map(\.slug), ["rws"])
        XCTAssertTrue(decks.isOwned(.classic))
    }

    /// Paid decks without an App Store SKU cannot be sold on iOS without
    /// breaking guideline 3.1.1, so they are not offered at all.
    func testPaidDecksNeedAnAppleSKUToBeListed() {
        let decks = store()
        decks.setRemote([
            deck("free", free: true),
            deck("paid-no-sku"),
            deck("paid-with-sku", appleSKU: "org.mediarise.tarot.deck.paid"),
            deck("owned-no-sku", owned: true),
        ], owned: [])
        XCTAssertEqual(decks.shopDecks().map(\.slug), ["rws", "free", "paid-with-sku", "owned-no-sku"])
    }

    func testOwnershipComesFromServerOrLocalRecord() {
        let decks = store()
        let paid = deck("japanese", appleSKU: "sku")
        decks.setRemote([paid], owned: [])
        XCTAssertFalse(decks.isOwned(paid))
        XCTAssertTrue(decks.isOwned(paid, locallyOwned: ["japanese"]))
        decks.setRemote([paid], owned: ["japanese"])
        XCTAssertTrue(decks.isOwned(paid))
    }

    /// A selected deck that is no longer owned (or no longer exists) falls
    /// back to the classic deck, so the app never shows art it cannot use.
    func testActiveDeckFallsBackToClassic() {
        let decks = store()
        let paid = deck("paid", appleSKU: "sku")
        decks.setRemote([paid, deck("free", free: true)], owned: [])
        XCTAssertEqual(decks.activeDeck(selectedSlug: "paid").slug, "rws")
        XCTAssertEqual(decks.activeDeck(selectedSlug: "gone").slug, "rws")
        XCTAssertEqual(decks.activeDeck(selectedSlug: "free").slug, "free")
    }

    func testArtFallsBackToClassicForMissingCards() throws {
        let decks = store()
        let fool = try XCTUnwrap(CardCatalog.load(.en).card(id: "the_fool"))
        let moon = try XCTUnwrap(CardCatalog.load(.en).card(id: "the_moon"))
        let foolURL = URL(string: "https://tarot.example/media/decks/j/the_fool.jpg")!
        decks.setRemote([deck("j", free: true, cards: ["the_fool": foolURL])], owned: [])

        XCTAssertEqual(decks.faceSource(for: fool, deckSlug: "j"), .remote(foolURL, fallbackFile: "the_fool.jpeg"))
        XCTAssertEqual(decks.faceSource(for: moon, deckSlug: "j"), .bundled("the_moon.jpeg"), "not in the deck")
        XCTAssertEqual(decks.faceSource(for: fool, deckSlug: "rws"), .bundled("the_fool.jpeg"))
        XCTAssertEqual(decks.backSource(deckSlug: "j"), .bundled(CardArt.backFile), "no back uploaded")
    }

    func testRemoteDeckFromServerPayload() throws {
        let json = """
        {"slug":"japanese","titles":{"ru":"Японская"},"descriptions":{},"theme":{"bg":"#101010","accent":"oops"},
         "priceKop":99900,"originalPriceKop":99900,"productId":"deck_japanese","isFree":true,"owned":true,
         "cardCount":78,"hasBack":true,"coverUrl":"/media/decks/japanese/cover.jpg",
         "backUrl":"/media/decks/japanese/back.jpg","cards":{"the_fool":"/media/decks/japanese/the_fool.jpg"}}
        """
        let shopDeck = try JSONCoding.makeDecoder().decode(ShopDeck.self, from: Data(json.utf8))
        let option = DeckOption(shopDeck, config: config)
        XCTAssertNil(option.appleProductId)
        XCTAssertEqual(option.coverURL?.absoluteString, "https://tarot.example/media/decks/japanese/cover.jpg")
        XCTAssertEqual(option.cardURLs["the_fool"]?.absoluteString, "https://tarot.example/media/decks/japanese/the_fool.jpg")
        XCTAssertEqual(option.colors.accent, AppColors.classic.accent, "an invalid colour falls back")
        XCTAssertNotEqual(option.colors.bg, AppColors.classic.bg)

        let l = Localizer(language: .en)
        XCTAssertEqual(option.title(in: .en, localizer: l), "Японская", "no English title: falls back to Russian")
        XCTAssertEqual(DeckOption.classic.title(in: .en, localizer: l), l.t("deck.classicName"))
    }
}

@MainActor
final class DeepLinkTests: XCTestCase {
    func testDeckLinks() {
        let links = DeepLinks()
        XCTAssertTrue(links.handle(URL(string: "mediarisetarot://deck/japanese")!))
        XCTAssertEqual(links.deckSlug, "japanese")

        XCTAssertTrue(links.handle(URL(string: "mediarisetarot://deck")!), "no slug still opens the Decks tab")
        XCTAssertEqual(links.deckSlug, "")

        links.deckSlug = nil
        XCTAssertFalse(links.handle(URL(string: "mediarisetarot://billing/complete?tx=1")!))
        XCTAssertFalse(links.handle(URL(string: "https://tarot.sorapure.fun/deck/japanese")!))
        XCTAssertNil(links.deckSlug)
    }
}

final class CardSearchTests: XCTestCase {
    func testSearchMatchesNameAndMeaningsCaseInsensitively() throws {
        let fool = try XCTUnwrap(CardCatalog.load(.ru).card(id: "the_fool"))
        XCTAssertTrue(fool.matches(""))
        XCTAssertTrue(fool.matches("  "))
        XCTAssertTrue(fool.matches(fool.name.uppercased()))
        XCTAssertTrue(fool.matches(String(fool.reversed.prefix(6)).lowercased()))
        XCTAssertFalse(fool.matches("zzz-no-such-card"))
    }
}

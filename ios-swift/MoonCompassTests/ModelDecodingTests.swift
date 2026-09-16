import XCTest

@testable import MoonCompass

final class ModelDecodingTests: XCTestCase {
    private let decoder = JSONCoding.makeDecoder()

    private func decode<T: Decodable>(_ type: T.Type, _ json: String) throws -> T {
        try decoder.decode(type, from: Data(json.utf8))
    }

    // MARK: userView

    /// A free user: the omitempty keys are absent and the expiry is an explicit null.
    func testUserDecodesWithOmittedFieldsAndNullExpiry() throws {
        let user = try decode(User.self, Fixtures.userJSON)
        XCTAssertEqual(user.id, "u1")
        XCTAssertFalse(user.hasPremium)
        XCTAssertNil(user.premiumExpiresAt)
        XCTAssertNil(user.premiumProductId)
        XCTAssertNil(user.premiumSource)
        XCTAssertTrue(user.emailVerified)
    }

    func testUserDecodesWithAllFields() throws {
        let user = try decode(User.self, """
        {"id":"u2","email":"p@b.c","hasPremium":true,"premiumExpiresAt":"2026-03-01T10:00:00Z",
         "premiumProductId":"premium_month","premiumSource":"apple",
         "emailVerified":false,"createdAt":"2025-09-13T01:12:03.456789+03:00"}
        """)
        XCTAssertTrue(user.hasPremium)
        XCTAssertEqual(user.premiumExpiresAt, Date(timeIntervalSince1970: 1_772_359_200))
        XCTAssertEqual(user.premiumProductId, "premium_month")
        XCTAssertEqual(user.premiumSource, "apple")
        XCTAssertFalse(user.emailVerified)
    }

    // MARK: Dates

    func testDateDecodingAcceptsFractionAndOffset() throws {
        let date = try XCTUnwrap(JSONCoding.parseDate("2025-09-13T01:12:03.456789+03:00"))
        // 2025-09-12T22:12:03.456789Z
        XCTAssertEqual(date.timeIntervalSince1970, 1_757_715_123.456789, accuracy: 0.000_5)
    }

    func testDateDecodingAcceptsPlainZulu() throws {
        let date = try XCTUnwrap(JSONCoding.parseDate("2026-03-01T10:00:00Z"))
        XCTAssertEqual(date.timeIntervalSince1970, 1_772_359_200, accuracy: 0.000_5)
    }

    func testDateDecodingThroughDecoder() throws {
        struct Box: Decodable { var a: Date; var b: Date }
        let box = try decode(Box.self, #"{"a":"2025-09-13T01:12:03.456789+03:00","b":"2026-03-01T10:00:00Z"}"#)
        XCTAssertEqual(box.a.timeIntervalSince1970, 1_757_715_123.456789, accuracy: 0.000_5)
        XCTAssertEqual(box.b.timeIntervalSince1970, 1_772_359_200, accuracy: 0.000_5)
    }

    func testDateDecodingRejectsGarbage() {
        XCTAssertNil(JSONCoding.parseDate("yesterday"))
        struct Box: Decodable { var a: Date }
        XCTAssertThrowsError(try decode(Box.self, #"{"a":"13/09/2025"}"#))
    }

    // MARK: Shop

    func testShopDeckDecodesWithOmittedFieldsAndResolvesMedia() throws {
        let response = try decode(ShopDecksResponse.self, """
        {"decks":[{"slug":"classic","titles":{"en":"Classic","ru":"Классика"},"descriptions":{"en":"d"},
          "theme":{"bg":"#040307","panel":"#1a1030","accent":"#6c5ce7","text":"#f7f4ea",
                   "muted":"#9a93b3","gold":"#d4af37","danger":"#ff6b6b","tabBar":"#0c0a14"},
          "priceKop":0,"productId":"deck_classic","isFree":true,"owned":true,"cardCount":78,
          "hasBack":false,"coverUrl":"/media/decks/classic/cover.jpg","backUrl":"",
          "cards":{"the_fool":"/media/decks/classic/the_fool.jpg"}}]}
        """)
        let deck = try XCTUnwrap(response.decks.first)
        XCTAssertEqual(deck.id, "classic")
        XCTAssertNil(deck.appleProductId)
        XCTAssertNil(deck.originalPriceKop)
        XCTAssertEqual(deck.titles["ru"], "Классика")
        XCTAssertEqual(deck.theme.accent, "#6c5ce7")
        XCTAssertEqual(deck.cards.count, 1, "a deck may ship fewer than 78 cards")

        let config = Fixtures.config
        let cover = try XCTUnwrap(deck.coverURL(in: config))
        XCTAssertEqual(cover.absoluteString, "https://stub.example.com/media/decks/classic/cover.jpg")
        XCTAssertEqual(cover.host, "stub.example.com")
        XCTAssertEqual(
            deck.cardURL("the_fool", in: config)?.absoluteString,
            "https://stub.example.com/media/decks/classic/the_fool.jpg"
        )
        XCTAssertNil(deck.backURL(in: config), "an empty string means no image")
        XCTAssertNil(deck.cardURL("the_magician", in: config), "a missing card resolves to nil")
    }

    func testShopDeckDecodesOptionalFieldsWhenPresent() throws {
        let deck = try decode(ShopDeck.self, """
        {"slug":"gold","titles":{},"descriptions":{},"theme":{},"priceKop":29900,"originalPriceKop":49900,
         "productId":"deck_gold","appleProductId":"org.mediarise.tarot.deck.gold","isFree":false,
         "owned":false,"cardCount":22,"hasBack":true,"coverUrl":"","backUrl":"/media/b.jpg","cards":{}}
        """)
        XCTAssertEqual(deck.originalPriceKop, 49_900)
        XCTAssertEqual(deck.appleProductId, "org.mediarise.tarot.deck.gold")
    }

    func testMediaURLResolution() {
        let config = Fixtures.config
        XCTAssertNil(config.mediaURL(""))
        XCTAssertNil(config.mediaURL("   "))
        XCTAssertNil(config.mediaURL(nil))
        XCTAssertEqual(config.mediaURL("media/x.jpg")?.absoluteString, "https://stub.example.com/media/x.jpg")
        XCTAssertEqual(
            config.mediaURL("https://cdn.example.com/x.jpg")?.absoluteString,
            "https://cdn.example.com/x.jpg",
            "an already absolute URL is kept"
        )
        XCTAssertEqual(
            config.mediaURL("/media/my deck.jpg")?.absoluteString,
            "https://stub.example.com/media/my%20deck.jpg"
        )
    }

    // MARK: Billing

    func testAppleVerifyDecodesActiveShape() throws {
        let response = try decode(AppleVerifyResponse.self, """
        {"ok":true,"recorded":true,"hasPremium":true,"premiumSource":"apple","productId":"premium_month",
         "deckSlug":"","expiresAt":"2026-10-16T10:00:00.123+00:00","environment":"Sandbox",
         "transactionId":"tx1","appleTransactionId":"200001","originalTransactionId":"200000"}
        """)
        XCTAssertTrue(response.ok)
        XCTAssertTrue(response.hasPremium)
        XCTAssertEqual(response.recorded, true)
        XCTAssertEqual(response.environment, "Sandbox")
        XCTAssertNotNil(response.expiresAt)
        XCTAssertNil(response.reason)
    }

    func testAppleVerifyDecodesExpiredShape() throws {
        let response = try decode(AppleVerifyResponse.self, #"{"ok":true,"hasPremium":false,"reason":"expired"}"#)
        XCTAssertTrue(response.ok)
        XCTAssertFalse(response.hasPremium)
        XCTAssertEqual(response.reason, "expired")
        XCTAssertNil(response.recorded)
        XCTAssertNil(response.expiresAt)
    }

    func testAppleVerifyDecodesNullExpiry() throws {
        let response = try decode(AppleVerifyResponse.self, #"{"ok":true,"hasPremium":true,"expiresAt":null}"#)
        XCTAssertNil(response.expiresAt)
    }

    func testAppleVerifyRequestOmitsNilFields() throws {
        let data = try JSONCoding.makeEncoder().encode(AppleVerifyRequest(originalTransactionId: "200000"))
        let object = try XCTUnwrap(JSONSerialization.jsonObject(with: data) as? [String: Any])
        XCTAssertEqual(object.keys.sorted(), ["originalTransactionId"])
    }
}

import CoreGraphics
import XCTest

@testable import MoonCompass

final class CardCatalogTests: XCTestCase {
    func testEveryLanguageHas78Cards() {
        for language in Language.allCases {
            let catalog = CardCatalog.load(language)
            XCTAssertEqual(catalog.cards.count, 78, "\(language)")
            XCTAssertEqual(Set(catalog.cards.map(\.id)).count, 78, "\(language): ids must be unique")
        }
    }

    /// Ids come from the art file, so a reading saved in one language resolves
    /// in every other — the React Native client's ids did not.
    func testIdsAreTheSameInEveryLanguage() {
        let english = CardCatalog.load(.en).cards.map(\.id)
        for language in [Language.ru, .th, .zh] {
            XCTAssertEqual(CardCatalog.load(language).cards.map(\.id), english, "\(language)")
        }
        XCTAssertNotEqual(CardCatalog.load(.ru).card(id: "the_fool")?.name,
                          CardCatalog.load(.en).card(id: "the_fool")?.name,
                          "names are still translated")
    }

    func testChineseFallsBackToEnglishText() {
        XCTAssertEqual(CardCatalog.load(.zh).card(id: "the_sun")?.upright,
                       CardCatalog.load(.en).card(id: "the_sun")?.upright)
    }

    /// Arcana and suit come from position, so they are right in every
    /// language. In the React Native client every Russian card was "minor".
    func testClassificationInEveryLanguage() {
        for language in [Language.en, .ru, .th] {
            let catalog = CardCatalog.load(language)
            XCTAssertEqual(catalog.cards.filter { $0.arcana == .major }.count, 22, "\(language)")
            for suit in TarotSuit.allCases {
                XCTAssertEqual(catalog.cards.filter { $0.suit == suit }.count, 14, "\(language) \(suit)")
            }
        }
        let catalog = CardCatalog.load(.ru)
        XCTAssertEqual(catalog.card(id: "the_fool")?.number, 0)
        XCTAssertEqual(catalog.card(id: "the_world")?.number, 21)
        XCTAssertEqual(catalog.card(id: "ace_of_cups")?.suit, .cups)
        XCTAssertEqual(catalog.card(id: "ace_of_cups")?.number, 1)
        XCTAssertEqual(catalog.card(id: "king_of_pentacles")?.suit, .pentacles)
        XCTAssertEqual(catalog.card(id: "king_of_pentacles")?.number, 14)
    }

    /// Every card's art is actually in the bundle, plus the back.
    func testEveryCardHasArt() {
        for card in CardCatalog.load(.en).cards {
            XCTAssertNotNil(CardArt.shared.image(file: card.imageFile, maxPixels: 64), card.imageFile)
        }
        XCTAssertNotNil(CardArt.shared.image(file: CardArt.backFile, maxPixels: 64))
    }
}

final class SpreadTests: XCTestCase {
    func testSpreadsMatchTheReactNativeDefinitions() {
        XCTAssertEqual(Spread.all.count, 7)
        XCTAssertEqual(Spread.all.reduce(0) { $0 + $1.cardCount }, 47)
        let counts = Dictionary(uniqueKeysWithValues: Spread.all.map { ($0.id, $0.cardCount) })
        XCTAssertEqual(counts, ["one-card": 1, "three-card": 3, "celtic-cross": 10, "love-relationship": 7,
                                "horseshoe": 7, "weekly": 7, "year-wheel": 12])
        XCTAssertEqual(Spread.all.filter(\.premium).map(\.id),
                       ["love-relationship", "horseshoe", "weekly", "year-wheel"])
        XCTAssertEqual(Spread.basicIds.compactMap(Spread.find).count, 3)
    }

    func testPositionsAreOnTheTableAndTranslated() {
        let l = Localizer(language: .en)
        for spread in Spread.all {
            XCTAssertEqual(spread.positions.map(\.index), Array(1...spread.cardCount), spread.id)
            XCTAssertNotEqual(l.t(spread.nameKey), spread.nameKey, spread.id)
            for position in spread.positions {
                XCTAssert((0...1).contains(position.x) && (0...1).contains(position.y), "\(spread.id) #\(position.index)")
                XCTAssertNotEqual(l.t(position.titleKey), position.titleKey, "\(spread.id) #\(position.index)")
            }
        }
    }
}

final class ReadingLayoutTests: XCTestCase {
    private let phone = CGSize(width: 358, height: 400)

    func testSingleCardUsesTheBaseWidth() {
        XCTAssertEqual(ReadingLayout.cardWidth(positions: Spread.find("one-card")!.positions, canvas: phone), 110)
    }

    func testWidthStaysWithinBounds() {
        for spread in Spread.all {
            let canvas = CGSize(width: 358, height: ReadingLayout.canvasHeight(cardCount: spread.cardCount))
            let width = ReadingLayout.cardWidth(positions: spread.positions, canvas: canvas)
            XCTAssert((ReadingLayout.minimumCardWidth...ReadingLayout.baseCardWidth).contains(width), "\(spread.id): \(width)")
        }
    }

    /// Cards on the same row must not overlap.
    func testThreeCardsDoNotOverlap() {
        let positions = Spread.find("three-card")!.positions
        let width = ReadingLayout.cardWidth(positions: positions, canvas: phone)
        let xs = positions.map { ReadingLayout.origin(for: $0, cardWidth: width, canvas: phone).x }.sorted()
        for (left, right) in zip(xs, xs.dropFirst()) {
            XCTAssertGreaterThanOrEqual(right - left, width, "cards overlap")
        }
    }

    func testCardsStayInsideTheTable() {
        for spread in Spread.all {
            let canvas = CGSize(width: 358, height: ReadingLayout.canvasHeight(cardCount: spread.cardCount))
            let width = ReadingLayout.cardWidth(positions: spread.positions, canvas: canvas)
            for position in spread.positions {
                let origin = ReadingLayout.origin(for: position, cardWidth: width, canvas: canvas)
                XCTAssertGreaterThanOrEqual(origin.x, ReadingLayout.padding - 0.5, spread.id)
                XCTAssertLessThanOrEqual(origin.x + width, canvas.width - ReadingLayout.padding + 0.5, spread.id)
                XCTAssertGreaterThanOrEqual(origin.y, ReadingLayout.padding - 0.5, spread.id)
            }
        }
    }

    func testCanvasHeights() {
        XCTAssertEqual(ReadingLayout.canvasHeight(cardCount: 1), 320)
        XCTAssertEqual(ReadingLayout.canvasHeight(cardCount: 7), 400)
        XCTAssertEqual(ReadingLayout.canvasHeight(cardCount: 12), 440)
    }
}

final class InterpretationTests: XCTestCase {
    func testSummaryUsesTheFirstThreePositions() throws {
        let catalog = CardCatalog.load(.en)
        let spread = try XCTUnwrap(Spread.find("celtic-cross"))
        let entries = spread.positions.enumerated().map { index, position in
            DrawnCard(card: catalog.cards[index], position: position, isReversed: index == 1)
        }
        let l = Localizer(language: .en)
        let summary = Interpretation.summary(spread: spread, entries: entries, localizer: l)

        XCTAssertTrue(summary.contains(l.t(spread.nameKey)), "names the spread")
        XCTAssertTrue(summary.contains("10"), "counts the positions")
        XCTAssertTrue(summary.contains(entries[0].card.upright))
        XCTAssertTrue(summary.contains(entries[1].card.reversed), "a reversed card uses its reversed meaning")
        XCTAssertTrue(summary.contains(entries[2].card.upright))
        XCTAssertFalse(summary.contains(entries[3].card.upright), "only the first three are quoted")
        XCTAssertFalse(summary.contains("{{"), "no placeholder left unfilled")
    }

    func testSavedReadingResolvesInAnotherLanguage() throws {
        let spread = try XCTUnwrap(Spread.find("three-card"))
        let reading = Reading(
            spreadId: spread.id, deckId: "rws",
            items: [ReadingItem(positionIndex: 1, cardId: "the_fool", isReversed: false),
                    ReadingItem(positionIndex: 2, cardId: "the_moon", isReversed: true),
                    ReadingItem(positionIndex: 9, cardId: "the_sun", isReversed: false),
                    ReadingItem(positionIndex: 3, cardId: "no_such_card", isReversed: false)],
            summaryText: "saved in English"
        )
        let entries = Interpretation.entries(for: reading, spread: spread, catalog: CardCatalog.load(.ru))
        XCTAssertEqual(entries.map(\.card.id), ["the_fool", "the_moon"], "unknown positions and cards are skipped")
        XCTAssertTrue(entries[1].isReversed)
    }

    func testRequestCarriesTranslatedPositionsAndBothMeanings() throws {
        let catalog = CardCatalog.load(.ru)
        let spread = try XCTUnwrap(Spread.find("three-card"))
        let entries = [DrawnCard(card: try XCTUnwrap(catalog.card(id: "the_star")), position: spread.positions[0], isReversed: true)]
        let l = Localizer(language: .ru)
        let request = InterpretationRequest(spread: spread, entries: entries, localizer: l)
        XCTAssertEqual(request.language, "ru")
        XCTAssertEqual(request.spreadName, l.t(spread.nameKey))
        XCTAssertEqual(request.cards.first?.positionTitle, l.t("position.past"))
        XCTAssertEqual(request.cards.first?.isReversed, true)
        XCTAssertFalse(request.cards.first?.uprightMeaning.isEmpty ?? true)
        XCTAssertFalse(request.cards.first?.reversedMeaning.isEmpty ?? true)
    }
}

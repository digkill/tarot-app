import XCTest

@testable import MoonCompass

@MainActor
final class HistoryStoreTests: XCTestCase {
    private var fileURL: URL!

    override func setUp() async throws {
        fileURL = FileManager.default.temporaryDirectory
            .appendingPathComponent("HistoryStoreTests-\(UUID().uuidString)")
            .appendingPathComponent("readings.json")
    }

    override func tearDown() async throws {
        try? FileManager.default.removeItem(at: fileURL.deletingLastPathComponent())
    }

    private func reading(_ spreadId: String = "three-card", at date: Date = Date(), kind: Reading.Kind? = nil) -> Reading {
        Reading(spreadId: spreadId, deckId: "rws", drawnAt: date,
                items: [ReadingItem(positionIndex: 1, cardId: "the_fool", isReversed: false)],
                summaryText: "s", kind: kind)
    }

    func testNewestFirstAndPersisted() {
        let store = HistoryStore(fileURL: fileURL)
        let first = store.add(reading())
        let second = store.add(reading())
        XCTAssertEqual(store.readings.map(\.id), [second.id, first.id])

        let reopened = HistoryStore(fileURL: fileURL)
        XCTAssertEqual(reopened.readings.map(\.id), [second.id, first.id])
        XCTAssertEqual(reopened.readings.first?.drawnAt.timeIntervalSince1970 ?? 0,
                       second.drawnAt.timeIntervalSince1970, accuracy: 0.001)
    }

    func testEditsPersist() {
        let store = HistoryStore(fileURL: fileURL)
        let saved = store.add(reading())
        store.toggleFavorite(id: saved.id)
        store.update(id: saved.id) { $0.notes = "заметка" }

        let reopened = HistoryStore(fileURL: fileURL)
        XCTAssertEqual(reopened.reading(id: saved.id)?.favorite, true)
        XCTAssertEqual(reopened.reading(id: saved.id)?.notes, "заметка")

        reopened.remove(id: saved.id)
        XCTAssertTrue(HistoryStore(fileURL: fileURL).readings.isEmpty)
    }

    func testTodaysDailyCardIgnoresBonusDrawsAndYesterday() {
        let store = HistoryStore(fileURL: fileURL)
        let now = Date()
        let yesterday = Calendar.current.date(byAdding: .day, value: -1, to: now)!
        store.add(reading(Spread.oneCardId, at: yesterday, kind: .daily))
        XCTAssertNil(store.todaysDailyCard(now: now), "yesterday's card is not today's")

        store.add(reading(Spread.oneCardId, at: now, kind: .bonus))
        XCTAssertNil(store.todaysDailyCard(now: now), "an extra draw is not the daily card")

        store.add(reading("three-card", at: now))
        XCTAssertNil(store.todaysDailyCard(now: now), "another spread is not the daily card")

        let daily = store.add(reading(Spread.oneCardId, at: now, kind: .daily))
        XCTAssertEqual(store.todaysDailyCard(now: now)?.id, daily.id)
    }

    func testCorruptFileStartsEmpty() throws {
        try FileManager.default.createDirectory(at: fileURL.deletingLastPathComponent(), withIntermediateDirectories: true)
        try Data("garbage".utf8).write(to: fileURL)
        XCTAssertTrue(HistoryStore(fileURL: fileURL).readings.isEmpty)
    }
}

final class DailyCardGateTests: XCTestCase {
    private var suiteName: String!
    private var defaults: UserDefaults!
    private var client: APIClient!

    override func setUp() async throws {
        StubURLProtocol.reset()
        suiteName = "DailyCardGateTests.\(UUID().uuidString)"
        defaults = UserDefaults(suiteName: suiteName)
        client = APIClient(config: Fixtures.config, session: StubURLProtocol.makeSession(),
                           tokenStore: InMemoryTokenStore(Fixtures.oldTokens))
    }

    override func tearDown() async throws {
        StubURLProtocol.reset()
        defaults.removePersistentDomain(forName: suiteName)
    }

    private var usageJSON: String {
        """
        {"date":"2026-09-16","timezone":"Europe/Moscow","hasPremium":false,
         "dailyCards":{"used":1,"limit":3,"remaining":2},
         "adUnlocks":{"used":0,"limit":5,"remaining":5},
         "interpretations":{"used":0,"limit":0,"remaining":0}}
        """
    }

    private func gate() -> DailyCardGate {
        DailyCardGate(usage: UsageAPI(client: client), ledger: LocalDailyLedger(defaults: defaults))
    }

    func testServerDecidesTheKind() async throws {
        StubURLProtocol.install { [usageJSON] _ in .json(200, usageJSON) }
        let first = try await gate().consume(hasTodaysCard: false, hasPremium: false)
        let extra = try await gate().consume(hasTodaysCard: true, hasPremium: false)
        XCTAssertEqual(first, .daily)
        XCTAssertEqual(extra, .bonus)
    }

    /// A real limit from the server is final; the offline ledger never overrides it.
    func testServerLimitIsNotOverridden() async {
        StubURLProtocol.install { _ in .json(429, Fixtures.errorJSON("quota_exceeded", "daily limit reached")) }
        do {
            _ = try await gate().consume(hasTodaysCard: true, hasPremium: false)
            XCTFail("expected quota_exceeded")
        } catch let error as APIError {
            XCTAssertEqual(error.code, "quota_exceeded")
        } catch {
            XCTFail("unexpected \(error)")
        }
    }

    /// Offline, the local ledger allows the same three free draws a day.
    func testOfflineFallsBackToTheLocalLimit() async throws {
        StubURLProtocol.install { _ in throw URLError(.notConnectedToInternet) }
        for _ in 0..<LocalDailyLedger.freeLimit {
            _ = try await gate().consume(hasTodaysCard: false, hasPremium: false)
        }
        do {
            _ = try await gate().consume(hasTodaysCard: true, hasPremium: false)
            XCTFail("expected the local limit")
        } catch let error as APIError {
            XCTAssertEqual(error.code, "quota_exceeded")
        }
    }

    func testServerErrorsAlsoFallBack() async throws {
        StubURLProtocol.install { _ in .json(503, Fixtures.errorJSON("unavailable", "down")) }
        let kind = try await gate().consume(hasTodaysCard: false, hasPremium: false)
        XCTAssertEqual(kind, .daily)
    }

    func testLedgerResetsTheNextDay() {
        var ledger = LocalDailyLedger(defaults: defaults)
        let today = Date()
        ledger.now = { today }
        XCTAssertTrue(ledger.consume(limit: 1))
        XCTAssertFalse(ledger.consume(limit: 1))

        let tomorrow = Calendar.current.date(byAdding: .day, value: 1, to: today)!
        ledger.now = { tomorrow }
        XCTAssertTrue(ledger.consume(limit: 1), "a new day starts a new count")
    }
}

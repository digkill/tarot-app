import XCTest

@testable import MoonCompass

private func fixture(_ name: String) throws -> Data {
    let url = try XCTUnwrap(Bundle(for: ArcanaDecodingTests.self).url(forResource: name, withExtension: "json"))
    return try Data(contentsOf: url)
}

/// The fixtures are real output of the Go server (see
/// arcana/internal/fixturegen), so these tests catch protocol drift.
final class ArcanaDecodingTests: XCTestCase {
    func testDecodesAServerStateFrame() throws {
        guard case let .state(matchId, payload) = try ArcanaServerEvent.decode(fixture("arcana_state")) else {
            return XCTFail("not a state frame")
        }
        XCTAssertEqual(matchId, "m-1")
        let v = payload.state
        XCTAssertEqual(v.phase, .playing)
        XCTAssertTrue(v.isMyTurn)
        XCTAssertEqual(v.you.hero.id, "the_moon")
        XCTAssertFalse(v.you.hand.isEmpty)
        XCTAssertGreaterThan(v.opponent.handCount, 0)
        XCTAssertEqual(payload.deadlineAt - payload.serverTime, 30_000)
        XCTAssertFalse(v.log.isEmpty)
    }

    func testDecodesTheCatalogAndDescribesEveryEffect() throws {
        let catalog = try ArcanaCoding.decoder.decode(ArcanaCatalog.self, from: fixture("arcana_catalog"))
        XCTAssertEqual(catalog.heroes.count, 4)
        XCTAssertGreaterThanOrEqual(catalog.cards.count, 20)

        let tarot = CardCatalog.load(.en)
        for language in [Language.en, .ru] {
            let l = Localizer(language: language)
            for card in catalog.cards {
                XCTAssertNotNil(tarot.card(id: card.id), "no art or name for \(card.id)")
                for side in [card.upright, card.reversed] {
                    let lines = ArcanaText.describe(side, l)
                    XCTAssertEqual(lines.count, side.ops.count, "\(card.id): an op has no description")
                    for line in lines {
                        XCTAssertFalse(line.contains("arcana."), "\(language) \(card.id): missing key in \(line)")
                        XCTAssertFalse(line.contains("{{"), "\(language) \(card.id): unfilled placeholder in \(line)")
                    }
                }
            }
            for hero in catalog.heroes {
                XCTAssertNotNil(tarot.card(id: hero.cardId), "no portrait for \(hero.id)")
                for part in ["passive", "ability", "ultimate"] {
                    XCTAssertTrue(l.has("arcana.heroText.\(hero.id).\(part)"), "\(language) \(hero.id).\(part)")
                }
            }
            for status in ArcanaStatusChip.order {
                XCTAssertTrue(l.has("arcana.status.\(status)") && l.has("arcana.statusHint.\(status)"), status)
            }
            for emote in catalog.emotes {
                XCTAssertTrue(l.has("arcana.emotes.\(emote)"), emote)
            }
        }
    }

    func testClientMessagesAreSnakeCaseIntents() throws {
        let message = ArcanaClientMessage(type: "card.play", matchId: "m", cardUid: "p1c3",
                                          target: ArcanaTarget(kind: .discardCard, cardUid: "p1c1"))
        let json = try XCTUnwrap(JSONSerialization.jsonObject(with: ArcanaCoding.encoder.encode(message)) as? [String: Any])
        XCTAssertEqual(json["match_id"] as? String, "m")
        XCTAssertEqual(json["card_uid"] as? String, "p1c3")
        XCTAssertEqual((json["target"] as? [String: Any])?["kind"] as? String, "discard_card")
        XCTAssertEqual(Set(json.keys), ["type", "match_id", "card_uid", "target"], "nil fields are omitted")
    }

    func testJWTExpiryIsReadWithoutVerifying() {
        // {"sub":"u","exp":1800000000}
        let token = "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJ1IiwiZXhwIjoxODAwMDAwMDAwfQ.sig"
        XCTAssertEqual(APIClient.expiry(ofJWT: token), Date(timeIntervalSince1970: 1_800_000_000))
        XCTAssertNil(APIClient.expiry(ofJWT: "garbage"))
    }
}

@MainActor
final class ArcanaStoreTests: XCTestCase {
    private func store() -> ArcanaStore {
        let config = APIConfig(baseURL: URL(string: "https://tarot.example")!)
        let api = ArcanaAPI(config: ArcanaConfig(baseURL: config.baseURL),
                            client: APIClient(config: config, tokenStore: InMemoryTokenStore()))
        let defaults = UserDefaults(suiteName: "ArcanaStoreTests.\(UUID().uuidString)")!
        return ArcanaStore(api: api, defaults: defaults)
    }

    private func stateEvent(seq: UInt64) throws -> ArcanaServerEvent {
        var json = try XCTUnwrap(JSONSerialization.jsonObject(with: fixture("arcana_state")) as? [String: Any])
        var payload = json["payload"] as! [String: Any]
        var state = payload["state"] as! [String: Any]
        state["seq"] = seq
        payload["state"] = state
        json["payload"] = payload
        return try ArcanaServerEvent.decode(JSONSerialization.data(withJSONObject: json))
    }

    func testStaleStatesAreIgnored() throws {
        let s = store()
        s.handle(.matchStarted(matchId: "m-1", ArcanaStartedPayload(
            you: .init(id: "me", hero: "the_moon"), opponent: .init(id: "them", hero: "death"))))
        XCTAssertEqual(s.stage, .match)
        s.handle(try stateEvent(seq: 7))
        s.handle(try stateEvent(seq: 5))
        XCTAssertEqual(s.view?.seq, 7, "an older state must not overwrite a newer one")
        s.handle(try stateEvent(seq: 8))
        XCTAssertEqual(s.view?.seq, 8)
        XCTAssertNotNil(s.deadline)
    }

    func testStatesForAnotherMatchAreIgnored() throws {
        let s = store()
        s.handle(.matchStarted(matchId: "other", ArcanaStartedPayload(
            you: .init(id: "me", hero: "the_moon"), opponent: .init(id: "them", hero: "death"))))
        s.handle(try stateEvent(seq: 3))
        XCTAssertNil(s.view)
    }

    func testFinishAndOpponentConnection() throws {
        let s = store()
        s.handle(.matchStarted(matchId: "m-1", ArcanaStartedPayload(
            you: .init(id: "me", hero: "the_moon"), opponent: .init(id: "them", hero: "death"))))
        s.handle(.opponentDisconnected)
        XCTAssertFalse(s.opponentConnected)
        s.handle(.opponentReconnected)
        XCTAssertTrue(s.opponentConnected)

        guard case let .state(_, payload) = try stateEvent(seq: 9) else { return XCTFail() }
        s.handle(.finished(matchId: "m-1", ArcanaFinishedPayload(winner: "me", reason: "surrender", result: "win", state: payload.state)))
        XCTAssertEqual(s.stage, .finished)
        XCTAssertEqual(s.finished?.result, "win")
        s.closeResult()
        XCTAssertEqual(s.stage, .lobby)
        XCTAssertNil(s.view)
    }

    /// The app ships the catalog so the lobby works without the server.
    func testBundledCatalogLoads() throws {
        let bundled = try XCTUnwrap(ArcanaStore.bundledCatalog())
        XCTAssertEqual(bundled.heroes.count, 4)
        XCTAssertEqual(bundled.protocolVersion, ArcanaCoding.protocolVersion)
    }

    func testErrorsAreCountedForDisplay() {
        let s = store()
        s.handle(.error(ArcanaErrorPayload(code: "not_enough_mana", message: "", ref: nil)))
        s.handle(.error(ArcanaErrorPayload(code: "not_enough_mana", message: "", ref: nil)))
        XCTAssertEqual(s.lastError, "not_enough_mana")
        XCTAssertEqual(s.errorCount, 2, "the same error twice must show twice")
        XCTAssertTrue(Localizer(language: .ru).has("arcana.errors.not_enough_mana"))
    }
}

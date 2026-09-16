import StoreKit
import StoreKitTest
import XCTest

@testable import MoonCompass

private final class RecordingVerifier: AppleTransactionVerifier, @unchecked Sendable {
    private let lock = NSLock()
    private var _calls: [(jws: String, token: UUID?)] = []
    var result: Result<AppleVerifyResponse, APIError>

    init(_ result: Result<AppleVerifyResponse, APIError>) {
        self.result = result
    }

    var calls: [(jws: String, token: UUID?)] { lock.withLock { _calls } }

    func verify(signedTransaction: String, appAccountToken: UUID?) async throws -> AppleVerifyResponse {
        lock.withLock { _calls.append((signedTransaction, appAccountToken)) }
        return try result.get()
    }
}

final class PurchaseDecisionTests: XCTestCase {
    private func api(_ code: String, _ status: Int) -> APIError {
        APIError(code: code, message: code, status: status)
    }

    /// Only a final answer from the server lets a paid transaction be
    /// finished; anything that may succeed on retry keeps it with StoreKit.
    func testWhichServerErrorsAreFinal() {
        XCTAssertEqual(PurchaseStore.failure(for: api("apple_account_mismatch", 403)), .otherAccount)
        XCTAssertEqual(PurchaseStore.failure(for: api("apple_family_shared", 403)), .otherAccount)
        XCTAssertEqual(PurchaseStore.failure(for: api("invalid_signature", 422)), .rejected)
        XCTAssertEqual(PurchaseStore.failure(for: api("validation_error", 422)), .rejected)

        XCTAssertEqual(PurchaseStore.failure(for: api("network", 0)), .verifyLater)
        XCTAssertEqual(PurchaseStore.failure(for: APIError.notAuthenticated), .verifyLater)
        XCTAssertEqual(PurchaseStore.failure(for: api("rate_limited", 429)), .verifyLater)
        XCTAssertEqual(PurchaseStore.failure(for: api("payment_provider_error", 502)), .verifyLater)
        XCTAssertEqual(PurchaseStore.failure(for: api("payments_unavailable", 503)), .verifyLater)
        XCTAssertEqual(PurchaseStore.failure(for: URLError(.timedOut)), .verifyLater)
    }

    func testEveryFailureHasATranslatedMessage() {
        let outcomes: [PurchaseOutcome] = [.pending, .failed(.productUnavailable), .failed(.verifyLater),
                                           .failed(.otherAccount), .failed(.rejected), .failed(.store)]
        for language in Language.allCases {
            let l = Localizer(language: language)
            for outcome in outcomes {
                let key = try? XCTUnwrap(PurchaseStore.messageKey(for: outcome))
                XCTAssertNotNil(key)
                XCTAssertNotEqual(l.t(key ?? ""), key, "\(language) \(outcome)")
            }
        }
        XCTAssertNil(PurchaseStore.messageKey(for: .cancelled))
    }

    /// The iOS-only strings are layered over the shared files without hiding them.
    func testIOSStringsOverlayTheSharedOnes() {
        for language in Language.allCases {
            let l = Localizer(language: language)
            for key in ["premiumIos.disclosure", "premiumIos.dailyCards", "premium.source.apple",
                        "premium.source.rustore", "premium.title", "premium.plans.yearly.name"] {
                XCTAssertTrue(l.has(key), "\(language): \(key)")
            }
        }
        XCTAssertFalse(Localizer(language: .ru).t("premiumIos.dailyCards").contains("ролик"))
    }

    /// The StoreKit configuration sells exactly the SKUs the app asks for.
    func testStoreKitConfigurationMatchesProductIDs() throws {
        let url = try XCTUnwrap(Bundle(for: Self.self).url(forResource: "MoonCompass", withExtension: "storekit"))
        let json = try XCTUnwrap(JSONSerialization.jsonObject(with: Data(contentsOf: url)) as? [String: Any])
        let products = (json["products"] as? [[String: Any]] ?? []).compactMap { $0["productID"] as? String }
        let subscriptions = (json["subscriptionGroups"] as? [[String: Any]] ?? [])
            .flatMap { $0["subscriptions"] as? [[String: Any]] ?? [] }
            .compactMap { $0["productID"] as? String }
        XCTAssertEqual(Set(products + subscriptions), Set(ProductIDs.premium))
    }
}

/// End-to-end against a local App Store: StoreKit signs the transactions,
/// only the server is faked.
@MainActor
final class PurchaseStoreStoreKitTests: XCTestCase {
    private var session: SKTestSession!
    private let userId = "8f14e45f-ceea-467a-9575-7d5a6b1f0b2c"

    override func setUp() async throws {
        session = try SKTestSession(configurationFileNamed: "MoonCompass")
        session.resetToDefaultState()
        session.disableDialogs = true
        session.clearTransactions()
    }

    override func tearDown() async throws {
        session.clearTransactions()
    }

    /// Records which transactions were finished.
    private final class FinishLog: @unchecked Sendable {
        private let lock = NSLock()
        private var ids: [UInt64] = []
        func add(_ id: UInt64) { lock.withLock { ids.append(id) } }
        var all: [UInt64] { lock.withLock { ids } }
    }

    private func makeStore(_ verifier: RecordingVerifier, _ log: FinishLog) -> PurchaseStore {
        PurchaseStore(verifier: verifier, finish: { transaction in log.add(transaction.id) })
    }

    private func granted() -> AppleVerifyResponse {
        AppleVerifyResponse(ok: true, hasPremium: true, productId: "premium_monthly")
    }

    func testPurchaseIsSentToTheServerThenFinished() async throws {
        let verifier = RecordingVerifier(.success(granted()))
        let finished = FinishLog()
        let store = makeStore(verifier, finished)
        store.userId = userId
        await store.loadProducts(ProductIDs.premium)
        XCTAssertEqual(Set(store.products.keys), Set(ProductIDs.premium))

        let outcome = await store.purchase(ProductIDs.premiumMonthly)
        XCTAssertEqual(outcome, .purchased(granted()))
        XCTAssertEqual(store.entitlementsVersion, 1)
        XCTAssertEqual(verifier.calls.count, 1)
        XCTAssertEqual(verifier.calls.first?.token, UUID(uuidString: userId), "purchase tied to the account")
        XCTAssertEqual(verifier.calls.first?.jws.split(separator: ".").count, 3, "a compact JWS, not decoded JSON")

        XCTAssertEqual(finished.all.count, 1, "finished once the server confirmed it")
    }

    /// Apple has charged but the server is unreachable: the transaction stays
    /// with StoreKit and the next sync delivers it.
    func testUnconfirmedPurchaseIsKeptAndRetried() async throws {
        let verifier = RecordingVerifier(.failure(APIError(code: "network", message: "offline", status: 0)))
        let finished = FinishLog()
        let store = makeStore(verifier, finished)
        store.userId = userId
        await store.loadProducts(ProductIDs.premium)

        let outcome = await store.purchase(ProductIDs.premiumLifetime)
        XCTAssertEqual(outcome, .failed(.verifyLater))
        XCTAssertEqual(store.entitlementsVersion, 0)
        XCTAssertTrue(finished.all.isEmpty, "not finished while the server has not confirmed it")

        verifier.result = .success(granted())
        let synced = await store.syncEntitlements()
        XCTAssertEqual(synced.count, 1, "delivered once, not once per list it appears in")
        XCTAssertEqual(store.entitlementsVersion, 1)

        XCTAssertEqual(finished.all.count, 1, "finished once the retry was confirmed")
    }

    func testSignedOutSyncDoesNothing() async {
        let verifier = RecordingVerifier(.success(granted()))
        let store = PurchaseStore(verifier: verifier)
        let synced = await store.syncEntitlements()
        XCTAssertTrue(synced.isEmpty)
        XCTAssertTrue(verifier.calls.isEmpty)
    }
}

import XCTest

@testable import MoonCompass

@MainActor
final class SessionStoreTests: XCTestCase {
    private var tokens: InMemoryTokenStore!
    private var client: APIClient!
    private var suiteName: String!
    private var defaults: UserDefaults!

    override func setUp() async throws {
        StubURLProtocol.reset()
        tokens = InMemoryTokenStore()
        client = APIClient(config: Fixtures.config, session: StubURLProtocol.makeSession(), tokenStore: tokens)
        suiteName = "SessionStoreTests.\(UUID().uuidString)"
        defaults = UserDefaults(suiteName: suiteName)
    }

    override func tearDown() async throws {
        StubURLProtocol.reset()
        defaults.removePersistentDomain(forName: suiteName)
    }

    private func makeStore() -> SessionStore {
        SessionStore(client: client, defaults: defaults)
    }

    /// Waits for an asynchronous state change driven by the session event stream.
    private func waitFor(_ store: SessionStore, timeout: TimeInterval = 2, _ predicate: (SessionStore.State) -> Bool) async {
        let deadline = Date().addingTimeInterval(timeout)
        while !predicate(store.state), Date() < deadline {
            try? await Task.sleep(nanoseconds: 20_000_000)
        }
    }

    // MARK: Launch

    func testNoStoredSessionIsSignedOutWithoutAnyRequest() async {
        let store = makeStore()
        await store.start()
        XCTAssertEqual(store.state, .signedOut)
        XCTAssertTrue(StubURLProtocol.requests.isEmpty)
    }

    func testStoredSessionIsRestored() async throws {
        try tokens.save(Fixtures.oldTokens)
        StubURLProtocol.install { _ in .json(200, Fixtures.userJSON) }
        let store = makeStore()
        await store.start()
        XCTAssertEqual(store.user?.email, "a@b.c")
    }

    func testRejectedSessionSignsOutAndClearsTokens() async throws {
        try tokens.save(Fixtures.oldTokens)
        StubURLProtocol.install { request in
            request.path == APIClient.refreshPath
                ? .json(401, Fixtures.errorJSON("invalid_token", "refresh token is invalid"))
                : .json(401, Fixtures.errorJSON("unauthorized", "invalid or expired token"))
        }
        let store = makeStore()
        await store.start()
        XCTAssertEqual(store.state, .signedOut)
        XCTAssertNil(try tokens.load())
    }

    /// Opening the app offline must not throw a signed-in user out.
    func testOfflineLaunchKeepsTheLastKnownAccount() async throws {
        // Sign in once so the account is cached.
        StubURLProtocol.install { _ in .json(200, Fixtures.authJSON(Fixtures.oldTokens)) }
        try await makeStore().login(email: "a@b.c", password: "password1")

        StubURLProtocol.install { _ in throw URLError(.notConnectedToInternet) }
        let store = makeStore()
        await store.start()
        XCTAssertEqual(store.user?.id, "u1")
        XCTAssertNotNil(try tokens.load(), "tokens survive an offline launch")
    }

    // MARK: Account actions

    func testLoginSignsInAndStoresTokens() async throws {
        StubURLProtocol.install { _ in .json(200, Fixtures.authJSON(Fixtures.newTokens)) }
        let store = makeStore()
        await store.start()
        try await store.login(email: "a@b.c", password: "password1")
        XCTAssertEqual(store.user?.id, "u1")
        XCTAssertEqual(try tokens.load(), Fixtures.newTokens)
    }

    /// The login screen routes an unverified account to the code screen, so
    /// the error code must come through intact and the state must not change.
    func testUnverifiedLoginSurfacesTheCode() async {
        StubURLProtocol.install { _ in .json(403, Fixtures.errorJSON("email_unverified", "email is not verified")) }
        let store = makeStore()
        await store.start()
        do {
            try await store.login(email: "a@b.c", password: "password1")
            XCTFail("expected email_unverified")
        } catch let error as APIError {
            XCTAssertEqual(error.code, "email_unverified")
        } catch {
            XCTFail("unexpected \(error)")
        }
        XCTAssertEqual(store.state, .signedOut)
    }

    func testRegisterSendsConsentAndDoesNotSignIn() async throws {
        StubURLProtocol.install { _ in .json(201, #"{"email":"a@b.c","verificationRequired":true}"#) }
        let store = makeStore()
        await store.start()
        try await store.register(email: "a@b.c", password: "password1", language: .ru,
                                 acceptedTerms: true, acceptedPrivacy: true)
        XCTAssertEqual(store.state, .signedOut)

        let request = try XCTUnwrap(StubURLProtocol.requests(to: "/api/v1/auth/register").first)
        let body = try XCTUnwrap(request.body)
        let json = try XCTUnwrap(JSONSerialization.jsonObject(with: body) as? [String: Any])
        XCTAssertEqual(json["consentVersion"] as? String, "1.0", "the server refuses any other version")
        XCTAssertEqual(json["acceptedPersonalData"] as? Bool, true)
        XCTAssertEqual(json["language"] as? String, "ru")
    }

    func testVerifyEmailSignsIn() async throws {
        StubURLProtocol.install { _ in .json(200, Fixtures.authJSON(Fixtures.newTokens)) }
        let store = makeStore()
        await store.start()
        try await store.verifyEmail(email: "a@b.c", code: "123456")
        XCTAssertEqual(store.user?.id, "u1")
    }

    func testLogoutSignsOutAndForgetsTheAccount() async throws {
        StubURLProtocol.install { request in
            request.path == "/api/v1/auth/logout" ? .json(200, #"{"ok":true}"#) : .json(200, Fixtures.authJSON(Fixtures.newTokens))
        }
        let store = makeStore()
        await store.start()
        try await store.login(email: "a@b.c", password: "password1")
        await store.logout()
        XCTAssertEqual(store.state, .signedOut)
        XCTAssertNil(try tokens.load())
        XCTAssertNil(defaults.data(forKey: SessionStore.cachedUserKey))
    }

    /// When a later request finds the session revoked, the app must drop back
    /// to the sign-in screen by itself.
    func testRevokedSessionDuringUseSignsOut() async throws {
        StubURLProtocol.install { _ in .json(200, Fixtures.authJSON(Fixtures.oldTokens)) }
        let store = makeStore()
        await store.start()
        try await store.login(email: "a@b.c", password: "password1")
        XCTAssertNotNil(store.user)

        StubURLProtocol.install { request in
            request.path == APIClient.refreshPath
                ? .json(401, Fixtures.errorJSON("invalid_token", "refresh token is invalid"))
                : .json(401, Fixtures.errorJSON("unauthorized", "invalid or expired token"))
        }
        await store.refreshUser()
        await waitFor(store) { $0 == .signedOut }
        XCTAssertEqual(store.state, .signedOut)
    }

    func testFailedAccountDeletionKeepsTheUserSignedIn() async throws {
        StubURLProtocol.install { request in
            request.request.httpMethod == "DELETE"
                ? .json(500, Fixtures.errorJSON("internal_error", "boom"))
                : .json(200, Fixtures.authJSON(Fixtures.newTokens))
        }
        let store = makeStore()
        await store.start()
        try await store.login(email: "a@b.c", password: "password1")
        do {
            try await store.deleteAccount()
            XCTFail("expected a failure")
        } catch {}
        XCTAssertNotNil(store.user, "the user can retry")
        XCTAssertNotNil(try tokens.load())
    }
}

import XCTest

@testable import MoonCompass

final class APIClientTests: XCTestCase {
    private var store: InMemoryTokenStore!
    private var client: APIClient!

    private let mePath = "/api/v1/me"
    private let refreshPath = APIClient.refreshPath

    override func setUp() {
        super.setUp()
        StubURLProtocol.reset()
        store = InMemoryTokenStore()
        client = APIClient(config: Fixtures.config, session: StubURLProtocol.makeSession(), tokenStore: store)
    }

    override func tearDown() {
        StubURLProtocol.reset()
        client = nil
        store = nil
        super.tearDown()
    }

    // MARK: Helpers

    private func expectAPIError(
        file: StaticString = #filePath,
        line: UInt = #line,
        _ body: () async throws -> Void
    ) async -> APIError? {
        do {
            try await body()
            XCTFail("expected an APIError", file: file, line: line)
            return nil
        } catch let error as APIError {
            return error
        } catch {
            XCTFail("expected an APIError, got \(error)", file: file, line: line)
            return nil
        }
    }

    /// The stream is buffered, so an event emitted before this call is delivered.
    private func nextSessionEvent() async -> SessionEvent? {
        for await event in client.sessionEvents {
            return event
        }
        return nil
    }

    /// A server whose `/me` accepts only the new access token and whose refresh
    /// hands out the new pair.
    private static func expiringTokenServer(refreshDelay: TimeInterval = 0) -> StubURLProtocol.Handler {
        { request in
            if request.path == APIClient.refreshPath {
                return .json(200, Fixtures.authJSON(Fixtures.newTokens), delay: refreshDelay)
            }
            if request.header("Authorization") == "Bearer \(Fixtures.newTokens.accessToken)" {
                return .json(200, Fixtures.userJSON)
            }
            return .json(401, Fixtures.errorJSON("unauthorized", "invalid or expired token"))
        }
    }

    // MARK: 3. Error envelope

    func testErrorEnvelopeMapsToAPIError() async {
        StubURLProtocol.install { _ in .json(403, Fixtures.errorJSON("email_unverified", "verify your email")) }
        let error = await expectAPIError {
            let _: OKResponse = try await self.client.send(.post, "/api/v1/x", auth: .none)
        }
        XCTAssertEqual(error?.code, "email_unverified")
        XCTAssertEqual(error?.message, "verify your email")
        XCTAssertEqual(error?.status, 403)
        XCTAssertNil(error?.premiumSource)
    }

    func testPremiumAlreadyActivePopulatesPremiumSource() async {
        StubURLProtocol.install { _ in
            .json(409, #"{"error":{"code":"premium_already_active","message":"premium is already active","premiumSource":"apple"}}"#)
        }
        let error = await expectAPIError {
            let _: OKResponse = try await self.client.send(.post, "/api/v1/x", auth: .none)
        }
        XCTAssertEqual(error?.code, "premium_already_active")
        XCTAssertEqual(error?.status, 409)
        XCTAssertEqual(error?.premiumSource, "apple")
    }

    func testNonJSONErrorBodyMapsToHTTPStatusCode() async {
        StubURLProtocol.install { _ in
            StubResponse(status: 502, body: Data("<html>Bad Gateway</html>".utf8), headers: ["Content-Type": "text/html"])
        }
        let error = await expectAPIError {
            let _: OKResponse = try await self.client.send(.get, "/api/v1/x", auth: .none)
        }
        XCTAssertEqual(error?.code, "http_502")
        XCTAssertEqual(error?.status, 502)
    }

    // MARK: 4. Refresh then retry

    func testUnauthorizedRefreshesOnceAndRetriesWithNewToken() async throws {
        try store.save(Fixtures.oldTokens)
        StubURLProtocol.install(Self.expiringTokenServer())

        let user = try await AuthAPI(client: client).me()

        XCTAssertEqual(user.id, "u1")
        let refreshes = StubURLProtocol.requests(to: refreshPath)
        XCTAssertEqual(refreshes.count, 1)
        let refreshBody = try XCTUnwrap(refreshes.first?.body)
        XCTAssertEqual(try JSONDecoder().decode([String: String].self, from: refreshBody), ["refreshToken": "old-refresh"])
        XCTAssertNil(refreshes.first?.header("Authorization"), "refresh is not itself an authed call")

        let meRequests = StubURLProtocol.requests(to: mePath)
        XCTAssertEqual(meRequests.map { $0.header("Authorization") }, ["Bearer old-access", "Bearer new-access"])
        XCTAssertEqual(try store.load(), Fixtures.newTokens, "both halves of the rotated pair are persisted")
    }

    // MARK: 5. Single-flight

    func testConcurrentUnauthorizedCallsShareOneRefresh() async throws {
        try store.save(Fixtures.oldTokens)
        // The delay keeps the refresh in flight while the other 401s arrive, so
        // they exercise joining the in-flight task, not just the stale-token check.
        StubURLProtocol.install(Self.expiringTokenServer(refreshDelay: 0.3))
        let api = AuthAPI(client: client)

        let users = try await withThrowingTaskGroup(of: User.self) { group in
            for _ in 0..<5 {
                group.addTask { try await api.me() }
            }
            var collected: [User] = []
            for try await user in group {
                collected.append(user)
            }
            return collected
        }

        XCTAssertEqual(users.count, 5)
        XCTAssertEqual(StubURLProtocol.requests(to: refreshPath).count, 1, "exactly one refresh for all callers")
        let withOldToken = StubURLProtocol.requests(to: mePath).filter { $0.header("Authorization") == "Bearer old-access" }
        XCTAssertEqual(withOldToken.count, 5, "all five were rejected before the refresh finished")
        XCTAssertEqual(try store.load(), Fixtures.newTokens)
    }

    // MARK: 6. Refresh rejected

    func testRejectedRefreshClearsSessionAndEmitsInvalidated() async throws {
        try store.save(Fixtures.oldTokens)
        StubURLProtocol.install { request in
            if request.path == APIClient.refreshPath {
                return .json(401, Fixtures.errorJSON("invalid_token", "refresh token is invalid, expired or revoked"))
            }
            return .json(401, Fixtures.errorJSON("unauthorized", "invalid or expired token"))
        }

        let error = await expectAPIError { _ = try await AuthAPI(client: self.client).me() }

        XCTAssertEqual(error?.code, "unauthorized", "the caller sees its own request's error")
        XCTAssertEqual(error?.status, 401)
        XCTAssertNil(try store.load())
        let event = await nextSessionEvent()
        XCTAssertEqual(event, .invalidated)
        XCTAssertEqual(StubURLProtocol.requests(to: refreshPath).count, 1)
        XCTAssertEqual(StubURLProtocol.requests(to: mePath).count, 1, "no retry without new tokens")
    }

    // MARK: 7. No loop

    func testSecondUnauthorizedAfterRefreshThrowsWithoutLooping() async throws {
        try store.save(Fixtures.oldTokens)
        StubURLProtocol.install { request in
            if request.path == APIClient.refreshPath {
                return .json(200, Fixtures.authJSON(Fixtures.newTokens))
            }
            return .json(401, Fixtures.errorJSON("unauthorized", "invalid or expired token"))
        }

        let error = await expectAPIError { _ = try await AuthAPI(client: self.client).me() }

        XCTAssertEqual(error?.status, 401)
        XCTAssertEqual(StubURLProtocol.requests(to: mePath).count, 2, "the original plus exactly one retry")
        XCTAssertEqual(StubURLProtocol.requests(to: refreshPath).count, 1)
        XCTAssertEqual(StubURLProtocol.requests.count, 3)
        XCTAssertEqual(try store.load(), Fixtures.newTokens, "a refreshed pair is kept; only a rejected refresh clears it")
    }

    // MARK: 8. Deleted user

    func testUserNotFoundInvalidatesSessionWithoutRefreshing() async throws {
        try store.save(Fixtures.oldTokens)
        StubURLProtocol.install { _ in .json(401, Fixtures.errorJSON("unauthorized", "user not found")) }

        let error = await expectAPIError { _ = try await AuthAPI(client: self.client).me() }

        XCTAssertEqual(error?.message, "user not found")
        XCTAssertEqual(StubURLProtocol.requests(to: refreshPath).count, 0)
        XCTAssertEqual(StubURLProtocol.requests.count, 1)
        XCTAssertNil(try store.load())
        let event = await nextSessionEvent()
        XCTAssertEqual(event, .invalidated)
    }

    // MARK: 9. 204

    func testNoContentResponseForNoResultCall() async throws {
        StubURLProtocol.install { _ in StubResponse(status: 204) }
        try await client.sendIgnoringResponse(.delete, "/api/v1/something", auth: .none)
    }

    func testCreatedIsTreatedLikeOK() async throws {
        StubURLProtocol.install { _ in .json(201, #"{"ok":true}"#) }
        let response: OKResponse = try await client.send(.post, "/api/v1/something", auth: .none)
        XCTAssertTrue(response.ok)
    }

    // MARK: 10. Rate limit

    func testRateLimitParsesRetryAfter() async {
        StubURLProtocol.install { _ in
            .json(429, Fixtures.errorJSON("rate_limited", "too many requests"), headers: ["Retry-After": "900"])
        }
        let error = await expectAPIError {
            try await AuthAPI(client: self.client).resendVerification(email: "a@b.c", language: "en")
        }
        XCTAssertEqual(error?.code, "rate_limited")
        XCTAssertEqual(error?.status, 429)
        XCTAssertEqual(error?.retryAfter, 900)
    }

    // MARK: 11. Transport failure

    func testTransportErrorMapsToNetwork() async {
        StubURLProtocol.install { _ in throw URLError(.notConnectedToInternet) }
        let error = await expectAPIError {
            let _: OKResponse = try await self.client.send(.get, "/api/v1/x", auth: .none)
        }
        XCTAssertEqual(error?.code, "network")
        XCTAssertEqual(error?.status, 0)
    }

    // MARK: 12. Headers

    func testTimezoneHeaderAlwaysSentAndNoAuthorizationOnLoginOrRegister() async throws {
        // A stored session must not leak into calls that create a new one.
        try store.save(Fixtures.oldTokens)
        StubURLProtocol.install { request in
            if request.path == "/api/v1/auth/register" {
                return .json(201, #"{"email":"a@b.c","verificationRequired":true}"#)
            }
            return .json(200, Fixtures.authJSON(Fixtures.newTokens))
        }
        let api = AuthAPI(client: client)
        _ = try await api.register(Self.registerRequest)
        _ = try await api.login(email: "a@b.c", password: "secret123")

        let requests = StubURLProtocol.requests
        XCTAssertEqual(requests.map(\.path), ["/api/v1/auth/register", "/api/v1/auth/login"])
        for request in requests {
            XCTAssertNil(request.header("Authorization"), request.path)
            XCTAssertEqual(request.header("X-Timezone"), TimeZone.current.identifier, request.path)
            XCTAssertEqual(request.header("Content-Type"), "application/json", request.path)
        }
        XCTAssertEqual(try store.load(), Fixtures.newTokens, "login persists its pair")
    }

    func testAuthedCallSendsBearerAndTimezone() async throws {
        try store.save(Fixtures.oldTokens)
        StubURLProtocol.install { _ in .json(200, #"{"slugs":["classic"]}"#) }
        let slugs = try await ShopAPI(client: client).ownedDeckSlugs()
        XCTAssertEqual(slugs, ["classic"])
        let request = try XCTUnwrap(StubURLProtocol.requests.first)
        XCTAssertEqual(request.header("Authorization"), "Bearer old-access")
        XCTAssertEqual(request.header("X-Timezone"), TimeZone.current.identifier)
    }

    func testOptionalAuthSendsTokenOnlyWhenPresent() async throws {
        StubURLProtocol.install { _ in .json(200, #"{"decks":[]}"#) }
        let shop = ShopAPI(client: client)
        _ = try await shop.decks()
        try store.save(Fixtures.oldTokens)
        _ = try await shop.decks()
        XCTAssertEqual(StubURLProtocol.requests.map { $0.header("Authorization") }, [nil, "Bearer old-access"])
    }

    func testRequiredAuthWithoutSessionFailsWithoutNetwork() async {
        StubURLProtocol.install { _ in .json(200, Fixtures.userJSON) }
        let error = await expectAPIError { _ = try await AuthAPI(client: self.client).me() }
        XCTAssertEqual(error?.status, 401)
        XCTAssertTrue(StubURLProtocol.requests.isEmpty)
    }

    // MARK: 13. Register

    func testRegisterDecodesAndDoesNotTouchTokenStore() async throws {
        StubURLProtocol.install { _ in .json(201, #"{"email":"a@b.c","verificationRequired":true}"#) }
        let response = try await AuthAPI(client: client).register(Self.registerRequest)
        XCTAssertEqual(response, RegisterResponse(email: "a@b.c", verificationRequired: true))
        XCTAssertNil(try store.load())

        let body = try XCTUnwrap(StubURLProtocol.requests.first?.body)
        let object = try XCTUnwrap(JSONSerialization.jsonObject(with: body) as? [String: Any])
        XCTAssertEqual(
            object.keys.sorted(),
            ["acceptedPersonalData", "acceptedPrivacy", "acceptedTerms", "consentVersion", "email", "language", "password"]
        )
    }

    // MARK: Session lifecycle

    func testVerifyEmailPersistsTokens() async throws {
        StubURLProtocol.install { _ in .json(200, Fixtures.authJSON(Fixtures.newTokens)) }
        let user = try await AuthAPI(client: client).verifyEmail(email: "a@b.c", code: "123456")
        XCTAssertEqual(user.id, "u1")
        XCTAssertEqual(try store.load(), Fixtures.newTokens)
    }

    func testLogoutClearsLocallyEvenWhenServerUnreachable() async throws {
        try store.save(Fixtures.oldTokens)
        StubURLProtocol.install { _ in throw URLError(.timedOut) }
        await AuthAPI(client: client).logout()
        XCTAssertNil(try store.load())
        let body = try XCTUnwrap(StubURLProtocol.requests(to: "/api/v1/auth/logout").first?.body)
        XCTAssertEqual(try JSONDecoder().decode([String: String].self, from: body), ["refreshToken": "old-refresh"])
    }

    func testAppleVerifyCall() async throws {
        try store.save(Fixtures.oldTokens)
        StubURLProtocol.install { _ in .json(200, #"{"ok":true,"hasPremium":false,"reason":"expired"}"#) }
        let token = UUID()
        let response = try await BillingAPI(client: client).verifyAppleTransaction(
            originalTransactionId: "200000",
            appAccountToken: token
        )
        XCTAssertEqual(response.reason, "expired")
        let body = try XCTUnwrap(StubURLProtocol.requests.first?.body)
        XCTAssertEqual(
            try JSONDecoder().decode([String: String].self, from: body),
            ["originalTransactionId": "200000", "appAccountToken": token.uuidString.lowercased()]
        )
    }

    private static let registerRequest = RegisterRequest(
        email: "a@b.c",
        password: "secret123",
        language: "en",
        acceptedTerms: true,
        acceptedPrivacy: true,
        acceptedPersonalData: true,
        consentVersion: "2025-09"
    )
}

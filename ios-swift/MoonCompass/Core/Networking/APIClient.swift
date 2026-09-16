import Foundation

enum HTTPMethod: String, Sendable {
    case get = "GET"
    case post = "POST"
    case put = "PUT"
    case patch = "PATCH"
    case delete = "DELETE"
}

/// How a request relates to the stored session.
enum AuthRequirement: Sendable {
    /// Never sends a token (login, register, …) — a stale token must not leak
    /// into a request that is about to create a new session.
    case none
    /// Sends the token if there is one. The server's optional-auth middleware
    /// ignores a bad token instead of answering 401, so these calls never
    /// refresh and never affect the session.
    case optional
    /// Requires a session; a 401 triggers the single-flight refresh.
    case required
}

enum SessionEvent: Sendable, Equatable {
    /// The stored tokens were rejected for good and have been cleared. The app
    /// should route back to sign-in. Not emitted for a user-initiated logout.
    case invalidated
}

/// The only place that talks HTTP.
///
/// An actor because it owns mutable session state — the in-flight refresh task —
/// that many concurrent requests read and write.
///
/// ## Single-flight refresh
/// The server rotates the refresh token on use (the old one is consumed), so two
/// parallel refreshes would make the second fail with `invalid_token` and log the
/// user out. Two mechanisms prevent that:
/// 1. `refreshTask` holds the in-flight refresh. The check and assignment happen
///    with no `await` between them, so actor isolation makes them atomic; every
///    other 401 that arrives meanwhile awaits the same task.
/// 2. A 401 that arrives *after* a refresh has already finished is compared with
///    the stored token: if the store already holds a different access token, the
///    request is simply retried with it instead of refreshing again.
///
/// The persisting of the new pair — or the clearing of the store and the
/// `invalidated` event — happens inside the shared task, so it runs exactly once
/// no matter how many requests are waiting on it.
///
/// ## Session-invalidated signal
/// An `AsyncStream<SessionEvent>` rather than an injected callback: the object
/// that reacts (a future session store) will itself hold the client, so a
/// callback would force a two-phase init or a weak back-reference, and would make
/// the client choose the callback's isolation. With a stream the observer just
/// iterates `sessionEvents` on its own actor (e.g. MainActor). The stream buffers
/// the newest event, so an invalidation that happens before observation starts is
/// not lost. It is single-consumer by design: one owner decides where to route.
actor APIClient {
    nonisolated let config: APIConfig
    nonisolated let sessionEvents: AsyncStream<SessionEvent>

    private let session: URLSession
    private let tokenStore: any TokenStore
    private let eventContinuation: AsyncStream<SessionEvent>.Continuation
    private let encoder = JSONCoding.makeEncoder()
    private let decoder = JSONCoding.makeDecoder()
    private var refreshTask: Task<AuthTokens, any Error>?

    static let refreshPath = "/api/v1/auth/refresh"

    init(config: APIConfig = .production, session: URLSession = .shared, tokenStore: any TokenStore) {
        self.config = config
        self.session = session
        self.tokenStore = tokenStore
        let (stream, continuation) = AsyncStream.makeStream(
            of: SessionEvent.self,
            bufferingPolicy: .bufferingNewest(1)
        )
        self.sessionEvents = stream
        self.eventContinuation = continuation
    }

    deinit {
        eventContinuation.finish()
    }

    // MARK: - Session

    var hasSession: Bool { storedTokens() != nil }

    /// Persists the pair from a login/verify/reset response.
    func storeSession(_ tokens: AuthTokens) throws {
        try tokenStore.save(tokens)
    }

    /// Local sign-out. Deliberately emits no event: the caller initiated it.
    func clearSession() {
        try? tokenStore.clear()
    }

    /// An access token valid for at least `margin`, refreshed first if needed —
    /// for connections that authenticate once instead of per request, like the
    /// Arcana Clash WebSocket. `force` refreshes even a token that looks
    /// valid, after the server rejected it.
    func validAccessToken(margin: TimeInterval = 60, force: Bool = false) async throws -> String {
        guard let tokens = storedTokens() else { throw APIError.notAuthenticated }
        if !force, let expiry = Self.expiry(ofJWT: tokens.accessToken), expiry > Date().addingTimeInterval(margin) {
            return tokens.accessToken
        }
        return try await refreshedTokens(replacing: tokens).accessToken
    }

    /// The `exp` claim of a JWT, read without verifying it: only to decide
    /// whether to refresh, never to trust the token.
    static func expiry(ofJWT token: String) -> Date? {
        let parts = token.split(separator: ".")
        guard parts.count == 3 else { return nil }
        var base64 = parts[1].replacingOccurrences(of: "-", with: "+").replacingOccurrences(of: "_", with: "/")
        base64 += String(repeating: "=", count: (4 - base64.count % 4) % 4)
        guard let data = Data(base64Encoded: base64),
              let claims = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
              let exp = claims["exp"] as? NSNumber else {
            return nil
        }
        return Date(timeIntervalSince1970: exp.doubleValue)
    }

    /// The refresh token, for logout — which must name the token to revoke.
    func currentRefreshToken() -> String? {
        storedTokens()?.refreshToken
    }

    // MARK: - Requests

    func send<Response: Decodable & Sendable>(
        _ method: HTTPMethod,
        _ path: String,
        body: (any Encodable & Sendable)? = nil,
        auth: AuthRequirement
    ) async throws -> Response {
        let bodyData = try encodeBody(body)
        let (data, status) = try await execute(method, path, body: bodyData, auth: auth)
        if Response.self == EmptyResponse.self {
            // Safe: the types were just compared.
            return EmptyResponse() as! Response
        }
        do {
            return try decoder.decode(Response.self, from: data)
        } catch {
            throw APIError.decoding(error, status: status)
        }
    }

    /// For calls whose body is irrelevant — including 204 No Content.
    func sendIgnoringResponse(
        _ method: HTTPMethod,
        _ path: String,
        body: (any Encodable & Sendable)? = nil,
        auth: AuthRequirement
    ) async throws {
        let _: EmptyResponse = try await send(method, path, body: body, auth: auth)
    }

    // MARK: - Pipeline

    private func execute(
        _ method: HTTPMethod,
        _ path: String,
        body: Data?,
        auth: AuthRequirement
    ) async throws -> (Data, Int) {
        switch auth {
        case .none:
            let request = makeRequest(method, path, body: body, accessToken: nil)
            return try validated(await transport(request))
        case .optional:
            let request = makeRequest(method, path, body: body, accessToken: storedTokens()?.accessToken)
            return try validated(await transport(request))
        case .required:
            return try await executeAuthed(method, path, body: body)
        }
    }

    private func executeAuthed(_ method: HTTPMethod, _ path: String, body: Data?) async throws -> (Data, Int) {
        guard let tokens = storedTokens() else { throw APIError.notAuthenticated }

        let first = try await transport(makeRequest(method, path, body: body, accessToken: tokens.accessToken))
        guard first.1.statusCode == 401 else { return try validated(first) }

        let original = Self.error(from: first.0, response: first.1)
        if original.isUserNotFound {
            // The JWT is valid but its user is gone; the refresh token belongs to
            // the same deleted user, so refreshing would only waste a round trip.
            invalidateSession()
            throw original
        }

        let fresh: AuthTokens
        do {
            fresh = try await refreshedTokens(replacing: tokens)
        } catch let refreshError as APIError where refreshError.status == 401 {
            // Session already cleared and announced by the refresh; surface what
            // the caller's own request got.
            throw original
        }

        // Exactly one retry. A second 401 means something other than expiry is
        // wrong (clock skew, revoked key); looping would hammer the server.
        let retry = try await transport(makeRequest(method, path, body: body, accessToken: fresh.accessToken))
        if retry.1.statusCode == 401 {
            let retryError = Self.error(from: retry.0, response: retry.1)
            if retryError.isUserNotFound { invalidateSession() }
            throw retryError
        }
        return try validated(retry)
    }

    /// Returns a usable pair after `rejected` got a 401, refreshing at most once
    /// across all concurrent callers (see the type's documentation).
    private func refreshedTokens(replacing rejected: AuthTokens) async throws -> AuthTokens {
        if let inFlight = refreshTask {
            return try await inFlight.value
        }
        guard let current = storedTokens() else {
            // A concurrent refresh already failed and cleared the session.
            throw APIError.notAuthenticated
        }
        if current.accessToken != rejected.accessToken {
            // Someone refreshed while this request was on the wire.
            return current
        }

        let task = Task { try await self.performRefresh(refreshToken: current.refreshToken) }
        refreshTask = task
        defer { refreshTask = nil }
        return try await task.value
    }

    private func performRefresh(refreshToken: String) async throws -> AuthTokens {
        let body = try encoder.encode(RefreshTokenRequest(refreshToken: refreshToken))
        let (data, response) = try await transport(
            makeRequest(.post, Self.refreshPath, body: body, accessToken: nil)
        )
        if response.statusCode == 401 {
            invalidateSession()
            throw Self.error(from: data, response: response)
        }
        // Any other failure (network, 5xx, 429) keeps the tokens: the session may
        // well be fine, and logging the user out over a flaky connection is worse.
        let (okData, status) = try validated((data, response))
        let auth: AuthResponse
        do {
            auth = try decoder.decode(AuthResponse.self, from: okData)
        } catch {
            throw APIError.decoding(error, status: status)
        }
        try tokenStore.save(auth.tokens)
        return auth.tokens
    }

    private func invalidateSession() {
        try? tokenStore.clear()
        eventContinuation.yield(.invalidated)
    }

    private func storedTokens() -> AuthTokens? {
        // An unreadable Keychain is treated as signed out rather than as an error
        // on every request.
        (try? tokenStore.load()) ?? nil
    }

    // MARK: - HTTP

    private func encodeBody(_ body: (any Encodable & Sendable)?) throws -> Data? {
        guard let body else { return nil }
        do {
            return try encoder.encode(body)
        } catch {
            throw APIError(code: "encoding_error", message: String(describing: error), status: 0)
        }
    }

    private func makeRequest(_ method: HTTPMethod, _ path: String, body: Data?, accessToken: String?) -> URLRequest {
        var request = URLRequest(url: config.endpointURL(path))
        request.httpMethod = method.rawValue
        request.timeoutInterval = 30
        request.setValue("application/json", forHTTPHeaderField: "Accept")
        // Read per request, not cached: the user may travel. Daily quotas on the
        // server reset at the user's local midnight.
        request.setValue(TimeZone.current.identifier, forHTTPHeaderField: "X-Timezone")
        if let body {
            request.httpBody = body
            request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        }
        if let accessToken {
            request.setValue("Bearer \(accessToken)", forHTTPHeaderField: "Authorization")
        }
        return request
    }

    private func transport(_ request: URLRequest) async throws -> (Data, HTTPURLResponse) {
        let data: Data
        let response: URLResponse
        do {
            (data, response) = try await session.data(for: request)
        } catch is CancellationError {
            throw CancellationError()
        } catch let urlError as URLError where urlError.code == .cancelled {
            // Cancellation is not a network failure; keep it distinguishable so
            // callers do not show an error for a screen the user left.
            throw CancellationError()
        } catch {
            throw APIError.network(error)
        }
        guard let http = response as? HTTPURLResponse else {
            throw APIError.network(URLError(.badServerResponse))
        }
        return (data, http)
    }

    /// Passes 2xx through (201 and 204 included) and turns anything else into an
    /// APIError.
    private func validated(_ result: (Data, HTTPURLResponse)) throws -> (Data, Int) {
        let (data, response) = result
        guard (200..<300).contains(response.statusCode) else {
            throw Self.error(from: data, response: response)
        }
        return (data, response.statusCode)
    }

    static func error(from data: Data, response: HTTPURLResponse) -> APIError {
        let status = response.statusCode
        let retryAfter = response.value(forHTTPHeaderField: "Retry-After")
            .flatMap { TimeInterval($0.trimmingCharacters(in: .whitespaces)) }

        if let envelope = try? JSONDecoder().decode(APIErrorEnvelope.self, from: data) {
            return APIError(
                code: envelope.error.code,
                message: envelope.error.message,
                status: status,
                retryAfter: retryAfter,
                premiumSource: envelope.error.premiumSource
            )
        }
        return APIError(
            code: "http_\(status)",
            message: HTTPURLResponse.localizedString(forStatusCode: status),
            status: status,
            retryAfter: retryAfter
        )
    }
}

import Foundation

/// Where the game server lives. By default the same host as the main API,
/// under `/api/v1/arcana`, so one certificate and one sign-in cover both.
struct ArcanaConfig: Sendable, Equatable {
    var baseURL: URL
    /// DEBUG: a fixed token instead of the signed-in session.
    var debugToken: String?

    var socketURL: URL {
        var components = URLComponents(url: baseURL.appending(path: "/api/v1/arcana/ws"), resolvingAgainstBaseURL: false)!
        components.scheme = baseURL.scheme == "http" ? "ws" : "wss"
        return components.url!
    }

    func endpoint(_ path: String) -> URL { baseURL.appending(path: "/api/v1/arcana" + path) }

    static func current(api: APIConfig) -> ArcanaConfig {
        #if DEBUG
        let base = DebugLaunch.arcanaURL.flatMap(URL.init(string:)) ?? api.baseURL
        return ArcanaConfig(baseURL: base, debugToken: DebugLaunch.arcanaToken)
        #else
        return ArcanaConfig(baseURL: api.baseURL)
        #endif
    }
}

/// The catalog and match history over plain HTTPS.
struct ArcanaAPI: Sendable {
    let config: ArcanaConfig
    let client: APIClient
    var session: URLSession = .shared

    func token(force: Bool = false) async throws -> String {
        if let debug = config.debugToken { return debug }
        return try await client.validAccessToken(force: force)
    }

    func catalog() async throws -> ArcanaCatalog {
        try await get("/catalog", authed: false)
    }

    func matches(limit: Int = 30) async throws -> [ArcanaMatchSummary] {
        struct Response: Decodable { var matches: [ArcanaMatchSummary] }
        let response: Response = try await get("/matches?limit=\(limit)", authed: true)
        return response.matches
    }

    private func get<T: Decodable>(_ path: String, authed: Bool) async throws -> T {
        guard let url = URL(string: config.endpoint("").absoluteString + path) else { throw URLError(.badURL) }
        var request = URLRequest(url: url, timeoutInterval: 20)
        if authed {
            request.setValue("Bearer \(try await token())", forHTTPHeaderField: "Authorization")
        }
        let (data, response): (Data, URLResponse)
        do {
            (data, response) = try await session.data(for: request)
        } catch {
            throw APIError.network(error)
        }
        guard let http = response as? HTTPURLResponse else { throw URLError(.badServerResponse) }
        guard (200..<300).contains(http.statusCode) else { throw APIClient.error(from: data, response: http) }
        do {
            return try ArcanaCoding.decoder.decode(T.self, from: data)
        } catch {
            throw APIError.decoding(error, status: http.statusCode)
        }
    }
}

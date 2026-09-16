import Foundation
import os

@testable import MoonCompass

/// A canned response for `StubURLProtocol`.
struct StubResponse: Sendable {
    var status: Int
    var body: Data
    var headers: [String: String]
    /// Holds the response back so concurrent requests genuinely overlap with it.
    var delay: TimeInterval

    init(status: Int, body: Data = Data(), headers: [String: String] = [:], delay: TimeInterval = 0) {
        self.status = status
        self.body = body
        self.headers = headers
        self.delay = delay
    }

    static func json(_ status: Int, _ json: String, headers: [String: String] = [:], delay: TimeInterval = 0) -> StubResponse {
        var merged = headers
        merged["Content-Type"] = "application/json"
        return StubResponse(status: status, body: Data(json.utf8), headers: merged, delay: delay)
    }
}

/// A request as the stub saw it. The body is captured separately because
/// URLSession hands protocols the body as a stream, with `httpBody` nil.
struct RecordedRequest: Sendable {
    var request: URLRequest
    var body: Data?

    var path: String { request.url?.path ?? "" }
    func header(_ name: String) -> String? { request.value(forHTTPHeaderField: name) }
}

/// Intercepts every request of a session built by `makeSession()`.
///
/// URLSession instantiates protocol classes itself, so the handler has to live
/// in static storage; it is guarded by `OSAllocatedUnfairLock` because loads run
/// on URLSession's own threads, concurrently in the single-flight test. Tests
/// using it must not run in parallel with each other (XCTest runs serially
/// unless parallel testing is enabled in the scheme, which it is not).
final class StubURLProtocol: URLProtocol {
    typealias Handler = @Sendable (RecordedRequest) throws -> StubResponse

    private struct State {
        var handler: Handler?
        var requests: [RecordedRequest] = []
    }

    private static let state = OSAllocatedUnfairLock(initialState: State())

    static func install(_ handler: @escaping Handler) {
        state.withLock { $0 = State(handler: handler) }
    }

    static func reset() {
        state.withLock { $0 = State() }
    }

    static var requests: [RecordedRequest] {
        state.withLock { $0.requests }
    }

    static func requests(to path: String) -> [RecordedRequest] {
        requests.filter { $0.path == path }
    }

    static func makeSession() -> URLSession {
        let configuration = URLSessionConfiguration.ephemeral
        configuration.protocolClasses = [StubURLProtocol.self]
        return URLSession(configuration: configuration)
    }

    override class func canInit(with request: URLRequest) -> Bool { true }
    override class func canonicalRequest(for request: URLRequest) -> URLRequest { request }

    override func startLoading() {
        let recorded = RecordedRequest(request: request, body: Self.readBody(of: request))
        let handler = Self.state.withLock { state -> Handler? in
            state.requests.append(recorded)
            return state.handler
        }

        let result: Result<StubResponse, any Error>
        if let handler {
            result = Result { try handler(recorded) }
        } else {
            result = .failure(URLError(.resourceUnavailable))
        }

        let delay = (try? result.get())?.delay ?? 0
        // URLProtocol is not Sendable, but the loading system expects exactly this:
        // callbacks on the client from any thread after startLoading returns.
        nonisolated(unsafe) let loader = self
        DispatchQueue.global().asyncAfter(deadline: .now() + delay) {
            loader.deliver(result)
        }
    }

    override func stopLoading() {}

    private func deliver(_ result: Result<StubResponse, any Error>) {
        switch result {
        case .success(let stub):
            let response = HTTPURLResponse(
                url: request.url!,
                statusCode: stub.status,
                httpVersion: "HTTP/1.1",
                headerFields: stub.headers
            )!
            client?.urlProtocol(self, didReceive: response, cacheStoragePolicy: .notAllowed)
            if !stub.body.isEmpty {
                client?.urlProtocol(self, didLoad: stub.body)
            }
            client?.urlProtocolDidFinishLoading(self)
        case .failure(let error):
            client?.urlProtocol(self, didFailWithError: error)
        }
    }

    private static func readBody(of request: URLRequest) -> Data? {
        if let body = request.httpBody { return body }
        guard let stream = request.httpBodyStream else { return nil }
        stream.open()
        defer { stream.close() }
        var data = Data()
        var buffer = [UInt8](repeating: 0, count: 4096)
        while stream.hasBytesAvailable {
            let count = stream.read(&buffer, maxLength: buffer.count)
            guard count > 0 else { break }
            data.append(buffer, count: count)
        }
        return data
    }
}

/// Fixtures shared by the networking tests.
enum Fixtures {
    static let config = APIConfig(baseURL: URL(string: "https://stub.example.com")!)
    static let oldTokens = AuthTokens(accessToken: "old-access", refreshToken: "old-refresh")
    static let newTokens = AuthTokens(accessToken: "new-access", refreshToken: "new-refresh")

    static let userJSON = """
    {"id":"u1","email":"a@b.c","hasPremium":false,"premiumExpiresAt":null,\
    "emailVerified":true,"createdAt":"2025-09-13T01:12:03.456789+03:00"}
    """

    static func authJSON(_ tokens: AuthTokens) -> String {
        #"{"user":\#(userJSON),"accessToken":"\#(tokens.accessToken)","refreshToken":"\#(tokens.refreshToken)"}"#
    }

    static func errorJSON(_ code: String, _ message: String) -> String {
        #"{"error":{"code":"\#(code)","message":"\#(message)"}}"#
    }
}

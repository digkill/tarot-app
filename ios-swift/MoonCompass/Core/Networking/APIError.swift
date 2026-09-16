import Foundation

/// The single error type the networking layer throws, so UI code switches on
/// the server's stable `code` rather than on transport details.
///
/// Codes come from the server's envelope `{"error":{"code","message"}}`. Two are
/// synthesised on the client: `network` (status 0 — no HTTP response at all) and
/// `http_<status>` (a response whose body is not the envelope, e.g. an HTML page
/// from a proxy).
struct APIError: Error, Sendable, Equatable {
    var code: String
    var message: String
    var status: Int
    /// Seconds from `Retry-After`, set on 429 so the UI can show a countdown
    /// instead of letting the user hammer a rate-limited endpoint.
    var retryAfter: TimeInterval?
    /// Only on `premium_already_active`: which store already granted premium,
    /// so the UI can tell the user where to manage it.
    var premiumSource: String?

    init(
        code: String,
        message: String,
        status: Int,
        retryAfter: TimeInterval? = nil,
        premiumSource: String? = nil
    ) {
        self.code = code
        self.message = message
        self.status = status
        self.retryAfter = retryAfter
        self.premiumSource = premiumSource
    }

    static func network(_ underlying: any Error) -> APIError {
        APIError(code: "network", message: underlying.localizedDescription, status: 0)
    }

    /// Thrown without touching the network when an authed call is made with no
    /// stored session. Status 401 so callers treat it like a server rejection.
    static let notAuthenticated = APIError(code: "unauthorized", message: "not signed in", status: 401)

    static func decoding(_ underlying: any Error, status: Int) -> APIError {
        APIError(code: "decoding_error", message: String(describing: underlying), status: status)
    }

    /// A valid JWT for a user that no longer exists. Refreshing cannot help — the
    /// refresh token belongs to the same deleted user — so the session is dead.
    var isUserNotFound: Bool {
        status == 401 && message.trimmingCharacters(in: .whitespaces).lowercased() == "user not found"
    }
}

extension APIError: LocalizedError {
    var errorDescription: String? { message }
}

/// Wire shape of the error envelope. Decoded separately from APIError so a
/// body that is not the envelope falls through to `http_<status>`.
struct APIErrorEnvelope: Decodable, Sendable {
    struct Detail: Decodable, Sendable {
        var code: String
        var message: String
        var premiumSource: String?
    }

    var error: Detail
}

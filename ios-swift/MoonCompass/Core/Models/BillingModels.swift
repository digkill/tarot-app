import Foundation

/// Body of `POST /billing/apple/verify`. All optional: the client sends either
/// the JWS from StoreKit or, when restoring, only the original transaction id.
/// Synthesised Encodable omits nil fields rather than sending null.
struct AppleVerifyRequest: Encodable, Sendable, Equatable {
    var signedTransaction: String?
    var originalTransactionId: String?
    var appAccountToken: String?
}

/// The server answers with different subsets depending on the outcome — a full
/// record for a granted purchase, or just `{ok, hasPremium, reason}` for e.g. an
/// expired subscription — so everything except `ok`/`hasPremium` is optional.
struct AppleVerifyResponse: Decodable, Sendable, Equatable {
    var ok: Bool
    var hasPremium: Bool
    var recorded: Bool?
    var premiumSource: String?
    var productId: String?
    var deckSlug: String?
    var expiresAt: Date?
    var environment: String?
    var transactionId: String?
    var appleTransactionId: String?
    var originalTransactionId: String?
    var reason: String?
}

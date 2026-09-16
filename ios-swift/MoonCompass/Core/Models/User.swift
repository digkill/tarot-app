import Foundation

/// The server's `userView`, returned bare by `GET /me` and inside every auth
/// response.
///
/// `premiumProductId` and `premiumSource` are `omitempty` on the server, so the
/// keys are absent (not null) for a free user; optionals cover both.
struct User: Codable, Sendable, Equatable, Identifiable {
    var id: String
    var email: String
    var hasPremium: Bool
    var premiumExpiresAt: Date?
    var premiumProductId: String?
    var premiumSource: String?
    var emailVerified: Bool
    var createdAt: Date
}

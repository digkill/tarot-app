import Foundation

struct QuotaBucket: Decodable, Sendable, Equatable {
    var used: Int
    var limit: Int
    var remaining: Int
}

/// Today's quotas as the server counts them, in the device's time zone
/// (the client sends `X-Timezone` on every request).
struct UsageSnapshot: Decodable, Sendable, Equatable {
    var date: String
    var timezone: String
    var hasPremium: Bool
    var dailyCards: QuotaBucket
    var interpretations: QuotaBucket
}

private struct EmptyBody: Encodable, Sendable {}

struct UsageAPI: Sendable {
    let client: APIClient

    func usage() async throws -> UsageSnapshot {
        try await client.send(.get, "/api/v1/usage", auth: .required)
    }

    /// Spends one daily-card draw. Throws `quota_exceeded` (429) at the limit.
    ///
    /// The ad-unlock flow of the React Native client is not ported: its "ad"
    /// was the app's own splash video, which App Review could fairly call
    /// misleading, so the iOS client never sends an ad token.
    func consumeDailyCard() async throws -> UsageSnapshot {
        try await client.send(.post, "/api/v1/usage/daily-card", body: EmptyBody(), auth: .required)
    }
}

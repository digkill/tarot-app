import Foundation

/// Decides whether another one-card draw is allowed today.
///
/// The server is the authority. If it cannot be reached — offline, or the
/// quota endpoint missing or failing — a local ledger stands in, so the daily
/// card still works without a connection. A real `quota_exceeded` from the
/// server is never overridden by the local ledger.
struct DailyCardGate: Sendable {
    let usage: UsageAPI
    let ledger: LocalDailyLedger

    /// Returns whether this draw is the day's first card or an extra one.
    func consume(hasTodaysCard: Bool, hasPremium: Bool) async throws -> Reading.Kind {
        let kind: Reading.Kind = hasTodaysCard ? .bonus : .daily
        do {
            _ = try await usage.consumeDailyCard()
            return kind
        } catch let error as APIError where Self.isUnavailable(error) {
            guard ledger.consume(limit: hasPremium ? LocalDailyLedger.premiumLimit : LocalDailyLedger.freeLimit) else {
                throw APIError.quotaExceeded
            }
            return kind
        }
    }

    static func isUnavailable(_ error: APIError) -> Bool {
        error.code == "network" || error.status == 404 || error.status >= 500
    }
}

extension APIError {
    static let quotaExceeded = APIError(code: "quota_exceeded", message: "daily limit reached", status: 429)
}

/// Offline fallback count of today's draws, in the device's calendar.
struct LocalDailyLedger: @unchecked Sendable {
    /// Mirrors the server: 3 free draws a day, 20 with premium.
    static let freeLimit = 3
    static let premiumLimit = 20
    static let storageKey = "tarot.dailyUsage.v1"

    // UserDefaults is documented as thread-safe.
    let defaults: UserDefaults
    var now: @Sendable () -> Date = { Date() }

    private struct Entry: Codable {
        var date: String
        var cards: Int
    }

    /// Records one draw if under `limit`; returns false at the limit.
    func consume(limit: Int) -> Bool {
        let today = Self.dayString(now())
        var entry = (defaults.data(forKey: Self.storageKey))
            .flatMap { try? JSONDecoder().decode(Entry.self, from: $0) }
            .flatMap { $0.date == today ? $0 : nil }
            ?? Entry(date: today, cards: 0)
        guard entry.cards < limit else { return false }
        entry.cards += 1
        if let data = try? JSONEncoder().encode(entry) {
            defaults.set(data, forKey: Self.storageKey)
        }
        return true
    }

    private static func dayString(_ date: Date) -> String {
        let components = Calendar.current.dateComponents([.year, .month, .day], from: date)
        return String(format: "%04d-%02d-%02d", components.year ?? 0, components.month ?? 0, components.day ?? 0)
    }
}

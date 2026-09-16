#if DEBUG
import Foundation

/// Launch arguments for looking at screens in the simulator without signing
/// in or tapping — e.g. for screenshots from the command line:
///
///     xcrun simctl launch booted org.mediarise.tarot -previewSignedIn \
///         -previewReading three-card
///
/// Compiled into Debug builds only.
enum DebugLaunch {
    private static var arguments: [String] { ProcessInfo.processInfo.arguments }

    private static func value(after flag: String) -> String? {
        guard let index = arguments.firstIndex(of: flag), arguments.indices.contains(index + 1) else {
            return nil
        }
        return arguments[index + 1]
    }

    /// `-previewSignedIn`: start signed in as a local placeholder account,
    /// with no network request. Add `-previewPremium` for a premium account.
    static var previewUser: User? {
        guard arguments.contains("-previewSignedIn") else { return nil }
        return User(id: "preview", email: "preview@moon.compass",
                    hasPremium: arguments.contains("-previewPremium"),
                    premiumExpiresAt: nil, premiumProductId: nil, premiumSource: nil,
                    emailVerified: true, createdAt: Date())
    }

    /// `-previewTab history|decks|settings`: open on that tab.
    static var tab: AppTab? {
        switch value(after: "-previewTab") {
        case "history": return .history
        case "decks": return .decks
        case "settings": return .settings
        default: return nil
        }
    }

    /// `-previewReading <spread-id>`: open the reading table for that spread.
    static var readingSpread: String? { value(after: "-previewReading") }

    /// `-previewInterpretation <spread-id>`: save a sample reading of that
    /// spread and open its interpretation.
    static var interpretationSpread: String? { value(after: "-previewInterpretation") }

    /// `-previewCatalog`: open the spread catalog.
    static var catalog: Bool { arguments.contains("-previewCatalog") }
}
#endif

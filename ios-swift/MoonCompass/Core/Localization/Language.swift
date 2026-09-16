import Foundation

/// The languages the app ships translations for. Mirrors `SUPPORTED_LANGUAGES`
/// in the React Native client, whose `i18n/*.json` files this app reads.
enum Language: String, CaseIterable, Codable, Sendable, Identifiable {
    case en
    case ru
    case th
    case zh

    var id: String { rawValue }

    /// The language's own name, for the picker — shown the same way whatever
    /// language the app is currently in.
    var nativeName: String {
        switch self {
        case .en: return "English"
        case .ru: return "Русский"
        case .th: return "ไทย"
        case .zh: return "中文"
        }
    }

    /// Picks the first supported language from the device's preferred list,
    /// falling back to English — the same rule as `resolveDeviceLanguage()`.
    /// Any Chinese variant (zh-Hans, zh-Hant, zh-TW…) maps to `zh`.
    static func resolve(preferred: [String] = Locale.preferredLanguages) -> Language {
        for tag in preferred {
            let primary = tag
                .lowercased()
                .replacingOccurrences(of: "_", with: "-")
                .split(separator: "-")
                .first
                .map(String.init) ?? ""
            if let language = Language(rawValue: primary) {
                return language
            }
        }
        return .en
    }
}

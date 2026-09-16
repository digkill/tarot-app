import SwiftUI

/// The app's palette. A deck carries its own, so buying or selecting a deck
/// re-skins everything including the tab bar — the same behaviour as the React
/// Native client, where `useAppColors()` returns the selected deck's theme.
///
/// Every value arrives from the server as a hex string and is therefore
/// validated: a malformed field falls back to the classic value rather than
/// rendering something unreadable.
struct AppColors: Equatable, Sendable {
    var bg: Color
    var panel: Color
    var accent: Color
    var text: Color
    var muted: Color
    var gold: Color
    var danger: Color
    var tabBar: Color

    static let classic = AppColors(
        bg: Color(hex: "#040307") ?? .black,
        panel: Color(hex: "#1a1030") ?? .black,
        accent: Color(hex: "#6c5ce7") ?? .purple,
        text: Color(hex: "#f7f4ea") ?? .white,
        muted: Color(hex: "#9a93b3") ?? .gray,
        gold: Color(hex: "#d4af37") ?? .yellow,
        danger: Color(hex: "#ff6b6b") ?? .red,
        tabBar: Color(hex: "#0c0a14") ?? .black
    )
}

/// The wire format of a deck's theme, mirroring `decks.Theme` on the server.
struct ThemePayload: Decodable, Sendable {
    var bg: String?
    var panel: String?
    var accent: String?
    var text: String?
    var muted: String?
    var gold: String?
    var danger: String?
    var tabBar: String?
}

extension AppColors {
    /// Builds a palette from a server payload, replacing each invalid or missing
    /// field with the classic one. Per-field rather than all-or-nothing, so one
    /// bad value in an otherwise good theme does not discard the whole thing.
    init(payload: ThemePayload?) {
        let fallback = AppColors.classic
        self.init(
            bg: Color(hex: payload?.bg) ?? fallback.bg,
            panel: Color(hex: payload?.panel) ?? fallback.panel,
            accent: Color(hex: payload?.accent) ?? fallback.accent,
            text: Color(hex: payload?.text) ?? fallback.text,
            muted: Color(hex: payload?.muted) ?? fallback.muted,
            gold: Color(hex: payload?.gold) ?? fallback.gold,
            danger: Color(hex: payload?.danger) ?? fallback.danger,
            tabBar: Color(hex: payload?.tabBar) ?? fallback.tabBar
        )
    }
}

extension Color {
    /// Parses `#rgb` and `#rrggbb`, with or without the leading `#`.
    /// Returns nil for anything else, which is what lets callers fall back
    /// per field instead of guessing.
    init?(hex: String?) {
        guard let raw = hex?.trimmingCharacters(in: .whitespacesAndNewlines), !raw.isEmpty else {
            return nil
        }
        var digits = raw.hasPrefix("#") ? String(raw.dropFirst()) : raw
        guard digits.allSatisfy(\.isHexDigit) else { return nil }

        if digits.count == 3 {
            // #abc -> #aabbcc
            digits = digits.map { "\($0)\($0)" }.joined()
        }
        guard digits.count == 6, let value = UInt32(digits, radix: 16) else { return nil }

        self.init(
            .sRGB,
            red: Double((value >> 16) & 0xff) / 255,
            green: Double((value >> 8) & 0xff) / 255,
            blue: Double(value & 0xff) / 255,
            opacity: 1
        )
    }
}

// MARK: - Environment

private struct AppColorsKey: EnvironmentKey {
    static let defaultValue = AppColors.classic
}

extension EnvironmentValues {
    var appColors: AppColors {
        get { self[AppColorsKey.self] }
        set { self[AppColorsKey.self] = newValue }
    }
}

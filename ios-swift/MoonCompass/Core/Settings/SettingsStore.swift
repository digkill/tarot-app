import Foundation
import Observation

/// Settings persisted on the device. Mirrors `Settings` in the React Native
/// client, minus three fields that client stores and toggles but never reads
/// (`disableSounds`, `showMysticMode`, `dailyReminder`) and `theme`, which only
/// ever switched the navigation chrome of an app that is dark regardless.
struct AppSettings: Codable, Equatable, Sendable {
    var language: Language
    /// Once the user picks a language in Settings it sticks; until then the app
    /// follows the device language on every launch.
    var languageExplicit = false
    var acceptedDisclaimer = false
    var disableAnimations = false
    /// Probability that a drawn card lands reversed.
    var reversedChance = 0.3
    var selectedDeckId = "rws"
    var ownedDeckIds: [String] = []

    init(language: Language) {
        self.language = language
    }

    /// Tolerant decoding: a field added in a later version, or a malformed
    /// value, falls back to its default instead of discarding everything.
    init(from decoder: any Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        language = (try? container.decode(Language.self, forKey: .language)) ?? .en
        languageExplicit = (try? container.decode(Bool.self, forKey: .languageExplicit)) ?? false
        acceptedDisclaimer = (try? container.decode(Bool.self, forKey: .acceptedDisclaimer)) ?? false
        disableAnimations = (try? container.decode(Bool.self, forKey: .disableAnimations)) ?? false
        reversedChance = (try? container.decode(Double.self, forKey: .reversedChance)) ?? 0.3
        selectedDeckId = (try? container.decode(String.self, forKey: .selectedDeckId)) ?? "rws"
        ownedDeckIds = (try? container.decode([String].self, forKey: .ownedDeckIds)) ?? []
    }
}

@MainActor
@Observable
final class SettingsStore {
    static let storageKey = "tarot.settings"

    private(set) var settings: AppSettings
    /// Rebuilt only when the language changes, since loading reads JSON files.
    private(set) var localizer: Localizer
    /// The card texts in the current language; rebuilt with the localizer.
    private(set) var catalog: CardCatalog

    @ObservationIgnored private let defaults: UserDefaults
    @ObservationIgnored private let bundle: Bundle

    init(
        defaults: UserDefaults = .standard,
        deviceLanguage: Language = Language.resolve(),
        bundle: Bundle = .main
    ) {
        self.defaults = defaults
        self.bundle = bundle
        let loaded = Self.load(from: defaults, deviceLanguage: deviceLanguage)
        settings = loaded
        localizer = Localizer(language: loaded.language, bundle: bundle)
        catalog = CardCatalog.load(loaded.language, bundle: bundle)
    }

    func acceptDisclaimer() {
        update { $0.acceptedDisclaimer = true }
    }

    /// An explicit choice, which from now on wins over the device language.
    func setLanguage(_ language: Language) {
        update {
            $0.language = language
            $0.languageExplicit = true
        }
    }

    func update(_ change: (inout AppSettings) -> Void) {
        var next = settings
        change(&next)
        guard next != settings else { return }
        let languageChanged = next.language != settings.language
        settings = next
        if languageChanged {
            localizer = Localizer(language: next.language, bundle: bundle)
            catalog = CardCatalog.load(next.language, bundle: bundle)
        }
        save()
    }

    private func save() {
        guard let data = try? JSONEncoder().encode(settings) else { return }
        defaults.set(data, forKey: Self.storageKey)
    }

    /// Unless the user chose a language explicitly, the device language wins —
    /// the same rule as `parseSettings` in the React Native client.
    static func load(from defaults: UserDefaults, deviceLanguage: Language) -> AppSettings {
        guard let data = defaults.data(forKey: storageKey),
              var stored = try? JSONDecoder().decode(AppSettings.self, from: data) else {
            return AppSettings(language: deviceLanguage)
        }
        if !stored.languageExplicit {
            stored.language = deviceLanguage
        }
        return stored
    }
}

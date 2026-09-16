import SwiftUI

/// The API wrappers screens need, built once around the app's single client.
struct AppServices: Sendable {
    let usage: UsageAPI
    let interpretations: InterpretationAPI
    let dailyCards: DailyCardGate

    init(client: APIClient, defaults: UserDefaults = .standard) {
        usage = UsageAPI(client: client)
        interpretations = InterpretationAPI(client: client)
        dailyCards = DailyCardGate(usage: usage, ledger: LocalDailyLedger(defaults: defaults))
    }
}

private struct AppServicesKey: EnvironmentKey {
    /// Previews only: talks to an unreachable host with no session.
    static let defaultValue = AppServices(
        client: APIClient(config: APIConfig(baseURL: URL(string: "https://preview.invalid")!),
                          tokenStore: InMemoryTokenStore())
    )
}

extension EnvironmentValues {
    var services: AppServices {
        get { self[AppServicesKey.self] }
        set { self[AppServicesKey.self] = newValue }
    }
}

/// The main tabs, so any screen can switch tab — e.g. "View all" on the home
/// screen opens History.
enum AppTab: Hashable {
    case home, arcana, decks, history, settings
}

private struct SelectTabKey: EnvironmentKey {
    static let defaultValue: @MainActor @Sendable (AppTab) -> Void = { _ in }
}

extension EnvironmentValues {
    var selectTab: @MainActor @Sendable (AppTab) -> Void {
        get { self[SelectTabKey.self] }
        set { self[SelectTabKey.self] = newValue }
    }
}

private struct ShowPremiumKey: EnvironmentKey {
    static let defaultValue: @MainActor @Sendable () -> Void = {}
}

extension EnvironmentValues {
    /// Opens the premium plans from anywhere in the signed-in app.
    var showPremium: @MainActor @Sendable () -> Void {
        get { self[ShowPremiumKey.self] }
        set { self[ShowPremiumKey.self] = newValue }
    }
}

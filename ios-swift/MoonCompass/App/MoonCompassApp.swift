import SwiftUI

@main
struct MoonCompassApp: App {
    @State private var settings: SettingsStore
    @State private var session: SessionStore
    @State private var history: HistoryStore
    private let services: AppServices

    init() {
        let client = APIClient(tokenStore: KeychainTokenStore())
        _settings = State(initialValue: SettingsStore())
        _session = State(initialValue: SessionStore(client: client))
        _history = State(initialValue: HistoryStore())
        services = AppServices(client: client)
    }

    var body: some Scene {
        WindowGroup {
            RootView()
                .environment(settings)
                .environment(session)
                .environment(history)
                .environment(\.services, services)
                .environment(\.appColors, .classic)
        }
    }
}

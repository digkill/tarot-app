import SwiftUI

@main
struct MoonCompassApp: App {
    @State private var settings: SettingsStore
    @State private var session: SessionStore
    @State private var history: HistoryStore
    @State private var decks: DeckStore
    @State private var deepLinks = DeepLinks()
    private let services: AppServices

    init() {
        let client = APIClient(tokenStore: KeychainTokenStore())
        _settings = State(initialValue: SettingsStore())
        _session = State(initialValue: SessionStore(client: client))
        _history = State(initialValue: HistoryStore())
        _decks = State(initialValue: DeckStore(shop: ShopAPI(client: client), config: client.config))
        services = AppServices(client: client)
    }

    var body: some Scene {
        WindowGroup {
            RootView()
                .environment(settings)
                .environment(session)
                .environment(history)
                .environment(decks)
                .environment(deepLinks)
                .environment(\.services, services)
                .onOpenURL { deepLinks.handle($0) }
        }
    }
}

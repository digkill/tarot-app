import SwiftUI

@main
struct MoonCompassApp: App {
    @State private var settings: SettingsStore
    @State private var session: SessionStore
    @State private var history: HistoryStore
    @State private var decks: DeckStore
    @State private var purchases: PurchaseStore
    @State private var arcana: ArcanaStore
    @State private var deepLinks = DeepLinks()
    private let services: AppServices

    init() {
        let client = APIClient(tokenStore: KeychainTokenStore())
        _settings = State(initialValue: SettingsStore())
        _session = State(initialValue: SessionStore(client: client))
        _history = State(initialValue: HistoryStore())
        _decks = State(initialValue: DeckStore(shop: ShopAPI(client: client), config: client.config))
        let purchases = PurchaseStore(verifier: BillingAPI(client: client))
        // From launch, not from a screen: renewals and Ask to Buy approvals
        // can arrive at any moment.
        purchases.startListening()
        _purchases = State(initialValue: purchases)
        _arcana = State(initialValue: ArcanaStore(api: ArcanaAPI(config: .current(api: client.config), client: client)))
        services = AppServices(client: client)
    }

    var body: some Scene {
        WindowGroup {
            RootView()
                .environment(settings)
                .environment(session)
                .environment(history)
                .environment(decks)
                .environment(purchases)
                .environment(arcana)
                .environment(deepLinks)
                .environment(\.services, services)
                .onOpenURL { deepLinks.handle($0) }
        }
    }
}

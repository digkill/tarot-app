import SwiftUI

/// The launch gate, built by swapping whole view trees rather than navigating:
/// the disclaimer until it is accepted, then the account flow until someone is
/// signed in, then the app — the same order as the React Native client.
struct RootView: View {
    @Environment(SettingsStore.self) private var settings
    @Environment(SessionStore.self) private var session
    @Environment(DeckStore.self) private var decks
    @Environment(PurchaseStore.self) private var purchases

    /// A deck is also a palette: selecting one re-skins the whole app.
    private var colors: AppColors {
        decks.activeDeck(selectedSlug: settings.settings.selectedDeckId,
                         locallyOwned: settings.settings.ownedDeckIds).colors
    }

    var body: some View {
        ZStack {
            AppBackground()
            content
        }
        .environment(\.appColors, colors)
        .animation(.easeInOut(duration: 0.3), value: colors)
        .preferredColorScheme(.dark)
        .environment(\.locale, Locale(identifier: settings.settings.language.rawValue))
        .task {
            await session.start()
        }
        .task(id: session.user?.id) {
            purchases.userId = session.user?.id
            // The catalog is public; ownership needs a session.
            await decks.refresh(signedIn: session.user != nil)
            // Binds App Store purchases to the account and picks up anything
            // bought while the server could not be reached.
            await purchases.syncEntitlements()
        }
        .onChange(of: purchases.entitlementsVersion) {
            Task {
                await session.refreshUser()
                await decks.refresh(signedIn: session.user != nil)
            }
        }
    }

    @ViewBuilder
    private var content: some View {
        if !settings.settings.acceptedDisclaimer {
            DisclaimerView()
                .transition(.opacity)
        } else {
            switch session.state {
            case .restoring:
                ProgressView()
                    .tint(colors.gold)
            case .signedOut:
                AuthFlowView()
                    .transition(.opacity)
            case .signedIn:
                MainTabView()
                    .transition(.opacity)
            }
        }
    }
}

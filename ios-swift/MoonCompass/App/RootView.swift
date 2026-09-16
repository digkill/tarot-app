import SwiftUI

/// The launch gate, built by swapping whole view trees rather than navigating:
/// the disclaimer until it is accepted, then the account flow until someone is
/// signed in, then the app — the same order as the React Native client.
struct RootView: View {
    @Environment(SettingsStore.self) private var settings
    @Environment(SessionStore.self) private var session
    @Environment(\.appColors) private var colors

    var body: some View {
        ZStack {
            AppBackground()
            content
        }
        .preferredColorScheme(.dark)
        .environment(\.locale, Locale(identifier: settings.settings.language.rawValue))
        .task {
            await session.start()
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

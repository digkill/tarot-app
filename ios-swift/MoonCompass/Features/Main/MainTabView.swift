import SwiftUI

/// The signed-in app.
struct MainTabView: View {
    @Environment(SettingsStore.self) private var settings
    @Environment(DeepLinks.self) private var deepLinks
    @Environment(\.appColors) private var colors

    @State private var tab: AppTab = .home

    var body: some View {
        let l = settings.localizer
        TabView(selection: $tab) {
            HomeView()
                .tabItem { Label(l.t("nav.home"), systemImage: "house") }
                .tag(AppTab.home)
            DecksView()
                .tabItem { Label(l.t("nav.decks"), systemImage: "rectangle.stack") }
                .tag(AppTab.decks)
            HistoryView()
                .tabItem { Label(l.t("nav.history"), systemImage: "clock") }
                .tag(AppTab.history)
            SettingsView()
                .tabItem { Label(l.t("nav.settings"), systemImage: "gearshape") }
                .tag(AppTab.settings)
        }
        .tint(colors.accent)
        .toolbarBackground(colors.tabBar, for: .tabBar)
        .toolbarBackground(.visible, for: .tabBar)
        .toolbarColorScheme(.dark, for: .tabBar)
        .environment(\.selectTab) { tab = $0 }
        .onChange(of: deepLinks.deckSlug, initial: true) { _, slug in
            if slug != nil { tab = .decks }
        }
        #if DEBUG
        .onAppear { if let debugTab = DebugLaunch.tab { tab = debugTab } }
        #endif
    }
}

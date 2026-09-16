import SwiftUI

/// The signed-in app. The Decks tab is still being ported.
struct MainTabView: View {
    @Environment(SettingsStore.self) private var settings
    @Environment(\.appColors) private var colors

    @State private var tab: AppTab = .home

    var body: some View {
        let l = settings.localizer
        TabView(selection: $tab) {
            HomeView()
                .tabItem { Label(l.t("nav.home"), systemImage: "house") }
                .tag(AppTab.home)
            PendingTab(title: l.t("nav.decks"), systemImage: "rectangle.stack")
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
        #if DEBUG
        .onAppear { if let debugTab = DebugLaunch.tab { tab = debugTab } }
        #endif
    }
}

/// Stand-in for a tab whose screen has not been ported yet.
private struct PendingTab: View {
    @Environment(\.appColors) private var colors

    let title: String
    let systemImage: String

    var body: some View {
        VStack(spacing: 14) {
            Image(systemName: systemImage)
                .font(.system(size: 44, weight: .light))
                .foregroundStyle(colors.gold)
            Text(title)
                .font(.title2.weight(.semibold))
                .foregroundStyle(colors.text)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .background(AppBackground())
    }
}

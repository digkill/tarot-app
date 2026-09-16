import SwiftUI

struct SpreadCatalogView: View {
    @Environment(SettingsStore.self) private var settings
    @Environment(SessionStore.self) private var session
    @Environment(HistoryStore.self) private var history
    @Environment(\.services) private var services
    @Environment(\.appColors) private var colors
    @Environment(\.selectTab) private var selectTab

    @Binding var path: [ReadingRoute]

    @State private var daily = DailyDrawFlow()
    @State private var lockedSpread: Spread?

    private var hasPremium: Bool { session.user?.hasPremium ?? false }

    /// Grouped by category in first-appearance order, fewest cards first.
    private var sections: [(category: SpreadCategory, spreads: [Spread])] {
        var order: [SpreadCategory] = []
        var grouped: [SpreadCategory: [Spread]] = [:]
        for spread in Spread.all {
            if grouped[spread.category] == nil { order.append(spread.category) }
            grouped[spread.category, default: []].append(spread)
        }
        return order.map { ($0, grouped[$0, default: []].sorted { $0.cardCount < $1.cardCount }) }
    }

    var body: some View {
        let l = settings.localizer
        ScrollView {
            LazyVStack(alignment: .leading, spacing: 12) {
                ForEach(sections, id: \.category) { section in
                    Text(l.t(section.category.titleKey))
                        .font(.headline)
                        .foregroundStyle(colors.gold)
                        .padding(.top, 12)
                    ForEach(section.spreads) { spread in
                        row(spread, l)
                    }
                }
            }
            .padding(20)
        }
        .navigationTitle(l.t("nav.spreads"))
        .navigationBarTitleDisplayMode(.inline)
        .toolbarBackground(colors.bg, for: .navigationBar)
        .toolbarColorScheme(.dark, for: .navigationBar)
        .sheet(isPresented: $daily.showingLimit) { DailyLimitSheet(path: $path) }
        .alert(l.t("dailyCard.unlockError"), isPresented: $daily.showingError) {
            Button("OK", role: .cancel) {}
        }
        .alert(l.t("premium.lockedTitle"), isPresented: Binding(
            get: { lockedSpread != nil }, set: { if !$0 { lockedSpread = nil } }
        )) {
            Button(l.t("premium.notNow"), role: .cancel) {}
            Button(l.t("premium.openSettings")) { selectTab(.settings) }
        } message: {
            Text(l.t("premium.lockedDescription"))
        }
    }

    private func row(_ spread: Spread, _ l: Localizer) -> some View {
        Button {
            open(spread)
        } label: {
            VStack(alignment: .leading, spacing: 8) {
                HStack(alignment: .top) {
                    Text(l.t(spread.nameKey))
                        .font(.headline)
                        .foregroundStyle(colors.text)
                    Spacer()
                    if spread.premium {
                        Text(l.t("spread.premium"))
                            .font(.caption.weight(.bold))
                            .foregroundStyle(.white)
                            .padding(.horizontal, 8)
                            .padding(.vertical, 3)
                            .background(colors.accent, in: Capsule())
                    }
                }
                Text(l.t(spread.descriptionKey))
                    .font(.subheadline)
                    .foregroundStyle(colors.text.opacity(0.85))
                    .multilineTextAlignment(.leading)
                    .wrapsText()
                HStack {
                    Text(l.t("spread.cards", ["count": spread.cardCount]))
                    Spacer()
                    Text(l.t(spread.category.titleKey))
                }
                .font(.caption.weight(.semibold))
                .foregroundStyle(colors.gold)
            }
            .padding(16)
            .frame(maxWidth: .infinity, alignment: .leading)
            .background(spread.premium ? colors.accent.opacity(0.12) : colors.panel,
                        in: RoundedRectangle(cornerRadius: 18, style: .continuous))
            .overlay(RoundedRectangle(cornerRadius: 18, style: .continuous)
                .stroke(spread.premium ? colors.accent.opacity(0.5) : colors.gold.opacity(0.25)))
        }
        .buttonStyle(.plain)
        .disabled(daily.busy)
    }

    private func open(_ spread: Spread) {
        if spread.premium, !hasPremium {
            lockedSpread = spread
            return
        }
        if spread.isOneCard {
            Task {
                await daily.start(
                    preferExisting: true, path: $path, history: history, services: services,
                    hasPremium: hasPremium, deckId: settings.settings.selectedDeckId
                )
            }
            return
        }
        path.append(.reading(spreadId: spread.id, deckId: settings.settings.selectedDeckId, dailyKind: nil))
    }
}

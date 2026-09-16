import SwiftUI

struct HomeView: View {
    @Environment(SettingsStore.self) private var settings
    @Environment(SessionStore.self) private var session
    @Environment(HistoryStore.self) private var history
    @Environment(\.services) private var services
    @Environment(\.appColors) private var colors
    @Environment(\.selectTab) private var selectTab

    @State private var path: [ReadingRoute] = []
    @State private var daily = DailyDrawFlow()

    var body: some View {
        let l = settings.localizer
        NavigationStack(path: $path) {
            ScrollView {
                VStack(alignment: .leading, spacing: 28) {
                    hero(l)
                    quickSpreads(l)
                    if let recent = history.readings.first {
                        recentReading(recent, l)
                    }
                    education(l)
                }
                .padding(.horizontal, 20)
                .padding(.bottom, 32)
            }
            .background(AppBackground())
            .toolbar(.hidden, for: .navigationBar)
            .readingDestinations(path: $path)
        }
        .tint(colors.gold)
        #if DEBUG
        .onAppear(perform: openDebugRoute)
        #endif
        .sheet(isPresented: $daily.showingLimit) { DailyLimitSheet(path: $path) }
        .alert(l.t("dailyCard.unlockError"), isPresented: $daily.showingError) {
            Button("OK", role: .cancel) {}
        }
    }

    #if DEBUG
    private func openDebugRoute() {
        guard path.isEmpty else { return }
        let deckId = settings.settings.selectedDeckId
        if DebugLaunch.catalog {
            path = [.catalog]
        } else if let spreadId = DebugLaunch.readingSpread {
            path = [.reading(spreadId: spreadId, deckId: deckId, dailyKind: .daily)]
        } else if let spreadId = DebugLaunch.interpretationSpread, let spread = Spread.find(spreadId) {
            let cards = settings.catalog.cards.shuffled()
            let entries = zip(spread.positions, cards).enumerated().map { index, pair in
                DrawnCard(card: pair.1, position: pair.0, isReversed: index % 3 == 1)
            }
            let reading = history.add(Reading(
                spreadId: spread.id, deckId: deckId,
                items: entries.map { ReadingItem(positionIndex: $0.position.index, cardId: $0.card.id, isReversed: $0.isReversed) },
                summaryText: Interpretation.summary(spread: spread, entries: entries, localizer: settings.localizer)
            ))
            path = [.interpretation(readingId: reading.id)]
        }
    }
    #endif

    private func startDaily(preferExisting: Bool) {
        Task {
            await daily.start(
                preferExisting: preferExisting, path: $path, history: history, services: services,
                hasPremium: session.user?.hasPremium ?? false, deckId: settings.settings.selectedDeckId
            )
        }
    }

    private var greetingKey: String {
        let hour = Calendar.current.component(.hour, from: Date())
        if hour < 12 { return "home.greeting.morning" }
        if hour < 18 { return "home.greeting.afternoon" }
        return "home.greeting.evening"
    }

    private func hero(_ l: Localizer) -> some View {
        let today = history.todaysDailyCard()
        return VStack(alignment: .leading, spacing: 10) {
            Text(l.t(greetingKey))
                .font(.subheadline.weight(.medium))
                .foregroundStyle(colors.text)
            Text(l.t("home.hero.title"))
                .font(.system(size: 26, weight: .bold))
                .foregroundStyle(colors.gold)
                .wrapsText()
            Text(l.t("home.hero.subtitle"))
                .font(.subheadline)
                .foregroundStyle(colors.text.opacity(0.9))
                .wrapsText()
            PrimaryButton(title: l.t(today == nil ? "home.hero.cta" : "dailyCard.ctaOpen"),
                          isLoading: daily.busy) {
                startDaily(preferExisting: true)
            }
            .padding(.top, 6)
            if today != nil {
                Button {
                    startDaily(preferExisting: false)
                } label: {
                    Text(l.t("home.hero.another"))
                        .font(.body.weight(.semibold))
                        .foregroundStyle(colors.gold)
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 14)
                        .overlay(RoundedRectangle(cornerRadius: 16, style: .continuous).stroke(colors.gold))
                }
                .buttonStyle(.plain)
                .disabled(daily.busy)
            }
        }
        .padding(20)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background {
            Image("HomeHero")
                .resizable()
                .scaledToFill()
                .overlay(Color.black.opacity(0.55))
        }
        .clipShape(RoundedRectangle(cornerRadius: 24, style: .continuous))
        .padding(.top, 12)
    }

    private func quickSpreads(_ l: Localizer) -> some View {
        VStack(alignment: .leading, spacing: 12) {
            sectionHeader(l.t("home.quickAccess.title"), link: l.t("home.quickAccess.more")) {
                path.append(.catalog)
            }
            HStack(spacing: 10) {
                ForEach(Spread.basicIds.compactMap(Spread.find)) { spread in
                    Button {
                        open(spread)
                    } label: {
                        VStack(alignment: .leading, spacing: 6) {
                            Text(l.t(spread.nameKey))
                                .font(.subheadline.weight(.bold))
                                .foregroundStyle(colors.gold)
                                .lineLimit(2)
                                .minimumScaleFactor(0.85)
                            Text(l.t("home.quickAccess.cardCount", ["count": spread.cardCount]))
                                .font(.caption)
                                .foregroundStyle(colors.text)
                        }
                        .padding(12)
                        .frame(maxWidth: .infinity, minHeight: 88, alignment: .topLeading)
                        .background(colors.accent.opacity(0.12), in: RoundedRectangle(cornerRadius: 16, style: .continuous))
                        .overlay(RoundedRectangle(cornerRadius: 16, style: .continuous).stroke(colors.accent.opacity(0.35)))
                    }
                    .buttonStyle(.plain)
                }
            }
        }
    }

    private func open(_ spread: Spread) {
        if spread.isOneCard {
            startDaily(preferExisting: true)
        } else {
            path.append(.reading(spreadId: spread.id, deckId: settings.settings.selectedDeckId, dailyKind: nil))
        }
    }

    private func recentReading(_ reading: Reading, _ l: Localizer) -> some View {
        VStack(alignment: .leading, spacing: 12) {
            sectionHeader(l.t("home.recent.title"), link: l.t("home.recent.viewAll")) {
                selectTab(.history)
            }
            Button {
                path.append(.interpretation(readingId: reading.id))
            } label: {
                VStack(alignment: .leading, spacing: 6) {
                    Text(l.t("home.recent.spread")).font(.caption).foregroundStyle(colors.muted)
                    Text(Spread.find(reading.spreadId).map { l.t($0.nameKey) } ?? reading.spreadId)
                        .font(.subheadline.weight(.semibold))
                        .foregroundStyle(colors.gold)
                    Text(l.t("home.recent.date")).font(.caption).foregroundStyle(colors.muted)
                    Text(reading.drawnAt.formatted(
                        Date.FormatStyle(date: .abbreviated, time: .shortened).locale(Locale(identifier: l.language.rawValue))
                    ))
                    .font(.subheadline.weight(.semibold))
                    .foregroundStyle(colors.gold)
                    Text(Interpretation.displaySummary(for: reading, catalog: settings.catalog, localizer: l))
                        .font(.subheadline)
                        .foregroundStyle(colors.text)
                        .lineLimit(3)
                        .padding(.top, 4)
                }
                .padding(16)
                .frame(maxWidth: .infinity, alignment: .leading)
                .background(colors.panel, in: RoundedRectangle(cornerRadius: 18, style: .continuous))
                .overlay(RoundedRectangle(cornerRadius: 18, style: .continuous).stroke(colors.gold.opacity(0.3)))
            }
            .buttonStyle(.plain)
        }
    }

    private func education(_ l: Localizer) -> some View {
        VStack(alignment: .leading, spacing: 12) {
            Text(l.t("home.education.title"))
                .font(.title3.weight(.bold))
                .foregroundStyle(colors.text)
            VStack(alignment: .leading, spacing: 14) {
                Text(l.t("home.education.body"))
                    .foregroundStyle(colors.text)
                    .wrapsText()
                Button {
                    selectTab(.decks)
                } label: {
                    Text(l.t("home.education.cta"))
                        .font(.body.weight(.semibold))
                        .foregroundStyle(colors.bg)
                        .padding(.horizontal, 18)
                        .padding(.vertical, 10)
                        .background(colors.gold, in: Capsule())
                }
                .buttonStyle(.plain)
            }
            .padding(16)
            .frame(maxWidth: .infinity, alignment: .leading)
            .background(colors.accent.opacity(0.12), in: RoundedRectangle(cornerRadius: 18, style: .continuous))
            .overlay(RoundedRectangle(cornerRadius: 18, style: .continuous).stroke(colors.accent.opacity(0.35)))
        }
    }

    private func sectionHeader(_ title: String, link: String, action: @escaping () -> Void) -> some View {
        HStack {
            Text(title)
                .font(.title3.weight(.bold))
                .foregroundStyle(colors.text)
            Spacer()
            Button(link, action: action)
                .font(.subheadline.weight(.semibold))
                .foregroundStyle(colors.accent)
        }
    }
}

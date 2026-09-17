import SwiftUI

/// The Arcana Clash tab: pick a hero, battle, and the reference screens.
struct ArcanaHomeView: View {
    @Environment(SettingsStore.self) private var settings
    @Environment(ArcanaStore.self) private var arcana
    @Environment(\.appColors) private var colors
    @Environment(\.scenePhase) private var scenePhase

    var body: some View {
        let l = settings.localizer
        @Bindable var arcana = arcana
        NavigationStack {
            ScrollView {
                VStack(alignment: .leading, spacing: 20) {
                    header(l)
                    if arcana.resumableMatch != nil {
                        resumeBanner(l)
                    }
                    heroPicker(l)
                    PrimaryButton(title: l.t("arcana.battle"), isEnabled: arcana.catalog != nil) {
                        arcana.battle()
                    }
                    if arcana.catalogFailed, arcana.catalog == nil {
                        StatusMessage(text: l.t("arcana.connectionFailed"))
                        LinkButton(title: l.t("arcana.retry")) { Task { await arcana.loadCatalog() } }
                    }
                    links(l)
                }
                .padding(20)
            }
            .background(AppBackground())
            .navigationTitle(l.t("arcana.title"))
            .toolbarBackground(colors.bg, for: .navigationBar)
            .toolbarColorScheme(.dark, for: .navigationBar)
            .navigationDestination(for: ArcanaPage.self) { page in
                switch page {
                case .rules: ArcanaRulesView()
                case .heroes: ArcanaHeroesView()
                case .history: ArcanaHistoryView()
                }
            }
        }
        .task { await arcana.loadCatalog() }
        .onAppear {
            arcana.enter()
            Task { await arcana.loadCatalog() }
        }
        .onDisappear { arcana.leave() }
        .onChange(of: scenePhase) { _, phase in
            if phase == .active { arcana.appDidBecomeActive() }
        }
        .fullScreenCover(isPresented: Binding(
            get: { arcana.stage != .lobby },
            set: { if !$0 { arcana.closeResult() } }
        )) {
            ArcanaMatchView()
        }
        #if DEBUG
        .task {
            if let hero = DebugLaunch.arcanaBattle {
                await arcana.loadCatalog()
                arcana.hero = hero == "random" ? nil : hero
                arcana.battle()
            }
        }
        #endif
    }

    private func header(_ l: Localizer) -> some View {
        HStack(alignment: .top, spacing: 14) {
            Image(systemName: "bolt.shield.fill")
                .font(.system(size: 36))
                .foregroundStyle(colors.gold)
            Text(l.t("arcana.tagline"))
                .foregroundStyle(colors.text)
                .wrapsText()
        }
    }

    private func resumeBanner(_ l: Localizer) -> some View {
        Button {
            arcana.resume()
        } label: {
            Label(l.t("arcana.resume"), systemImage: "arrow.uturn.forward.circle.fill")
                .font(.headline)
                .foregroundStyle(colors.bg)
                .frame(maxWidth: .infinity)
                .padding(.vertical, 14)
                .background(colors.gold, in: RoundedRectangle(cornerRadius: 16, style: .continuous))
        }
        .buttonStyle(.plain)
    }

    private func heroPicker(_ l: Localizer) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            Text(l.t("arcana.chooseHero"))
                .font(.headline)
                .foregroundStyle(colors.gold)
            ScrollView(.horizontal, showsIndicators: false) {
                HStack(spacing: 12) {
                    heroChip(id: nil, title: l.t("arcana.randomHero"), cardId: nil, hp: nil)
                    ForEach(arcana.catalog?.heroes ?? []) { hero in
                        heroChip(id: hero.id, title: heroName(hero), cardId: hero.cardId, hp: hero.hp)
                    }
                }
                .padding(.vertical, 4)
            }
            .scrollClipDisabled()
        }
    }

    private func heroChip(id: String?, title: String, cardId: String?, hp: Int?) -> some View {
        let selected = arcana.hero == id
        return Button {
            arcana.hero = id
        } label: {
            VStack(spacing: 6) {
                ArcanaCardArt(cardId: cardId, width: 84)
                    .clipShape(RoundedRectangle(cornerRadius: 8, style: .continuous))
                Text(title)
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(selected ? colors.gold : colors.text)
                    .lineLimit(1)
                    .minimumScaleFactor(0.7)
                if let hp {
                    Label("\(hp)", systemImage: "heart.fill")
                        .font(.caption2)
                        .foregroundStyle(colors.muted)
                }
            }
            .frame(width: 96)
            .padding(8)
            .background(selected ? colors.accent.opacity(0.25) : colors.panel,
                        in: RoundedRectangle(cornerRadius: 14, style: .continuous))
            .overlay(RoundedRectangle(cornerRadius: 14, style: .continuous)
                .stroke(selected ? colors.accent : colors.gold.opacity(0.2), lineWidth: selected ? 2 : 1))
        }
        .buttonStyle(.plain)
        .accessibilityAddTraits(selected ? .isSelected : [])
    }

    private func links(_ l: Localizer) -> some View {
        VStack(spacing: 10) {
            ForEach([ArcanaPage.rules, .heroes, .history], id: \.self) { page in
                NavigationLink(value: page) {
                    HStack {
                        Image(systemName: page.icon).frame(width: 28)
                        Text(l.t(page.titleKey))
                        Spacer()
                        Image(systemName: "chevron.right").foregroundStyle(colors.muted)
                    }
                    .font(.body.weight(.semibold))
                    .foregroundStyle(colors.text)
                    .padding(16)
                    .background(colors.panel, in: RoundedRectangle(cornerRadius: 14, style: .continuous))
                }
            }
        }
    }

    private func heroName(_ hero: ArcanaHeroDef) -> String {
        settings.catalog.card(id: hero.cardId)?.name ?? hero.id
    }
}

enum ArcanaPage: Hashable {
    case rules, heroes, history

    var titleKey: String {
        switch self {
        case .rules: "arcana.rules"
        case .heroes: "arcana.heroes"
        case .history: "arcana.history"
        }
    }

    var icon: String {
        switch self {
        case .rules: "book.closed"
        case .heroes: "person.3"
        case .history: "clock.arrow.circlepath"
        }
    }
}

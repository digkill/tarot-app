import SwiftUI

struct ArcanaRulesView: View {
    @Environment(SettingsStore.self) private var settings
    @Environment(ArcanaStore.self) private var arcana
    @Environment(\.appColors) private var colors

    var body: some View {
        let l = settings.localizer
        let rules = arcana.catalog?.rules
        ScrollView {
            VStack(alignment: .leading, spacing: 14) {
                ForEach(Array(l.list("arcana.rulesText").enumerated()), id: \.offset) { _, paragraph in
                    Text(Localizer.interpolate(paragraph, [
                        "turn": rules?.turnSeconds ?? 40, "reconnect": rules?.reconnectSeconds ?? 45,
                    ]))
                    .foregroundStyle(colors.text)
                    .wrapsText()
                }
                Text(l.t("arcana.statusTitle"))
                    .font(.headline)
                    .foregroundStyle(colors.gold)
                    .padding(.top, 8)
                ForEach(ArcanaStatusChip.order, id: \.self) { name in
                    VStack(alignment: .leading, spacing: 2) {
                        Label(l.t("arcana.status.\(name)"), systemImage: ArcanaStatusChip.icon(name))
                            .font(.subheadline.weight(.semibold))
                            .foregroundStyle(colors.gold)
                        Text(l.t("arcana.statusHint.\(name)"))
                            .font(.subheadline)
                            .foregroundStyle(colors.text)
                            .wrapsText()
                    }
                }
            }
            .padding(20)
        }
        .background(AppBackground())
        .navigationTitle(l.t("arcana.rules"))
        .navigationBarTitleDisplayMode(.inline)
    }
}

struct ArcanaHeroesView: View {
    @Environment(SettingsStore.self) private var settings
    @Environment(ArcanaStore.self) private var arcana
    @Environment(\.appColors) private var colors

    var body: some View {
        let l = settings.localizer
        ScrollView {
            VStack(spacing: 16) {
                ForEach(arcana.catalog?.heroes ?? []) { hero in
                    HStack(alignment: .top, spacing: 14) {
                        CardImage(file: settings.catalog.card(id: hero.cardId)?.imageFile ?? CardArt.backFile, width: 90)
                            .clipShape(RoundedRectangle(cornerRadius: 8, style: .continuous))
                        VStack(alignment: .leading, spacing: 8) {
                            Text(settings.catalog.card(id: hero.cardId)?.name ?? hero.id)
                                .font(.headline)
                                .foregroundStyle(colors.gold)
                            Label(l.t("arcana.hp", ["count": hero.hp]), systemImage: "heart.fill")
                                .font(.caption.weight(.semibold))
                                .foregroundStyle(colors.danger)
                            power(l.t("arcana.passive"), l.t("arcana.heroText.\(hero.id).passive"))
                            power(l.t("arcana.ability") + " · " + l.t("arcana.cost", ["cost": hero.ability.cost]),
                                  l.t("arcana.heroText.\(hero.id).ability"))
                            power(l.t("arcana.ultimate"), l.t("arcana.heroText.\(hero.id).ultimate"))
                        }
                    }
                    .padding(14)
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .background(colors.panel, in: RoundedRectangle(cornerRadius: 16, style: .continuous))
                }
            }
            .padding(20)
        }
        .background(AppBackground())
        .navigationTitle(l.t("arcana.heroes"))
        .navigationBarTitleDisplayMode(.inline)
    }

    private func power(_ title: String, _ text: String) -> some View {
        VStack(alignment: .leading, spacing: 2) {
            Text(title).font(.caption.weight(.bold)).foregroundStyle(colors.accent)
            Text(text).font(.subheadline).foregroundStyle(colors.text).wrapsText()
        }
    }
}

struct ArcanaHistoryView: View {
    @Environment(SettingsStore.self) private var settings
    @Environment(ArcanaStore.self) private var arcana
    @Environment(\.appColors) private var colors

    @State private var matches: [ArcanaMatchSummary]?
    @State private var failed = false

    var body: some View {
        let l = settings.localizer
        ScrollView {
            VStack(spacing: 12) {
                if failed {
                    StatusMessage(text: l.t("arcana.historyError"))
                    LinkButton(title: l.t("arcana.retry")) { Task { await load() } }
                } else if let matches {
                    if matches.isEmpty {
                        Text(l.t("arcana.historyEmpty"))
                            .foregroundStyle(colors.muted)
                            .multilineTextAlignment(.center)
                            .padding(.top, 40)
                    }
                    ForEach(matches) { row($0, l) }
                } else {
                    ProgressView().tint(colors.accent).padding(.top, 40)
                }
            }
            .padding(20)
        }
        .background(AppBackground())
        .navigationTitle(l.t("arcana.history"))
        .navigationBarTitleDisplayMode(.inline)
        .task { await load() }
        .refreshable { await load() }
    }

    private func load() async {
        do {
            matches = try await arcana.matchHistory()
            failed = false
        } catch {
            failed = true
        }
    }

    private func row(_ m: ArcanaMatchSummary, _ l: Localizer) -> some View {
        let color: Color = m.result == "win" ? colors.gold : (m.result == "loss" ? colors.danger : colors.muted)
        return VStack(alignment: .leading, spacing: 6) {
            HStack {
                Text(l.t("arcana.result.\(m.result)"))
                    .font(.headline)
                    .foregroundStyle(color)
                Spacer()
                Text(m.finishedAt.formatted(.dateTime.day().month().hour().minute()
                    .locale(Locale(identifier: settings.settings.language.rawValue))))
                    .font(.caption)
                    .foregroundStyle(colors.muted)
            }
            Text("\(heroName(m.hero)) — \(heroName(m.opponentHero))")
                .foregroundStyle(colors.text)
            Text([l.t("arcana.reason.\(m.reason)"), l.t("arcana.turns", ["count": m.turns]),
                  l.t("arcana.minutes", ["count": max(1, Int((Double(m.durationMs) / 60000).rounded()))])]
                .joined(separator: " · "))
                .font(.caption)
                .foregroundStyle(colors.muted)
                .wrapsText()
        }
        .padding(14)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(colors.panel, in: RoundedRectangle(cornerRadius: 14, style: .continuous))
    }

    private func heroName(_ id: String) -> String {
        let cardId = id == "strength" ? "the_strength" : id
        return settings.catalog.card(id: cardId)?.name ?? id
    }
}

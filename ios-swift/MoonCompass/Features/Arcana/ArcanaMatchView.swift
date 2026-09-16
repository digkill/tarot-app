import SwiftUI

/// Everything after Battle: searching, the battlefield, the result.
struct ArcanaMatchView: View {
    @Environment(SettingsStore.self) private var settings
    @Environment(ArcanaStore.self) private var arcana
    @Environment(\.appColors) private var colors

    var body: some View {
        ZStack {
            AppBackground()
            switch arcana.stage {
            case .searching:
                searching
            case .match, .finished:
                if arcana.view != nil {
                    ArcanaBattlefieldView()
                } else {
                    connecting
                }
            case .lobby:
                EmptyView()
            }
        }
        .preferredColorScheme(.dark)
        .environment(\.appColors, colors)
    }

    private var searching: some View {
        let l = settings.localizer
        return VStack(spacing: 24) {
            Spacer()
            ProgressView().controlSize(.large).tint(colors.gold)
            Text(arcana.connection == .online ? l.t("arcana.searching") : l.t("arcana.connecting"))
                .font(.title3.weight(.semibold))
                .foregroundStyle(colors.text)
            if arcana.lastError == "connection_failed" {
                StatusMessage(text: l.t("arcana.connectionFailed"))
                    .multilineTextAlignment(.center)
            }
            Spacer()
            LinkButton(title: l.t("arcana.cancelSearch")) { arcana.cancelSearch() }
                .padding(.bottom, 24)
        }
        .padding(24)
    }

    private var connecting: some View {
        let l = settings.localizer
        return VStack(spacing: 16) {
            ProgressView().tint(colors.gold)
            Text(l.t("arcana.connecting")).foregroundStyle(colors.text)
            LinkButton(title: l.t("arcana.close")) { arcana.closeResult() }
        }
    }
}

struct ArcanaBattlefieldView: View {
    @Environment(SettingsStore.self) private var settings
    @Environment(ArcanaStore.self) private var arcana
    @Environment(\.appColors) private var colors

    @State private var detail: ArcanaCardView?
    @State private var power: PowerChoice?
    @State private var mulliganPicks: Set<String> = []
    @State private var confirmingSurrender = false
    @State private var errorText: String?

    enum PowerChoice: String, Identifiable {
        case ability, ultimate
        var id: String { rawValue }
    }

    var body: some View {
        let l = settings.localizer
        if let v = arcana.view {
            VStack(spacing: 10) {
                opponentPanel(v, l)
                center(v, l)
                myPanel(v, l)
                hand(v, l)
                bottomBar(v, l)
            }
            .padding(.horizontal, 12)
            .padding(.top, 6)
            .overlay { overlays(v, l) }
            .sheet(item: $detail) { card in
                ArcanaCardSheet(card: card, canAct: v.isMyTurn) { target in
                    arcana.play(card.uid, target: target)
                }
                .presentationDetents([.large])
            }
            .sheet(item: $power) { choice in
                ArcanaPowerSheet(choice: choice, view: v) { target in
                    choice == .ability ? arcana.ability(target: target) : arcana.ultimate(target: target)
                }
                .presentationDetents([.medium, .large])
            }
            .confirmationDialog(l.t("arcana.surrenderConfirm"), isPresented: $confirmingSurrender, titleVisibility: .visible) {
                Button(l.t("arcana.surrender"), role: .destructive) { arcana.surrender() }
                Button(l.t("history.cancel"), role: .cancel) {}
            }
            .onChange(of: arcana.errorCount) {
                guard let code = arcana.lastError else { return }
                errorText = l.has("arcana.errors.\(code)") ? l.t("arcana.errors.\(code)") : l.t("arcana.errors.generic")
                Task {
                    try? await Task.sleep(for: .seconds(2.5))
                    errorText = nil
                }
            }
        }
    }

    // MARK: - Panels

    private func heroName(_ id: String) -> String {
        let cardId = id == "strength" ? "the_strength" : id
        return settings.catalog.card(id: cardId)?.name ?? id
    }

    private func heroPortrait(_ id: String, width: CGFloat) -> some View {
        let cardId = id == "strength" ? "the_strength" : id
        return CardImage(file: settings.catalog.card(id: cardId)?.imageFile ?? CardArt.backFile, width: width)
            .clipShape(RoundedRectangle(cornerRadius: 8, style: .continuous))
    }

    private func opponentPanel(_ v: ArcanaGameView, _ l: Localizer) -> some View {
        let o = v.opponent
        return HStack(alignment: .top, spacing: 10) {
            heroPortrait(o.hero.id, width: 52)
                .overlay(alignment: .bottomTrailing) {
                    if !arcana.opponentConnected {
                        Image(systemName: "wifi.slash")
                            .font(.caption.weight(.bold))
                            .foregroundStyle(.white)
                            .padding(4)
                            .background(colors.danger, in: Circle())
                    }
                }
            VStack(alignment: .leading, spacing: 5) {
                HStack {
                    Text(heroName(o.hero.id)).font(.subheadline.weight(.bold)).foregroundStyle(colors.gold)
                    Spacer()
                    Label("\(o.handCount)", systemImage: "rectangle.portrait.on.rectangle.portrait")
                    Label("\(o.deckCount)", systemImage: "square.stack.3d.up")
                    Label("\(o.mana)/\(o.maxMana)", systemImage: "drop.fill")
                }
                .font(.caption.weight(.semibold))
                .foregroundStyle(colors.text)
                ArcanaHealthBar(hero: o.hero)
                statuses(o.statuses)
            }
        }
        .padding(10)
        .background(colors.panel.opacity(0.9), in: RoundedRectangle(cornerRadius: 16, style: .continuous))
        .overlay(RoundedRectangle(cornerRadius: 16, style: .continuous)
            .stroke(!v.isMyTurn && v.phase == .playing ? colors.danger.opacity(0.7) : .clear, lineWidth: 2))
    }

    private func statuses(_ list: [ArcanaStatusView]) -> some View {
        ScrollView(.horizontal, showsIndicators: false) {
            HStack(spacing: 5) {
                ForEach(list, id: \.name) { ArcanaStatusChip(status: $0) }
            }
        }
        .frame(height: list.isEmpty ? 0 : 24)
    }

    private func center(_ v: ArcanaGameView, _ l: Localizer) -> some View {
        VStack(spacing: 8) {
            HStack {
                Text(v.phase == .mulligan ? l.t("arcana.mulliganTitle")
                     : (v.isMyTurn ? l.t("arcana.yourTurn") : l.t("arcana.opponentTurn")))
                    .font(.headline)
                    .foregroundStyle(v.isMyTurn ? colors.gold : colors.text)
                Spacer()
                if let deadline = arcana.deadline, v.phase != .finished {
                    TimelineView(.periodic(from: .now, by: 1)) { context in
                        let left = max(0, Int(deadline.timeIntervalSince(context.date).rounded(.up)))
                        Label("\(left)", systemImage: "timer")
                            .font(.headline.monospacedDigit())
                            .foregroundStyle(left <= 10 ? colors.danger : colors.text)
                    }
                }
            }
            VStack(alignment: .leading, spacing: 4) {
                ForEach(v.log.suffix(4).reversed()) { entry in
                    if let line = ArcanaText.logLine(entry, me: v.you.id, cardName: cardName, l) {
                        HStack(spacing: 6) {
                            Text(line)
                                .font(.caption)
                                .foregroundStyle(entry.player == v.you.id ? colors.gold : colors.text)
                                .lineLimit(1)
                            Spacer(minLength: 0)
                            Text(eventSummary(entry, me: v.you.id))
                                .font(.caption.weight(.bold).monospacedDigit())
                                .foregroundStyle(colors.muted)
                                .lineLimit(1)
                        }
                    }
                }
            }
            .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
            .padding(10)
            .background(colors.bg.opacity(0.55), in: RoundedRectangle(cornerRadius: 14, style: .continuous))
        }
        .frame(maxHeight: .infinity)
    }

    private func cardName(_ id: String) -> String {
        settings.catalog.card(id: id)?.name ?? id
    }

    /// "−5 ❤︎" style totals of what an action did to each side.
    private func eventSummary(_ entry: ArcanaLogEntry, me: String) -> String {
        var parts: [String] = []
        for ev in entry.events ?? [] {
            let side = ev.player == me ? "↓" : "↑"
            switch ev.kind {
            case "damage": if let n = ev.amount, n > 0 { parts.append("\(side)−\(n)\(ev.crit == true ? "!" : "")") }
            case "heal": if let n = ev.amount, n > 0 { parts.append("\(side)+\(n)") }
            default: break
            }
        }
        return parts.prefix(3).joined(separator: " ")
    }

    private func myPanel(_ v: ArcanaGameView, _ l: Localizer) -> some View {
        let me = v.you
        return HStack(alignment: .top, spacing: 10) {
            heroPortrait(me.hero.id, width: 52)
            VStack(alignment: .leading, spacing: 5) {
                HStack(spacing: 8) {
                    Text(heroName(me.hero.id)).font(.subheadline.weight(.bold)).foregroundStyle(colors.gold)
                    Spacer()
                    manaPips(me)
                }
                ArcanaHealthBar(hero: me.hero)
                statuses(me.statuses)
                HStack(spacing: 8) {
                    powerButton(title: l.t("arcana.ability") + " · \(me.abilityCost)", icon: "wand.and.stars",
                                enabled: me.abilityUsable) { power = .ability }
                    let charge = me.hero.id == "death"
                        ? "\(min(me.hero.souls, me.hero.ultimateCost))/\(me.hero.ultimateCost)"
                        : "\(min(me.hero.charge, me.hero.ultimateCost))/\(me.hero.ultimateCost)"
                    powerButton(title: l.t("arcana.ultimate") + " · " + charge, icon: "star.circle.fill",
                                enabled: me.ultimateUsable, highlight: me.hero.ultimateReady) { power = .ultimate }
                }
            }
        }
        .padding(10)
        .background(colors.panel.opacity(0.9), in: RoundedRectangle(cornerRadius: 16, style: .continuous))
        .overlay(RoundedRectangle(cornerRadius: 16, style: .continuous)
            .stroke(v.isMyTurn ? colors.gold.opacity(0.8) : .clear, lineWidth: 2))
    }

    private func manaPips(_ me: ArcanaPlayerView) -> some View {
        HStack(spacing: 3) {
            ForEach(0..<max(me.maxMana, me.mana), id: \.self) { i in
                Circle()
                    .fill(i < me.mana ? Color.cyan : colors.text.opacity(0.15))
                    .frame(width: 10, height: 10)
            }
            Text("\(me.mana)")
                .font(.caption.weight(.bold).monospacedDigit())
                .foregroundStyle(.cyan)
        }
        .accessibilityElement(children: .ignore)
        .accessibilityLabel(settings.localizer.t("arcana.mana") + " \(me.mana)/\(me.maxMana)")
    }

    private func powerButton(title: String, icon: String, enabled: Bool, highlight: Bool = false,
                             action: @escaping () -> Void) -> some View {
        Button(action: action) {
            Label(title, systemImage: icon)
                .font(.caption.weight(.bold))
                .lineLimit(1)
                .minimumScaleFactor(0.7)
                .foregroundStyle(highlight ? colors.bg : .white)
                .frame(maxWidth: .infinity)
                .padding(.vertical, 8)
                .background(highlight ? colors.gold : colors.accent, in: RoundedRectangle(cornerRadius: 10, style: .continuous))
                .opacity(enabled ? 1 : 0.4)
        }
        .buttonStyle(.plain)
        .disabled(!enabled)
    }

    private func hand(_ v: ArcanaGameView, _ l: Localizer) -> some View {
        ScrollView(.horizontal, showsIndicators: false) {
            HStack(spacing: 8) {
                ForEach(v.you.hand) { card in
                    Button {
                        detail = card
                    } label: {
                        ArcanaCardTile(card: card, width: 92, dimmed: v.isMyTurn && !card.isPlayable)
                    }
                    .buttonStyle(.plain)
                }
            }
            .padding(.horizontal, 4)
            .padding(.top, 8)
        }
        .frame(height: 190)
    }

    private func bottomBar(_ v: ArcanaGameView, _ l: Localizer) -> some View {
        HStack(spacing: 10) {
            Menu {
                ForEach(arcana.catalog?.emotes ?? [], id: \.self) { emote in
                    Button(l.t("arcana.emotes.\(emote)")) { arcana.emote(emote) }
                }
                Divider()
                Button(l.t("arcana.surrender"), role: .destructive) { confirmingSurrender = true }
            } label: {
                Image(systemName: "ellipsis.bubble.fill")
                    .font(.title3)
                    .foregroundStyle(colors.text)
                    .frame(width: 52, height: 48)
                    .background(colors.panel, in: RoundedRectangle(cornerRadius: 14, style: .continuous))
            }
            .accessibilityLabel(l.t("arcana.emote"))
            Button {
                arcana.endTurn()
            } label: {
                Text(l.t("arcana.endTurn"))
                    .font(.headline)
                    .foregroundStyle(v.isMyTurn ? colors.bg : colors.muted)
                    .frame(maxWidth: .infinity)
                    .frame(height: 48)
                    .background(v.isMyTurn ? colors.gold : colors.panel, in: RoundedRectangle(cornerRadius: 14, style: .continuous))
            }
            .buttonStyle(.plain)
            .disabled(!v.isMyTurn)
        }
        .padding(.bottom, 8)
    }

    // MARK: - Overlays

    @ViewBuilder
    private func overlays(_ v: ArcanaGameView, _ l: Localizer) -> some View {
        ZStack {
            VStack(spacing: 8) {
                if arcana.connection != .online && arcana.stage == .match {
                    banner(l.t("arcana.reconnecting"), icon: "wifi.exclamationmark")
                }
                if !arcana.opponentConnected && arcana.stage == .match {
                    banner(l.t("arcana.opponentDisconnected"), icon: "person.crop.circle.badge.exclamationmark")
                }
                if let errorText {
                    banner(errorText, icon: "exclamationmark.triangle.fill")
                }
                ForEach(arcana.emotes) { bubble in
                    Text(l.t("arcana.emotes.\(bubble.emote)"))
                        .font(.headline)
                        .foregroundStyle(colors.bg)
                        .padding(.horizontal, 16)
                        .padding(.vertical, 10)
                        .background(bubble.mine ? colors.gold : colors.text, in: Capsule())
                        .frame(maxWidth: .infinity, alignment: bubble.mine ? .trailing : .leading)
                        .transition(.scale.combined(with: .opacity))
                }
                Spacer()
            }
            .padding(.top, 110)
            .padding(.horizontal, 16)
            .animation(.spring(duration: 0.3), value: arcana.emotes)

            if v.phase == .mulligan {
                mulligan(v, l)
            }
            if arcana.stage == .finished, let result = arcana.finished {
                ArcanaResultView(result: result)
            }
        }
    }

    private func banner(_ text: String, icon: String) -> some View {
        Label(text, systemImage: icon)
            .font(.subheadline.weight(.semibold))
            .foregroundStyle(colors.text)
            .padding(.horizontal, 14)
            .padding(.vertical, 10)
            .background(.ultraThinMaterial, in: Capsule())
    }

    private func mulligan(_ v: ArcanaGameView, _ l: Localizer) -> some View {
        let max = arcana.catalog?.rules.mulliganMax ?? 3
        return VStack(spacing: 16) {
            Text(l.t("arcana.mulliganTitle"))
                .font(.title2.weight(.bold))
                .foregroundStyle(colors.gold)
            if v.you.mulliganed {
                ProgressView().tint(colors.gold)
                Text(l.t("arcana.waitingOpponent")).foregroundStyle(colors.text)
            } else {
                Text(l.t("arcana.mulliganHint", ["count": max]))
                    .foregroundStyle(colors.text)
                    .multilineTextAlignment(.center)
                    .wrapsText()
                LazyVGrid(columns: [GridItem(.adaptive(minimum: 96), spacing: 8)], spacing: 8) {
                    ForEach(v.you.hand) { card in
                        let picked = mulliganPicks.contains(card.uid)
                        Button {
                            if picked {
                                mulliganPicks.remove(card.uid)
                            } else if mulliganPicks.count < max {
                                mulliganPicks.insert(card.uid)
                            }
                        } label: {
                            ArcanaCardTile(card: card, width: 92, selected: picked)
                                .overlay {
                                    if picked {
                                        Image(systemName: "arrow.triangle.2.circlepath")
                                            .font(.title.weight(.bold))
                                            .foregroundStyle(colors.gold)
                                            .shadow(radius: 4)
                                    }
                                }
                        }
                        .buttonStyle(.plain)
                    }
                }
                PrimaryButton(title: mulliganPicks.isEmpty ? l.t("arcana.keepHand")
                              : l.t("arcana.replace", ["count": mulliganPicks.count])) {
                    arcana.mulligan(Array(mulliganPicks))
                    mulliganPicks = []
                }
            }
        }
        .padding(20)
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .background(colors.bg.opacity(0.94))
    }
}

/// Card details: art, cost and both sides' effects; play from here.
struct ArcanaCardSheet: View {
    @Environment(SettingsStore.self) private var settings
    @Environment(ArcanaStore.self) private var arcana
    @Environment(\.appColors) private var colors
    @Environment(\.dismiss) private var dismiss

    let card: ArcanaCardView
    let canAct: Bool
    let play: (ArcanaTarget?) -> Void

    var body: some View {
        let l = settings.localizer
        let tarot = card.cardId.flatMap { settings.catalog.card(id: $0) }
        let def = arcana.catalog?.card(card.cardId)
        ScrollView {
            VStack(spacing: 14) {
                CardImage(source: .bundled(tarot?.imageFile ?? CardArt.backFile), width: 180, contentMode: .fit)
                    .rotationEffect(card.isReversed ? .degrees(180) : .zero)
                    .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
                Text(tarot?.name ?? "")
                    .font(.title2.weight(.bold))
                    .foregroundStyle(colors.gold)
                Text(l.t("arcana.cost", ["cost": card.cost]) + " · " + l.t(card.isReversed ? "arcana.reversed" : "arcana.upright"))
                    .font(.subheadline.weight(.semibold))
                    .foregroundStyle(colors.text)
                if let def {
                    effects(def.side(reversed: card.isReversed), l, active: true)
                    effects(def.side(reversed: !card.isReversed), l, active: false)
                }
                ArcanaTargetButtons(
                    kinds: card.isPlayable && canAct ? (card.targets ?? []) : [],
                    excludeUID: card.uid, playTitle: l.t("arcana.play")
                ) { target in
                    play(target)
                    dismiss()
                }
                if !(card.isPlayable && canAct) {
                    Text(l.t("arcana.notPlayable"))
                        .font(.subheadline)
                        .foregroundStyle(colors.muted)
                }
            }
            .padding(20)
        }
        .background(colors.panel.ignoresSafeArea())
    }

    private func effects(_ side: ArcanaSide, _ l: Localizer, active: Bool) -> some View {
        VStack(alignment: .leading, spacing: 6) {
            Text(l.t(active == !card.isReversed ? "arcana.upright" : "arcana.reversed"))
                .font(.caption.weight(.bold))
                .foregroundStyle(active ? colors.accent : colors.muted)
            ForEach(ArcanaText.describe(side, l), id: \.self) { line in
                Label(line, systemImage: "sparkle")
                    .font(.subheadline)
                    .foregroundStyle(active ? colors.text : colors.muted)
                    .wrapsText()
            }
        }
        .padding(12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(colors.bg.opacity(active ? 0.6 : 0.3), in: RoundedRectangle(cornerRadius: 12, style: .continuous))
    }
}

/// The hero ability or ultimate, with its description and targets.
struct ArcanaPowerSheet: View {
    @Environment(SettingsStore.self) private var settings
    @Environment(\.appColors) private var colors
    @Environment(\.dismiss) private var dismiss

    let choice: ArcanaBattlefieldView.PowerChoice
    let view: ArcanaGameView
    let use: (ArcanaTarget?) -> Void

    var body: some View {
        let l = settings.localizer
        let me = view.you
        let usable = choice == .ability ? me.abilityUsable : me.ultimateUsable
        let kinds = choice == .ability ? me.abilityTargets : me.ultimateTargets
        VStack(spacing: 16) {
            Text(l.t(choice == .ability ? "arcana.ability" : "arcana.ultimate"))
                .font(.title2.weight(.bold))
                .foregroundStyle(colors.gold)
            Text(l.t("arcana.heroText.\(me.hero.id).\(choice.rawValue)"))
                .foregroundStyle(colors.text)
                .multilineTextAlignment(.center)
                .wrapsText()
            ArcanaTargetButtons(kinds: usable ? (kinds ?? []) : [], excludeUID: nil,
                                playTitle: l.t(choice == .ability ? "arcana.ability" : "arcana.ultimate")) { target in
                use(target)
                dismiss()
            }
            if !usable {
                Text(l.t("arcana.notPlayable")).font(.subheadline).foregroundStyle(colors.muted)
            }
        }
        .padding(24)
        .frame(maxHeight: .infinity, alignment: .top)
        .background(colors.panel.ignoresSafeArea())
    }
}

/// One button per allowed target; card targets open a picker of the cards
/// the server allows.
struct ArcanaTargetButtons: View {
    @Environment(SettingsStore.self) private var settings
    @Environment(ArcanaStore.self) private var arcana
    @Environment(\.appColors) private var colors

    let kinds: [ArcanaTargetKind]
    let excludeUID: String?
    let playTitle: String
    let choose: (ArcanaTarget?) -> Void

    @State private var picking: ArcanaTargetKind?

    var body: some View {
        let l = settings.localizer
        VStack(spacing: 10) {
            if kinds.count > 1 {
                Text(l.t("arcana.chooseTarget")).font(.headline).foregroundStyle(colors.text)
            }
            ForEach(kinds, id: \.self) { kind in
                switch kind {
                case .none:
                    PrimaryButton(title: playTitle) { choose(nil) }
                case .enemy:
                    PrimaryButton(title: kinds.count > 1 ? l.t("arcana.targetEnemy") : playTitle) {
                        choose(ArcanaTarget(kind: .enemy))
                    }
                case .self:
                    PrimaryButton(title: kinds.count > 1 ? l.t("arcana.targetSelf") : playTitle) {
                        choose(ArcanaTarget(kind: .self))
                    }
                case .handCard, .discardCard:
                    if picking == kind {
                        cardPicker(kind, l)
                    } else {
                        PrimaryButton(title: l.t(kind == .handCard ? "arcana.targetHandCard" : "arcana.targetDiscardCard")) {
                            picking = kind
                        }
                    }
                }
            }
        }
    }

    private func cardPicker(_ kind: ArcanaTargetKind, _ l: Localizer) -> some View {
        let you = arcana.view?.you
        let cards = (kind == .handCard ? you?.hand : you?.discard)?.filter { $0.uid != excludeUID && $0.cardId != nil } ?? []
        return VStack(alignment: .leading, spacing: 8) {
            Text(l.t(kind == .handCard ? "arcana.targetHandCard" : "arcana.targetDiscardCard"))
                .font(.headline)
                .foregroundStyle(colors.text)
            ScrollView(.horizontal, showsIndicators: false) {
                HStack(spacing: 8) {
                    ForEach(cards) { card in
                        Button {
                            choose(ArcanaTarget(kind: kind, cardUid: card.uid))
                        } label: {
                            ArcanaCardTile(card: ArcanaCardView(uid: card.uid, cardId: card.cardId, reversed: card.reversed,
                                                                cost: card.cost, baseCost: card.baseCost), width: 80)
                        }
                        .buttonStyle(.plain)
                    }
                }
            }
        }
    }
}

/// Victory or defeat.
struct ArcanaResultView: View {
    @Environment(SettingsStore.self) private var settings
    @Environment(ArcanaStore.self) private var arcana
    @Environment(\.appColors) private var colors

    let result: ArcanaFinishedPayload

    var body: some View {
        let l = settings.localizer
        let won = result.result == "win"
        VStack(spacing: 18) {
            Image(systemName: won ? "crown.fill" : (result.result == "loss" ? "moon.zzz.fill" : "questionmark.circle"))
                .font(.system(size: 64))
                .foregroundStyle(won ? colors.gold : colors.muted)
            Text(l.t(won ? "arcana.victory" : (result.result == "loss" ? "arcana.defeat" : "arcana.noResult")))
                .font(.largeTitle.weight(.bold))
                .foregroundStyle(won ? colors.gold : colors.text)
            Text(l.t("arcana.reason.\(result.reason)"))
                .foregroundStyle(colors.text)
                .multilineTextAlignment(.center)
            PrimaryButton(title: l.t("arcana.playAgain")) { arcana.battle() }
            LinkButton(title: l.t("arcana.close")) { arcana.closeResult() }
        }
        .padding(28)
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .background(colors.bg.opacity(0.92))
    }
}

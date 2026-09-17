import SwiftUI

/// A status effect as a small chip; tap for its meaning.
struct ArcanaStatusChip: View {
    @Environment(SettingsStore.self) private var settings
    @Environment(\.appColors) private var colors

    let status: ArcanaStatusView
    @State private var showingHint = false

    static let order = ["burn", "regen", "strength", "weakness", "vulnerable", "ward", "riposte",
                        "empower", "fury", "surge", "exhausted", "confused", "veil", "bond"]
    static let negative: Set<String> = ["burn", "weakness", "vulnerable", "exhausted", "confused"]

    static func icon(_ name: String) -> String {
        switch name {
        case "burn": "flame.fill"
        case "regen": "leaf.fill"
        case "strength": "figure.strengthtraining.traditional"
        case "weakness": "arrow.down.circle.fill"
        case "vulnerable": "shield.slash.fill"
        case "ward": "shield.lefthalf.filled"
        case "riposte": "arrow.uturn.backward.circle.fill"
        case "empower": "sparkles"
        case "fury": "bolt.fill"
        case "surge": "drop.fill"
        case "exhausted": "zzz"
        case "confused": "questionmark.circle.fill"
        case "veil": "eye.slash.fill"
        case "bond": "heart.fill"
        default: "circle.fill"
        }
    }

    var body: some View {
        let l = settings.localizer
        let bad = Self.negative.contains(status.name)
        Button { showingHint = true } label: {
            HStack(spacing: 3) {
                Image(systemName: Self.icon(status.name))
                Text("\(status.amount)")
                if let turns = status.turns, turns > 0 {
                    Text("·\(turns)").foregroundStyle(colors.muted)
                }
            }
            .font(.caption2.weight(.bold))
            .foregroundStyle(bad ? colors.danger : colors.gold)
            .padding(.horizontal, 7)
            .padding(.vertical, 4)
            .background((bad ? colors.danger : colors.gold).opacity(0.15), in: Capsule())
        }
        .buttonStyle(.plain)
        .accessibilityLabel(l.t("arcana.status.\(status.name)") + " \(status.amount)")
        .popover(isPresented: $showingHint) {
            VStack(alignment: .leading, spacing: 4) {
                Text(l.t("arcana.status.\(status.name)")).font(.headline)
                Text(l.t("arcana.statusHint.\(status.name)")).font(.subheadline).wrapsText()
            }
            .padding()
            .frame(width: 240)
            .presentationCompactAdaptation(.popover)
        }
    }
}

/// Health and armor as a bar.
struct ArcanaHealthBar: View {
    @Environment(\.appColors) private var colors
    let hero: ArcanaHeroView

    var body: some View {
        VStack(alignment: .leading, spacing: 3) {
            GeometryReader { geo in
                ZStack(alignment: .leading) {
                    Capsule().fill(colors.text.opacity(0.12))
                    Capsule().fill(colors.danger)
                        .frame(width: geo.size.width * CGFloat(max(0, hero.hp)) / CGFloat(max(1, hero.maxHp)))
                }
            }
            .frame(height: 10)
            HStack(spacing: 10) {
                Label("\(max(0, hero.hp))/\(hero.maxHp)", systemImage: "heart.fill")
                    .foregroundStyle(colors.text)
                if hero.armor > 0 {
                    Label("\(hero.armor)", systemImage: "shield.fill")
                        .foregroundStyle(colors.gold)
                }
            }
            .font(.caption.weight(.bold))
            .monospacedDigit()
        }
        .animation(.easeOut(duration: 0.3), value: hero.hp)
    }
}

/// A card in hand: art, cost and whether it can be played.
struct ArcanaCardTile: View {
    @Environment(SettingsStore.self) private var settings
    @Environment(\.appColors) private var colors

    let card: ArcanaCardView
    var width: CGFloat = 96
    var selected = false
    var dimmed = false
    var deckSlug: String?

    var body: some View {
        let tarot = card.cardId.flatMap { settings.catalog.card(id: $0) }
        VStack(spacing: 4) {
            ZStack(alignment: .topLeading) {
                ArcanaCardArt(cardId: card.cardId, width: width, deckSlug: deckSlug)
                    .rotationEffect(card.isReversed ? .degrees(180) : .zero)
                    .clipShape(RoundedRectangle(cornerRadius: 8, style: .continuous))
                Text("\(card.cost)")
                    .font(.system(size: 15, weight: .heavy, design: .rounded))
                    .foregroundStyle(.white)
                    .frame(width: 28, height: 28)
                    .background(card.cost < card.baseCost ? Color.green : (card.cost > card.baseCost ? colors.danger : colors.accent),
                                in: Circle())
                    .overlay(Circle().stroke(.white.opacity(0.8), lineWidth: 1.5))
                    .offset(x: -6, y: -6)
                if card.isReversed {
                    Image(systemName: "arrow.up.arrow.down.circle.fill")
                        .foregroundStyle(colors.gold)
                        .background(Circle().fill(colors.bg))
                        .frame(maxWidth: .infinity, alignment: .topTrailing)
                        .offset(x: 4, y: -4)
                }
            }
            Text(tarot?.name ?? "")
                .font(.caption2.weight(.semibold))
                .foregroundStyle(colors.text)
                .lineLimit(2)
                .multilineTextAlignment(.center)
                .frame(width: width, height: 28, alignment: .top)
        }
        .padding(4)
        .background(RoundedRectangle(cornerRadius: 12, style: .continuous)
            .stroke(selected ? colors.gold : (card.isPlayable ? colors.accent : .clear), lineWidth: selected ? 3 : 2))
        .opacity(dimmed ? 0.5 : 1)
        .accessibilityElement(children: .combine)
    }
}

/// Card art for the game, drawn from the deck the player has selected — the
/// same art they see in readings, so a bought deck is theirs in battle too.
/// Falls back per card to the classic art when the deck lacks a picture.
struct ArcanaCardArt: View {
    @Environment(SettingsStore.self) private var settings
    @Environment(DeckStore.self) private var decks

    /// A tarot card id (`the_tower`); nil or unknown shows the card back.
    let cardId: String?
    let width: CGFloat
    var contentMode: ContentMode = .fill
    /// Another player's deck; nil draws with the player's own selected deck.
    var deckSlug: String?

    var body: some View {
        let slug = deckSlug ?? decks.activeDeck(selectedSlug: settings.settings.selectedDeckId,
                                                locallyOwned: settings.settings.ownedDeckIds).slug
        let source: CardArtSource = if let card = cardId.flatMap({ settings.catalog.card(id: $0) }) {
            decks.faceSource(for: card, deckSlug: slug)
        } else {
            decks.backSource(deckSlug: slug)
        }
        CardImage(source: source, width: width, contentMode: contentMode)
    }
}

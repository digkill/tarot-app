import SwiftUI

/// What a sheet about one card is opened with.
struct CardMeaningItem: Identifiable, Hashable {
    let card: TarotCard
    let isReversed: Bool
    /// The deck whose art to show.
    let deckId: String
    var positionTitle: String?

    var id: String { card.id }
}

/// A card's art and meaning, with a toggle between upright and reversed that
/// starts on the orientation the card was drawn in.
///
/// The React Native sheet also had keyword chips and love/work/health/advice
/// sections, but the bundled card data has none of those fields, so they were
/// always empty and are not ported.
struct CardMeaningSheet: View {
    @Environment(SettingsStore.self) private var settings
    @Environment(DeckStore.self) private var decks
    @Environment(\.appColors) private var colors
    @Environment(\.dismiss) private var dismiss

    let item: CardMeaningItem

    @State private var showReversed: Bool

    init(item: CardMeaningItem) {
        self.item = item
        _showReversed = State(initialValue: item.isReversed)
    }

    var body: some View {
        let l = settings.localizer
        ScrollView {
            VStack(spacing: 18) {
                if let title = item.positionTitle {
                    Text(title)
                        .font(.subheadline.weight(.semibold))
                        .foregroundStyle(colors.muted)
                }
                Text(item.card.name)
                    .font(.title2.weight(.bold))
                    .foregroundStyle(colors.gold)
                    .multilineTextAlignment(.center)

                Picker("", selection: $showReversed) {
                    Text(l.t("cardMeaning.upright")).tag(false)
                    Text(l.t("cardMeaning.reversed")).tag(true)
                }
                .pickerStyle(.segmented)

                CardImage(source: decks.faceSource(for: item.card, deckSlug: item.deckId), width: 200, contentMode: .fit)
                    .rotationEffect(.degrees(showReversed ? 180 : 0))
                    .animation(settings.settings.disableAnimations ? nil : .easeInOut(duration: 0.3), value: showReversed)
                    .shadow(color: .black.opacity(0.4), radius: 10, y: 4)

                VStack(alignment: .leading, spacing: 8) {
                    Text(l.t("cardMeaning.general"))
                        .font(.headline)
                        .foregroundStyle(colors.gold)
                    Text(item.card.meaning(reversed: showReversed))
                        .font(.body)
                        .foregroundStyle(colors.text)
                        .wrapsText()
                }
                .frame(maxWidth: .infinity, alignment: .leading)

                PrimaryButton(title: l.t("cardMeaning.close")) {
                    dismiss()
                }
            }
            .padding(24)
        }
        .background(colors.bg.ignoresSafeArea())
        .presentationDetents([.large])
        .presentationDragIndicator(.visible)
    }
}

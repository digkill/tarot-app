import SwiftUI

/// Deep links the app understands. Kept observable so a link that arrives
/// before the signed-in tabs exist is still acted on once they appear.
@MainActor
@Observable
final class DeepLinks {
    /// A deck to open in the Decks tab: `mediarisetarot://deck/<slug>`.
    var deckSlug: String?

    /// Returns false for links this app does not handle.
    @discardableResult
    func handle(_ url: URL) -> Bool {
        guard url.scheme?.lowercased() == "mediarisetarot", url.host?.lowercased() == "deck" else {
            return false
        }
        let slug = url.pathComponents.first { $0 != "/" }?.trimmingCharacters(in: .whitespaces)
        deckSlug = slug?.nilIfEmpty ?? ""
        return true
    }
}

/// The deck shop and the 78-card gallery.
struct DecksView: View {
    @Environment(SettingsStore.self) private var settings
    @Environment(DeckStore.self) private var decks
    @Environment(DeepLinks.self) private var deepLinks
    @Environment(PurchaseStore.self) private var purchases
    @Environment(SessionStore.self) private var session
    @Environment(\.appColors) private var colors

    private enum ArcanaFilter: Hashable { case all, major, minor }
    private enum SuitFilter: Hashable { case all, suit(TarotSuit) }

    @State private var arcana: ArcanaFilter = .all
    @State private var suit: SuitFilter = .all
    @State private var search = ""
    @State private var selected: CardMeaningItem?
    @State private var highlightedSlug: String?
    @State private var purchaseMessage: String?

    private var activeSlug: String {
        decks.activeDeck(selectedSlug: settings.settings.selectedDeckId,
                         locallyOwned: settings.settings.ownedDeckIds).slug
    }

    private var filteredCards: [TarotCard] {
        settings.catalog.cards.filter { card in
            let arcanaMatches = switch arcana {
            case .all: true
            case .major: card.arcana == .major
            case .minor: card.arcana == .minor
            }
            let suitMatches = switch suit {
            case .all: true
            case let .suit(value): card.suit == value
            }
            return arcanaMatches && suitMatches && card.matches(search)
        }
    }

    var body: some View {
        let l = settings.localizer
        NavigationStack {
            ScrollViewReader { proxy in
                ScrollView {
                    VStack(alignment: .leading, spacing: 16) {
                        Text(l.t("deck.shopHint"))
                            .font(.subheadline)
                            .foregroundStyle(colors.muted)
                            .wrapsText()

                        shop(l)
                        filters(l)
                        grid(l)
                    }
                    .padding(20)
                }
                .onChange(of: deepLinks.deckSlug, initial: true) { _, slug in
                    guard let slug else { return }
                    highlightedSlug = slug
                    withAnimation { proxy.scrollTo("deck-\(slug)", anchor: .center) }
                    deepLinks.deckSlug = nil
                }
            }
            .background(AppBackground())
            .navigationTitle(l.t("deck.shopTitle"))
            .toolbarBackground(colors.bg, for: .navigationBar)
            .toolbarColorScheme(.dark, for: .navigationBar)
            .searchable(text: $search, prompt: l.t("deck.searchPlaceholder"))
            .refreshable { await decks.refresh(signedIn: true) }
        }
        .sheet(item: $selected) { CardMeaningSheet(item: $0) }
        .task(id: deckSKUs) { await purchases.loadProducts(deckSKUs) }
        .alert(purchaseMessage ?? "", isPresented: Binding(
            get: { purchaseMessage != nil }, set: { if !$0 { purchaseMessage = nil } }
        )) {
            Button("OK", role: .cancel) {}
        }
    }

    private var deckSKUs: [String] {
        decks.shopDecks(locallyOwned: settings.settings.ownedDeckIds).compactMap(\.appleProductId)
    }

    private func buy(_ deck: DeckOption) async {
        guard let sku = deck.appleProductId else { return }
        let l = settings.localizer
        let outcome = await purchases.purchase(sku)
        if case .purchased = outcome {
            await decks.refresh(signedIn: true)
            if decks.isOwned(deck, locallyOwned: settings.settings.ownedDeckIds) {
                settings.update { $0.selectedDeckId = deck.slug }
            }
            purchaseMessage = l.t("premiumIos.deckPurchased")
        } else if let key = PurchaseStore.messageKey(for: outcome) {
            purchaseMessage = l.t(key)
        }
    }

    // MARK: - Shop

    private func shop(_ l: Localizer) -> some View {
        ScrollView(.horizontal, showsIndicators: false) {
            HStack(alignment: .top, spacing: 12) {
                ForEach(decks.shopDecks(locallyOwned: settings.settings.ownedDeckIds)) { deck in
                    deckCard(deck, l).id("deck-\(deck.slug)")
                }
                if decks.loading {
                    ProgressView().tint(colors.accent).padding()
                }
            }
            .padding(.vertical, 4)
        }
        .scrollClipDisabled()
    }

    private func deckCard(_ deck: DeckOption, _ l: Localizer) -> some View {
        let owned = decks.isOwned(deck, locallyOwned: settings.settings.ownedDeckIds)
        let active = activeSlug == deck.slug
        let highlighted = highlightedSlug == deck.slug
        return VStack(alignment: .leading, spacing: 8) {
            CardImage(source: decks.coverSource(deck), width: 156, contentMode: .fit)
                .frame(maxWidth: .infinity)
                .background(deck.colors.bg, in: RoundedRectangle(cornerRadius: 12, style: .continuous))

            Text(deck.title(in: settings.settings.language, localizer: l))
                .font(.headline)
                .foregroundStyle(colors.gold)
                .lineLimit(2)
            Text(deck.description(in: settings.settings.language, localizer: l))
                .font(.caption)
                .foregroundStyle(colors.text)
                .lineLimit(3)
                .frame(maxHeight: .infinity, alignment: .top)

            priceRow(deck, owned: owned, l)

            if owned {
                Button {
                    settings.update { $0.selectedDeckId = deck.slug }
                } label: {
                    Text(l.t(active ? "deck.selected" : "deck.select"))
                        .font(.subheadline.weight(.semibold))
                        .foregroundStyle(active ? colors.bg : .white)
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 10)
                        .background(active ? colors.gold : colors.accent,
                                    in: RoundedRectangle(cornerRadius: 12, style: .continuous))
                }
                .buttonStyle(.plain)
                // Not `.disabled`: that dims the gold "Selected" state as if broken.
                .allowsHitTesting(!active)
                .accessibilityAddTraits(active ? .isSelected : [])
            } else {
                // Paid decks reach this list only with an App Store SKU.
                let sku = deck.appleProductId ?? ""
                let available = purchases.products[sku] != nil
                let buying = purchases.purchasingId == sku
                Button {
                    Task { await buy(deck) }
                } label: {
                    ZStack {
                        Text(l.t("deck.buy")).opacity(buying ? 0 : 1)
                        if buying { ProgressView().tint(.white) }
                    }
                    .font(.subheadline.weight(.semibold))
                    .foregroundStyle(.white)
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 10)
                    .background(colors.accent.opacity(available ? 1 : 0.45),
                                in: RoundedRectangle(cornerRadius: 12, style: .continuous))
                }
                .buttonStyle(.plain)
                .disabled(!available || purchases.purchasingId != nil)
            }
        }
        .padding(12)
        .frame(width: 180, height: 430)
        .background(colors.panel, in: RoundedRectangle(cornerRadius: 18, style: .continuous))
        .overlay(
            RoundedRectangle(cornerRadius: 18, style: .continuous)
                .stroke(highlighted ? colors.gold : (active ? colors.accent : colors.gold.opacity(0.45)),
                        lineWidth: highlighted ? 2 : 1)
        )
    }

    private func priceRow(_ deck: DeckOption, owned: Bool, _ l: Localizer) -> some View {
        HStack(spacing: 6) {
            Text(priceLabel(deck, owned: owned, l))
                .foregroundStyle(colors.text)
        }
        .font(.caption.weight(.semibold))
        .lineLimit(1)
        .minimumScaleFactor(0.8)
    }

    /// The App Store price, never the server's rubles: those are the Android
    /// price, and on iOS the storefront decides price and currency.
    private func priceLabel(_ deck: DeckOption, owned: Bool, _ l: Localizer) -> String {
        if deck.isFree || deck.isBundled { return l.t("deck.free") }
        if owned { return l.t("deck.owned") }
        return deck.appleProductId.flatMap { purchases.products[$0]?.displayPrice } ?? "—"
    }

    // MARK: - Gallery

    private func filters(_ l: Localizer) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            Text(l.t("deck.arcana"))
                .font(.subheadline.weight(.semibold))
                .foregroundStyle(colors.text)
            FlowChips(options: [ArcanaFilter.all, .major, .minor], selected: arcana, label: { filter in
                switch filter {
                case .all: l.t("deck.arcanaFilter.all")
                case .major: l.t("deck.arcanaFilter.major")
                case .minor: l.t("deck.arcanaFilter.minor")
                }
            }) { arcana = $0 }

            Text(l.t("deck.suits"))
                .font(.subheadline.weight(.semibold))
                .foregroundStyle(colors.text)
                .padding(.top, 6)
            ScrollView(.horizontal, showsIndicators: false) {
                HStack(spacing: 8) {
                    ForEach([SuitFilter.all] + TarotSuit.allCases.map(SuitFilter.suit), id: \.self) { filter in
                        let isSelected = filter == suit
                        Button {
                            suit = filter
                        } label: {
                            Text(suitLabel(filter, l))
                                .font(.subheadline.weight(.semibold))
                                .foregroundStyle(isSelected ? .white : colors.text)
                                .padding(.horizontal, 14)
                                .padding(.vertical, 8)
                                .background(isSelected ? colors.accent : colors.text.opacity(0.08), in: Capsule())
                        }
                        .buttonStyle(.plain)
                    }
                }
            }
        }
    }

    private func suitLabel(_ filter: SuitFilter, _ l: Localizer) -> String {
        switch filter {
        case .all: l.t("deck.suitFilter.all")
        case let .suit(value): l.t("deck.suitFilter.\(value.rawValue)")
        }
    }

    private func grid(_ l: Localizer) -> some View {
        let cards = filteredCards
        return Group {
            if cards.isEmpty {
                Text(l.t("deck.searchEmpty"))
                    .foregroundStyle(colors.muted)
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 32)
            } else {
                LazyVGrid(columns: [GridItem(.flexible(), spacing: 16), GridItem(.flexible(), spacing: 16)], spacing: 16) {
                    ForEach(cards) { card in
                        Button {
                            selected = CardMeaningItem(card: card, isReversed: false, deckId: activeSlug)
                        } label: {
                            VStack(alignment: .leading, spacing: 6) {
                                CardImage(source: decks.faceSource(for: card, deckSlug: activeSlug),
                                          width: 132, contentMode: .fit)
                                    .frame(maxWidth: .infinity)
                                Text(card.name)
                                    .font(.subheadline.weight(.semibold))
                                    .foregroundStyle(colors.gold)
                                    .lineLimit(2)
                                Text(card.upright)
                                    .font(.caption)
                                    .foregroundStyle(colors.text)
                                    .lineLimit(3)
                                    .multilineTextAlignment(.leading)
                            }
                            .padding(10)
                            .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
                            .background(colors.panel, in: RoundedRectangle(cornerRadius: 16, style: .continuous))
                            .overlay(RoundedRectangle(cornerRadius: 16, style: .continuous).stroke(colors.gold.opacity(0.45)))
                        }
                        .buttonStyle(.plain)
                    }
                }
            }
        }
    }
}

extension TarotCard {
    /// Case-insensitive match on the name and both meanings, as the React
    /// Native `cardMatchesQuery` did.
    func matches(_ query: String) -> Bool {
        let needle = query.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !needle.isEmpty else { return true }
        return [name, id, upright, reversed].contains { $0.localizedCaseInsensitiveContains(needle) }
    }
}

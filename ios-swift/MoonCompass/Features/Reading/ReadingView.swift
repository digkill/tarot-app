import SwiftUI

/// The reading table: shuffle, deal, then flip cards to reveal them.
struct ReadingView: View {
    @Environment(SettingsStore.self) private var settings
    @Environment(SessionStore.self) private var session
    @Environment(HistoryStore.self) private var history
    @Environment(\.services) private var services
    @Environment(\.appColors) private var colors
    @Environment(\.dismiss) private var dismiss

    let spreadId: String
    let deckId: String
    /// Non-nil for a one-card draw already cleared by the daily quota.
    let dailyKind: Reading.Kind?
    @Binding var path: [ReadingRoute]

    private enum Phase { case shuffle, dealing, review }

    @State private var phase: Phase = .shuffle
    @State private var entries: [DrawnCard] = []
    @State private var dealt: [Bool] = []
    @State private var drawToken = UUID()
    @State private var dealTask: Task<Void, Never>?
    @State private var savedReadingId: String?
    @State private var saving = false
    @State private var selected: CardMeaningItem?
    @State private var quotaFailure: String?
    @State private var started = false

    private var spread: Spread? { Spread.find(spreadId) }
    private var isDailyDraw: Bool { spread?.isOneCard == true }

    var body: some View {
        let l = settings.localizer
        Group {
            if let spread {
                content(spread, l)
            } else {
                Text(l.t("reading.missingSpread"))
                    .foregroundStyle(colors.text)
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            }
        }
        .navigationBarTitleDisplayMode(.inline)
        .toolbar(.hidden, for: .tabBar)
        .toolbarBackground(colors.bg, for: .navigationBar)
        .toolbarColorScheme(.dark, for: .navigationBar)
        .sheet(item: $selected) { CardMeaningSheet(item: $0) }
        .alert(l.t("dailyCard.limitTitle"), isPresented: Binding(
            get: { quotaFailure != nil },
            set: { if !$0 { quotaFailure = nil; dismiss() } }
        )) {
            Button("OK", role: .cancel) {}
        } message: {
            Text(quotaFailure ?? "")
        }
        .task {
            guard !started else { return }
            started = true
            await begin()
        }
        .onDisappear { dealTask?.cancel() }
    }

    private func content(_ spread: Spread, _ l: Localizer) -> some View {
        VStack(spacing: 0) {
            header(spread, l)
                .padding(.horizontal, 20)
                .padding(.top, 8)

            table(spread, l)
                .padding(.horizontal, 16)
                .padding(.top, 12)

            Text(l.t("reading.zoomHint"))
                .font(.caption)
                .foregroundStyle(colors.muted)
                .multilineTextAlignment(.center)
                .opacity(phase == .review ? 1 : 0)
                .padding(.horizontal, 24)
                .padding(.vertical, 8)

            ScrollView {
                VStack(alignment: .leading, spacing: 14) {
                    ForEach(entries, id: \.position.index) { entry in
                        Button {
                            selected = CardMeaningItem(card: entry.card, isReversed: entry.isReversed, deckId: deckId,
                                                       positionTitle: l.t(entry.position.titleKey))
                        } label: {
                            VStack(alignment: .leading, spacing: 3) {
                                Text(l.t(entry.position.titleKey))
                                    .font(.subheadline.weight(.semibold))
                                    .foregroundStyle(colors.text)
                                Text(l.t(entry.position.descriptionKey))
                                    .font(.footnote)
                                    .foregroundStyle(colors.muted)
                                    .wrapsText()
                                Text(entry.card.name + (entry.isReversed ? " " + l.t("reading.reversed") : ""))
                                    .font(.subheadline.weight(.semibold))
                                    .foregroundStyle(colors.gold)
                            }
                            .frame(maxWidth: .infinity, alignment: .leading)
                        }
                        .buttonStyle(.plain)
                        .disabled(phase != .review)
                    }
                }
                .padding(.horizontal, 20)
                .padding(.bottom, 20)
            }

            footer(l)
                .padding(20)
        }
    }

    private func header(_ spread: Spread, _ l: Localizer) -> some View {
        HStack(alignment: .firstTextBaseline) {
            VStack(alignment: .leading, spacing: 2) {
                Text(l.t(spread.nameKey))
                    .font(.title3.weight(.bold))
                    .foregroundStyle(colors.gold)
                Text(l.t("reading.cardCount", ["count": spread.cardCount]))
                    .font(.footnote)
                    .foregroundStyle(colors.muted)
            }
            Spacer()
            if !isDailyDraw {
                Button(l.t("reading.redraw")) { startReading(skipAnimation: false) }
                    .font(.body.weight(.semibold))
                    .foregroundStyle(colors.accent)
                    .disabled(phase == .shuffle)
            }
        }
    }

    private func table(_ spread: Spread, _ l: Localizer) -> some View {
        let height = ReadingLayout.canvasHeight(cardCount: spread.cardCount)
        return GeometryReader { geometry in
            let canvas = geometry.size
            let cardWidth = ReadingLayout.cardWidth(positions: spread.positions, canvas: canvas)
            ZoomableCanvas(
                isEnabled: phase == .review,
                resetKey: drawToken.uuidString,
                // Shown under the table instead: over it, it hid the bottom card.
                hint: nil,
                resetLabel: l.t("reading.resetZoom")
            ) {
                ZStack(alignment: .topLeading) {
                    ForEach(Array(entries.enumerated()), id: \.element.position.index) { index, entry in
                        cardCell(entry, index: index, width: cardWidth, l)
                            .offset(
                                x: ReadingLayout.origin(for: entry.position, cardWidth: cardWidth, canvas: canvas).x,
                                y: ReadingLayout.origin(for: entry.position, cardWidth: cardWidth, canvas: canvas).y
                            )
                    }
                }
                .frame(width: canvas.width, height: canvas.height, alignment: .topLeading)
            }
            .id(drawToken)
            .overlay {
                if phase == .shuffle {
                    Text(l.t("reading.shuffling"))
                        .font(.headline)
                        .foregroundStyle(colors.text)
                }
            }
        }
        .frame(height: height)
        .background(colors.panel, in: RoundedRectangle(cornerRadius: 20, style: .continuous))
        .overlay(RoundedRectangle(cornerRadius: 20, style: .continuous).stroke(colors.gold.opacity(0.25)))
        .clipShape(RoundedRectangle(cornerRadius: 20, style: .continuous))
    }

    private func cardCell(_ entry: DrawnCard, index: Int, width: CGFloat, _ l: Localizer) -> some View {
        let isDealt = dealt.indices.contains(index) && dealt[index]
        return VStack(spacing: 2) {
            TarotCardView(
                card: entry.card,
                isReversed: entry.isReversed,
                deckId: deckId,
                width: width,
                startFaceDown: true,
                interactive: phase == .review
            ) {
                selected = CardMeaningItem(card: entry.card, isReversed: entry.isReversed, deckId: deckId,
                                           positionTitle: l.t(entry.position.titleKey))
            }
            Text(l.t(entry.position.titleKey))
                .font(.system(size: 10, weight: .medium))
                .foregroundStyle(colors.text)
                .lineLimit(2)
                .minimumScaleFactor(0.75)
                .multilineTextAlignment(.center)
                .frame(width: width + 28, height: ReadingLayout.labelHeight + 2, alignment: .top)
        }
        .frame(width: width)
        .opacity(isDealt ? 1 : 0)
        .scaleEffect(isDealt ? 1 : 0.9)
        .offset(y: isDealt ? 0 : 40)
    }

    private func footer(_ l: Localizer) -> some View {
        HStack(spacing: 12) {
            Button {
                revealImmediately()
            } label: {
                Text(l.t("reading.skipAnimations"))
                    .font(.subheadline.weight(.semibold))
                    .foregroundStyle(colors.gold)
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 14)
                    .overlay(RoundedRectangle(cornerRadius: 16, style: .continuous).stroke(colors.gold.opacity(0.4)))
            }
            .buttonStyle(.plain)
            .disabled(phase == .review)
            .opacity(phase == .review ? 0.5 : 1)

            PrimaryButton(
                title: l.t(saving ? "reading.saving" : "reading.continue"),
                isEnabled: phase == .review && !saving && !entries.isEmpty
            ) {
                continueToInterpretation()
            }
        }
    }

    // MARK: - Flow

    private func begin() async {
        // A one-card draw normally arrives already cleared by the daily
        // quota. If not, clear it here, as the React Native screen did.
        if isDailyDraw, dailyKind == nil {
            do {
                _ = try await services.dailyCards.consume(
                    hasTodaysCard: history.todaysDailyCard() != nil,
                    hasPremium: session.user?.hasPremium ?? false
                )
            } catch let error as APIError where error.code == "quota_exceeded" {
                quotaFailure = settings.localizer.t("dailyCard.quotaReached")
                return
            } catch {
                quotaFailure = settings.localizer.t("dailyCard.unlockError")
                return
            }
        }
        startReading(skipAnimation: false)
    }

    private func startReading(skipAnimation: Bool) {
        guard let spread else { return }
        dealTask?.cancel()
        let animate = !(skipAnimation || settings.settings.disableAnimations)

        dealTask = Task { @MainActor in
            if animate {
                phase = .shuffle
                entries = []
                try? await Task.sleep(nanoseconds: 800_000_000)
                guard !Task.isCancelled else { return }
            }

            let drawn = draw(spread)
            drawToken = UUID()
            entries = drawn
            if isDailyDraw {
                persist(drawn)
            }

            guard animate else {
                dealt = Array(repeating: true, count: drawn.count)
                phase = .review
                return
            }

            dealt = Array(repeating: false, count: drawn.count)
            phase = .dealing
            for index in drawn.indices {
                withAnimation(.easeOut(duration: 0.6).delay(Double(index) * 0.14)) {
                    dealt[index] = true
                }
            }
            let total = 0.6 + Double(max(drawn.count - 1, 0)) * 0.14
            try? await Task.sleep(nanoseconds: UInt64(total * 1_000_000_000))
            guard !Task.isCancelled else { return }
            phase = .review
        }
    }

    /// Finishes the deal at once. Unlike the React Native screen, skipping does
    /// not quietly draw a different set of cards.
    private func revealImmediately() {
        guard phase != .review else { return }
        if phase == .shuffle {
            startReading(skipAnimation: true)
            return
        }
        dealTask?.cancel()
        var instantly = Transaction()
        instantly.disablesAnimations = true
        withTransaction(instantly) {
            dealt = Array(repeating: true, count: entries.count)
        }
        phase = .review
    }

    private func draw(_ spread: Spread) -> [DrawnCard] {
        var deck = settings.catalog.cards.shuffled()
        let chance = settings.settings.reversedChance
        return spread.positions.compactMap { position in
            guard !deck.isEmpty else { return nil }
            let card = deck.removeFirst()
            return DrawnCard(card: card, position: position, isReversed: Double.random(in: 0..<1) < chance)
        }
    }

    /// A daily card is saved the moment it is drawn, so it cannot be redrawn by
    /// leaving and coming back.
    @discardableResult
    private func persist(_ drawn: [DrawnCard]) -> String? {
        if let savedReadingId { return savedReadingId }
        guard let spread, !drawn.isEmpty else { return nil }
        let reading = history.add(Reading(
            spreadId: spread.id,
            deckId: deckId,
            items: drawn.map { ReadingItem(positionIndex: $0.position.index, cardId: $0.card.id, isReversed: $0.isReversed) },
            summaryText: Interpretation.summary(spread: spread, entries: drawn, localizer: settings.localizer),
            kind: isDailyDraw ? (dailyKind ?? .daily) : nil
        ))
        savedReadingId = reading.id
        return reading.id
    }

    private func continueToInterpretation() {
        guard !entries.isEmpty else { return }
        saving = true
        defer { saving = false }
        guard let readingId = persist(entries) else { return }
        var next = path
        if !next.isEmpty { next.removeLast() }
        next.append(.interpretation(readingId: readingId))
        path = next
    }
}

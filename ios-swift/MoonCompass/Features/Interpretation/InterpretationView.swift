import SwiftUI

struct InterpretationView: View {
    @Environment(SettingsStore.self) private var settings
    @Environment(SessionStore.self) private var session
    @Environment(HistoryStore.self) private var history
    @Environment(\.services) private var services
    @Environment(\.appColors) private var colors
    @Environment(\.displayScale) private var displayScale
    @Environment(\.selectTab) private var selectTab

    let readingId: String
    @Binding var path: [ReadingRoute]

    @State private var notes = ""
    @State private var notesLoaded = false
    @State private var notesSaved = false
    @State private var usage: UsageSnapshot?
    @State private var generating = false
    @State private var aiError: String?
    @State private var selected: CardMeaningItem?
    @State private var shareImage: ShareableImage?
    @State private var shareFailed = false

    private var hasPremium: Bool { session.user?.hasPremium ?? false }

    var body: some View {
        let l = settings.localizer
        Group {
            if let reading = history.reading(id: readingId), let spread = Spread.find(reading.spreadId) {
                content(reading, spread, l)
            } else {
                VStack(spacing: 12) {
                    Text(l.t("interpretation.missingReading")).foregroundStyle(colors.text)
                    LinkButton(title: l.t("interpretation.goBack")) {
                        if !path.isEmpty { path.removeLast() }
                    }
                }
                .frame(maxWidth: .infinity, maxHeight: .infinity)
            }
        }
        .navigationBarTitleDisplayMode(.inline)
        .toolbar(.hidden, for: .tabBar)
        .toolbarBackground(colors.bg, for: .navigationBar)
        .toolbarColorScheme(.dark, for: .navigationBar)
        .sheet(item: $selected) { CardMeaningSheet(item: $0) }
        .sheet(item: $shareImage) { ShareSheet(items: [$0.image, l.t("interpretation.shareDialog")]) }
        .alert(l.t("interpretation.notesSaved"), isPresented: $notesSaved) { Button("OK", role: .cancel) {} }
        .alert(l.t("interpretation.shareError"), isPresented: $shareFailed) { Button("OK", role: .cancel) {} }
        .task(id: readingId) {
            if !notesLoaded, let reading = history.reading(id: readingId) {
                notes = reading.notes
                notesLoaded = true
            }
            if hasPremium {
                usage = try? await services.usage.usage()
            }
        }
    }

    private func content(_ reading: Reading, _ spread: Spread, _ l: Localizer) -> some View {
        let entries = Interpretation.entries(for: reading, spread: spread, catalog: settings.catalog)
        return ScrollView {
            VStack(alignment: .leading, spacing: 18) {
                ReadingSummaryCard(
                    reading: reading, spread: spread, entries: entries, localizer: l,
                    onSelect: { entry in
                        selected = CardMeaningItem(card: entry.card, isReversed: entry.isReversed,
                                                   positionTitle: l.t(entry.position.titleKey))
                    }
                )

                HStack(spacing: 12) {
                    Button {
                        history.toggleFavorite(id: reading.id)
                    } label: {
                        Label(l.t(reading.favorite ? "interpretation.unfavorite" : "interpretation.favorite"),
                              systemImage: reading.favorite ? "star.fill" : "star")
                            .font(.subheadline.weight(.semibold))
                            .foregroundStyle(colors.gold)
                            .frame(maxWidth: .infinity)
                            .padding(.vertical, 14)
                            .overlay(RoundedRectangle(cornerRadius: 16, style: .continuous).stroke(colors.gold.opacity(0.4)))
                    }
                    .buttonStyle(.plain)

                    PrimaryButton(title: l.t("interpretation.share")) {
                        share(reading, spread, entries, l)
                    }
                }

                aiSection(reading, spread, entries, l)
                notesSection(reading, l)
            }
            .padding(20)
        }
        .scrollDismissesKeyboard(.interactively)
    }

    // MARK: - AI

    @ViewBuilder
    private func aiSection(_ reading: Reading, _ spread: Spread, _ entries: [DrawnCard], _ l: Localizer) -> some View {
        if hasPremium {
            VStack(alignment: .leading, spacing: 12) {
                Text(l.t("aiInterpretation.title"))
                    .font(.headline)
                    .foregroundStyle(colors.gold)
                if let usage {
                    Text(l.t("aiInterpretation.quotaHint", [
                        "remaining": usage.interpretations.remaining, "limit": usage.interpretations.limit,
                    ]))
                    .font(.footnote)
                    .foregroundStyle(colors.muted)
                }

                if generating {
                    HStack(spacing: 10) {
                        ProgressView().tint(colors.accent)
                        Text(l.t("aiInterpretation.loading")).foregroundStyle(colors.text)
                    }
                } else if let insight = reading.aiInsights {
                    Text(insight.summary)
                        .foregroundStyle(colors.text)
                        .wrapsText()
                    ForEach(insight.positions, id: \.positionIndex) { position in
                        VStack(alignment: .leading, spacing: 4) {
                            Text("\(position.positionTitle) — \(position.cardName)")
                                .font(.subheadline.weight(.semibold))
                                .foregroundStyle(colors.gold)
                            Text(position.meaning)
                                .foregroundStyle(colors.text)
                                .wrapsText()
                        }
                    }
                    LinkButton(title: l.t("aiInterpretation.refreshInterpretation")) {
                        Task { await generate(reading, spread, entries) }
                    }
                } else {
                    PrimaryButton(title: l.t("aiInterpretation.generate")) {
                        Task { await generate(reading, spread, entries) }
                    }
                }

                if let aiError, !generating {
                    StatusMessage(text: aiError)
                    LinkButton(title: l.t("aiInterpretation.retry")) {
                        Task { await generate(reading, spread, entries) }
                    }
                }
            }
            .padding(16)
            .frame(maxWidth: .infinity, alignment: .leading)
            .background(colors.panel, in: RoundedRectangle(cornerRadius: 18, style: .continuous))
            .overlay(RoundedRectangle(cornerRadius: 18, style: .continuous).stroke(colors.accent.opacity(0.4)))
        } else {
            VStack(alignment: .leading, spacing: 10) {
                Text(l.t("aiInterpretation.premiumFeature"))
                    .font(.headline)
                    .foregroundStyle(colors.gold)
                Text(l.t("aiInterpretation.unlockMessage"))
                    .foregroundStyle(colors.text)
                    .wrapsText()
                PrimaryButton(title: l.t("aiInterpretation.unlock")) {
                    selectTab(.settings)
                }
            }
            .padding(16)
            .frame(maxWidth: .infinity, alignment: .leading)
            .background(colors.accent.opacity(0.1), in: RoundedRectangle(cornerRadius: 18, style: .continuous))
            .overlay(RoundedRectangle(cornerRadius: 18, style: .continuous).stroke(colors.accent.opacity(0.3)))
        }
    }

    private func generate(_ reading: Reading, _ spread: Spread, _ entries: [DrawnCard]) async {
        guard !entries.isEmpty, !generating else { return }
        let l = settings.localizer
        generating = true
        aiError = nil
        defer { generating = false }
        do {
            let response = try await services.interpretations.interpret(
                InterpretationRequest(spread: spread, entries: entries, localizer: l)
            )
            history.update(id: reading.id) {
                $0.aiInsights = AIInsight(
                    summary: response.summary,
                    positions: response.positions.map {
                        .init(positionIndex: $0.positionIndex, positionTitle: $0.positionTitle,
                              cardName: $0.cardName, orientation: $0.orientation, meaning: $0.meaning)
                    },
                    language: l.language,
                    generatedAt: Date()
                )
            }
            usage = try? await services.usage.usage()
        } catch is CancellationError {
            return
        } catch let error as APIError {
            switch error.code {
            case "unauthorized": aiError = l.t("aiInterpretation.needLogin")
            case "premium_required": aiError = l.t("aiInterpretation.premiumRequired")
            case "quota_exceeded":
                aiError = l.t("aiInterpretation.quotaReached", ["limit": usage?.interpretations.limit ?? 50])
            case "rate_limited": aiError = l.t("aiInterpretation.rateLimited")
            default: aiError = l.t("aiInterpretation.unavailable")
            }
        } catch {
            aiError = l.t("aiInterpretation.unavailable")
        }
    }

    // MARK: - Notes

    private func notesSection(_ reading: Reading, _ l: Localizer) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            Text(l.t("interpretation.notesTitle"))
                .font(.headline)
                .foregroundStyle(colors.text)
            ZStack(alignment: .topLeading) {
                if notes.isEmpty {
                    Text(l.t("interpretation.notesPlaceholder"))
                        .foregroundStyle(colors.muted)
                        .padding(.horizontal, 5)
                        .padding(.vertical, 8)
                        .allowsHitTesting(false)
                }
                TextEditor(text: $notes)
                    .scrollContentBackground(.hidden)
                    .foregroundStyle(colors.text)
                    .frame(minHeight: 110)
            }
            PrimaryButton(title: l.t("interpretation.saveNotes"), isEnabled: notes != reading.notes) {
                history.update(id: reading.id) { $0.notes = notes }
                notesSaved = true
            }
        }
        .padding(16)
        .background(colors.panel, in: RoundedRectangle(cornerRadius: 18, style: .continuous))
        .overlay(RoundedRectangle(cornerRadius: 18, style: .continuous).stroke(colors.accent.opacity(0.2)))
    }

    // MARK: - Share

    /// Renders the summary card to an image, the counterpart of the React
    /// Native `ViewShot` capture.
    private func share(_ reading: Reading, _ spread: Spread, _ entries: [DrawnCard], _ l: Localizer) {
        let card = ReadingSummaryCard(reading: reading, spread: spread, entries: entries, localizer: l, onSelect: nil)
            .frame(width: 390)
            .padding(16)
            .background(colors.bg)
            .environment(\.appColors, colors)
        let renderer = ImageRenderer(content: card)
        renderer.scale = displayScale
        guard let image = renderer.uiImage else {
            shareFailed = true
            return
        }
        shareImage = ShareableImage(image: image)
    }
}

struct ShareableImage: Identifiable {
    let id = UUID()
    let image: UIImage
}

/// The shareable part of a reading: title, date, summary and every card.
struct ReadingSummaryCard: View {
    @Environment(\.appColors) private var colors

    let reading: Reading
    let spread: Spread
    let entries: [DrawnCard]
    let localizer: Localizer
    let onSelect: ((DrawnCard) -> Void)?

    var body: some View {
        let l = localizer
        VStack(alignment: .leading, spacing: 16) {
            VStack(alignment: .leading, spacing: 4) {
                Text(l.t(spread.nameKey))
                    .font(.title2.weight(.bold))
                    .foregroundStyle(colors.gold)
                Text(reading.drawnAt.formatted(
                    Date.FormatStyle(date: .long, time: .shortened).locale(Locale(identifier: l.language.rawValue))
                ))
                .font(.footnote)
                .foregroundStyle(colors.muted)
            }

            if !entries.isEmpty {
                VStack(alignment: .leading, spacing: 6) {
                    Text(l.t("interpretation.summaryTitle"))
                        .font(.headline)
                        .foregroundStyle(colors.text)
                    Text(Interpretation.summary(spread: spread, entries: entries, localizer: l))
                        .foregroundStyle(colors.text)
                        .wrapsText()
                }
            }

            ForEach(entries, id: \.position.index) { entry in
                VStack(alignment: .leading, spacing: 8) {
                    Text(l.t(entry.position.titleKey))
                        .font(.subheadline.weight(.semibold))
                        .foregroundStyle(colors.text)
                    Text(l.t(entry.position.descriptionKey))
                        .font(.footnote)
                        .foregroundStyle(colors.muted)
                        .wrapsText()
                    HStack(alignment: .top, spacing: 14) {
                        CardImage(file: entry.card.imageFile, width: 84)
                            .rotationEffect(.degrees(entry.isReversed ? 180 : 0))
                            .onTapGesture { onSelect?(entry) }
                        VStack(alignment: .leading, spacing: 6) {
                            Text(entry.card.name + (entry.isReversed ? " " + l.t("reading.reversed") : ""))
                                .font(.subheadline.weight(.semibold))
                                .foregroundStyle(colors.gold)
                            Text(entry.card.meaning(reversed: entry.isReversed))
                                .font(.subheadline)
                                .foregroundStyle(colors.text)
                                .wrapsText()
                        }
                    }
                }
            }
        }
        .padding(18)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(colors.panel, in: RoundedRectangle(cornerRadius: 20, style: .continuous))
        .overlay(RoundedRectangle(cornerRadius: 20, style: .continuous).stroke(colors.gold.opacity(0.2)))
    }
}

/// UIKit's share sheet.
struct ShareSheet: UIViewControllerRepresentable {
    let items: [Any]

    func makeUIViewController(context: Context) -> UIActivityViewController {
        UIActivityViewController(activityItems: items, applicationActivities: nil)
    }

    func updateUIViewController(_ controller: UIActivityViewController, context: Context) {}
}

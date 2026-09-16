import SwiftUI

/// Screens pushed from the home and history tabs.
enum ReadingRoute: Hashable {
    case catalog
    /// `dailyKind` is set when a one-card draw has already been cleared by the
    /// daily quota; for every other spread it is nil.
    case reading(spreadId: String, deckId: String, dailyKind: Reading.Kind?)
    case interpretation(readingId: String)
}

extension View {
    /// Registers every `ReadingRoute` destination on a navigation stack.
    func readingDestinations(path: Binding<[ReadingRoute]>) -> some View {
        navigationDestination(for: ReadingRoute.self) { route in
            Group {
                switch route {
                case .catalog:
                    SpreadCatalogView(path: path)
                case let .reading(spreadId, deckId, dailyKind):
                    ReadingView(spreadId: spreadId, deckId: deckId, dailyKind: dailyKind, path: path)
                case let .interpretation(readingId):
                    InterpretationView(readingId: readingId, path: path)
                }
            }
            .background(AppBackground())
        }
    }
}

/// Starts a one-card draw, going through the daily quota.
@MainActor
@Observable
final class DailyDrawFlow {
    var busy = false
    var showingLimit = false
    var showingError = false

    func start(
        preferExisting: Bool,
        path: Binding<[ReadingRoute]>,
        history: HistoryStore,
        services: AppServices,
        hasPremium: Bool,
        deckId: String
    ) async {
        let today = history.todaysDailyCard()
        if preferExisting, let today {
            path.wrappedValue.append(.interpretation(readingId: today.id))
            return
        }
        busy = true
        defer { busy = false }
        do {
            let kind = try await services.dailyCards.consume(hasTodaysCard: today != nil, hasPremium: hasPremium)
            path.wrappedValue.append(.reading(spreadId: Spread.oneCardId, deckId: deckId, dailyKind: kind))
        } catch let error as APIError where error.code == "quota_exceeded" {
            showingLimit = true
        } catch is CancellationError {
            return
        } catch {
            showingError = true
        }
    }
}

/// Shown when today's draws are used up.
///
/// The React Native version offered a video to unlock another card; that flow
/// is not ported (see `UsageAPI`), so the sheet offers today's card and
/// premium instead.
struct DailyLimitSheet: View {
    @Environment(SettingsStore.self) private var settings
    @Environment(HistoryStore.self) private var history
    @Environment(\.appColors) private var colors
    @Environment(\.dismiss) private var dismiss
    @Environment(\.selectTab) private var selectTab

    @Binding var path: [ReadingRoute]

    var body: some View {
        let l = settings.localizer
        VStack(spacing: 14) {
            Text(l.t("dailyCard.limitTitle"))
                .font(.title3.weight(.bold))
                .foregroundStyle(colors.gold)
                .multilineTextAlignment(.center)
            Text(l.t("dailyCard.quotaReached"))
                .foregroundStyle(colors.text)
                .multilineTextAlignment(.center)
                .wrapsText()

            if let today = history.todaysDailyCard() {
                PrimaryButton(title: l.t("dailyCard.ctaOpen")) {
                    dismiss()
                    path.append(.interpretation(readingId: today.id))
                }
            }
            Button {
                dismiss()
                selectTab(.settings)
            } label: {
                Text(l.t("dailyCard.subscribe"))
                    .font(.body.weight(.semibold))
                    .foregroundStyle(colors.bg)
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 14)
                    .background(colors.gold, in: RoundedRectangle(cornerRadius: 16, style: .continuous))
            }
            .buttonStyle(.plain)
            LinkButton(title: l.t("premium.notNow")) { dismiss() }
        }
        .padding(24)
        .background(colors.panel.ignoresSafeArea())
        .presentationDetents([.medium])
    }
}

import SwiftUI

/// The reading journal. Local to the device for now; syncing with the server's
/// readings API comes next.
struct HistoryView: View {
    @Environment(SettingsStore.self) private var settings
    @Environment(HistoryStore.self) private var history
    @Environment(\.appColors) private var colors

    private enum Filter: Hashable { case all, favorites }

    @State private var path: [ReadingRoute] = []
    @State private var filter: Filter = .all
    @State private var pendingDelete: Reading?
    @State private var confirmingClear = false

    private var visible: [Reading] {
        filter == .favorites ? history.readings.filter(\.favorite) : history.readings
    }

    var body: some View {
        let l = settings.localizer
        NavigationStack(path: $path) {
            VStack(spacing: 12) {
                HStack {
                    FlowChips(options: [Filter.all, .favorites], selected: filter,
                              label: { l.t($0 == .all ? "history.filter.all" : "history.filter.favorites") }) {
                        filter = $0
                    }
                    .frame(maxWidth: 240)
                    Spacer()
                    if !history.readings.isEmpty {
                        Button(l.t("history.clearAll")) { confirmingClear = true }
                            .font(.subheadline.weight(.semibold))
                            .foregroundStyle(colors.danger)
                    }
                }
                .padding(.horizontal, 20)

                if visible.isEmpty {
                    VStack(spacing: 8) {
                        Image(systemName: "moon.stars")
                            .font(.system(size: 40, weight: .light))
                            .foregroundStyle(colors.gold)
                        Text(l.t("history.emptyTitle"))
                            .font(.headline)
                            .foregroundStyle(colors.text)
                        Text(l.t("history.emptySubtitle"))
                            .font(.subheadline)
                            .foregroundStyle(colors.muted)
                            .multilineTextAlignment(.center)
                    }
                    .padding(32)
                    .frame(maxHeight: .infinity)
                } else {
                    List {
                        ForEach(visible) { reading in
                            row(reading, l)
                                .listRowBackground(Color.clear)
                                .listRowSeparator(.hidden)
                                .listRowInsets(EdgeInsets(top: 6, leading: 20, bottom: 6, trailing: 20))
                                .swipeActions {
                                    Button(l.t("history.delete"), role: .destructive) {
                                        pendingDelete = reading
                                    }
                                }
                        }
                    }
                    .listStyle(.plain)
                    .scrollContentBackground(.hidden)
                }
            }
            .padding(.top, 8)
            .background(AppBackground())
            .navigationTitle(l.t("nav.history"))
            .toolbarBackground(colors.bg, for: .navigationBar)
            .toolbarColorScheme(.dark, for: .navigationBar)
            .readingDestinations(path: $path)
        }
        .tint(colors.gold)
        .confirmationDialog(l.t("history.deleteTitle"), isPresented: Binding(
            get: { pendingDelete != nil }, set: { if !$0 { pendingDelete = nil } }
        ), titleVisibility: .visible) {
            Button(l.t("history.confirm"), role: .destructive) {
                if let pendingDelete { history.remove(id: pendingDelete.id) }
            }
            Button(l.t("history.cancel"), role: .cancel) {}
        } message: {
            Text(l.t("history.deleteDescription"))
        }
        .confirmationDialog(l.t("history.clearTitle"), isPresented: $confirmingClear, titleVisibility: .visible) {
            Button(l.t("history.confirm"), role: .destructive) { history.removeAll() }
            Button(l.t("history.cancel"), role: .cancel) {}
        } message: {
            Text(l.t("history.clearDescription"))
        }
    }

    private func row(_ reading: Reading, _ l: Localizer) -> some View {
        Button {
            path.append(.interpretation(readingId: reading.id))
        } label: {
            HStack(alignment: .top, spacing: 12) {
                VStack(alignment: .leading, spacing: 4) {
                    Text(Spread.find(reading.spreadId).map { l.t($0.nameKey) } ?? reading.spreadId)
                        .font(.headline)
                        .foregroundStyle(colors.gold)
                    Text(reading.drawnAt.formatted(
                        Date.FormatStyle(date: .abbreviated, time: .shortened).locale(Locale(identifier: l.language.rawValue))
                    ))
                    .font(.caption)
                    .foregroundStyle(colors.muted)
                    Text(Interpretation.displaySummary(for: reading, catalog: settings.catalog, localizer: l))
                        .font(.subheadline)
                        .foregroundStyle(colors.text)
                        .lineLimit(3)
                        .multilineTextAlignment(.leading)
                }
                Spacer(minLength: 0)
                Button {
                    history.toggleFavorite(id: reading.id)
                } label: {
                    Image(systemName: reading.favorite ? "star.fill" : "star")
                        .font(.title3)
                        .foregroundStyle(colors.gold)
                }
                .buttonStyle(.plain)
                .accessibilityLabel(l.t(reading.favorite ? "interpretation.unfavorite" : "interpretation.favorite"))
            }
            .padding(16)
            .background(colors.panel, in: RoundedRectangle(cornerRadius: 18, style: .continuous))
            .overlay(RoundedRectangle(cornerRadius: 18, style: .continuous).stroke(colors.gold.opacity(0.2)))
        }
        .buttonStyle(.plain)
    }
}

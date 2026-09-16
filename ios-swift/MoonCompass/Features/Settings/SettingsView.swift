import SwiftUI

/// Settings, ported so far: account, premium and language.
/// Animations and the reversed-card chance follow with those features.
struct SettingsView: View {
    @Environment(SettingsStore.self) private var settings
    @Environment(SessionStore.self) private var session
    @Environment(\.appColors) private var colors
    @Environment(\.showPremium) private var showPremium

    @State private var confirmingLogout = false
    @State private var confirmingDelete = false
    @State private var deleting = false
    @State private var deleteFailed = false
    @State private var legalDocument: LegalDocument?

    var body: some View {
        let l = settings.localizer
        NavigationStack {
            ScrollView {
                VStack(alignment: .leading, spacing: 20) {
                    accountSection(l)
                    premiumSection(l)
                    languageSection(l)
                }
                .padding(20)
            }
            .background(AppBackground())
            .navigationTitle(l.t("nav.settings"))
            .toolbarBackground(colors.bg, for: .navigationBar)
            .toolbarColorScheme(.dark, for: .navigationBar)
            .navigationDestination(item: $legalDocument) { document in
                LegalDocumentView(document: document)
                    .background(AppBackground())
            }
        }
        .confirmationDialog(l.t("auth.logoutTitle"), isPresented: $confirmingLogout, titleVisibility: .visible) {
            Button(l.t("auth.logoutCta")) {
                Task { await session.logout() }
            }
            Button(l.t("history.cancel"), role: .cancel) {}
        } message: {
            Text(l.t("auth.logoutDescription"))
        }
        .confirmationDialog(l.t("auth.deleteTitle"), isPresented: $confirmingDelete, titleVisibility: .visible) {
            Button(l.t("auth.deleteCta"), role: .destructive) {
                Task { await deleteAccount() }
            }
            Button(l.t("history.cancel"), role: .cancel) {}
        } message: {
            Text(l.t("auth.deleteDescription"))
        }
        .alert(l.t("auth.deleteTitle"), isPresented: $deleteFailed) {
            Button("OK", role: .cancel) {}
        } message: {
            Text(l.t("auth.errors.generic"))
        }
    }

    private func accountSection(_ l: Localizer) -> some View {
        SettingsCard(title: l.t("settings.accountSection")) {
            if let user = session.user {
                Text(user.email)
                    .font(.body.weight(.semibold))
                    .foregroundStyle(colors.text)
            }
            HStack(spacing: 16) {
                Button(l.t("auth.termsLink")) { legalDocument = .terms }
                Button(l.t("auth.privacyLink")) { legalDocument = .privacy }
            }
            .font(.subheadline.weight(.semibold))
            .foregroundStyle(colors.accent)
            .buttonStyle(.plain)

            Button {
                confirmingLogout = true
            } label: {
                Text(l.t("auth.logoutCta"))
                    .font(.body.weight(.semibold))
                    .foregroundStyle(colors.gold)
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 12)
                    .overlay(RoundedRectangle(cornerRadius: 12).stroke(colors.gold.opacity(0.5)))
            }
            .buttonStyle(.plain)

            Button {
                confirmingDelete = true
            } label: {
                ZStack {
                    Text(l.t("auth.deleteCta")).opacity(deleting ? 0 : 1)
                    if deleting { ProgressView().tint(colors.danger) }
                }
                .font(.body.weight(.semibold))
                .foregroundStyle(colors.danger)
                .frame(maxWidth: .infinity)
                .padding(.vertical, 12)
                .overlay(RoundedRectangle(cornerRadius: 12).stroke(colors.danger.opacity(0.5)))
            }
            .buttonStyle(.plain)
            .disabled(deleting)
        }
    }

    private func premiumSection(_ l: Localizer) -> some View {
        let user = session.user
        let active = user?.hasPremium ?? false
        return SettingsCard(title: l.t("settings.premiumSection")) {
            HStack {
                Text(l.t("premium.title"))
                    .font(.body.weight(.semibold))
                    .foregroundStyle(colors.text)
                Spacer()
                Text(l.t(active ? "premium.status.active" : "premium.status.inactive"))
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(active ? colors.bg : colors.muted)
                    .padding(.horizontal, 10)
                    .padding(.vertical, 4)
                    .background(active ? colors.gold : colors.text.opacity(0.08), in: Capsule())
            }
            if let user, active {
                Text(PremiumView.expiryText(user, l))
                    .font(.subheadline)
                    .foregroundStyle(colors.muted)
            } else {
                Text(l.t("premium.benefits"))
                    .font(.subheadline)
                    .foregroundStyle(colors.muted)
                    .wrapsText()
            }
            PrimaryButton(title: l.t(active ? "premium.manage" : "premium.openSettings")) {
                showPremium()
            }
        }
    }

    private func languageSection(_ l: Localizer) -> some View {
        SettingsCard(title: l.t("settings.languageTitle")) {
            FlowChips(
                options: Language.allCases,
                selected: settings.settings.language,
                label: { $0.nativeName }
            ) { language in
                settings.setLanguage(language)
            }
        }
    }

    private func deleteAccount() async {
        deleting = true
        defer { deleting = false }
        do {
            try await session.deleteAccount()
        } catch {
            deleteFailed = true
        }
    }
}

struct SettingsCard<Content: View>: View {
    @Environment(\.appColors) private var colors

    let title: String
    @ViewBuilder let content: Content

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            Text(title)
                .font(.headline)
                .foregroundStyle(colors.gold)
            content
        }
        .padding(16)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(colors.panel.opacity(0.85), in: RoundedRectangle(cornerRadius: 18, style: .continuous))
        .overlay(RoundedRectangle(cornerRadius: 18, style: .continuous).stroke(colors.gold.opacity(0.15)))
    }
}

/// Selectable chips that wrap onto new lines when they don't fit.
struct FlowChips<Option: Hashable>: View {
    @Environment(\.appColors) private var colors

    let options: [Option]
    let selected: Option
    let label: (Option) -> String
    let onSelect: (Option) -> Void

    var body: some View {
        HStack(spacing: 8) {
            ForEach(options, id: \.self) { option in
                let isSelected = option == selected
                Button {
                    onSelect(option)
                } label: {
                    Text(label(option))
                        .font(.subheadline.weight(.semibold))
                        .lineLimit(1)
                        .minimumScaleFactor(0.8)
                        .foregroundStyle(isSelected ? .white : colors.text)
                        .padding(.horizontal, 12)
                        .padding(.vertical, 8)
                        .frame(maxWidth: .infinity)
                        .background(
                            isSelected ? colors.accent : colors.text.opacity(0.08),
                            in: Capsule()
                        )
                }
                .buttonStyle(.plain)
                .accessibilityAddTraits(isSelected ? .isSelected : [])
            }
        }
    }
}

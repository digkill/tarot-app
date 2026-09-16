import StoreKit
import SwiftUI

/// Premium plans, bought through the App Store.
///
/// Prices come from StoreKit only: Apple shows each storefront its own price
/// and currency, so nothing here is hard-coded. The subscription terms, the
/// restore button and the terms/privacy links are what guideline 3.1.2 asks a
/// subscription screen to show.
struct PremiumView: View {
    @Environment(SettingsStore.self) private var settings
    @Environment(SessionStore.self) private var session
    @Environment(PurchaseStore.self) private var purchases
    @Environment(\.appColors) private var colors
    @Environment(\.dismiss) private var dismiss

    @State private var selectedPlan = ProductIDs.premiumYearly
    @State private var message: (text: String, tone: StatusMessage.Tone)?
    @State private var legalDocument: LegalDocument?
    @State private var managingSubscription = false

    private var user: User? { session.user }

    var body: some View {
        let l = settings.localizer
        NavigationStack {
            ScrollView {
                VStack(alignment: .leading, spacing: 18) {
                    header(l)
                    features(l)
                    if let user, user.hasPremium {
                        status(user, l)
                    } else {
                        plans(l)
                    }
                    if let message {
                        StatusMessage(text: message.text, tone: message.tone)
                    }
                    footer(l)
                }
                .padding(20)
            }
            .background(AppBackground())
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button(l.t("premium.cancel")) { dismiss() }
                        .foregroundStyle(colors.text)
                }
            }
            .toolbarBackground(colors.bg, for: .navigationBar)
            .toolbarColorScheme(.dark, for: .navigationBar)
            .navigationDestination(item: $legalDocument) { document in
                LegalDocumentView(document: document)
                    .background(AppBackground())
            }
        }
        .manageSubscriptionsSheet(isPresented: $managingSubscription)
        .task { await purchases.loadProducts(ProductIDs.premium) }
    }

    // MARK: - Sections

    private func header(_ l: Localizer) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            Image(systemName: "moon.stars.fill")
                .font(.system(size: 40))
                .foregroundStyle(colors.gold)
            Text(l.t("premium.title"))
                .font(.largeTitle.weight(.bold))
                .foregroundStyle(colors.gold)
            Text(l.t("premium.benefits"))
                .foregroundStyle(colors.text)
                .wrapsText()
        }
    }

    private func features(_ l: Localizer) -> some View {
        SettingsCard(title: l.t("premium.features.title")) {
            ForEach(["premium.features.aiInterpretations", "premium.features.deeperInsights",
                     "premiumIos.dailyCards", "premium.features.unlimitedReadings",
                     "premium.features.prioritySupport"], id: \.self) { key in
                HStack(alignment: .firstTextBaseline, spacing: 10) {
                    Image(systemName: "checkmark.seal.fill")
                        .foregroundStyle(colors.accent)
                    Text(l.t(key))
                        .font(.subheadline)
                        .foregroundStyle(colors.text)
                        .wrapsText()
                }
            }
        }
    }

    private func plans(_ l: Localizer) -> some View {
        VStack(alignment: .leading, spacing: 12) {
            Text(l.t("premium.selectPlan"))
                .font(.headline)
                .foregroundStyle(colors.gold)

            let available = ProductIDs.premium.compactMap { purchases.products[$0] }
            if available.isEmpty {
                if purchases.loadingProducts {
                    ProgressView().tint(colors.accent).frame(maxWidth: .infinity).padding()
                } else {
                    StatusMessage(text: l.t("premiumIos.productsUnavailable"))
                    LinkButton(title: l.t("aiInterpretation.retry")) {
                        Task { await purchases.loadProducts(ProductIDs.premium) }
                    }
                }
            } else {
                ForEach(available, id: \.id) { product in
                    planRow(product, l)
                }
                PrimaryButton(
                    title: l.t("premium.subscribe"),
                    isLoading: purchases.purchasingId != nil,
                    isEnabled: purchases.products[selectedPlan] != nil && !purchases.restoring
                ) {
                    Task { await buy(selectedPlan) }
                }
            }
        }
    }

    private func planRow(_ product: Product, _ l: Localizer) -> some View {
        let plan = Self.planKey(product.id)
        let isSelected = selectedPlan == product.id
        return Button {
            selectedPlan = product.id
        } label: {
            HStack(spacing: 12) {
                Image(systemName: isSelected ? "largecircle.fill.circle" : "circle")
                    .font(.title3)
                    .foregroundStyle(isSelected ? colors.accent : colors.muted)
                VStack(alignment: .leading, spacing: 2) {
                    Text(l.t("premium.plans.\(plan).name"))
                        .font(.headline)
                        .foregroundStyle(colors.text)
                    Text(l.t("premium.plans.\(plan).period"))
                        .font(.caption)
                        .foregroundStyle(colors.muted)
                }
                Spacer(minLength: 8)
                Text(product.displayPrice)
                    .font(.headline)
                    .foregroundStyle(colors.gold)
            }
            .padding(14)
            .background(isSelected ? colors.accent.opacity(0.18) : colors.panel,
                        in: RoundedRectangle(cornerRadius: 16, style: .continuous))
            .overlay(RoundedRectangle(cornerRadius: 16, style: .continuous)
                .stroke(isSelected ? colors.accent : colors.gold.opacity(0.25), lineWidth: isSelected ? 2 : 1))
        }
        .buttonStyle(.plain)
        .accessibilityAddTraits(isSelected ? .isSelected : [])
    }

    private func status(_ user: User, _ l: Localizer) -> some View {
        SettingsCard(title: l.t("settings.premiumActive")) {
            Text(Self.expiryText(user, l))
                .foregroundStyle(colors.text)
            if let source = user.premiumSource, l.has("premium.source.\(source)") {
                Text(l.t("premium.source.\(source)"))
                    .font(.caption)
                    .foregroundStyle(colors.muted)
            }
            if user.premiumSource == "apple" {
                if user.premiumProductId != "premium_lifetime" {
                    PrimaryButton(title: l.t("premium.manage")) { managingSubscription = true }
                }
            } else if user.premiumSource != nil {
                Text(l.t("premiumIos.managedElsewhereTitle"))
                    .font(.subheadline.weight(.semibold))
                    .foregroundStyle(colors.gold)
                Text(l.t("premiumIos.managedElsewhereBody"))
                    .font(.subheadline)
                    .foregroundStyle(colors.text)
                    .wrapsText()
            }
        }
    }

    private func footer(_ l: Localizer) -> some View {
        VStack(alignment: .leading, spacing: 12) {
            LinkButton(title: l.t("premium.restore"), isEnabled: !purchases.restoring && purchases.purchasingId == nil) {
                Task { await restore() }
            }
            .frame(maxWidth: .infinity)

            Text(l.t("premiumIos.disclosure"))
                .font(.caption)
                .foregroundStyle(colors.muted)
                .wrapsText()

            HStack(spacing: 16) {
                Button(l.t("auth.termsLink")) { legalDocument = .terms }
                Button(l.t("auth.privacyLink")) { legalDocument = .privacy }
            }
            .font(.caption.weight(.semibold))
            .foregroundStyle(colors.accent)
            .buttonStyle(.plain)
        }
    }

    // MARK: - Actions

    private func buy(_ productId: String) async {
        let l = settings.localizer
        message = nil
        let outcome = await purchases.purchase(productId)
        if case .purchased = outcome {
            await session.refreshUser()
            message = (l.t("premium.purchaseSuccess"), .info)
        } else if let key = PurchaseStore.messageKey(for: outcome) {
            message = (l.t(key), outcome == .pending ? .info : .error)
        }
    }

    private func restore() async {
        let l = settings.localizer
        message = nil
        do {
            let granted = try await purchases.restore()
            await session.refreshUser()
            if session.user?.hasPremium == true {
                message = (l.t("premium.restoreSuccess"), .info)
            } else {
                message = (l.t(granted.isEmpty ? "premium.restoreNone" : "premiumIos.restoreDone"), .info)
            }
        } catch {
            // Cancelling the Apple Account prompt lands here too.
            message = (l.t("premium.purchaseError"), .error)
        }
    }

    // MARK: - Formatting

    static func planKey(_ productId: String) -> String {
        switch productId {
        case ProductIDs.premiumMonthly: "monthly"
        case ProductIDs.premiumYearly: "yearly"
        default: "lifetime"
        }
    }

    static func expiryText(_ user: User, _ l: Localizer) -> String {
        guard let expires = user.premiumExpiresAt, user.premiumProductId != "premium_lifetime" else {
            return l.t("premium.lifetimeAccess")
        }
        let date = expires.formatted(Date.FormatStyle(date: .long, time: .omitted)
            .locale(Locale(identifier: l.language.rawValue)))
        return l.t("premium.expiresOn", ["date": date])
    }
}

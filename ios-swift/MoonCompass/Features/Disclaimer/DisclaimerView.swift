import SwiftUI

/// Shown once, before anything else: readings are for entertainment and
/// reflection, not medical, legal or financial advice.
struct DisclaimerView: View {
    @Environment(SettingsStore.self) private var settings
    @Environment(\.appColors) private var colors

    var body: some View {
        let l = settings.localizer
        VStack(spacing: 0) {
            ScrollView {
                VStack(alignment: .leading, spacing: 16) {
                    Text(l.t("disclaimer.title"))
                        .font(.system(size: 24, weight: .bold))
                        .foregroundStyle(colors.gold)

                    ForEach(["disclaimer.description1", "disclaimer.description2", "disclaimer.nextHint"], id: \.self) { key in
                        Text(l.t(key))
                            .font(.system(size: 16))
                            .lineSpacing(4)
                            .foregroundStyle(colors.text)
                            .wrapsText()
                    }

                    VStack(alignment: .leading, spacing: 8) {
                        Text(l.t("disclaimer.remember"))
                            .font(.system(size: 18, weight: .semibold))
                            .foregroundStyle(colors.accent)
                        Text(l.t("disclaimer.rememberDescription"))
                            .font(.system(size: 15))
                            .lineSpacing(4)
                            .foregroundStyle(colors.text)
                            .wrapsText()
                    }
                    .padding(18)
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .background(colors.accent.opacity(0.1), in: RoundedRectangle(cornerRadius: 16, style: .continuous))
                    .overlay(
                        RoundedRectangle(cornerRadius: 16, style: .continuous)
                            .stroke(colors.accent.opacity(0.4), lineWidth: 1)
                    )
                    .padding(.top, 12)
                }
                .padding(24)
            }

            PrimaryButton(title: l.t("disclaimer.accept")) {
                settings.acceptDisclaimer()
            }
            .padding(24)
        }
    }
}

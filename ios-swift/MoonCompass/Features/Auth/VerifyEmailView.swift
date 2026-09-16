import SwiftUI

struct VerifyEmailView: View {
    @Environment(SettingsStore.self) private var settings
    @Environment(SessionStore.self) private var session
    @Environment(\.appColors) private var colors

    let email: String
    @Binding var path: [AuthRoute]

    @State private var code = ""
    @State private var submitting = false
    @State private var error: String?
    @State private var info: String?

    private var canSubmit: Bool { code.count == 6 && !submitting }

    var body: some View {
        let l = settings.localizer
        ScrollView {
            VStack(alignment: .leading, spacing: 8) {
                Text(l.t("auth.verifyTitle"))
                    .font(.system(size: 28, weight: .bold))
                    .foregroundStyle(colors.gold)
                    .padding(.bottom, 8)
                Text(l.t("auth.verifySubtitle", ["email": email]))
                    .foregroundStyle(colors.text.opacity(0.75))
                    .lineSpacing(4)
                    .wrapsText()
                    .padding(.bottom, 16)

                FormField(label: l.t("auth.code"), text: $code, placeholder: "000000", kind: .code)
                    .padding(.top, 8)

                if let error { StatusMessage(text: error).padding(.top, 8) }
                if let info { StatusMessage(text: info, tone: .info).padding(.top, 8) }

                PrimaryButton(title: l.t("auth.verifyCta"), isLoading: submitting, isEnabled: canSubmit) {
                    Task { await verify() }
                }
                .padding(.top, 16)

                LinkButton(title: l.t("auth.resendCode"), isEnabled: !submitting) {
                    Task { await resend() }
                }
                .padding(.top, 18)

                LinkButton(title: l.t("auth.backToLogin")) {
                    path.removeAll()
                }
                .padding(.top, 18)
            }
            .padding(24)
        }
        .scrollDismissesKeyboard(.interactively)
    }

    private func verify() async {
        guard canSubmit else { return }
        error = nil
        submitting = true
        defer { submitting = false }
        do {
            try await session.verifyEmail(email: email, code: code)
        } catch {
            self.error = settings.localizer.authErrorMessage(error)
        }
    }

    private func resend() async {
        error = nil
        info = nil
        submitting = true
        defer { submitting = false }
        do {
            try await session.resendVerification(email: email, language: settings.settings.language)
            info = settings.localizer.t("auth.codeSent")
        } catch {
            self.error = settings.localizer.authErrorMessage(error)
        }
    }
}

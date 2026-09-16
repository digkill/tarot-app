import SwiftUI

/// Two steps: request a code by email, then set a new password with it.
struct ForgotPasswordView: View {
    @Environment(SettingsStore.self) private var settings
    @Environment(SessionStore.self) private var session
    @Environment(\.appColors) private var colors

    @Binding var path: [AuthRoute]

    private enum Step { case request, reset }

    @State private var email: String
    @State private var code = ""
    @State private var password = ""
    @State private var step: Step = .request
    @State private var submitting = false
    @State private var error: String?
    @State private var info: String?

    init(initialEmail: String, path: Binding<[AuthRoute]>) {
        _email = State(initialValue: initialEmail)
        _path = path
    }

    private var trimmedEmail: String { email.trimmingCharacters(in: .whitespacesAndNewlines) }
    private var canRequest: Bool { trimmedEmail.contains("@") && !submitting }
    private var canReset: Bool { code.count == 6 && password.count >= 8 && !submitting }

    var body: some View {
        let l = settings.localizer
        ScrollView {
            VStack(alignment: .leading, spacing: 8) {
                Text(l.t("auth.forgotTitle"))
                    .font(.system(size: 28, weight: .bold))
                    .foregroundStyle(colors.gold)
                    .padding(.bottom, 8)
                Text(l.t(step == .request ? "auth.forgotSubtitle" : "auth.resetSubtitle"))
                    .foregroundStyle(colors.text.opacity(0.75))
                    .lineSpacing(4)
                    .wrapsText()
                    .padding(.bottom, 16)

                FormField(label: l.t("auth.email"), text: $email, placeholder: l.t("auth.emailPlaceholder"),
                          kind: .email, isEditable: step == .request)
                    .padding(.top, 8)

                if step == .reset {
                    FormField(label: l.t("auth.code"), text: $code, placeholder: "000000", kind: .code)
                        .padding(.top, 8)
                    FormField(label: l.t("auth.newPassword"), text: $password,
                              placeholder: l.t("auth.passwordPlaceholder"), kind: .newPassword)
                        .padding(.top, 8)
                }

                if let error { StatusMessage(text: error).padding(.top, 8) }
                if let info { StatusMessage(text: info, tone: .info).padding(.top, 8) }

                PrimaryButton(
                    title: l.t(step == .request ? "auth.sendResetCta" : "auth.resetCta"),
                    isLoading: submitting,
                    isEnabled: step == .request ? canRequest : canReset
                ) {
                    Task { step == .request ? await sendCode() : await reset() }
                }
                .padding(.top, 16)

                LinkButton(title: l.t("auth.backToLogin")) {
                    path.removeAll()
                }
                .padding(.top, 18)
            }
            .padding(24)
        }
        .scrollDismissesKeyboard(.interactively)
    }

    private func sendCode() async {
        guard canRequest else { return }
        error = nil
        submitting = true
        defer { submitting = false }
        do {
            try await session.forgotPassword(email: trimmedEmail, language: settings.settings.language)
            step = .reset
            info = settings.localizer.t("auth.resetCodeSent")
        } catch {
            self.error = settings.localizer.authErrorMessage(error)
        }
    }

    /// A successful reset signs the user in, which swaps this whole flow out.
    private func reset() async {
        guard canReset else { return }
        error = nil
        submitting = true
        defer { submitting = false }
        do {
            try await session.resetPassword(email: trimmedEmail, code: code, password: password)
        } catch {
            self.error = settings.localizer.authErrorMessage(error)
        }
    }
}

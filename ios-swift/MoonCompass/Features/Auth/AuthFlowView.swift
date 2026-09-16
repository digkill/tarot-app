import SwiftUI

enum LegalDocument: String, Hashable, Sendable {
    case terms
    case privacy
}

enum AuthRoute: Hashable {
    case verifyEmail(email: String)
    case forgotPassword(email: String)
    case legal(LegalDocument)
}

/// Everything a signed-out user can reach. Once a session starts, `RootView`
/// swaps this whole stack out for the main tabs.
struct AuthFlowView: View {
    @Environment(\.appColors) private var colors
    @State private var path: [AuthRoute] = []

    var body: some View {
        NavigationStack(path: $path) {
            AuthView(path: $path)
                .navigationDestination(for: AuthRoute.self) { route in
                    Group {
                        switch route {
                        case let .verifyEmail(email):
                            VerifyEmailView(email: email, path: $path)
                        case let .forgotPassword(email):
                            ForgotPasswordView(initialEmail: email, path: $path)
                        case let .legal(document):
                            LegalDocumentView(document: document)
                        }
                    }
                    .background(AppBackground())
                }
                .background(AppBackground())
        }
        .tint(colors.gold)
    }
}

struct AuthView: View {
    @Environment(SettingsStore.self) private var settings
    @Environment(SessionStore.self) private var session
    @Environment(\.appColors) private var colors

    @Binding var path: [AuthRoute]

    private enum Mode { case login, register }

    @State private var mode: Mode = .login
    @State private var email = ""
    @State private var password = ""
    @State private var confirmPassword = ""
    @State private var acceptedTerms = false
    @State private var acceptedPrivacy = false
    @State private var submitting = false
    @State private var error: String?

    private var trimmedEmail: String { email.trimmingCharacters(in: .whitespacesAndNewlines) }

    private var canSubmit: Bool {
        guard !trimmedEmail.isEmpty, password.count >= 8, !submitting else { return false }
        if mode == .register {
            return password == confirmPassword && acceptedTerms && acceptedPrivacy
        }
        return true
    }

    var body: some View {
        let l = settings.localizer
        ScrollView {
            VStack(alignment: .leading, spacing: 8) {
                Text(l.t(mode == .login ? "auth.loginTitle" : "auth.registerTitle"))
                    .font(.system(size: 28, weight: .bold))
                    .foregroundStyle(colors.gold)
                    .padding(.bottom, 8)
                Text(l.t(mode == .login ? "auth.loginSubtitle" : "auth.registerSubtitle"))
                    .foregroundStyle(colors.text.opacity(0.75))
                    .lineSpacing(4)
                    .wrapsText()
                    .padding(.bottom, 16)

                FormField(label: l.t("auth.email"), text: $email,
                          placeholder: l.t("auth.emailPlaceholder"), kind: .email)
                    .padding(.top, 8)
                FormField(label: l.t("auth.password"), text: $password,
                          placeholder: l.t("auth.passwordPlaceholder"),
                          kind: mode == .register ? .newPassword : .password)
                    .padding(.top, 8)

                if mode == .register {
                    registerFields(l)
                }

                if let error {
                    StatusMessage(text: error).padding(.top, 8)
                }

                PrimaryButton(
                    title: l.t(mode == .login ? "auth.loginCta" : "auth.registerCta"),
                    isLoading: submitting,
                    isEnabled: canSubmit
                ) {
                    Task { await submit() }
                }
                .padding(.top, 16)

                if mode == .login {
                    LinkButton(title: l.t("auth.forgotLink")) {
                        path.append(.forgotPassword(email: trimmedEmail))
                    }
                    .padding(.top, 18)
                }

                LinkButton(title: l.t(mode == .login ? "auth.needAccount" : "auth.haveAccount")) {
                    mode = mode == .login ? .register : .login
                    error = nil
                }
                .padding(.top, 18)
            }
            .padding(24)
            .padding(.bottom, 24)
        }
        .scrollDismissesKeyboard(.interactively)
        .toolbar(.hidden, for: .navigationBar)
    }

    @ViewBuilder
    private func registerFields(_ l: Localizer) -> some View {
        FormField(label: l.t("auth.confirmPassword"), text: $confirmPassword,
                  placeholder: l.t("auth.confirmPasswordPlaceholder"), kind: .newPassword)
            .padding(.top, 8)
        if !password.isEmpty, password.count < 8 {
            StatusMessage(text: l.t("auth.passwordHint"))
        }
        if !confirmPassword.isEmpty, confirmPassword != password {
            StatusMessage(text: l.t("auth.passwordMismatch"))
        }

        ConsentRow(isOn: $acceptedTerms, prefix: l.t("auth.acceptTermsPrefix"),
                   linkTitle: l.t("auth.termsLink")) {
            path.append(.legal(.terms))
        }
        ConsentRow(isOn: $acceptedPrivacy, prefix: l.t("auth.acceptPrivacyPrefix"),
                   linkTitle: l.t("auth.privacyLink")) {
            path.append(.legal(.privacy))
        }

        Text(l.t("auth.ageNote"))
            .font(.system(size: 13))
            .foregroundStyle(colors.text.opacity(0.6))
            .wrapsText()
            .padding(.top, 4)
    }

    private func submit() async {
        guard canSubmit else { return }
        let l = settings.localizer
        error = nil
        submitting = true
        defer { submitting = false }

        switch mode {
        case .login:
            do {
                try await session.login(email: trimmedEmail, password: password)
            } catch let apiError as APIError where apiError.code == "email_unverified" {
                path.append(.verifyEmail(email: trimmedEmail))
            } catch {
                self.error = l.authErrorMessage(error)
            }
        case .register:
            do {
                try await session.register(
                    email: trimmedEmail, password: password, language: settings.settings.language,
                    acceptedTerms: acceptedTerms, acceptedPrivacy: acceptedPrivacy
                )
                path.append(.verifyEmail(email: trimmedEmail))
            } catch let apiError as APIError where apiError.code == "mail_failed" || apiError.code == "code_cooldown" {
                // The account exists; the user can still ask for a new code.
                path.append(.verifyEmail(email: trimmedEmail))
            } catch {
                self.error = l.authErrorMessage(error)
            }
        }
    }
}

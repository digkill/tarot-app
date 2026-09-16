import SwiftUI

/// The app-wide backdrop: the palette's background with the printed pattern
/// tiled over it at low opacity, as in the React Native `AppBackground`.
struct AppBackground: View {
    @Environment(\.appColors) private var colors

    var body: some View {
        ZStack {
            colors.bg
            Image("Pattern")
                .resizable(resizingMode: .tile)
                .opacity(0.22)
        }
        .ignoresSafeArea()
        .accessibilityHidden(true)
    }
}

struct PrimaryButton: View {
    @Environment(\.appColors) private var colors

    let title: String
    var isLoading = false
    var isEnabled = true
    let action: () -> Void

    var body: some View {
        Button(action: action) {
            ZStack {
                Text(title)
                    .font(.system(size: 18, weight: .semibold))
                    .opacity(isLoading ? 0 : 1)
                if isLoading {
                    ProgressView().tint(.white)
                }
            }
            .foregroundStyle(.white)
            .frame(maxWidth: .infinity)
            .padding(.vertical, 16)
            .background(colors.accent, in: RoundedRectangle(cornerRadius: 16, style: .continuous))
            .opacity(isEnabled ? 1 : 0.45)
        }
        .buttonStyle(.plain)
        .disabled(!isEnabled || isLoading)
    }
}

/// A secondary, text-only action in the gold accent ("Forgot password?").
struct LinkButton: View {
    @Environment(\.appColors) private var colors

    let title: String
    var isEnabled = true
    let action: () -> Void

    var body: some View {
        Button(action: action) {
            Text(title)
                .font(.body.weight(.semibold))
                .foregroundStyle(colors.gold)
                .frame(maxWidth: .infinity)
                .multilineTextAlignment(.center)
        }
        .buttonStyle(.plain)
        .disabled(!isEnabled)
        .opacity(isEnabled ? 1 : 0.5)
    }
}

enum FieldKind {
    case email
    case password
    case newPassword
    case code

    var isSecure: Bool { self == .password || self == .newPassword }
}

struct FormField: View {
    @Environment(\.appColors) private var colors

    let label: String
    @Binding var text: String
    var placeholder = ""
    var kind: FieldKind = .email
    var isEditable = true

    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            Text(label)
                .font(.body.weight(.semibold))
                .foregroundStyle(colors.text)
            field
                .font(.system(size: 16))
                .foregroundStyle(colors.text)
                .tint(colors.accent)
                .padding(.horizontal, 14)
                .padding(.vertical, 12)
                .background(colors.text.opacity(0.08), in: RoundedRectangle(cornerRadius: 14, style: .continuous))
                .overlay(
                    RoundedRectangle(cornerRadius: 14, style: .continuous)
                        .stroke(colors.gold.opacity(0.25), lineWidth: 1)
                )
                .opacity(isEditable ? 1 : 0.6)
                .disabled(!isEditable)
        }
    }

    @ViewBuilder
    private var field: some View {
        let prompt = Text(placeholder).foregroundStyle(colors.text.opacity(0.4))
        switch kind {
        case .email:
            TextField("", text: $text, prompt: prompt)
                .keyboardType(.emailAddress)
                .textContentType(.emailAddress)
                .textInputAutocapitalization(.never)
                .autocorrectionDisabled()
        case .password:
            SecureField("", text: $text, prompt: prompt)
                .textContentType(.password)
        case .newPassword:
            SecureField("", text: $text, prompt: prompt)
                .textContentType(.newPassword)
        case .code:
            TextField("", text: $text, prompt: prompt)
                .keyboardType(.numberPad)
                .textContentType(.oneTimeCode)
                .onChange(of: text) { _, newValue in
                    // Six digits only, as the server expects.
                    let digits = String(newValue.filter(\.isNumber).prefix(6))
                    if digits != newValue { text = digits }
                }
        }
    }
}

/// An error or confirmation line under a form.
struct StatusMessage: View {
    @Environment(\.appColors) private var colors

    enum Tone { case error, info }

    let text: String
    var tone: Tone = .error

    var body: some View {
        Text(text)
            .font(.subheadline)
            .foregroundStyle(tone == .error ? colors.danger : colors.gold)
            .frame(maxWidth: .infinity, alignment: .leading)
            .wrapsText()
    }
}

/// A consent checkbox whose label ends in a tappable link to the document.
struct ConsentRow: View {
    @Environment(\.appColors) private var colors

    @Binding var isOn: Bool
    let prefix: String
    let linkTitle: String
    let openLink: () -> Void

    var body: some View {
        HStack(alignment: .top, spacing: 10) {
            Button {
                isOn.toggle()
            } label: {
                Image(systemName: isOn ? "checkmark.square.fill" : "square")
                    .font(.system(size: 22))
                    .foregroundStyle(isOn ? colors.accent : colors.text.opacity(0.7))
            }
            .buttonStyle(.plain)
            .accessibilityAddTraits(.isToggle)
            .accessibilityValue(isOn ? Text("✓") : Text(""))

            Text(attributedLabel)
                .font(.system(size: 14))
                .foregroundStyle(colors.text)
                .wrapsText()
                .environment(\.openURL, OpenURLAction { _ in
                    openLink()
                    return .handled
                })
        }
        .padding(.vertical, 8)
    }

    private var attributedLabel: AttributedString {
        var label = AttributedString(prefix)
        var link = AttributedString(linkTitle)
        // The URL only marks the run as a link; the action is `openLink`.
        link.link = URL(string: "moon-compass://legal")
        link.foregroundColor = colors.accent
        link.inlinePresentationIntent = .stronglyEmphasized
        label.append(link)
        return label
    }
}

extension View {
    /// Lets multi-line text wrap instead of truncating inside stacks.
    func wrapsText() -> some View {
        fixedSize(horizontal: false, vertical: true)
    }
}

extension Localizer {
    /// Maps a failed account request to a message, preferring a specific
    /// `auth.errors.<code>` translation when one exists — the same lookup the
    /// React Native screens use.
    func authErrorMessage(_ error: any Error) -> String {
        if let apiError = error as? APIError {
            let key = "auth.errors.\(apiError.code)"
            if has(key) {
                return t(key)
            }
        }
        return t("auth.errors.generic")
    }
}

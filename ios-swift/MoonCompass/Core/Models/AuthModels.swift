import Foundation

/// Returned by login, verify-email, reset-password and refresh. Every one of
/// them carries a fresh pair that must be persisted.
struct AuthResponse: Codable, Sendable, Equatable {
    var user: User
    var accessToken: String
    var refreshToken: String

    var tokens: AuthTokens {
        AuthTokens(accessToken: accessToken, refreshToken: refreshToken)
    }
}

struct RegisterRequest: Encodable, Sendable, Equatable {
    var email: String
    var password: String
    var language: String
    var acceptedTerms: Bool
    var acceptedPrivacy: Bool
    var acceptedPersonalData: Bool
    var consentVersion: String
}

/// Registration issues no tokens: the account only becomes usable after the
/// emailed code is confirmed via verify-email.
struct RegisterResponse: Decodable, Sendable, Equatable {
    var email: String
    var verificationRequired: Bool
}

struct LoginRequest: Encodable, Sendable {
    var email: String
    var password: String
}

struct VerifyEmailRequest: Encodable, Sendable {
    var email: String
    var code: String
}

/// Shared by resend-verification and forgot-password; `language` picks the
/// language of the email the server sends.
struct EmailLanguageRequest: Encodable, Sendable {
    var email: String
    var language: String
}

struct ResetPasswordRequest: Encodable, Sendable {
    var email: String
    var code: String
    var password: String
}

struct RefreshTokenRequest: Encodable, Sendable {
    var refreshToken: String
}

/// `{"ok":true}`.
struct OKResponse: Decodable, Sendable, Equatable {
    var ok: Bool
}

/// Stand-in response type for calls whose body is ignored (204, or a body the
/// caller does not need). The client never decodes it.
struct EmptyResponse: Decodable, Sendable, Equatable {}

import Foundation

/// Account endpoints. Every response that carries a token pair is persisted
/// here, so screens only ever see the `User`.
struct AuthAPI: Sendable {
    let client: APIClient

    /// Creates the account and sends a verification code. No tokens yet — the
    /// session starts at `verifyEmail`.
    func register(_ request: RegisterRequest) async throws -> RegisterResponse {
        try await client.send(.post, "/api/v1/auth/register", body: request, auth: .none)
    }

    /// Throws `invalid_credentials` (401) or `email_unverified` (403).
    func login(email: String, password: String) async throws -> User {
        try await establishSession(
            client.send(.post, "/api/v1/auth/login", body: LoginRequest(email: email, password: password), auth: .none)
        )
    }

    func verifyEmail(email: String, code: String) async throws -> User {
        try await establishSession(
            client.send(.post, "/api/v1/auth/verify-email", body: VerifyEmailRequest(email: email, code: code), auth: .none)
        )
    }

    /// Throws `code_cooldown` (429, with `retryAfter`) when asked too soon.
    func resendVerification(email: String, language: String) async throws {
        try await client.sendIgnoringResponse(
            .post, "/api/v1/auth/resend-verification",
            body: EmailLanguageRequest(email: email, language: language), auth: .none
        )
    }

    func forgotPassword(email: String, language: String) async throws {
        try await client.sendIgnoringResponse(
            .post, "/api/v1/auth/forgot-password",
            body: EmailLanguageRequest(email: email, language: language), auth: .none
        )
    }

    func resetPassword(email: String, code: String, password: String) async throws -> User {
        try await establishSession(
            client.send(
                .post, "/api/v1/auth/reset-password",
                body: ResetPasswordRequest(email: email, code: code, password: password), auth: .none
            )
        )
    }

    /// Always ends the local session, even if the server cannot be reached: the
    /// user asked to sign out, and an unrevoked refresh token simply expires.
    func logout() async {
        if let refreshToken = await client.currentRefreshToken() {
            try? await client.sendIgnoringResponse(
                .post, "/api/v1/auth/logout",
                body: RefreshTokenRequest(refreshToken: refreshToken), auth: .none
            )
        }
        await client.clearSession()
    }

    func me() async throws -> User {
        try await client.send(.get, "/api/v1/me", auth: .required)
    }

    /// Clears the local session only once the server confirmed the deletion, so
    /// a failed request leaves the user signed in and able to retry.
    func deleteAccount() async throws {
        try await client.sendIgnoringResponse(.delete, "/api/v1/me", auth: .required)
        await client.clearSession()
    }

    private func establishSession(_ response: AuthResponse) async throws -> User {
        try await client.storeSession(response.tokens)
        return response.user
    }
}

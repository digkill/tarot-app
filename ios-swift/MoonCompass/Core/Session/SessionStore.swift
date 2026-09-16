import Foundation
import Observation

/// Who is signed in, and the account actions. The single owner of the API
/// client's session-invalidated stream.
@MainActor
@Observable
final class SessionStore {
    enum State: Equatable {
        /// Launch: checking a stored session with the server.
        case restoring
        case signedOut
        case signedIn(User)
    }

    /// Version of the terms and privacy policy the user consents to. Must match
    /// `currentConsentVersion` on the server, or registration is refused.
    static let consentVersion = "1.0"
    static let cachedUserKey = "tarot.session.user"

    private(set) var state: State = .restoring

    var user: User? {
        if case let .signedIn(user) = state { return user }
        return nil
    }

    @ObservationIgnored private let client: APIClient
    @ObservationIgnored private let auth: AuthAPI
    @ObservationIgnored private let defaults: UserDefaults
    @ObservationIgnored private var eventsTask: Task<Void, Never>?

    init(client: APIClient, defaults: UserDefaults = .standard) {
        self.client = client
        self.auth = AuthAPI(client: client)
        self.defaults = defaults
    }

    // MARK: - Launch

    /// Restores a stored session and starts listening for invalidation.
    ///
    /// Unlike the React Native client, a network failure at launch does not
    /// sign the user out: with tokens still stored, the last known account is
    /// shown and the next request refreshes as usual. Only the server saying
    /// the session is gone ends it.
    func start() async {
        #if DEBUG
        if let user = DebugLaunch.previewUser {
            state = .signedIn(user)
            return
        }
        #endif
        observeSessionEvents()

        guard await client.hasSession else {
            state = .signedOut
            return
        }
        do {
            signIn(try await auth.me())
        } catch let error as APIError where error.status == 401 {
            await signOutLocally()
        } catch {
            if let cached = cachedUser() {
                state = .signedIn(cached)
            } else {
                state = .signedOut
            }
        }
    }

    private func observeSessionEvents() {
        guard eventsTask == nil else { return }
        let events = client.sessionEvents
        eventsTask = Task { [weak self] in
            for await event in events {
                guard let self else { return }
                if event == .invalidated {
                    await self.signOutLocally()
                }
            }
        }
    }

    // MARK: - Account actions

    func login(email: String, password: String) async throws {
        signIn(try await auth.login(email: email, password: password))
    }

    /// Creates the account. No session yet: the user must enter the emailed code.
    func register(email: String, password: String, language: Language,
                  acceptedTerms: Bool, acceptedPrivacy: Bool) async throws {
        _ = try await auth.register(RegisterRequest(
            email: email,
            password: password,
            language: language.rawValue,
            acceptedTerms: acceptedTerms,
            acceptedPrivacy: acceptedPrivacy,
            // The registration screen has one privacy checkbox, which covers
            // the privacy policy and the processing of personal data — the
            // same as the React Native client sends.
            acceptedPersonalData: acceptedPrivacy,
            consentVersion: Self.consentVersion
        ))
    }

    func verifyEmail(email: String, code: String) async throws {
        signIn(try await auth.verifyEmail(email: email, code: code))
    }

    func resendVerification(email: String, language: Language) async throws {
        try await auth.resendVerification(email: email, language: language.rawValue)
    }

    func forgotPassword(email: String, language: Language) async throws {
        try await auth.forgotPassword(email: email, language: language.rawValue)
    }

    func resetPassword(email: String, code: String, password: String) async throws {
        signIn(try await auth.resetPassword(email: email, code: code, password: password))
    }

    /// Re-reads the account, e.g. after a purchase changed premium status.
    func refreshUser() async {
        guard case .signedIn = state, let user = try? await auth.me() else { return }
        signIn(user)
    }

    func logout() async {
        await auth.logout()
        clearCachedUser()
        state = .signedOut
    }

    func deleteAccount() async throws {
        try await auth.deleteAccount()
        clearCachedUser()
        state = .signedOut
    }

    // MARK: - State

    private func signIn(_ user: User) {
        if let data = try? JSONCoding.makeEncoder().encode(user) {
            defaults.set(data, forKey: Self.cachedUserKey)
        }
        state = .signedIn(user)
    }

    private func signOutLocally() async {
        await client.clearSession()
        clearCachedUser()
        state = .signedOut
    }

    private func cachedUser() -> User? {
        guard let data = defaults.data(forKey: Self.cachedUserKey) else { return nil }
        return try? JSONCoding.makeDecoder().decode(User.self, from: data)
    }

    private func clearCachedUser() {
        defaults.removeObject(forKey: Self.cachedUserKey)
    }
}

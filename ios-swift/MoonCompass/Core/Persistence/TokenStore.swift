import Foundation
import os
import Security

/// The access/refresh pair. Stored and replaced together: the server rotates
/// the refresh token on every refresh, so persisting only one half would leave
/// a pair that can never be refreshed again.
struct AuthTokens: Codable, Sendable, Equatable {
    var accessToken: String
    var refreshToken: String
}

/// Synchronous on purpose: Keychain calls are fast, and a synchronous API lets
/// the APIClient actor read and replace tokens without a suspension point in
/// the middle of the refresh bookkeeping.
protocol TokenStore: Sendable {
    func load() throws -> AuthTokens?
    func save(_ tokens: AuthTokens) throws
    func clear() throws
}

struct KeychainError: Error, Sendable, Equatable {
    var status: OSStatus

    /// The test host runs unsigned (`CODE_SIGNING_ALLOWED=NO`), and without a
    /// keychain-access-group entitlement some simulator runtimes refuse access.
    var isMissingEntitlement: Bool { status == errSecMissingEntitlement }
}

/// Keychain rather than UserDefaults: a refresh token is a long-lived credential.
///
/// `AfterFirstUnlock` (not `WhenUnlocked`) so a background refresh — e.g. while
/// processing a StoreKit transaction with the screen locked — can still read it.
struct KeychainTokenStore: TokenStore {
    static let defaultService = "org.mediarise.tarot.session"

    let service: String
    /// Injectable so tests use a throwaway item and never clobber a real session
    /// in the same simulator.
    let account: String

    init(service: String = KeychainTokenStore.defaultService, account: String = "tokens") {
        self.service = service
        self.account = account
    }

    private var baseQuery: [String: Any] {
        [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: account,
        ]
    }

    func load() throws -> AuthTokens? {
        var query = baseQuery
        query[kSecReturnData as String] = true
        query[kSecMatchLimit as String] = kSecMatchLimitOne

        var item: CFTypeRef?
        let status = SecItemCopyMatching(query as CFDictionary, &item)
        switch status {
        case errSecSuccess:
            guard let data = item as? Data else { return nil }
            // A corrupt item is treated as no session rather than an error the
            // user can never get past: they simply sign in again.
            return try? JSONDecoder().decode(AuthTokens.self, from: data)
        case errSecItemNotFound:
            return nil
        default:
            throw KeychainError(status: status)
        }
    }

    func save(_ tokens: AuthTokens) throws {
        let data = try JSONEncoder().encode(tokens)
        let attributes: [String: Any] = [
            kSecValueData as String: data,
            kSecAttrAccessible as String: kSecAttrAccessibleAfterFirstUnlock,
        ]
        let updateStatus = SecItemUpdate(baseQuery as CFDictionary, attributes as CFDictionary)
        switch updateStatus {
        case errSecSuccess:
            return
        case errSecItemNotFound:
            var add = baseQuery
            add.merge(attributes) { _, new in new }
            let addStatus = SecItemAdd(add as CFDictionary, nil)
            guard addStatus == errSecSuccess else { throw KeychainError(status: addStatus) }
        default:
            throw KeychainError(status: updateStatus)
        }
    }

    func clear() throws {
        let status = SecItemDelete(baseQuery as CFDictionary)
        guard status == errSecSuccess || status == errSecItemNotFound else {
            throw KeychainError(status: status)
        }
    }
}

/// For tests and SwiftUI previews. `OSAllocatedUnfairLock` rather than
/// `Synchronization.Mutex`, which needs iOS 18.
final class InMemoryTokenStore: TokenStore {
    private let state: OSAllocatedUnfairLock<AuthTokens?>

    init(_ tokens: AuthTokens? = nil) {
        state = OSAllocatedUnfairLock(initialState: tokens)
    }

    func load() throws -> AuthTokens? {
        state.withLock { $0 }
    }

    func save(_ tokens: AuthTokens) throws {
        state.withLock { $0 = tokens }
    }

    func clear() throws {
        state.withLock { $0 = nil }
    }
}

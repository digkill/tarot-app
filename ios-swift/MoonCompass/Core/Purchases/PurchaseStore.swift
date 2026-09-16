import Foundation
import Observation
import StoreKit

/// App Store product identifiers. They must match App Store Connect and
/// `AppleProductID` in the server catalog byte for byte; deck SKUs come from
/// the server with each deck.
enum ProductIDs {
    static let premiumMonthly = "org.mediarise.tarot.premium.monthly"
    static let premiumYearly = "org.mediarise.tarot.premium.yearly"
    static let premiumLifetime = "org.mediarise.tarot.premium.lifetime"

    static let premium = [premiumMonthly, premiumYearly, premiumLifetime]
}

/// Sends a signed StoreKit transaction to the server. A protocol so tests can
/// stand in for the network.
protocol AppleTransactionVerifier: Sendable {
    func verify(signedTransaction: String, appAccountToken: UUID?) async throws -> AppleVerifyResponse
}

extension BillingAPI: AppleTransactionVerifier {
    func verify(signedTransaction: String, appAccountToken: UUID?) async throws -> AppleVerifyResponse {
        try await verifyAppleTransaction(signedTransaction: signedTransaction, appAccountToken: appAccountToken)
    }
}

enum PurchaseFailure: Error, Equatable, Sendable {
    /// StoreKit returned no product for the id.
    case productUnavailable
    /// Apple charged, but the server has not confirmed it yet. The transaction
    /// is left unfinished, so StoreKit delivers it again and it is retried.
    case verifyLater
    /// The purchase belongs to a different account of this app.
    case otherAccount
    /// The server refused the transaction for good.
    case rejected
    /// StoreKit itself failed.
    case store
}

enum PurchaseOutcome: Equatable, Sendable {
    case purchased(AppleVerifyResponse)
    /// Waiting for Ask to Buy or a payment method update.
    case pending
    case cancelled
    case failed(PurchaseFailure)
}

/// StoreKit 2 purchases, with the server as the only source of entitlements.
///
/// Apple's rule of thumb applies: a transaction is finished only once the
/// server has recorded it. Until then StoreKit keeps offering it — through
/// `Transaction.updates` while the app runs and `Transaction.unfinished` at the
/// next sign-in — so a purchase made while the server is unreachable is never
/// lost.
@MainActor
@Observable
final class PurchaseStore {
    private(set) var products: [String: Product] = [:]
    private(set) var loadingProducts = false
    /// The product being bought, to show progress on its button.
    private(set) var purchasingId: String?
    private(set) var restoring = false
    /// Bumped whenever the server grants something, so the app re-reads the
    /// account and the deck shop.
    private(set) var entitlementsVersion = 0

    /// The signed-in account. Transactions arriving while signed out are left
    /// unfinished for the next sign-in to pick up.
    var userId: String?

    @ObservationIgnored private let verifier: any AppleTransactionVerifier
    @ObservationIgnored private let finishTransaction: @MainActor (Transaction) async -> Void
    @ObservationIgnored private var updatesTask: Task<Void, Never>?

    /// `finish` is injectable so tests can observe exactly when a transaction
    /// is finished; StoreKit's test environment reports that unreliably.
    init(verifier: any AppleTransactionVerifier,
         finish: @escaping @MainActor (Transaction) async -> Void = { await $0.finish() }) {
        self.verifier = verifier
        self.finishTransaction = finish
    }

    // MARK: - Products

    func loadProducts(_ ids: [String]) async {
        let missing = Set(ids).subtracting(products.keys)
        guard !missing.isEmpty else { return }
        loadingProducts = true
        defer { loadingProducts = false }
        // A cold StoreKit daemon can answer the very first request with
        // nothing at all, so an empty answer is tried once more.
        for attempt in 0..<2 {
            if attempt > 0 { try? await Task.sleep(for: .seconds(1)) }
            let loaded = (try? await Product.products(for: missing)) ?? []
            for product in loaded {
                products[product.id] = product
            }
            if !loaded.isEmpty { return }
        }
    }

    // MARK: - Purchasing

    func purchase(_ productId: String) async -> PurchaseOutcome {
        guard purchasingId == nil else { return .cancelled }
        guard let product = products[productId] else { return .failed(.productUnavailable) }
        purchasingId = productId
        defer { purchasingId = nil }

        var options: Set<Product.PurchaseOption> = []
        // Ties the purchase to the account, including every future renewal
        // the server hears about through App Store notifications.
        if let token = accountToken {
            options.insert(.appAccountToken(token))
        }

        let result: Product.PurchaseResult
        do {
            result = try await product.purchase(options: options)
        } catch {
            return .failed(.store)
        }
        switch result {
        case let .success(verification):
            return await deliver(verification)
        case .pending:
            return .pending
        case .userCancelled:
            return .cancelled
        @unknown default:
            return .cancelled
        }
    }

    /// Asks the App Store for the latest transactions (this may prompt for the
    /// Apple Account password), then re-sends everything the user owns.
    /// Returns what the server granted.
    func restore() async throws -> [AppleVerifyResponse] {
        restoring = true
        defer { restoring = false }
        try await AppStore.sync()
        return await syncEntitlements()
    }

    /// Sends every current entitlement and every unfinished transaction to the
    /// server. Run at sign-in: it binds purchases to the account before the
    /// first renewal notification arrives, and backs up missed notifications.
    @discardableResult
    func syncEntitlements() async -> [AppleVerifyResponse] {
        guard userId != nil else { return [] }
        var granted: [AppleVerifyResponse] = []
        var seen = Set<UInt64>()
        for await verification in Transaction.unfinished {
            if case let .purchased(response) = await deliver(verification) {
                granted.append(response)
            }
            if case let .verified(transaction) = verification { seen.insert(transaction.id) }
        }
        for await verification in Transaction.currentEntitlements {
            if case let .verified(transaction) = verification, seen.contains(transaction.id) { continue }
            if case let .purchased(response) = await deliver(verification) {
                granted.append(response)
            }
        }
        return granted
    }

    /// Listens for renewals, Ask to Buy approvals and purchases made outside
    /// the app for as long as the app runs.
    func startListening() {
        guard updatesTask == nil else { return }
        updatesTask = Task { [weak self] in
            for await verification in Transaction.updates {
                guard let self else { return }
                _ = await self.deliver(verification)
            }
        }
    }

    // MARK: - Server

    private var accountToken: UUID? {
        userId.flatMap(UUID.init(uuidString:))
    }

    private func deliver(_ verification: VerificationResult<Transaction>) async -> PurchaseOutcome {
        // A transaction whose signature StoreKit could not check is not sent:
        // the server would refuse it anyway.
        guard case let .verified(transaction) = verification else { return .failed(.rejected) }
        guard userId != nil else { return .failed(.verifyLater) }

        do {
            let response = try await verifier.verify(signedTransaction: verification.jwsRepresentation,
                                                     appAccountToken: accountToken)
            await finishTransaction(transaction)
            entitlementsVersion += 1
            return .purchased(response)
        } catch {
            let failure = Self.failure(for: error)
            if failure != .verifyLater {
                // Final answer from the server; keeping it unfinished would only
                // replay the same refusal. It stays in `currentEntitlements`, so
                // signing in to the right account still restores it.
                await finishTransaction(transaction)
            }
            return .failed(failure)
        }
    }

    /// Which server errors are final. Network failures, 401 (the session is
    /// being refreshed or is gone), rate limits and 5xx are retried later.
    nonisolated static func failure(for error: any Error) -> PurchaseFailure {
        guard let apiError = error as? APIError else { return .verifyLater }
        switch apiError.code {
        case "apple_account_mismatch", "apple_family_shared":
            return .otherAccount
        default:
            break
        }
        switch apiError.status {
        case 401, 408, 429: return .verifyLater
        case 400..<500: return .rejected
        default: return .verifyLater
        }
    }

    /// The message key for a failed or pending purchase.
    nonisolated static func messageKey(for outcome: PurchaseOutcome) -> String? {
        switch outcome {
        case .purchased, .cancelled: nil
        case .pending: "premiumIos.pending"
        case .failed(.productUnavailable): "premiumIos.productsUnavailable"
        case .failed(.verifyLater): "premiumIos.verifyFailed"
        case .failed(.otherAccount): "premiumIos.otherAccount"
        case .failed(.rejected), .failed(.store): "premium.purchaseError"
        }
    }
}

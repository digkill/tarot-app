import Foundation

struct BillingAPI: Sendable {
    let client: APIClient

    /// Sends a StoreKit transaction to the server, which verifies it with Apple
    /// and grants premium or a deck.
    ///
    /// Errors: `apple_account_mismatch` / `apple_family_shared` (403),
    /// `invalid_signature` / `validation_error` (422), `payment_provider_error`
    /// (502), `payments_unavailable` (503).
    func verifyAppleTransaction(
        signedTransaction: String? = nil,
        originalTransactionId: String? = nil,
        appAccountToken: UUID? = nil
    ) async throws -> AppleVerifyResponse {
        let body = AppleVerifyRequest(
            signedTransaction: signedTransaction,
            originalTransactionId: originalTransactionId,
            appAccountToken: appAccountToken?.uuidString.lowercased()
        )
        return try await client.send(.post, "/api/v1/billing/apple/verify", body: body, auth: .required)
    }
}

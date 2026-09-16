import Foundation

struct ShopAPI: Sendable {
    let client: APIClient

    /// Optional auth: signed-in users get `owned` filled in. With an expired
    /// access token the server silently answers as for a guest (it never 401s
    /// here), so use `ownedDeckSlugs()` when ownership must be authoritative.
    func decks() async throws -> [ShopDeck] {
        let response: ShopDecksResponse = try await client.send(.get, "/api/v1/shop/decks", auth: .optional)
        return response.decks
    }

    func ownedDeckSlugs() async throws -> [String] {
        let response: OwnedDecksResponse = try await client.send(.get, "/api/v1/me/decks", auth: .required)
        return response.slugs
    }
}

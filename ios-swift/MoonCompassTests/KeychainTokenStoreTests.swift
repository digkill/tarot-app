import Security
import XCTest

@testable import MoonCompass

final class KeychainTokenStoreTests: XCTestCase {
    func testSaveLoadClearRoundTrip() throws {
        // A throwaway account so a real session in the same simulator survives.
        let store = KeychainTokenStore(account: "test-\(UUID().uuidString)")
        defer { try? store.clear() }

        let initial: AuthTokens?
        do {
            initial = try store.load()
            try store.save(Fixtures.oldTokens)
        } catch let error as KeychainError where error.isMissingEntitlement {
            throw XCTSkip("Keychain is unavailable in this unsigned test host (errSecMissingEntitlement).")
        }

        XCTAssertNil(initial, "a fresh account starts empty")
        XCTAssertEqual(try store.load(), Fixtures.oldTokens)

        try store.save(Fixtures.newTokens)
        XCTAssertEqual(try store.load(), Fixtures.newTokens, "saving again replaces the pair")

        try store.clear()
        XCTAssertNil(try store.load())
        XCTAssertNoThrow(try store.clear(), "clearing an empty store is not an error")
    }

    func testInMemoryStoreRoundTrip() throws {
        let store = InMemoryTokenStore()
        XCTAssertNil(try store.load())
        try store.save(Fixtures.oldTokens)
        XCTAssertEqual(try store.load(), Fixtures.oldTokens)
        try store.clear()
        XCTAssertNil(try store.load())
    }
}

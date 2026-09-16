import XCTest

@testable import MoonCompass

@MainActor
final class SettingsStoreTests: XCTestCase {
    private var suiteName: String!
    private var defaults: UserDefaults!

    override func setUp() async throws {
        suiteName = "SettingsStoreTests.\(UUID().uuidString)"
        defaults = UserDefaults(suiteName: suiteName)
    }

    override func tearDown() async throws {
        defaults.removePersistentDomain(forName: suiteName)
    }

    func testFirstLaunchFollowsTheDevice() {
        let store = SettingsStore(defaults: defaults, deviceLanguage: .th)
        XCTAssertEqual(store.settings.language, .th)
        XCTAssertFalse(store.settings.acceptedDisclaimer)
        XCTAssertFalse(store.settings.languageExplicit)
        XCTAssertEqual(store.settings.reversedChance, 0.3)
        XCTAssertEqual(store.localizer.language, .th)
    }

    func testAcceptingTheDisclaimerPersists() {
        SettingsStore(defaults: defaults, deviceLanguage: .en).acceptDisclaimer()
        XCTAssertTrue(SettingsStore(defaults: defaults, deviceLanguage: .en).settings.acceptedDisclaimer)
    }

    /// Until the user picks a language, a device language change is followed.
    func testImplicitLanguageFollowsTheDeviceOnNextLaunch() {
        let first = SettingsStore(defaults: defaults, deviceLanguage: .en)
        first.acceptDisclaimer()
        let next = SettingsStore(defaults: defaults, deviceLanguage: .ru)
        XCTAssertEqual(next.settings.language, .ru)
        XCTAssertTrue(next.settings.acceptedDisclaimer)
    }

    func testExplicitLanguageSticks() {
        let store = SettingsStore(defaults: defaults, deviceLanguage: .en)
        store.setLanguage(.zh)
        XCTAssertEqual(store.localizer.language, .zh, "the localizer is rebuilt on change")
        XCTAssertTrue(store.settings.languageExplicit)

        let relaunched = SettingsStore(defaults: defaults, deviceLanguage: .ru)
        XCTAssertEqual(relaunched.settings.language, .zh)
    }

    func testCorruptStorageFallsBackToDefaults() {
        defaults.set(Data("not json".utf8), forKey: SettingsStore.storageKey)
        let store = SettingsStore(defaults: defaults, deviceLanguage: .ru)
        XCTAssertEqual(store.settings.language, .ru)
        XCTAssertFalse(store.settings.acceptedDisclaimer)
    }

    /// Settings written by an older build, or with a field of the wrong type,
    /// keep everything that is still readable.
    func testPartialStorageKeepsWhatItCan() {
        defaults.set(Data(#"{"acceptedDisclaimer":true,"reversedChance":"oops","language":"xx"}"#.utf8),
                     forKey: SettingsStore.storageKey)
        let store = SettingsStore(defaults: defaults, deviceLanguage: .en)
        XCTAssertTrue(store.settings.acceptedDisclaimer)
        XCTAssertEqual(store.settings.reversedChance, 0.3)
        XCTAssertEqual(store.settings.selectedDeckId, "rws")
    }
}

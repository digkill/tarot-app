import XCTest

@testable import MoonCompass

final class LocalizerTests: XCTestCase {
    /// The shared `i18n/` folder must actually be in the app bundle, for every
    /// language, or every screen shows raw keys.
    func testEveryLanguageLoadsFromTheBundle() {
        for language in Language.allCases {
            let l = Localizer(language: language)
            for key in ["disclaimer.title", "auth.loginTitle", "auth.errors.generic", "nav.settings"] {
                XCTAssertNotEqual(l.t(key), key, "\(language): \(key) is not translated")
            }
            XCTAssertFalse(l.list("legal.privacyParagraphs").isEmpty, "\(language): privacy policy missing")
            XCTAssertFalse(l.list("legal.termsParagraphs").isEmpty, "\(language): terms missing")
        }
    }

    func testLanguagesDiffer() {
        XCTAssertNotEqual(Localizer(language: .ru).t("nav.settings"), Localizer(language: .en).t("nav.settings"))
    }

    func testFallsBackToEnglishThenToTheKey() {
        let l = Localizer(
            language: .ru,
            strings: ["a": "по-русски"],
            fallbackStrings: ["a": "english", "b": "only in english"]
        )
        XCTAssertEqual(l.t("a"), "по-русски")
        XCTAssertEqual(l.t("b"), "only in english")
        XCTAssertEqual(l.t("missing.key"), "missing.key")
        XCTAssertTrue(l.has("b"))
        XCTAssertFalse(l.has("missing.key"))
    }

    func testInterpolatesPlaceholders() {
        let l = Localizer(language: .en, strings: ["verify": "Code sent to {{email}} ({{email}})"])
        XCTAssertEqual(l.t("verify", ["email": "a@b.c"]), "Code sent to a@b.c (a@b.c)")
        XCTAssertEqual(l.t("verify"), "Code sent to {{email}} ({{email}})", "no arguments, no substitution")
    }

    func testEnglishPlurals() {
        let l = Localizer(language: .en, strings: ["cards_one": "{{count}} card", "cards_other": "{{count}} cards"])
        XCTAssertEqual(l.t("cards", ["count": 1]), "1 card")
        XCTAssertEqual(l.t("cards", ["count": 3]), "3 cards")
        XCTAssertEqual(l.t("cards", ["count": 0]), "0 cards")
    }

    func testRussianPluralsFallBackToOther() {
        let l = Localizer(language: .ru, strings: [
            "cards_one": "{{count}} карта",
            "cards_few": "{{count}} карты",
            "cards_many": "{{count}} карт",
        ], fallbackStrings: ["cards_other": "{{count}} cards"])
        XCTAssertEqual(l.t("cards", ["count": 1]), "1 карта")
        XCTAssertEqual(l.t("cards", ["count": 22]), "22 карты")
        XCTAssertEqual(l.t("cards", ["count": 11]), "11 карт")
        XCTAssertEqual(l.t("cards", ["count": 25]), "25 карт")

        let sparse = Localizer(language: .ru, strings: ["cards_other": "{{count}} шт."])
        XCTAssertEqual(sparse.t("cards", ["count": 5]), "5 шт.", "a missing form falls back to _other")
    }

    func testRussianPluralRules() {
        let expected: [Int: PluralRules.Category] = [
            0: .many, 1: .one, 2: .few, 4: .few, 5: .many, 11: .many, 12: .many,
            14: .many, 21: .one, 22: .few, 101: .one, 111: .many, 112: .many,
        ]
        for (count, category) in expected {
            XCTAssertEqual(PluralRules.category(for: count, in: .ru), category, "\(count)")
        }
        XCTAssertEqual(PluralRules.category(for: 1, in: .zh), .other)
        XCTAssertEqual(PluralRules.category(for: 1, in: .th), .other)
    }

    func testDeviceLanguageResolution() {
        XCTAssertEqual(Language.resolve(preferred: ["ru-RU"]), .ru)
        XCTAssertEqual(Language.resolve(preferred: ["zh-Hant-TW"]), .zh)
        XCTAssertEqual(Language.resolve(preferred: ["de-DE", "th_TH"]), .th, "first supported wins")
        XCTAssertEqual(Language.resolve(preferred: ["fr-FR"]), .en)
        XCTAssertEqual(Language.resolve(preferred: []), .en)
    }

    func testAuthErrorMessagePrefersSpecificTranslation() {
        let l = Localizer(language: .en, strings: [
            "auth.errors.invalid_credentials": "Wrong email or password",
            "auth.errors.network": "No connection",
            "auth.errors.generic": "Something went wrong",
        ])
        XCTAssertEqual(l.authErrorMessage(APIError(code: "invalid_credentials", message: "", status: 401)),
                       "Wrong email or password")
        XCTAssertEqual(l.authErrorMessage(APIError(code: "network", message: "", status: 0)), "No connection")
        XCTAssertEqual(l.authErrorMessage(APIError(code: "brand_new_code", message: "", status: 500)),
                       "Something went wrong")
        XCTAssertEqual(l.authErrorMessage(CancellationError()), "Something went wrong")
    }
}

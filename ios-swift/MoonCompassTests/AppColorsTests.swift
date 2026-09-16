import SwiftUI
import XCTest

@testable import MoonCompass

final class AppColorsTests: XCTestCase {
    func testParsesSixDigitHex() {
        XCTAssertNotNil(Color(hex: "#6c5ce7"))
        XCTAssertNotNil(Color(hex: "6c5ce7"), "the leading # is optional")
        XCTAssertNotNil(Color(hex: "  #6C5CE7  "), "surrounding whitespace is tolerated")
    }

    func testParsesThreeDigitShorthand() {
        XCTAssertNotNil(Color(hex: "#abc"))
    }

    func testRejectsMalformedHex() {
        for bad in ["", "   ", "#", "#12", "#12345", "#1234567", "#gggggg", "rgb(1,2,3)", "purple"] {
            XCTAssertNil(Color(hex: bad), "\(bad.debugDescription) must not parse")
        }
        XCTAssertNil(Color(hex: nil))
    }

    /// A deck theme arrives from the server, so one bad field must not discard
    /// the whole palette — each field falls back on its own.
    func testPayloadFallsBackPerField() {
        let payload = ThemePayload(
            bg: "#101010",
            panel: "not a colour",
            accent: nil,
            text: "#fff",
            muted: "",
            gold: "#d4af37",
            danger: "#ff6b6b",
            tabBar: "#0c0a14"
        )
        let colors = AppColors(payload: payload)
        let classic = AppColors.classic

        XCTAssertNotEqual(colors.bg, classic.bg, "a valid field is used")
        XCTAssertEqual(colors.panel, classic.panel, "an invalid field falls back")
        XCTAssertEqual(colors.accent, classic.accent, "a missing field falls back")
        XCTAssertEqual(colors.muted, classic.muted, "an empty field falls back")
        XCTAssertEqual(colors.gold, classic.gold)
    }

    func testMissingPayloadIsClassic() {
        XCTAssertEqual(AppColors(payload: nil), AppColors.classic)
    }
}

import Foundation

/// JSON coders configured for the Go backend.
enum JSONCoding {
    static func makeDecoder() -> JSONDecoder {
        let decoder = JSONDecoder()
        decoder.dateDecodingStrategy = .custom(decodeDate)
        return decoder
    }

    static func makeEncoder() -> JSONEncoder {
        let encoder = JSONEncoder()
        encoder.dateEncodingStrategy = .iso8601
        return encoder
    }

    /// Go marshals `time.Time` as RFC 3339 with as many fractional digits as the
    /// value has (Postgres keeps microseconds) and a numeric offset, but a value
    /// with a zero fraction has none at all. `.iso8601` rejects the fraction, and
    /// a fraction-only parser rejects the other form, so both are tried.
    ///
    /// `Date.ISO8601FormatStyle` is used instead of `ISO8601DateFormatter`
    /// because it is a Sendable value — the strategy closure must be @Sendable.
    @Sendable
    static func decodeDate(from decoder: any Decoder) throws -> Date {
        let container = try decoder.singleValueContainer()
        let raw = try container.decode(String.self)
        if let date = parseDate(raw) {
            return date
        }
        throw DecodingError.dataCorruptedError(
            in: container,
            debugDescription: "Expected an RFC 3339 date, got \(raw.debugDescription)"
        )
    }

    static func parseDate(_ raw: String) -> Date? {
        let withFraction = Date.ISO8601FormatStyle(includingFractionalSeconds: true)
        let withoutFraction = Date.ISO8601FormatStyle()
        if let date = try? withFraction.parse(raw) { return date }
        if let date = try? withoutFraction.parse(raw) { return date }

        // Older Foundation builds are strict about the number of fractional
        // digits. Strip the fraction, parse the rest, and add it back.
        guard let tIndex = raw.firstIndex(of: "T"),
              let dot = raw[tIndex...].firstIndex(of: ".") else {
            return nil
        }
        let afterDot = raw.index(after: dot)
        let fractionEnd = raw[afterDot...].firstIndex(where: { !$0.isNumber }) ?? raw.endIndex
        let digits = raw[afterDot..<fractionEnd]
        guard !digits.isEmpty, let fraction = Double("0." + digits) else { return nil }
        let stripped = String(raw[..<dot]) + String(raw[fractionEnd...])
        guard let whole = try? withoutFraction.parse(stripped) else { return nil }
        return whole.addingTimeInterval(fraction)
    }
}

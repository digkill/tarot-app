import Foundation
import Observation

/// The reading journal, newest first, kept in a JSON file on the device.
///
/// A file rather than a database: the journal is a short flat list that is
/// read whole and edited one item at a time, so atomic rewrites of one small
/// file are simpler and easier to inspect than a schema to migrate.
@MainActor
@Observable
final class HistoryStore {
    private(set) var readings: [Reading] = []

    @ObservationIgnored private let fileURL: URL

    init(fileURL: URL = HistoryStore.defaultFileURL) {
        self.fileURL = fileURL
        readings = Self.load(from: fileURL)
    }

    static var defaultFileURL: URL {
        let directory = FileManager.default.urls(for: .applicationSupportDirectory, in: .userDomainMask)[0]
        return directory.appendingPathComponent("readings.json")
    }

    func reading(id: String) -> Reading? {
        readings.first { $0.id == id }
    }

    @discardableResult
    func add(_ reading: Reading) -> Reading {
        readings.insert(reading, at: 0)
        save()
        return reading
    }

    func update(id: String, _ change: (inout Reading) -> Void) {
        guard let index = readings.firstIndex(where: { $0.id == id }) else { return }
        change(&readings[index])
        save()
    }

    func toggleFavorite(id: String) {
        update(id: id) { $0.favorite.toggle() }
    }

    func remove(id: String) {
        readings.removeAll { $0.id == id }
        save()
    }

    func removeAll() {
        readings.removeAll()
        save()
    }

    /// The first one-card reading drawn today, if any — "today" in the
    /// device's own calendar, like the React Native client.
    func todaysDailyCard(now: Date = Date(), calendar: Calendar = .current) -> Reading? {
        let startOfDay = calendar.startOfDay(for: now)
        return readings.first {
            $0.spreadId == Spread.oneCardId && $0.kind != .bonus && $0.drawnAt >= startOfDay
        }
    }

    private func save() {
        do {
            try FileManager.default.createDirectory(at: fileURL.deletingLastPathComponent(),
                                                    withIntermediateDirectories: true)
            let data = try JSONEncoder.history.encode(readings)
            try data.write(to: fileURL, options: [.atomic, .completeFileProtectionUntilFirstUserAuthentication])
        } catch {
            assertionFailure("could not save the reading journal: \(error)")
        }
    }

    private static func load(from url: URL) -> [Reading] {
        guard let data = try? Data(contentsOf: url) else { return [] }
        return (try? JSONDecoder.history.decode([Reading].self, from: data)) ?? []
    }
}

private extension JSONEncoder {
    static var history: JSONEncoder {
        let encoder = JSONEncoder()
        encoder.dateEncodingStrategy = .millisecondsSince1970
        return encoder
    }
}

private extension JSONDecoder {
    static var history: JSONDecoder {
        let decoder = JSONDecoder()
        decoder.dateDecodingStrategy = .millisecondsSince1970
        return decoder
    }
}

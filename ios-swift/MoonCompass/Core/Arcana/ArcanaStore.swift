import Foundation
import Observation

/// The Arcana Clash client state: one WebSocket, the matchmaking stage and
/// the latest match state from the server.
///
/// The server is authoritative. This store only forwards intents and shows
/// whatever state arrives, dropping any older than what it already has
/// (by `seq`). A dropped connection during a match is retried with backoff
/// and resumed with `match.resume`, inside the server's reconnect window.
@MainActor
@Observable
final class ArcanaStore {
    enum Connection: Equatable { case offline, connecting, online }
    enum Stage: Equatable { case lobby, searching, match, finished }

    struct EmoteBubble: Identifiable, Equatable {
        let id = UUID()
        let mine: Bool
        let emote: String
    }

    private(set) var connection: Connection = .offline
    private(set) var stage: Stage = .lobby
    private(set) var catalog: ArcanaCatalog?
    private(set) var catalogFailed = false
    @ObservationIgnored private var usingBundledCatalog = false
    private(set) var matchId: String?
    private(set) var started: ArcanaStartedPayload?
    private(set) var view: ArcanaGameView?
    /// The end of the current turn or mulligan, on this device's clock.
    private(set) var deadline: Date?
    private(set) var opponentConnected = true
    private(set) var finished: ArcanaFinishedPayload?
    /// A match the server still holds for this player (e.g. after the app was
    /// closed mid-match).
    private(set) var resumableMatch: String?
    /// The code of the last rejected action or connection problem.
    private(set) var lastError: String?
    private(set) var errorCount = 0
    private(set) var emotes: [EmoteBubble] = []
    private(set) var userId: String?

    var hero: String? {
        didSet { defaults.set(hero, forKey: Self.heroKey) }
    }

    @ObservationIgnored private let api: ArcanaAPI
    @ObservationIgnored private let urlSession: URLSession
    @ObservationIgnored private let defaults: UserDefaults
    @ObservationIgnored private var socket: URLSessionWebSocketTask?
    @ObservationIgnored private var tasks: [Task<Void, Never>] = []
    @ObservationIgnored private var wantsQueue = false
    @ObservationIgnored private var reconnectAttempt = 0
    @ObservationIgnored private var refreshedForAuth = false
    @ObservationIgnored private var keepConnected = false

    static let heroKey = "arcana.hero"

    init(api: ArcanaAPI, urlSession: URLSession = .shared, defaults: UserDefaults = .standard) {
        self.api = api
        self.urlSession = urlSession
        self.defaults = defaults
        hero = defaults.string(forKey: Self.heroKey)
    }

    // MARK: - Catalog

    /// The server's catalog, or the copy bundled with the app when the server
    /// can't be reached — so heroes and rules show offline. The server's copy
    /// replaces the bundled one as soon as it loads.
    func loadCatalog() async {
        guard catalog == nil || usingBundledCatalog else { return }
        do {
            catalog = try await api.catalog()
            usingBundledCatalog = false
            catalogFailed = false
        } catch {
            catalogFailed = true
            if catalog == nil, let bundled = Self.bundledCatalog() {
                catalog = bundled
                usingBundledCatalog = true
            }
        }
    }

    static func bundledCatalog(bundle: Bundle = .main) -> ArcanaCatalog? {
        guard let url = bundle.url(forResource: "arcana_catalog", withExtension: "json"),
              let data = try? Data(contentsOf: url) else { return nil }
        return try? ArcanaCoding.decoder.decode(ArcanaCatalog.self, from: data)
    }

    func matchHistory() async throws -> [ArcanaMatchSummary] {
        try await api.matches()
    }

    // MARK: - Lifecycle

    /// The Arcana screen is visible: connect to learn about a match to resume.
    func enter() {
        keepConnected = true
        if connection == .offline { connect() }
    }

    /// The screen went away. Outside a match the socket is not needed.
    func leave() {
        keepConnected = false
        if stage == .lobby || stage == .finished { disconnect() }
    }

    /// The app came back to the foreground; iOS may have dropped the socket.
    func appDidBecomeActive() {
        if keepConnected || stage == .match || stage == .searching, connection == .offline {
            connect()
        }
    }

    // MARK: - Intents

    func battle() {
        finished = nil
        stage = .searching
        wantsQueue = true
        if connection == .online {
            send(.init(type: "queue.join", hero: hero))
        } else if connection == .offline {
            connect()
        }
    }

    func cancelSearch() {
        wantsQueue = false
        send(.init(type: "queue.leave"))
        stage = .lobby
    }

    func resume() {
        guard let id = resumableMatch else { return }
        matchId = id
        stage = .match
        send(.init(type: "match.resume", matchId: id))
    }

    func mulligan(_ uids: [String]) { matchAction("mulligan", cardUids: uids) }
    func play(_ uid: String, target: ArcanaTarget?) { matchAction("card.play", cardUid: uid, target: target) }
    func ability(target: ArcanaTarget?) { matchAction("hero.ability", target: target) }
    func ultimate(target: ArcanaTarget?) { matchAction("hero.ultimate", target: target) }
    func endTurn() { matchAction("turn.end") }
    func surrender() { matchAction("match.surrender") }

    func emote(_ emote: String) {
        matchAction("player.emote", emote: emote)
    }

    /// Back to the lobby after the result screen.
    func closeResult() {
        stage = .lobby
        view = nil
        started = nil
        matchId = nil
        finished = nil
        if !keepConnected { disconnect() }
    }

    func clearError() { lastError = nil }

    private func matchAction(_ type: String, cardUid: String? = nil, cardUids: [String]? = nil,
                             target: ArcanaTarget? = nil, emote: String? = nil) {
        guard let matchId else { return }
        send(.init(type: type, matchId: matchId, cardUid: cardUid, cardUids: cardUids, target: target, emote: emote))
    }

    // MARK: - Connection

    private func connect() {
        guard connection == .offline else { return }
        connection = .connecting
        let attempt = Task { [weak self] in
            guard let self else { return }
            do {
                let token = try await self.api.token(force: self.refreshedForAuth)
                self.open(token: token)
            } catch {
                self.dropped(code: nil)
            }
        }
        tasks.append(attempt)
    }

    private func open(token: String) {
        let task = urlSession.webSocketTask(with: api.config.socketURL)
        task.maximumMessageSize = 1 << 20
        socket = task
        task.resume()
        send(.init(type: "hello", protocol: ArcanaCoding.protocolVersion, token: token), on: task)
        tasks.append(Task { [weak self] in await self?.receiveLoop(task) })
        tasks.append(Task { [weak self] in await self?.pingLoop(task) })
    }

    private func disconnect() {
        wantsQueue = false
        tasks.forEach { $0.cancel() }
        tasks = []
        socket?.cancel(with: .normalClosure, reason: nil)
        socket = nil
        connection = .offline
    }

    private func receiveLoop(_ task: URLSessionWebSocketTask) async {
        while !Task.isCancelled {
            do {
                let message = try await task.receive()
                guard task === socket else { return }
                let data: Data
                switch message {
                case let .string(text): data = Data(text.utf8)
                case let .data(raw): data = raw
                @unknown default: continue
                }
                if let event = try? ArcanaServerEvent.decode(data) {
                    handle(event)
                }
            } catch {
                guard task === socket else { return }
                dropped(code: task.closeCode.rawValue, reason: task.closeReason)
                return
            }
        }
    }

    /// The server pings too, but a client-side ping notices a dead link
    /// (e.g. Wi-Fi → LTE) without waiting for the OS.
    private func pingLoop(_ task: URLSessionWebSocketTask) async {
        while !Task.isCancelled {
            try? await Task.sleep(for: .seconds(15))
            guard task === socket, !Task.isCancelled else { return }
            let alive = await withCheckedContinuation { continuation in
                task.sendPing { error in continuation.resume(returning: error == nil) }
            }
            if !alive, task === socket {
                task.cancel(with: .goingAway, reason: nil)
                dropped(code: nil)
                return
            }
        }
    }

    private func dropped(code: Int?, reason: Data? = nil) {
        socket = nil
        connection = .offline
        let text = reason.flatMap { String(data: $0, encoding: .utf8) } ?? ""
        if code == 4001 {
            // The same account connected from another device: do not fight it.
            lastError = "session_replaced"
            errorCount += 1
            stage = stage == .match ? .lobby : stage
            return
        }
        if text == "unauthorized", !refreshedForAuth {
            refreshedForAuth = true
        }
        guard stage == .match || stage == .searching || keepConnected else { return }
        reconnectAttempt += 1
        if reconnectAttempt == 2 {
            // Keep retrying, but tell the player something is wrong.
            lastError = "connection_failed"
            errorCount += 1
        }
        if reconnectAttempt > 8 {
            lastError = "connection_failed"
            errorCount += 1
            if stage == .searching { stage = .lobby }
            return
        }
        let delay = min(pow(2, Double(reconnectAttempt - 1)), 8)
        tasks.append(Task { [weak self] in
            try? await Task.sleep(for: .seconds(delay))
            guard let self, !Task.isCancelled, self.connection == .offline else { return }
            self.connect()
        })
    }

    private func send(_ message: ArcanaClientMessage, on task: URLSessionWebSocketTask? = nil) {
        guard let task = task ?? socket,
              let data = try? ArcanaCoding.encoder.encode(message),
              let text = String(data: data, encoding: .utf8) else { return }
        task.send(.string(text)) { _ in }
    }

    // MARK: - Server events

    func handle(_ event: ArcanaServerEvent) {
        switch event {
        case let .hello(hello):
            connection = .online
            if lastError == "connection_failed" { lastError = nil }
            reconnectAttempt = 0
            refreshedForAuth = false
            userId = hello.userId
            resumableMatch = hello.activeMatch
            if let active = hello.activeMatch, stage == .match || stage == .searching {
                matchId = active
                stage = .match
                send(.init(type: "match.resume", matchId: active))
            } else if stage == .match {
                // The match ended while we were away.
                stage = .lobby
                view = nil
            } else if wantsQueue, stage == .searching {
                send(.init(type: "queue.join", hero: hero))
            }
        case .queueWaiting:
            stage = .searching
        case .queueLeft:
            if stage == .searching { stage = .lobby }
        case let .matchStarted(id, payload):
            wantsQueue = false
            matchId = id
            started = payload
            resumableMatch = nil
            view = nil
            finished = nil
            opponentConnected = true
            stage = .match
        case let .state(id, payload):
            if matchId == nil { matchId = id }
            guard id == matchId else { return }
            if let current = view, current.seq > payload.state.seq { return }
            view = payload.state
            let offset = Double(payload.serverTime) / 1000 - Date().timeIntervalSince1970
            deadline = Date(timeIntervalSince1970: Double(payload.deadlineAt) / 1000 - offset)
            opponentConnected = payload.opponentConnected
            resumableMatch = nil
            if stage != .finished { stage = .match }
        case let .finished(id, payload):
            guard id == matchId || matchId == nil else { return }
            finished = payload
            view = payload.state
            deadline = nil
            resumableMatch = nil
            stage = .finished
        case let .emote(payload):
            let bubble = EmoteBubble(mine: payload.player == userId, emote: payload.emote)
            emotes.append(bubble)
            Task { [weak self] in
                try? await Task.sleep(for: .seconds(3))
                self?.emotes.removeAll { $0.id == bubble.id }
            }
        case .opponentDisconnected:
            opponentConnected = false
        case .opponentReconnected:
            opponentConnected = true
        case let .error(error):
            if error.code == "already_in_match" {
                stage = .match
                return
            }
            lastError = error.code
            errorCount += 1
            if stage == .searching, error.code == "invalid_hero" { stage = .lobby }
        case .unknown:
            break
        }
    }

    #if DEBUG
    /// Previews and tests: set a state without a server.
    func setPreview(stage: Stage, view: ArcanaGameView?, catalog: ArcanaCatalog?, started: ArcanaStartedPayload? = nil) {
        self.stage = stage
        self.view = view
        self.catalog = catalog
        self.started = started
        self.matchId = view == nil ? nil : "preview"
        self.deadline = Date().addingTimeInterval(32)
    }
    #endif
}

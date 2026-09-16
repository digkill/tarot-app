# Arcana Clash server

Server-authoritative PvP card game for Moon Compass. Go, `net/http`,
`github.com/coder/websocket`, PostgreSQL. No Redis: live matches are held in
memory, and each finished match is written once.

```
cmd/server            HTTP + WebSocket entry point
cmd/bot               a protocol-level bot to play against
internal/game         rules engine — no network, clock or database
internal/cards        card definitions (data)
internal/heroes       hero definitions (data)
internal/match        match actor: one goroutine per match, timers, reconnect
internal/matchmaking  queue with a pluggable compatibility rule
internal/ws           connections, sessions, heartbeat, routing
internal/api          catalog and match history over HTTP
internal/auth         verifies the main backend's access tokens
internal/storage      Postgres (and in-memory) persistence, migrations
```

## Run locally

```bash
JWT_SECRET=<same as the main backend> DATABASE_URL=<main backend DB> go run ./cmd/server
go run ./cmd/bot -secret "$JWT_SECRET"          # an opponent that queues forever
go test -race ./...
```

Without `DATABASE_URL` finished matches are kept in memory. The migration
needs the main backend's `users` table and tracks its version in
`arcana_goose_db_version`.

iOS Debug builds can point at a local server:
`-arcanaURL http://localhost:8090 -arcanaToken <jwt> -arcanaBattle death`.

## Engine

```go
g, _ := game.New(seed, game.PlayerSetup{ID: "a", Hero: "death"}, game.PlayerSetup{ID: "b", Hero: "strength"})
g.Mulligan("a", []string{uid}); g.Mulligan("b", nil)
g.PlayCard(g.ActivePlayerID(), uid, game.Target{Kind: cards.TargetEnemy})
g.EndTurn(g.ActivePlayerID())
view := g.ViewFor("a") // what player a may see
game.Replay(rules, seed, a, b, g.Log())
```

- **Cards** are a cost and two sides (upright, reversed); each side is a list
  of ops. A new card is one entry in `internal/cards`. A new effect kind is one
  handler in `internal/game/ops.go`.
- **Heroes** name a passive (hooks in `internal/game/passives.go`) and reuse
  ops for the ability and the ultimate.
- **Randomness** — decks, crits, shuffles — comes only from the match seed, so
  the stored action log replays exactly. `RulesVersion` changes with the rules.

## Protocol v1

JSON text frames. Clients send intents only; the server computes every
number. Server frames: `{"type", "seq", "match_id", "payload"}` — `seq` is the
match state sequence and only grows; drop anything older than you have.

| Client → server | Fields |
| --- | --- |
| `hello` (first frame, within 10 s) | `protocol`, `token` |
| `queue.join` / `queue.leave` | `hero` (optional; random when empty) |
| `match.resume` | `match_id` |
| `mulligan` | `match_id`, `card_uids` (up to 3) |
| `card.play` | `match_id`, `card_uid`, `target {kind, card_uid}` |
| `hero.ability` / `hero.ultimate` | `match_id`, `target` |
| `turn.end` / `match.surrender` | `match_id` |
| `player.emote` | `match_id`, `emote` (fixed list) |

| Server → client | Payload |
| --- | --- |
| `hello` | `user_id`, `protocol`, `rules_version`, `active_match`, `emotes` |
| `queue.waiting` / `queue.left` | |
| `match.started` | `you`, `opponent` (`id`, `hero`) |
| `match.state` | `state` (player view), `server_time`, `deadline_at`, `opponent_connected` |
| `opponent.disconnected` / `opponent.reconnected` | `reconnect_by` |
| `player.emote` | `player`, `emote` |
| `match.finished` | `winner`, `reason`, `result` (win/loss/none), `state` |
| `error` | `code`, `message`, `ref` (echo of the client's `ref`) |

Target kinds: `enemy`, `self`, `hand_card`, `discard_card`; a view lists
`none` for cards that take no target.

Timers belong to the server: turn 40 s (three timeouts in a row lose),
mulligan 30 s, reconnect window 45 s. The server pings every 15 s; a missing
pong drops the connection, idle thinking does not. Close code 4001 means the
same account connected elsewhere. More than 20 frames a second disconnects.

End reasons: `normal`, `surrender`, `disconnect_timeout`, `turn_timeout`,
`server_error`.

HTTP: `GET /api/v1/arcana/catalog` (public), `GET /api/v1/arcana/matches`
and `GET /api/v1/arcana/matches/{id}` (Bearer; a replay is redacted for the
requesting player). WebSocket at `GET /api/v1/arcana/ws`.

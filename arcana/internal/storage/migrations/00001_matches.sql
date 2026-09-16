-- Arcana Clash lives in the Moon Compass database, next to the `users` table
-- owned by the main backend, and keeps its own goose version table
-- (arcana_goose_db_version) so the two services migrate independently.

-- +goose Up
CREATE TABLE IF NOT EXISTS arcana_matches (
    id               UUID PRIMARY KEY,
    -- SET NULL, not CASCADE: deleting one account must not erase the other
    -- player's history.
    player1_id       UUID REFERENCES users(id) ON DELETE SET NULL,
    player2_id       UUID REFERENCES users(id) ON DELETE SET NULL,
    player1_hero     TEXT NOT NULL,
    player2_hero     TEXT NOT NULL,
    winner_id        UUID REFERENCES users(id) ON DELETE SET NULL,
    result_reason    TEXT NOT NULL CHECK (result_reason IN
                        ('normal','surrender','disconnect_timeout','turn_timeout','server_error')),
    -- The uint64 seed's bits, stored as a signed BIGINT.
    seed             BIGINT NOT NULL,
    rules_version    INTEGER NOT NULL,
    protocol_version INTEGER NOT NULL,
    turns            INTEGER NOT NULL DEFAULT 0,
    started_at       TIMESTAMPTZ NOT NULL,
    finished_at      TIMESTAMPTZ NOT NULL,
    duration_ms      BIGINT NOT NULL,
    rating_delta     INTEGER,
    replay           JSONB NOT NULL DEFAULT '[]'
);

CREATE INDEX IF NOT EXISTS idx_arcana_matches_p1 ON arcana_matches (player1_id, finished_at DESC);
CREATE INDEX IF NOT EXISTS idx_arcana_matches_p2 ON arcana_matches (player2_id, finished_at DESC);

-- +goose Down
DROP TABLE IF EXISTS arcana_matches;

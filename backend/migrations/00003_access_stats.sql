-- +goose Up
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS consent_ip TEXT,
    ADD COLUMN IF NOT EXISTS consent_user_agent TEXT;

CREATE TABLE IF NOT EXISTS access_stats (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    event           TEXT NOT NULL,
    ip_enc          TEXT,
    user_agent_enc  TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_access_stats_user_created
    ON access_stats (user_id, created_at DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_access_stats_user_created;
DROP TABLE IF EXISTS access_stats;

-- +goose Up
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS premium_expires_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS premium_product_id TEXT,
    ADD COLUMN IF NOT EXISTS premium_source TEXT;

CREATE INDEX IF NOT EXISTS idx_users_premium_expires
    ON users (premium_expires_at)
    WHERE has_premium = TRUE AND premium_expires_at IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_users_premium_expires;
ALTER TABLE users
    DROP COLUMN IF EXISTS premium_expires_at,
    DROP COLUMN IF EXISTS premium_product_id,
    DROP COLUMN IF EXISTS premium_source;

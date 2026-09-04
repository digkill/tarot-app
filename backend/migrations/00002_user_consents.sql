-- +goose Up
ALTER TABLE users
    ADD COLUMN terms_accepted_at TIMESTAMPTZ,
    ADD COLUMN privacy_accepted_at TIMESTAMPTZ,
    ADD COLUMN pdn_accepted_at TIMESTAMPTZ,
    ADD COLUMN consent_version TEXT NOT NULL DEFAULT '1.0',
    ADD COLUMN consent_ip TEXT,
    ADD COLUMN consent_user_agent TEXT;

-- +goose Down
ALTER TABLE users
    DROP COLUMN IF EXISTS terms_accepted_at,
    DROP COLUMN IF EXISTS privacy_accepted_at,
    DROP COLUMN IF EXISTS pdn_accepted_at,
    DROP COLUMN IF EXISTS consent_version,
    DROP COLUMN IF EXISTS consent_ip,
    DROP COLUMN IF EXISTS consent_user_agent;

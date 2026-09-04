-- +goose Up
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS role TEXT NOT NULL DEFAULT 'user';

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
ALTER TABLE users
    ADD CONSTRAINT users_role_check CHECK (role IN ('user', 'admin'));

CREATE TABLE IF NOT EXISTS transactions (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id              UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    product_id           TEXT NOT NULL,
    kind                 TEXT NOT NULL DEFAULT 'subscription',
    status               TEXT NOT NULL DEFAULT 'paid',
    provider             TEXT NOT NULL DEFAULT 'admin',
    provider_invoice_id  TEXT,
    provider_purchase_id TEXT,
    amount_kop           INTEGER NOT NULL DEFAULT 0,
    currency             TEXT NOT NULL DEFAULT 'RUB',
    sandbox              BOOLEAN NOT NULL DEFAULT FALSE,
    notes                TEXT NOT NULL DEFAULT '',
    paid_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    refunded_at          TIMESTAMPTZ,
    refunded_by          TEXT,
    refund_reason        TEXT NOT NULL DEFAULT '',
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT transactions_status_check CHECK (status IN ('pending', 'paid', 'refunded', 'canceled')),
    CONSTRAINT transactions_kind_check CHECK (kind IN ('subscription', 'one_time', 'promo', 'restore')),
    CONSTRAINT transactions_provider_check CHECK (provider IN ('rustore', 'admin', 'promo', 'dev'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_transactions_provider_invoice
    ON transactions (provider, provider_invoice_id)
    WHERE provider_invoice_id IS NOT NULL AND provider_invoice_id <> '';

CREATE INDEX IF NOT EXISTS idx_transactions_user_created
    ON transactions (user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_transactions_status_created
    ON transactions (status, created_at DESC);

CREATE TABLE IF NOT EXISTS admin_audit (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    admin_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    admin_email   TEXT NOT NULL,
    action        TEXT NOT NULL,
    target_type   TEXT NOT NULL,
    target_id     TEXT NOT NULL DEFAULT '',
    details       TEXT NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_admin_audit_created ON admin_audit (created_at DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_admin_audit_created;
DROP TABLE IF EXISTS admin_audit;
DROP INDEX IF EXISTS idx_transactions_status_created;
DROP INDEX IF EXISTS idx_transactions_user_created;
DROP INDEX IF EXISTS idx_transactions_provider_invoice;
DROP TABLE IF EXISTS transactions;
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
ALTER TABLE users DROP COLUMN IF EXISTS role;

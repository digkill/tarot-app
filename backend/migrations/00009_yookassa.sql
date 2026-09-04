-- +goose Up
ALTER TABLE transactions DROP CONSTRAINT IF EXISTS transactions_provider_check;
ALTER TABLE transactions
    ADD CONSTRAINT transactions_provider_check
    CHECK (provider IN ('rustore', 'admin', 'promo', 'dev', 'yookassa'));

CREATE TABLE IF NOT EXISTS checkout_sessions (
    transaction_id     UUID PRIMARY KEY REFERENCES transactions(id) ON DELETE CASCADE,
    confirmation_url   TEXT NOT NULL,
    provider_payment_id TEXT,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_checkout_provider_payment
    ON checkout_sessions (provider_payment_id)
    WHERE provider_payment_id IS NOT NULL AND provider_payment_id <> '';

-- +goose Down
DROP INDEX IF EXISTS idx_checkout_provider_payment;
DROP TABLE IF EXISTS checkout_sessions;
ALTER TABLE transactions DROP CONSTRAINT IF EXISTS transactions_provider_check;
ALTER TABLE transactions
    ADD CONSTRAINT transactions_provider_check
    CHECK (provider IN ('rustore', 'admin', 'promo', 'dev'));

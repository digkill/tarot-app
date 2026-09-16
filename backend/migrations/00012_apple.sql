-- +goose Up
ALTER TABLE transactions DROP CONSTRAINT IF EXISTS transactions_provider_check;
ALTER TABLE transactions
    ADD CONSTRAINT transactions_provider_check
    CHECK (provider IN ('rustore', 'admin', 'promo', 'dev', 'yookassa', 'cloudpayments', 'apple'));

-- Decks are sold as non-consumable IAP on iOS. Mirrors rustore_product_id.
ALTER TABLE decks ADD COLUMN IF NOT EXISTS apple_product_id TEXT;
CREATE UNIQUE INDEX IF NOT EXISTS idx_decks_apple_product
    ON decks (apple_product_id)
    WHERE apple_product_id IS NOT NULL AND apple_product_id <> '';

-- Maps an Apple subscription back to a user. Required because App Store Server
-- Notifications carry no user id of any kind: appAccountToken is the primary
-- mechanism, this table is the backstop for purchases that carry no token
-- (for example a re-subscribe started from the App Store subscription page).
CREATE TABLE IF NOT EXISTS apple_subscriptions (
    original_transaction_id TEXT PRIMARY KEY,
    user_id                 UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    product_id              TEXT NOT NULL DEFAULT '',
    apple_product_id        TEXT NOT NULL DEFAULT '',
    environment             TEXT NOT NULL DEFAULT 'Production',
    status                  INTEGER,
    auto_renew_status       INTEGER,
    expires_at              TIMESTAMPTZ,
    last_transaction_id     TEXT NOT NULL DEFAULT '',
    last_notification_type  TEXT NOT NULL DEFAULT '',
    last_notification_uuid  TEXT NOT NULL DEFAULT '',
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_apple_subscriptions_user
    ON apple_subscriptions (user_id);
CREATE INDEX IF NOT EXISTS idx_apple_subscriptions_expires
    ON apple_subscriptions (expires_at)
    WHERE expires_at IS NOT NULL;

-- Notification de-duplication. Apple retries a notification up to 5 times over
-- ~3 days, and a retry must not grant or revoke twice.
CREATE TABLE IF NOT EXISTS apple_notifications (
    notification_uuid       TEXT PRIMARY KEY,
    notification_type       TEXT NOT NULL DEFAULT '',
    subtype                 TEXT NOT NULL DEFAULT '',
    original_transaction_id TEXT NOT NULL DEFAULT '',
    received_at             TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE IF EXISTS apple_notifications;
DROP INDEX IF EXISTS idx_apple_subscriptions_expires;
DROP INDEX IF EXISTS idx_apple_subscriptions_user;
DROP TABLE IF EXISTS apple_subscriptions;
DROP INDEX IF EXISTS idx_decks_apple_product;
ALTER TABLE decks DROP COLUMN IF EXISTS apple_product_id;
ALTER TABLE transactions DROP CONSTRAINT IF EXISTS transactions_provider_check;
ALTER TABLE transactions
    ADD CONSTRAINT transactions_provider_check
    CHECK (provider IN ('rustore', 'admin', 'promo', 'dev', 'yookassa', 'cloudpayments'));

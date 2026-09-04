-- +goose Up
CREATE TABLE decks (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug                TEXT NOT NULL UNIQUE,
    title_i18n          JSONB NOT NULL DEFAULT '{}'::jsonb,
    description_i18n    JSONB NOT NULL DEFAULT '{}'::jsonb,
    theme               JSONB NOT NULL DEFAULT '{}'::jsonb,
    price_kop           INTEGER NOT NULL DEFAULT 0,
    currency            TEXT NOT NULL DEFAULT 'RUB',
    rustore_product_id  TEXT,
    is_free             BOOLEAN NOT NULL DEFAULT FALSE,
    is_published        BOOLEAN NOT NULL DEFAULT FALSE,
    sort_order          INTEGER NOT NULL DEFAULT 0,
    card_count          INTEGER NOT NULL DEFAULT 0,
    has_back            BOOLEAN NOT NULL DEFAULT FALSE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT decks_slug_check CHECK (slug ~ '^[a-z0-9][a-z0-9_-]{1,62}$')
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_decks_rustore_product
    ON decks (rustore_product_id)
    WHERE rustore_product_id IS NOT NULL AND rustore_product_id <> '';

CREATE TABLE user_decks (
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    deck_id         UUID NOT NULL REFERENCES decks(id) ON DELETE CASCADE,
    source          TEXT NOT NULL DEFAULT 'admin',
    transaction_id  UUID REFERENCES transactions(id) ON DELETE SET NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, deck_id)
);

CREATE INDEX IF NOT EXISTS idx_user_decks_user ON user_decks (user_id);

-- +goose Down
DROP INDEX IF EXISTS idx_user_decks_user;
DROP TABLE IF EXISTS user_decks;
DROP INDEX IF EXISTS idx_decks_rustore_product;
DROP TABLE IF EXISTS decks;

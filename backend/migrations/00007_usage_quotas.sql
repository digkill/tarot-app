-- +goose Up
CREATE TABLE daily_usage (
    user_id           UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    day               DATE NOT NULL,
    daily_cards       INTEGER NOT NULL DEFAULT 0,
    interpretations   INTEGER NOT NULL DEFAULT 0,
    ad_unlocks        INTEGER NOT NULL DEFAULT 0,
    last_ad_at        TIMESTAMPTZ,
    last_interpret_at TIMESTAMPTZ,
    PRIMARY KEY (user_id, day)
);

-- +goose Down
DROP TABLE IF EXISTS daily_usage;

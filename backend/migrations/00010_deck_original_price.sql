-- +goose Up
ALTER TABLE decks
    ADD COLUMN IF NOT EXISTS original_price_kop INTEGER;

-- +goose Down
ALTER TABLE decks DROP COLUMN IF EXISTS original_price_kop;

-- +goose Up
ALTER TABLE arcana_matches ADD COLUMN IF NOT EXISTS player1_deck TEXT NOT NULL DEFAULT '';
ALTER TABLE arcana_matches ADD COLUMN IF NOT EXISTS player2_deck TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE arcana_matches DROP COLUMN IF EXISTS player2_deck;
ALTER TABLE arcana_matches DROP COLUMN IF EXISTS player1_deck;

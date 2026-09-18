-- The winner as a seat (1 or 2), not only as a user id: a player without an
-- account row (a test bot) or a deleted account must not erase the result for
-- the other player.

-- +goose Up
ALTER TABLE arcana_matches ADD COLUMN IF NOT EXISTS winner_seat SMALLINT
    CHECK (winner_seat IS NULL OR winner_seat IN (1, 2));

UPDATE arcana_matches SET winner_seat = CASE
    WHEN winner_id IS NOT NULL AND winner_id = player1_id THEN 1
    WHEN winner_id IS NOT NULL AND winner_id = player2_id THEN 2
    ELSE winner_seat END
WHERE winner_seat IS NULL;

-- +goose Down
ALTER TABLE arcana_matches DROP COLUMN IF EXISTS winner_seat;

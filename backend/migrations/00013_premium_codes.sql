-- +goose Up
CREATE TABLE premium_codes (
    code_hash TEXT PRIMARY KEY CHECK (length(code_hash) = 64),
    redeemed_by UUID REFERENCES users(id) ON DELETE SET NULL,
    redeemed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (redeemed_by IS NULL OR redeemed_at IS NOT NULL)
);
-- Retain redeemed_at after account deletion so a used code never becomes reusable.
INSERT INTO premium_codes (code_hash) VALUES
    ('c8dacfc0cb0dd086805bff158e34c24cca47c25106f086fceddda0becacd1ea3'),
    ('5a7e3f8655d4ad54a8b1fce79973b49c20ceb8d9b3d25ef28ac482fd89d1fc1e');

-- +goose Down
DROP TABLE premium_codes;

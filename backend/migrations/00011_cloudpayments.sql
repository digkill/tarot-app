-- +goose Up
ALTER TABLE transactions DROP CONSTRAINT IF EXISTS transactions_provider_check;
ALTER TABLE transactions
    ADD CONSTRAINT transactions_provider_check
    CHECK (provider IN ('rustore', 'admin', 'promo', 'dev', 'yookassa', 'cloudpayments'));

-- +goose Down
ALTER TABLE transactions DROP CONSTRAINT IF EXISTS transactions_provider_check;
ALTER TABLE transactions
    ADD CONSTRAINT transactions_provider_check
    CHECK (provider IN ('rustore', 'admin', 'promo', 'dev', 'yookassa'));

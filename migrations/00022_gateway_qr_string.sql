-- +goose Up
ALTER TABLE transaction_gateway_payments ADD COLUMN IF NOT EXISTS qr_string TEXT;

-- +goose Down
ALTER TABLE transaction_gateway_payments DROP COLUMN IF EXISTS qr_string;

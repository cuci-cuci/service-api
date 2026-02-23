-- +goose Up
ALTER TABLE tenant_service_prices
  ADD COLUMN IF NOT EXISTS is_quick_add BOOLEAN NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE tenant_service_prices
  DROP COLUMN IF EXISTS is_quick_add;

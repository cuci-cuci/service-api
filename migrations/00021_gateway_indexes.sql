-- +goose Up
-- +goose NO TRANSACTION
CREATE INDEX CONCURRENTLY idx_tgp_tenant_status_created
  ON transaction_gateway_payments(tenant_id, gateway_status, created_at DESC);

CREATE INDEX CONCURRENTLY idx_tgp_tenant_created
  ON transaction_gateway_payments(tenant_id, created_at DESC);

DROP INDEX CONCURRENTLY IF EXISTS idx_tgp_tenant;

-- +goose Down
-- +goose NO TRANSACTION
CREATE INDEX CONCURRENTLY idx_tgp_tenant ON transaction_gateway_payments(tenant_id);
DROP INDEX CONCURRENTLY IF EXISTS idx_tgp_tenant_status_created;
DROP INDEX CONCURRENTLY IF EXISTS idx_tgp_tenant_created;

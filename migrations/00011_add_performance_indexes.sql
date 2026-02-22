-- +goose Up
CREATE INDEX IF NOT EXISTS idx_shifts_cashier_outlet_status ON shifts(cashier_id, outlet_id, status);
CREATE INDEX IF NOT EXISTS idx_orders_outlet_status ON orders(outlet_id, status);
CREATE INDEX IF NOT EXISTS idx_transactions_tenant_member ON transactions(tenant_id, member_id);

-- +goose Down
DROP INDEX IF EXISTS idx_shifts_cashier_outlet_status;
DROP INDEX IF EXISTS idx_orders_outlet_status;
DROP INDEX IF EXISTS idx_transactions_tenant_member;

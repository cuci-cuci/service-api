-- +goose Up

CREATE TABLE shifts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    outlet_id UUID NOT NULL REFERENCES outlets(id),
    cashier_id UUID NOT NULL REFERENCES users(id),
    opening_cash BIGINT NOT NULL DEFAULT 0,
    closing_cash BIGINT,
    expected_cash BIGINT,
    cash_difference BIGINT,
    status VARCHAR(10) NOT NULL DEFAULT 'open',
    opened_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    closed_at TIMESTAMPTZ,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE shift_transactions (
    shift_id UUID NOT NULL REFERENCES shifts(id) ON DELETE CASCADE,
    transaction_id UUID NOT NULL REFERENCES transactions(id),
    PRIMARY KEY (shift_id, transaction_id)
);

CREATE INDEX idx_shifts_tenant_id ON shifts(tenant_id);
CREATE INDEX idx_shifts_outlet_id ON shifts(outlet_id);
CREATE INDEX idx_shifts_cashier_id ON shifts(cashier_id);
CREATE INDEX idx_shifts_status ON shifts(status);

ALTER TABLE shifts ENABLE ROW LEVEL SECURITY;
ALTER TABLE shift_transactions ENABLE ROW LEVEL SECURITY;

CREATE POLICY shifts_tenant_isolation ON shifts
    USING (tenant_id = COALESCE(current_setting('app.current_tenant_id', true)::uuid, tenant_id));

CREATE POLICY shift_transactions_tenant_isolation ON shift_transactions
    USING (shift_id IN (SELECT id FROM shifts WHERE tenant_id = COALESCE(current_setting('app.current_tenant_id', true)::uuid, tenant_id)));

-- +goose Down
DROP POLICY IF EXISTS shift_transactions_tenant_isolation ON shift_transactions;
DROP POLICY IF EXISTS shifts_tenant_isolation ON shifts;
DROP TABLE IF EXISTS shift_transactions;
DROP TABLE IF EXISTS shifts;

-- +goose Up
CREATE TABLE IF NOT EXISTS payment_methods (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    type VARCHAR(20) NOT NULL CHECK (type IN ('cash', 'qris', 'bank_transfer', 'ewallet', 'other')),
    is_active BOOLEAN NOT NULL DEFAULT true,
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_payment_methods_tenant_id ON payment_methods(tenant_id);

ALTER TABLE payment_methods ENABLE ROW LEVEL SECURITY;

CREATE POLICY payment_methods_tenant_isolation ON payment_methods
    FOR ALL USING (
        current_setting('app.current_role', true) = 'superadmin'
        OR tenant_id::text = current_setting('app.current_tenant_id', true)
    );

-- +goose Down
DROP TABLE IF EXISTS payment_methods;

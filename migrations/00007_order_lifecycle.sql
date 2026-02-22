-- +goose Up

CREATE TABLE orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    transaction_id UUID UNIQUE REFERENCES transactions(id),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    outlet_id UUID NOT NULL REFERENCES outlets(id),
    status VARCHAR(20) NOT NULL DEFAULT 'received',
    estimated_completion_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    picked_up_at TIMESTAMPTZ,
    notes TEXT,
    created_by UUID NOT NULL REFERENCES users(id),
    updated_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE order_status_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    from_status VARCHAR(20),
    to_status VARCHAR(20) NOT NULL,
    changed_by UUID NOT NULL REFERENCES users(id),
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_orders_tenant_id ON orders(tenant_id);
CREATE INDEX idx_orders_outlet_id ON orders(outlet_id);
CREATE INDEX idx_orders_status ON orders(status);
CREATE INDEX idx_orders_transaction_id ON orders(transaction_id);
CREATE INDEX idx_order_status_logs_order_id ON order_status_logs(order_id);

ALTER TABLE orders ENABLE ROW LEVEL SECURITY;
ALTER TABLE order_status_logs ENABLE ROW LEVEL SECURITY;

CREATE POLICY orders_tenant_isolation ON orders
    USING (tenant_id = COALESCE(
        current_setting('app.current_tenant_id', true)::uuid,
        tenant_id
    ));

CREATE POLICY order_status_logs_tenant_isolation ON order_status_logs
    USING (order_id IN (
        SELECT id FROM orders WHERE tenant_id = COALESCE(
            current_setting('app.current_tenant_id', true)::uuid,
            tenant_id
        )
    ));

-- +goose Down
DROP POLICY IF EXISTS order_status_logs_tenant_isolation ON order_status_logs;
DROP POLICY IF EXISTS orders_tenant_isolation ON orders;
DROP TABLE IF EXISTS order_status_logs;
DROP TABLE IF EXISTS orders;

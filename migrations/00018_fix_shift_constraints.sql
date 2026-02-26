-- +goose Up

-- Fix race condition: enforce at most 1 open shift per cashier at DB level
CREATE UNIQUE INDEX idx_shifts_one_open_per_cashier
    ON shifts(cashier_id) WHERE status = 'open';

-- Composite index for GetCurrentShift query performance
CREATE INDEX idx_shifts_cashier_open_opened
    ON shifts(cashier_id, opened_at DESC) WHERE status = 'open';

-- Fix RLS policies to match correct pattern from 00001_init_schema.sql
DROP POLICY IF EXISTS shifts_tenant_isolation ON shifts;
CREATE POLICY shifts_tenant_isolation ON shifts
    USING (
        current_setting('app.current_role', true) = 'superadmin'
        OR tenant_id::text = current_setting('app.current_tenant_id', true)
    );

DROP POLICY IF EXISTS shift_transactions_tenant_isolation ON shift_transactions;
CREATE POLICY shift_transactions_tenant_isolation ON shift_transactions
    USING (
        current_setting('app.current_role', true) = 'superadmin'
        OR shift_id IN (
            SELECT id FROM shifts
            WHERE tenant_id::text = current_setting('app.current_tenant_id', true)
        )
    );

-- +goose Down
DROP INDEX IF EXISTS idx_shifts_one_open_per_cashier;
DROP INDEX IF EXISTS idx_shifts_cashier_open_opened;

DROP POLICY IF EXISTS shifts_tenant_isolation ON shifts;
CREATE POLICY shifts_tenant_isolation ON shifts
    USING (tenant_id = COALESCE(current_setting('app.current_tenant_id', true)::uuid, tenant_id));

DROP POLICY IF EXISTS shift_transactions_tenant_isolation ON shift_transactions;
CREATE POLICY shift_transactions_tenant_isolation ON shift_transactions
    USING (shift_id IN (SELECT id FROM shifts WHERE tenant_id = COALESCE(current_setting('app.current_tenant_id', true)::uuid, tenant_id)));

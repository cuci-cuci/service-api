-- +goose Up

-- Add delivery fields to orders
ALTER TABLE orders ADD COLUMN IF NOT EXISTS delivery_type VARCHAR(20) DEFAULT 'pickup';
ALTER TABLE orders ADD COLUMN IF NOT EXISTS delivery_address TEXT;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS delivery_fee BIGINT DEFAULT 0;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS scheduled_pickup_at TIMESTAMPTZ;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS customer_phone VARCHAR(20);

-- Staff activity log (void, cancel, refund tracking)
CREATE TABLE staff_activity_logs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id),
  user_id UUID NOT NULL REFERENCES users(id),
  activity_type VARCHAR(50) NOT NULL,
  reference_id UUID,
  reference_type VARCHAR(50),
  amount BIGINT DEFAULT 0,
  notes TEXT,
  created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_staff_activity_tenant ON staff_activity_logs(tenant_id, created_at DESC);
CREATE INDEX idx_staff_activity_user ON staff_activity_logs(user_id, created_at DESC);
CREATE INDEX idx_staff_activity_type ON staff_activity_logs(tenant_id, activity_type);

ALTER TABLE staff_activity_logs ENABLE ROW LEVEL SECURITY;

CREATE POLICY staff_activity_logs_tenant_isolation ON staff_activity_logs
  USING (tenant_id = COALESCE(current_setting('app.current_tenant_id', true)::uuid, tenant_id));

-- +goose Down
ALTER TABLE orders DROP COLUMN IF EXISTS delivery_type;
ALTER TABLE orders DROP COLUMN IF EXISTS delivery_address;
ALTER TABLE orders DROP COLUMN IF EXISTS delivery_fee;
ALTER TABLE orders DROP COLUMN IF EXISTS scheduled_pickup_at;
ALTER TABLE orders DROP COLUMN IF EXISTS customer_phone;
DROP TABLE IF EXISTS staff_activity_logs;

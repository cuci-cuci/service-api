-- +goose Up

-- Delivery zones per outlet
CREATE TABLE IF NOT EXISTS delivery_zones (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id),
  outlet_id UUID NOT NULL REFERENCES outlets(id),
  name VARCHAR(100) NOT NULL,
  district VARCHAR(200),
  fee BIGINT NOT NULL DEFAULT 0,
  estimated_minutes INT DEFAULT 60,
  is_active BOOLEAN DEFAULT TRUE,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_delivery_zones_outlet ON delivery_zones(tenant_id, outlet_id, is_active);
ALTER TABLE delivery_zones ENABLE ROW LEVEL SECURITY;
CREATE POLICY delivery_zones_tenant_isolation ON delivery_zones
  USING (tenant_id = COALESCE(current_setting('app.current_tenant_id', true)::uuid, tenant_id));

-- Pickup requests from customers
CREATE TABLE IF NOT EXISTS pickup_requests (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id),
  outlet_id UUID NOT NULL REFERENCES outlets(id),
  zone_id UUID REFERENCES delivery_zones(id),
  order_id UUID REFERENCES orders(id),
  customer_name VARCHAR(200) NOT NULL,
  customer_phone VARCHAR(20) NOT NULL,
  address TEXT NOT NULL,
  pickup_type VARCHAR(20) NOT NULL DEFAULT 'pickup',
  status VARCHAR(20) NOT NULL DEFAULT 'pending',
  scheduled_at TIMESTAMPTZ,
  completed_at TIMESTAMPTZ,
  notes TEXT,
  delivery_fee BIGINT DEFAULT 0,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_pickup_requests_outlet ON pickup_requests(tenant_id, outlet_id, status);
CREATE INDEX idx_pickup_requests_status ON pickup_requests(tenant_id, status, created_at DESC);
ALTER TABLE pickup_requests ENABLE ROW LEVEL SECURITY;
CREATE POLICY pickup_requests_tenant_isolation ON pickup_requests
  USING (tenant_id = COALESCE(current_setting('app.current_tenant_id', true)::uuid, tenant_id));

-- Link orders to delivery zones
ALTER TABLE orders ADD COLUMN IF NOT EXISTS delivery_zone_id UUID REFERENCES delivery_zones(id);

-- +goose Down
ALTER TABLE orders DROP COLUMN IF EXISTS delivery_zone_id;
DROP TABLE IF EXISTS pickup_requests;
DROP TABLE IF EXISTS delivery_zones;

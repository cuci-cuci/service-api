-- +goose Up

-- Supply categories (deterjen, plastik, pewangi, dll)
CREATE TABLE supply_categories (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id),
  name VARCHAR(100) NOT NULL,
  icon VARCHAR(50) DEFAULT 'Package',
  is_active BOOLEAN DEFAULT true,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW(),
  UNIQUE(tenant_id, name)
);

-- Supplies (individual items with stock tracking)
CREATE TABLE supplies (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id),
  category_id UUID REFERENCES supply_categories(id),
  outlet_id UUID REFERENCES outlets(id),
  name VARCHAR(200) NOT NULL,
  unit VARCHAR(50) NOT NULL DEFAULT 'pcs',
  current_stock NUMERIC(12,2) NOT NULL DEFAULT 0,
  min_stock NUMERIC(12,2) DEFAULT 0,
  cost_per_unit BIGINT DEFAULT 0,
  is_active BOOLEAN DEFAULT true,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_supplies_tenant ON supplies(tenant_id);
CREATE INDEX idx_supplies_category ON supplies(category_id);
CREATE INDEX idx_supplies_low_stock ON supplies(tenant_id, current_stock, min_stock)
  WHERE is_active = true;

-- Stock movements (in/out log)
CREATE TABLE stock_movements (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id),
  supply_id UUID NOT NULL REFERENCES supplies(id) ON DELETE CASCADE,
  movement_type VARCHAR(20) NOT NULL CHECK (movement_type IN ('in', 'out', 'adjustment')),
  quantity NUMERIC(12,2) NOT NULL,
  notes TEXT,
  created_by UUID NOT NULL REFERENCES users(id),
  created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_stock_movements_supply ON stock_movements(supply_id, created_at DESC);
CREATE INDEX idx_stock_movements_tenant ON stock_movements(tenant_id, created_at DESC);

-- RLS
ALTER TABLE supply_categories ENABLE ROW LEVEL SECURITY;
ALTER TABLE supplies ENABLE ROW LEVEL SECURITY;
ALTER TABLE stock_movements ENABLE ROW LEVEL SECURITY;

CREATE POLICY supply_categories_tenant_isolation ON supply_categories
  USING (tenant_id = COALESCE(current_setting('app.current_tenant_id', true)::uuid, tenant_id));

CREATE POLICY supplies_tenant_isolation ON supplies
  USING (tenant_id = COALESCE(current_setting('app.current_tenant_id', true)::uuid, tenant_id));

CREATE POLICY stock_movements_tenant_isolation ON stock_movements
  USING (tenant_id = COALESCE(current_setting('app.current_tenant_id', true)::uuid, tenant_id));

-- +goose Down
DROP TABLE IF EXISTS stock_movements;
DROP TABLE IF EXISTS supplies;
DROP TABLE IF EXISTS supply_categories;

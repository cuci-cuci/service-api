-- +goose Up

CREATE TABLE expense_categories (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id),
  name VARCHAR(100) NOT NULL,
  icon VARCHAR(50) DEFAULT 'receipt',
  is_active BOOLEAN DEFAULT true,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);

ALTER TABLE expense_categories ENABLE ROW LEVEL SECURITY;
CREATE POLICY expense_categories_tenant ON expense_categories
  USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE TABLE expenses (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id),
  outlet_id UUID REFERENCES outlets(id),
  category_id UUID REFERENCES expense_categories(id),
  amount BIGINT NOT NULL CHECK (amount > 0),
  description TEXT,
  expense_date DATE NOT NULL,
  receipt_url TEXT,
  created_by UUID REFERENCES users(id),
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);

ALTER TABLE expenses ENABLE ROW LEVEL SECURITY;
CREATE POLICY expenses_tenant ON expenses
  USING (tenant_id = current_setting('app.tenant_id')::uuid);
CREATE INDEX idx_expenses_tenant_date ON expenses(tenant_id, expense_date);
CREATE INDEX idx_expenses_tenant_category ON expenses(tenant_id, category_id);

CREATE TABLE recurring_expenses (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id),
  category_id UUID REFERENCES expense_categories(id),
  outlet_id UUID REFERENCES outlets(id),
  amount BIGINT NOT NULL CHECK (amount > 0),
  description TEXT,
  frequency VARCHAR(20) NOT NULL CHECK (frequency IN ('daily','weekly','monthly','yearly')),
  next_due_date DATE NOT NULL,
  is_active BOOLEAN DEFAULT true,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);

ALTER TABLE recurring_expenses ENABLE ROW LEVEL SECURITY;
CREATE POLICY recurring_expenses_tenant ON recurring_expenses
  USING (tenant_id = current_setting('app.tenant_id')::uuid);

-- +goose Down
DROP TABLE IF EXISTS recurring_expenses;
DROP TABLE IF EXISTS expenses;
DROP TABLE IF EXISTS expense_categories;

-- +goose Up
CREATE TABLE dashboard_goals (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id),
  goal_type VARCHAR(50) NOT NULL,
  target_value BIGINT NOT NULL CHECK (target_value > 0),
  is_active BOOLEAN DEFAULT true,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW(),
  UNIQUE(tenant_id, goal_type)
);

CREATE INDEX idx_dashboard_goals_tenant ON dashboard_goals(tenant_id);

-- +goose Down
DROP TABLE IF EXISTS dashboard_goals;

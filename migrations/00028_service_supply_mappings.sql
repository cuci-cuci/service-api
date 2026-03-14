-- +goose Up
CREATE TABLE service_supply_mappings (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id),
  service_template_id UUID NOT NULL REFERENCES service_templates(id),
  supply_id UUID NOT NULL REFERENCES supplies(id),
  quantity_per_unit NUMERIC(10,3) NOT NULL DEFAULT 1.0,
  unit VARCHAR(20) DEFAULT 'pcs',
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW(),
  UNIQUE(tenant_id, service_template_id, supply_id)
);

CREATE INDEX idx_ssm_tenant ON service_supply_mappings(tenant_id);
CREATE INDEX idx_ssm_service ON service_supply_mappings(tenant_id, service_template_id);

ALTER TABLE service_supply_mappings ENABLE ROW LEVEL SECURITY;

CREATE POLICY ssm_tenant_isolation ON service_supply_mappings
  USING (tenant_id = COALESCE(current_setting('app.current_tenant_id', true)::uuid, tenant_id));

-- +goose Down
DROP TABLE IF EXISTS service_supply_mappings;

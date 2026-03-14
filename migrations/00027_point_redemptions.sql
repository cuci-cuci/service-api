-- +goose Up
CREATE TABLE IF NOT EXISTS point_redemptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    member_id UUID NOT NULL REFERENCES members(id),
    points_redeemed INT NOT NULL CHECK (points_redeemed > 0),
    discount_amount BIGINT NOT NULL CHECK (discount_amount > 0),
    transaction_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_point_redemptions_member ON point_redemptions(member_id);
CREATE INDEX IF NOT EXISTS idx_point_redemptions_tenant ON point_redemptions(tenant_id);

ALTER TABLE point_redemptions ENABLE ROW LEVEL SECURITY;

CREATE POLICY point_redemptions_tenant_isolation ON point_redemptions
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);

-- +goose Down
DROP TABLE IF EXISTS point_redemptions;

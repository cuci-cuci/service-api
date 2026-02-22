-- +goose Up
ALTER TABLE members DROP CONSTRAINT IF EXISTS members_phone_key;
CREATE UNIQUE INDEX idx_members_tenant_phone ON members (COALESCE(tenant_id, '00000000-0000-0000-0000-000000000000'::uuid), phone);

-- +goose Down
DROP INDEX IF EXISTS idx_members_tenant_phone;
ALTER TABLE members ADD CONSTRAINT members_phone_key UNIQUE (phone);

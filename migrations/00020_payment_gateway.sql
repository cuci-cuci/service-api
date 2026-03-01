-- +goose Up
-- +goose StatementBegin

-- Per-tenant gateway configuration (one row per tenant per gateway provider)
CREATE TABLE payment_gateway_configs (
    id                      UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id               UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    gateway                 VARCHAR(50) NOT NULL DEFAULT 'xendit' CHECK (gateway IN ('xendit')),
    is_enabled              BOOLEAN NOT NULL DEFAULT false,
    secret_key_encrypted    TEXT,
    public_key              TEXT,
    webhook_token_encrypted TEXT,
    enabled_types           TEXT[] NOT NULL DEFAULT '{}',
    key_version             INTEGER NOT NULL DEFAULT 1,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, gateway)
);

CREATE INDEX idx_pgw_configs_tenant ON payment_gateway_configs(tenant_id);

ALTER TABLE payment_gateway_configs ENABLE ROW LEVEL SECURITY;
CREATE POLICY pgw_configs_tenant_isolation ON payment_gateway_configs
    USING (
        current_setting('app.current_role', true) = 'superadmin'
        OR tenant_id::text = current_setting('app.current_tenant_id', true)
    );
ALTER TABLE payment_gateway_configs FORCE ROW LEVEL SECURITY;

-- Satellite table: gateway payment attempts linked to transactions
CREATE TABLE transaction_gateway_payments (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id           UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    transaction_id      UUID NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
    payment_item_id     UUID,
    gateway             VARCHAR(50) NOT NULL DEFAULT 'xendit',
    gateway_type        VARCHAR(50) NOT NULL CHECK (gateway_type IN ('qris', 'virtual_account', 'ewallet')),
    external_id         VARCHAR(255) NOT NULL UNIQUE,
    gateway_ref_id      VARCHAR(255),
    amount              BIGINT NOT NULL,
    gateway_status      VARCHAR(50) NOT NULL DEFAULT 'PENDING'
                            CHECK (gateway_status IN ('PENDING','ACTIVE','PAID','EXPIRED','FAILED','CANCELLED')),
    gateway_payment_url TEXT,
    gateway_response    JSONB,
    webhook_payload     JSONB,
    expires_at          TIMESTAMPTZ,
    paid_at             TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tgp_tenant ON transaction_gateway_payments(tenant_id);
CREATE INDEX idx_tgp_transaction ON transaction_gateway_payments(transaction_id);
CREATE INDEX idx_tgp_external_id ON transaction_gateway_payments(external_id);
CREATE INDEX idx_tgp_status ON transaction_gateway_payments(gateway_status) WHERE gateway_status IN ('PENDING','ACTIVE');

ALTER TABLE transaction_gateway_payments ENABLE ROW LEVEL SECURITY;
CREATE POLICY tgp_tenant_isolation ON transaction_gateway_payments
    USING (
        current_setting('app.current_role', true) = 'superadmin'
        OR tenant_id::text = current_setting('app.current_tenant_id', true)
    );
ALTER TABLE transaction_gateway_payments FORCE ROW LEVEL SECURITY;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS transaction_gateway_payments;
DROP TABLE IF EXISTS payment_gateway_configs;
-- +goose StatementEnd

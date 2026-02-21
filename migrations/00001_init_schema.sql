-- +goose Up
-- +goose StatementBegin

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Tenants
CREATE TABLE tenants (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) NOT NULL UNIQUE,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tenants_slug ON tenants(slug);
CREATE INDEX idx_tenants_is_active ON tenants(is_active);

-- Users
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL CHECK (role IN ('superadmin', 'tenant_owner', 'cashier')),
    tenant_id UUID REFERENCES tenants(id) ON DELETE SET NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_tenant_id ON users(tenant_id);
CREATE INDEX idx_users_role ON users(role);

-- Outlets
CREATE TABLE outlets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    address TEXT NOT NULL DEFAULT '',
    phone VARCHAR(50) NOT NULL DEFAULT '',
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_outlets_tenant_id ON outlets(tenant_id);

-- Service Categories
CREATE TABLE service_categories (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    icon VARCHAR(100) NOT NULL DEFAULT '',
    sort_order INT NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_service_categories_sort_order ON service_categories(sort_order);

-- Service Templates
CREATE TABLE service_templates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    category_id UUID NOT NULL REFERENCES service_categories(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    pricing_unit VARCHAR(20) NOT NULL CHECK (pricing_unit IN ('kg', 'pcs', 'meter', 'pair', 'sqm')),
    base_price BIGINT NOT NULL DEFAULT 0,
    estimated_duration_hours INT NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT true,
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_service_templates_category_id ON service_templates(category_id);
CREATE INDEX idx_service_templates_sort_order ON service_templates(sort_order);

-- Tenant Service Prices
CREATE TABLE tenant_service_prices (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    service_template_id UUID NOT NULL REFERENCES service_templates(id) ON DELETE CASCADE,
    price BIGINT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    UNIQUE(tenant_id, service_template_id)
);

CREATE INDEX idx_tenant_service_prices_tenant_id ON tenant_service_prices(tenant_id);

-- Config Versions
CREATE TABLE config_versions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    version INT NOT NULL,
    data JSONB NOT NULL,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, version)
);

CREATE INDEX idx_config_versions_tenant_id ON config_versions(tenant_id);
CREATE INDEX idx_config_versions_tenant_version ON config_versions(tenant_id, version DESC);

-- Feature Flags
CREATE TABLE feature_flags (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    key VARCHAR(100) NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    default_enabled BOOLEAN NOT NULL DEFAULT false
);

CREATE INDEX idx_feature_flags_key ON feature_flags(key);

-- Tenant Feature Flags
CREATE TABLE tenant_feature_flags (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    feature_flag_id UUID NOT NULL REFERENCES feature_flags(id) ON DELETE CASCADE,
    enabled BOOLEAN NOT NULL DEFAULT false,
    UNIQUE(tenant_id, feature_flag_id)
);

CREATE INDEX idx_tenant_feature_flags_tenant_id ON tenant_feature_flags(tenant_id);

-- Transactions
CREATE TABLE transactions (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    outlet_id UUID NOT NULL REFERENCES outlets(id) ON DELETE CASCADE,
    local_order_number VARCHAR(100) NOT NULL DEFAULT '',
    customer_name VARCHAR(255),
    items JSONB NOT NULL DEFAULT '[]'::jsonb,
    subtotal BIGINT NOT NULL DEFAULT 0,
    discount_amount BIGINT NOT NULL DEFAULT 0,
    tax_amount BIGINT NOT NULL DEFAULT 0,
    total_amount BIGINT NOT NULL DEFAULT 0,
    payment_status VARCHAR(50) NOT NULL DEFAULT 'pending',
    payments JSONB NOT NULL DEFAULT '[]'::jsonb,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    config_version_id UUID NOT NULL,
    notes TEXT,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    synced_at TIMESTAMPTZ
);

CREATE INDEX idx_transactions_tenant_id ON transactions(tenant_id);
CREATE INDEX idx_transactions_outlet_id ON transactions(outlet_id);
CREATE INDEX idx_transactions_created_at ON transactions(created_at);
CREATE INDEX idx_transactions_tenant_created ON transactions(tenant_id, created_at DESC);

-- Members
CREATE TABLE members (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    phone VARCHAR(50) NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL DEFAULT '',
    tier VARCHAR(20) NOT NULL DEFAULT 'bronze' CHECK (tier IN ('bronze', 'silver', 'gold', 'platinum')),
    discount_percent INT NOT NULL DEFAULT 0,
    total_points INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_members_phone ON members(phone);
CREATE INDEX idx_members_tenant_id ON members(tenant_id);

-- Audit Logs
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    actor_id UUID NOT NULL,
    actor_name VARCHAR(255) NOT NULL,
    action VARCHAR(100) NOT NULL,
    entity_type VARCHAR(100) NOT NULL,
    entity_id VARCHAR(255) NOT NULL,
    old_value JSONB,
    new_value JSONB,
    tenant_id UUID REFERENCES tenants(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_logs_actor_id ON audit_logs(actor_id);
CREATE INDEX idx_audit_logs_entity ON audit_logs(entity_type, entity_id);
CREATE INDEX idx_audit_logs_tenant_id ON audit_logs(tenant_id);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at DESC);

-- Sync Sessions
CREATE TABLE sync_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    outlet_id UUID NOT NULL REFERENCES outlets(id) ON DELETE CASCADE,
    direction VARCHAR(20) NOT NULL CHECK (direction IN ('upload', 'download')),
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'in_progress', 'completed', 'failed')),
    transaction_count INT NOT NULL DEFAULT 0,
    error_message TEXT,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

CREATE INDEX idx_sync_sessions_tenant_id ON sync_sessions(tenant_id);
CREATE INDEX idx_sync_sessions_started_at ON sync_sessions(started_at DESC);

-- Enable Row Level Security on tenant-scoped tables
ALTER TABLE outlets ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenant_service_prices ENABLE ROW LEVEL SECURITY;
ALTER TABLE config_versions ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenant_feature_flags ENABLE ROW LEVEL SECURITY;
ALTER TABLE transactions ENABLE ROW LEVEL SECURITY;
ALTER TABLE members ENABLE ROW LEVEL SECURITY;
ALTER TABLE sync_sessions ENABLE ROW LEVEL SECURITY;

-- RLS Policies: Superadmin bypasses, tenant users see own data
-- Outlets
CREATE POLICY outlets_tenant_isolation ON outlets
    USING (
        current_setting('app.current_role', true) = 'superadmin'
        OR tenant_id::text = current_setting('app.current_tenant_id', true)
    );

-- Tenant Service Prices
CREATE POLICY tenant_service_prices_isolation ON tenant_service_prices
    USING (
        current_setting('app.current_role', true) = 'superadmin'
        OR tenant_id::text = current_setting('app.current_tenant_id', true)
    );

-- Config Versions
CREATE POLICY config_versions_isolation ON config_versions
    USING (
        current_setting('app.current_role', true) = 'superadmin'
        OR tenant_id::text = current_setting('app.current_tenant_id', true)
    );

-- Tenant Feature Flags
CREATE POLICY tenant_feature_flags_isolation ON tenant_feature_flags
    USING (
        current_setting('app.current_role', true) = 'superadmin'
        OR tenant_id::text = current_setting('app.current_tenant_id', true)
    );

-- Transactions
CREATE POLICY transactions_tenant_isolation ON transactions
    USING (
        current_setting('app.current_role', true) = 'superadmin'
        OR tenant_id::text = current_setting('app.current_tenant_id', true)
    );

-- Members (global members have NULL tenant_id)
CREATE POLICY members_tenant_isolation ON members
    USING (
        current_setting('app.current_role', true) = 'superadmin'
        OR tenant_id IS NULL
        OR tenant_id::text = current_setting('app.current_tenant_id', true)
    );

-- Sync Sessions
CREATE POLICY sync_sessions_tenant_isolation ON sync_sessions
    USING (
        current_setting('app.current_role', true) = 'superadmin'
        OR tenant_id::text = current_setting('app.current_tenant_id', true)
    );

-- Force RLS for table owner too (important for application-level RLS)
ALTER TABLE outlets FORCE ROW LEVEL SECURITY;
ALTER TABLE tenant_service_prices FORCE ROW LEVEL SECURITY;
ALTER TABLE config_versions FORCE ROW LEVEL SECURITY;
ALTER TABLE tenant_feature_flags FORCE ROW LEVEL SECURITY;
ALTER TABLE transactions FORCE ROW LEVEL SECURITY;
ALTER TABLE members FORCE ROW LEVEL SECURITY;
ALTER TABLE sync_sessions FORCE ROW LEVEL SECURITY;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP POLICY IF EXISTS sync_sessions_tenant_isolation ON sync_sessions;
DROP POLICY IF EXISTS members_tenant_isolation ON members;
DROP POLICY IF EXISTS transactions_tenant_isolation ON transactions;
DROP POLICY IF EXISTS tenant_feature_flags_isolation ON tenant_feature_flags;
DROP POLICY IF EXISTS config_versions_isolation ON config_versions;
DROP POLICY IF EXISTS tenant_service_prices_isolation ON tenant_service_prices;
DROP POLICY IF EXISTS outlets_tenant_isolation ON outlets;

DROP TABLE IF EXISTS sync_sessions;
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS members;
DROP TABLE IF EXISTS transactions;
DROP TABLE IF EXISTS tenant_feature_flags;
DROP TABLE IF EXISTS feature_flags;
DROP TABLE IF EXISTS config_versions;
DROP TABLE IF EXISTS tenant_service_prices;
DROP TABLE IF EXISTS service_templates;
DROP TABLE IF EXISTS service_categories;
DROP TABLE IF EXISTS outlets;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS tenants;

DROP EXTENSION IF EXISTS "uuid-ossp";

-- +goose StatementEnd

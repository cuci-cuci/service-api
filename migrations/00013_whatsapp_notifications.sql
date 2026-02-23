-- +goose Up

-- Tenant WhatsApp notification settings
CREATE TABLE tenant_notification_settings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    whatsapp_enabled BOOLEAN NOT NULL DEFAULT false,
    fonnte_api_token TEXT, -- per-tenant Fonnte API token
    notify_on_received BOOLEAN NOT NULL DEFAULT true,
    notify_on_done BOOLEAN NOT NULL DEFAULT true,
    notify_on_picked_up BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(tenant_id)
);

-- Notification log for tracking delivery
CREATE TABLE notification_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    order_id UUID REFERENCES orders(id) ON DELETE SET NULL,
    phone TEXT NOT NULL,
    message TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending', -- pending, sent, failed
    provider_response TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_notification_logs_tenant ON notification_logs(tenant_id);
CREATE INDEX idx_notification_logs_order ON notification_logs(order_id);

-- Add customer_phone to orders for direct notification
ALTER TABLE orders ADD COLUMN IF NOT EXISTS customer_phone TEXT;

-- Add tracking_token for public order tracking (short random code)
ALTER TABLE orders ADD COLUMN IF NOT EXISTS tracking_token TEXT;
CREATE UNIQUE INDEX idx_orders_tracking_token ON orders(tracking_token) WHERE tracking_token IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_orders_tracking_token;
ALTER TABLE orders DROP COLUMN IF EXISTS tracking_token;
ALTER TABLE orders DROP COLUMN IF EXISTS customer_phone;
DROP TABLE IF EXISTS notification_logs;
DROP TABLE IF EXISTS tenant_notification_settings;

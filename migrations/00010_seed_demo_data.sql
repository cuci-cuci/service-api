-- +goose Up
-- +goose StatementBegin

-- Demo Tenant
INSERT INTO tenants (id, name, slug, is_active, created_at, updated_at)
VALUES (
    'e701d2d3-f5d6-45d2-8202-fb514a695377',
    'Laundry Bersih Cemerlang',
    'bersih-cemerlang',
    true, NOW(), NOW()
) ON CONFLICT (id) DO NOTHING;

-- Demo Outlets
INSERT INTO outlets (id, tenant_id, name, address, phone, is_active, created_at, updated_at)
VALUES
    ('b1000001-0000-0000-0000-000000000001', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'Cabang Pusat', 'Jl. Sudirman No. 10, Jakarta', '081234567890', true, NOW(), NOW()),
    ('b1000001-0000-0000-0000-000000000002', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'Cabang Selatan', 'Jl. Gatot Subroto No. 25, Jakarta', '081234567891', true, NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- Owner account (password: owner123456)
INSERT INTO users (id, email, name, password_hash, role, tenant_id, is_active, created_at, updated_at)
VALUES (
    'a1000001-0000-0000-0000-000000000001',
    'owner@demo.cuci.app',
    'Demo Owner',
    '$2a$10$ETrFNETHjBEXRbQeA09qTeNXvZHMasFPQdzUk0..QBV4Ps0L0ZEjO',
    'tenant_owner',
    'e701d2d3-f5d6-45d2-8202-fb514a695377',
    true, NOW(), NOW()
) ON CONFLICT (email) DO NOTHING;

-- Cashier account (password: kasir123456)
INSERT INTO users (id, email, name, password_hash, role, tenant_id, is_active, created_at, updated_at)
VALUES (
    'a1000001-0000-0000-0000-000000000002',
    'kasir@demo.cuci.app',
    'Demo Kasir',
    '$2a$10$nXrdFfM13BMClRITX8YgteJSN6riSf3xMmN19OrpAegDlXN7G0WTW',
    'cashier',
    'e701d2d3-f5d6-45d2-8202-fb514a695377',
    true, NOW(), NOW()
) ON CONFLICT (email) DO NOTHING;

-- Set cashier's outlet_id (added by migration 00006)
UPDATE users SET outlet_id = 'b1000001-0000-0000-0000-000000000001' WHERE id = 'a1000001-0000-0000-0000-000000000002' AND outlet_id IS NULL;

-- Payment methods for demo tenant
INSERT INTO payment_methods (id, tenant_id, name, type, is_active, sort_order, created_at)
VALUES
    ('d1000001-0000-0000-0000-000000000001', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'Tunai', 'cash', true, 1, NOW()),
    ('d1000001-0000-0000-0000-000000000002', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'QRIS', 'qris', true, 2, NOW()),
    ('d1000001-0000-0000-0000-000000000003', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'Transfer Bank', 'bank_transfer', true, 3, NOW())
ON CONFLICT (id) DO NOTHING;

-- Tenant service prices (copy from base prices)
INSERT INTO tenant_service_prices (id, tenant_id, service_template_id, price, is_active)
SELECT uuid_generate_v4(), 'e701d2d3-f5d6-45d2-8202-fb514a695377', id, base_price, true
FROM service_templates WHERE is_active = true
ON CONFLICT (tenant_id, service_template_id) DO NOTHING;

-- Members
INSERT INTO members (id, tenant_id, name, phone, email, tier, discount_percent, total_points, total_spending, created_at)
VALUES
    ('f1000001-0000-0000-0000-000000000001', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'Budi Santoso', '081200000001', 'budi@example.com', 'gold', 10, 0, 1500000, NOW()),
    ('f1000001-0000-0000-0000-000000000002', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'Siti Rahayu', '081200000002', 'siti@example.com', 'silver', 5, 0, 750000, NOW()),
    ('f1000001-0000-0000-0000-000000000003', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'Ahmad Pratama', '081200000003', '', 'bronze', 0, 0, 150000, NOW())
ON CONFLICT (id) DO NOTHING;

-- Feature Flags
INSERT INTO feature_flags (id, key, description, default_enabled)
VALUES
    ('ff000001-0000-0000-0000-000000000001', 'enable_loyalty', 'Enable member loyalty and tier system', true),
    ('ff000001-0000-0000-0000-000000000002', 'enable_multi_outlet', 'Allow tenant to manage multiple outlets', true),
    ('ff000001-0000-0000-0000-000000000003', 'enable_analytics', 'Show analytics dashboard for owners', true),
    ('ff000001-0000-0000-0000-000000000004', 'enable_whatsapp_notif', 'Send order updates via WhatsApp', false)
ON CONFLICT (id) DO NOTHING;

-- Tenant Feature Flag Overrides
INSERT INTO tenant_feature_flags (id, tenant_id, feature_flag_id, enabled)
VALUES
    (uuid_generate_v4(), 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'ff000001-0000-0000-0000-000000000001', true),
    (uuid_generate_v4(), 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'ff000001-0000-0000-0000-000000000002', true),
    (uuid_generate_v4(), 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'ff000001-0000-0000-0000-000000000003', true)
ON CONFLICT (tenant_id, feature_flag_id) DO UPDATE SET enabled = EXCLUDED.enabled;

-- Config version for demo tenant
INSERT INTO config_versions (id, tenant_id, version, data, created_by, created_at)
VALUES (
    'cc000001-0000-0000-0000-000000000001',
    'e701d2d3-f5d6-45d2-8202-fb514a695377',
    1,
    '{"storeName": "Laundry Bersih Cemerlang", "taxRate": 11, "currency": "IDR", "receiptFooter": "Terima kasih atas kepercayaan Anda!", "autoCloseShiftHours": 12}'::jsonb,
    'a1000001-0000-0000-0000-000000000001',
    NOW()
) ON CONFLICT (tenant_id, version) DO NOTHING;

-- Demo Transactions
INSERT INTO transactions (id, tenant_id, outlet_id, local_order_number, customer_name, member_id, items, subtotal, discount_amount, tax_amount, total_amount, payment_status, payments, status, config_version_id, notes, created_by, created_at, synced_at)
VALUES
    ('tx000001-0000-0000-0000-000000000001', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'b1000001-0000-0000-0000-000000000001', 'INV-20260220-001', 'Budi Santoso', 'f1000001-0000-0000-0000-000000000001', '[{"name":"Cuci & Setrika Reguler","qty":3,"unit":"kg","price":7000,"total":21000}]'::jsonb, 21000, 0, 2310, 23310, 'paid', '[{"method":"cash","amount":23310}]'::jsonb, 'completed', 'cc000001-0000-0000-0000-000000000001', NULL, 'a1000001-0000-0000-0000-000000000002', NOW() - INTERVAL '2 days', NOW() - INTERVAL '2 days'),
    ('tx000001-0000-0000-0000-000000000002', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'b1000001-0000-0000-0000-000000000001', 'INV-20260220-002', 'Siti Rahayu', 'f1000001-0000-0000-0000-000000000002', '[{"name":"Express Cuci & Setrika","qty":2,"unit":"kg","price":12000,"total":24000}]'::jsonb, 24000, 0, 2640, 26640, 'paid', '[{"method":"qris","amount":26640}]'::jsonb, 'completed', 'cc000001-0000-0000-0000-000000000001', NULL, 'a1000001-0000-0000-0000-000000000002', NOW() - INTERVAL '1 day', NOW() - INTERVAL '1 day'),
    ('tx000001-0000-0000-0000-000000000003', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'b1000001-0000-0000-0000-000000000001', 'INV-20260220-003', 'Ahmad Pratama', 'f1000001-0000-0000-0000-000000000003', '[{"name":"Cuci Saja Reguler","qty":5,"unit":"kg","price":5000,"total":25000},{"name":"Setrika Express","qty":2,"unit":"kg","price":8000,"total":16000}]'::jsonb, 41000, 0, 4510, 45510, 'paid', '[{"method":"cash","amount":45510}]'::jsonb, 'completed', 'cc000001-0000-0000-0000-000000000001', NULL, 'a1000001-0000-0000-0000-000000000002', NOW() - INTERVAL '1 day', NOW() - INTERVAL '1 day'),
    ('tx000001-0000-0000-0000-000000000004', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'b1000001-0000-0000-0000-000000000001', 'INV-20260221-001', 'Dewi Lestari', NULL, '[{"name":"Dry Cleaning Jas","qty":1,"unit":"pcs","price":35000,"total":35000}]'::jsonb, 35000, 0, 3850, 38850, 'pending', '[{"method":"bank_transfer","amount":38850}]'::jsonb, 'pending', 'cc000001-0000-0000-0000-000000000001', 'Pengambilan besok sore', 'a1000001-0000-0000-0000-000000000002', NOW() - INTERVAL '3 hours', NOW() - INTERVAL '3 hours'),
    ('tx000001-0000-0000-0000-000000000005', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'b1000001-0000-0000-0000-000000000002', 'INV-20260221-002', 'Rini Wulandari', NULL, '[{"name":"Cuci Sepatu Sneakers","qty":2,"unit":"pair","price":35000,"total":70000}]'::jsonb, 70000, 0, 7700, 77700, 'pending', '[{"method":"qris","amount":77700}]'::jsonb, 'pending', 'cc000001-0000-0000-0000-000000000001', NULL, 'a1000001-0000-0000-0000-000000000002', NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- Demo Shift (closed)
INSERT INTO shifts (id, tenant_id, outlet_id, cashier_id, opening_cash, closing_cash, status, opened_at, closed_at, notes, created_at)
VALUES (
    'sh000001-0000-0000-0000-000000000001',
    'e701d2d3-f5d6-45d2-8202-fb514a695377',
    'b1000001-0000-0000-0000-000000000001',
    'a1000001-0000-0000-0000-000000000002',
    500000,
    635460,
    'closed',
    NOW() - INTERVAL '1 day',
    NOW() - INTERVAL '1 day' + INTERVAL '8 hours',
    'Shift demo',
    NOW() - INTERVAL '1 day'
) ON CONFLICT (id) DO NOTHING;

-- Link transactions to shift
INSERT INTO shift_transactions (shift_id, transaction_id)
VALUES
    ('sh000001-0000-0000-0000-000000000001', 'tx000001-0000-0000-0000-000000000001'),
    ('sh000001-0000-0000-0000-000000000001', 'tx000001-0000-0000-0000-000000000002'),
    ('sh000001-0000-0000-0000-000000000001', 'tx000001-0000-0000-0000-000000000003')
ON CONFLICT DO NOTHING;

-- Demo Sync Sessions
INSERT INTO sync_sessions (id, tenant_id, outlet_id, direction, status, transaction_count, error_message, started_at, completed_at)
VALUES
    ('sy000001-0000-0000-0000-000000000001', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'b1000001-0000-0000-0000-000000000001', 'upload', 'completed', 3, NULL, NOW() - INTERVAL '1 day', NOW() - INTERVAL '1 day' + INTERVAL '5 seconds'),
    ('sy000001-0000-0000-0000-000000000002', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'b1000001-0000-0000-0000-000000000002', 'upload', 'failed', 0, 'Connection timeout after 30s', NOW() - INTERVAL '6 hours', NULL)
ON CONFLICT (id) DO NOTHING;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM sync_sessions WHERE id IN ('sy000001-0000-0000-0000-000000000001', 'sy000001-0000-0000-0000-000000000002');
DELETE FROM shift_transactions WHERE shift_id = 'sh000001-0000-0000-0000-000000000001';
DELETE FROM shifts WHERE id = 'sh000001-0000-0000-0000-000000000001';
DELETE FROM transactions WHERE id::text LIKE 'tx000001%';
DELETE FROM config_versions WHERE id = 'cc000001-0000-0000-0000-000000000001';
DELETE FROM tenant_feature_flags WHERE tenant_id = 'e701d2d3-f5d6-45d2-8202-fb514a695377';
DELETE FROM feature_flags WHERE id::text LIKE 'ff000001%';
DELETE FROM members WHERE id::text LIKE 'f1000001%';
DELETE FROM tenant_service_prices WHERE tenant_id = 'e701d2d3-f5d6-45d2-8202-fb514a695377';
DELETE FROM payment_methods WHERE id::text LIKE 'd1000001%';
DELETE FROM users WHERE id IN ('a1000001-0000-0000-0000-000000000001', 'a1000001-0000-0000-0000-000000000002');
DELETE FROM outlets WHERE id IN ('b1000001-0000-0000-0000-000000000001', 'b1000001-0000-0000-0000-000000000002');
DELETE FROM tenants WHERE id = 'e701d2d3-f5d6-45d2-8202-fb514a695377';
-- +goose StatementEnd

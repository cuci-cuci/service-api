-- +goose Up

-- Service Categories (c is valid hex)
INSERT INTO service_categories (id, name, icon, sort_order, is_active) VALUES
    ('c0000001-0000-0000-0000-000000000001', 'Cuci & Setrika', 'shirt', 1, true),
    ('c0000001-0000-0000-0000-000000000002', 'Setrika Saja', 'iron', 2, true),
    ('c0000001-0000-0000-0000-000000000003', 'Cuci Saja', 'droplets', 3, true),
    ('c0000001-0000-0000-0000-000000000004', 'Dry Cleaning', 'sparkles', 4, true),
    ('c0000001-0000-0000-0000-000000000005', 'Cuci Sepatu', 'footprints', 5, true),
    ('c0000001-0000-0000-0000-000000000006', 'Cuci Karpet & Gorden', 'layout-grid', 6, true),
    ('c0000001-0000-0000-0000-000000000007', 'Cuci Bed Cover & Selimut', 'bed-double', 7, true)
ON CONFLICT (id) DO NOTHING;

-- Service Templates: Cuci & Setrika
INSERT INTO service_templates (id, category_id, name, pricing_unit, base_price, estimated_duration_hours, sort_order) VALUES
    ('e0000001-0000-0000-0000-000000000001', 'c0000001-0000-0000-0000-000000000001', 'Reguler (3 Hari)', 'kg', 7000, 72, 1),
    ('e0000001-0000-0000-0000-000000000002', 'c0000001-0000-0000-0000-000000000001', 'Express (1 Hari)', 'kg', 12000, 24, 2),
    ('e0000001-0000-0000-0000-000000000003', 'c0000001-0000-0000-0000-000000000001', 'Kilat (6 Jam)', 'kg', 18000, 6, 3)
ON CONFLICT (id) DO NOTHING;

-- Service Templates: Setrika Saja
INSERT INTO service_templates (id, category_id, name, pricing_unit, base_price, estimated_duration_hours, sort_order) VALUES
    ('e0000001-0000-0000-0000-000000000004', 'c0000001-0000-0000-0000-000000000002', 'Setrika Reguler', 'kg', 5000, 48, 1),
    ('e0000001-0000-0000-0000-000000000005', 'c0000001-0000-0000-0000-000000000002', 'Setrika Express', 'kg', 8000, 12, 2)
ON CONFLICT (id) DO NOTHING;

-- Service Templates: Cuci Saja
INSERT INTO service_templates (id, category_id, name, pricing_unit, base_price, estimated_duration_hours, sort_order) VALUES
    ('e0000001-0000-0000-0000-000000000006', 'c0000001-0000-0000-0000-000000000003', 'Cuci Reguler', 'kg', 5000, 48, 1),
    ('e0000001-0000-0000-0000-000000000007', 'c0000001-0000-0000-0000-000000000003', 'Cuci Express', 'kg', 9000, 12, 2)
ON CONFLICT (id) DO NOTHING;

-- Service Templates: Dry Cleaning
INSERT INTO service_templates (id, category_id, name, pricing_unit, base_price, estimated_duration_hours, sort_order) VALUES
    ('e0000001-0000-0000-0000-000000000008', 'c0000001-0000-0000-0000-000000000004', 'Jas / Blazer', 'pcs', 35000, 72, 1),
    ('e0000001-0000-0000-0000-000000000009', 'c0000001-0000-0000-0000-000000000004', 'Gaun / Dress', 'pcs', 40000, 72, 2),
    ('e0000001-0000-0000-0000-000000000010', 'c0000001-0000-0000-0000-000000000004', 'Kemeja', 'pcs', 20000, 72, 3),
    ('e0000001-0000-0000-0000-000000000011', 'c0000001-0000-0000-0000-000000000004', 'Celana', 'pcs', 18000, 72, 4),
    ('e0000001-0000-0000-0000-000000000012', 'c0000001-0000-0000-0000-000000000004', 'Jaket Kulit', 'pcs', 50000, 96, 5)
ON CONFLICT (id) DO NOTHING;

-- Service Templates: Cuci Sepatu
INSERT INTO service_templates (id, category_id, name, pricing_unit, base_price, estimated_duration_hours, sort_order) VALUES
    ('e0000001-0000-0000-0000-000000000013', 'c0000001-0000-0000-0000-000000000005', 'Sneakers', 'pair', 35000, 72, 1),
    ('e0000001-0000-0000-0000-000000000014', 'c0000001-0000-0000-0000-000000000005', 'Sepatu Kulit', 'pair', 45000, 72, 2),
    ('e0000001-0000-0000-0000-000000000015', 'c0000001-0000-0000-0000-000000000005', 'Sandal', 'pair', 20000, 48, 3)
ON CONFLICT (id) DO NOTHING;

-- Service Templates: Cuci Karpet & Gorden
INSERT INTO service_templates (id, category_id, name, pricing_unit, base_price, estimated_duration_hours, sort_order) VALUES
    ('e0000001-0000-0000-0000-000000000016', 'c0000001-0000-0000-0000-000000000006', 'Karpet Kecil', 'pcs', 30000, 72, 1),
    ('e0000001-0000-0000-0000-000000000017', 'c0000001-0000-0000-0000-000000000006', 'Karpet Besar', 'sqm', 25000, 96, 2),
    ('e0000001-0000-0000-0000-000000000018', 'c0000001-0000-0000-0000-000000000006', 'Gorden', 'meter', 10000, 72, 3)
ON CONFLICT (id) DO NOTHING;

-- Service Templates: Cuci Bed Cover & Selimut
INSERT INTO service_templates (id, category_id, name, pricing_unit, base_price, estimated_duration_hours, sort_order) VALUES
    ('e0000001-0000-0000-0000-000000000019', 'c0000001-0000-0000-0000-000000000007', 'Bed Cover Single', 'pcs', 25000, 72, 1),
    ('e0000001-0000-0000-0000-000000000020', 'c0000001-0000-0000-0000-000000000007', 'Bed Cover Double/King', 'pcs', 35000, 72, 2),
    ('e0000001-0000-0000-0000-000000000021', 'c0000001-0000-0000-0000-000000000007', 'Selimut', 'pcs', 20000, 72, 3),
    ('e0000001-0000-0000-0000-000000000022', 'c0000001-0000-0000-0000-000000000007', 'Bantal / Guling', 'pcs', 15000, 48, 4)
ON CONFLICT (id) DO NOTHING;

-- Seed default payment methods for existing tenants
INSERT INTO payment_methods (tenant_id, name, type, sort_order) SELECT id, 'Tunai', 'cash', 1 FROM tenants WHERE NOT EXISTS (SELECT 1 FROM payment_methods WHERE payment_methods.tenant_id = tenants.id AND type = 'cash');
INSERT INTO payment_methods (tenant_id, name, type, sort_order) SELECT id, 'QRIS', 'qris', 2 FROM tenants WHERE NOT EXISTS (SELECT 1 FROM payment_methods WHERE payment_methods.tenant_id = tenants.id AND type = 'qris');
INSERT INTO payment_methods (tenant_id, name, type, sort_order) SELECT id, 'Transfer Bank', 'bank_transfer', 3 FROM tenants WHERE NOT EXISTS (SELECT 1 FROM payment_methods WHERE payment_methods.tenant_id = tenants.id AND type = 'bank_transfer');

-- Seed tenant_service_prices for existing tenants (use base_price from templates)
INSERT INTO tenant_service_prices (tenant_id, service_template_id, price, is_active)
SELECT t.id, st.id, st.base_price, true
FROM tenants t CROSS JOIN service_templates st
WHERE st.is_active = true
ON CONFLICT (tenant_id, service_template_id) DO NOTHING;

-- +goose Down
DELETE FROM tenant_service_prices WHERE service_template_id IN (SELECT id FROM service_templates WHERE id::text LIKE 'e0000001%');
DELETE FROM payment_methods WHERE name IN ('Tunai', 'QRIS', 'Transfer Bank');
DELETE FROM service_templates WHERE id::text LIKE 'e0000001%';
DELETE FROM service_categories WHERE id::text LIKE 'c0000001%';

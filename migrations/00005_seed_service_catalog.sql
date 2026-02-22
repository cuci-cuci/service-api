-- +goose Up
-- +goose StatementBegin

-- =============================================
-- Seed: Default Laundry Service Catalog
-- =============================================

-- Service Categories
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
    ('t0000001-0000-0000-0000-000000000001', 'c0000001-0000-0000-0000-000000000001', 'Reguler (3 Hari)', 'kg', 7000, 72, 1),
    ('t0000001-0000-0000-0000-000000000002', 'c0000001-0000-0000-0000-000000000001', 'Express (1 Hari)', 'kg', 12000, 24, 2),
    ('t0000001-0000-0000-0000-000000000003', 'c0000001-0000-0000-0000-000000000001', 'Kilat (6 Jam)', 'kg', 18000, 6, 3)
ON CONFLICT (id) DO NOTHING;

-- Service Templates: Setrika Saja
INSERT INTO service_templates (id, category_id, name, pricing_unit, base_price, estimated_duration_hours, sort_order) VALUES
    ('t0000001-0000-0000-0000-000000000004', 'c0000001-0000-0000-0000-000000000002', 'Setrika Reguler', 'kg', 5000, 48, 1),
    ('t0000001-0000-0000-0000-000000000005', 'c0000001-0000-0000-0000-000000000002', 'Setrika Express', 'kg', 8000, 12, 2)
ON CONFLICT (id) DO NOTHING;

-- Service Templates: Cuci Saja
INSERT INTO service_templates (id, category_id, name, pricing_unit, base_price, estimated_duration_hours, sort_order) VALUES
    ('t0000001-0000-0000-0000-000000000006', 'c0000001-0000-0000-0000-000000000003', 'Cuci Reguler', 'kg', 5000, 48, 1),
    ('t0000001-0000-0000-0000-000000000007', 'c0000001-0000-0000-0000-000000000003', 'Cuci Express', 'kg', 9000, 12, 2)
ON CONFLICT (id) DO NOTHING;

-- Service Templates: Dry Cleaning
INSERT INTO service_templates (id, category_id, name, pricing_unit, base_price, estimated_duration_hours, sort_order) VALUES
    ('t0000001-0000-0000-0000-000000000008', 'c0000001-0000-0000-0000-000000000004', 'Jas / Blazer', 'pcs', 35000, 72, 1),
    ('t0000001-0000-0000-0000-000000000009', 'c0000001-0000-0000-0000-000000000004', 'Gaun / Dress', 'pcs', 40000, 72, 2),
    ('t0000001-0000-0000-0000-000000000010', 'c0000001-0000-0000-0000-000000000004', 'Kemeja', 'pcs', 20000, 72, 3),
    ('t0000001-0000-0000-0000-000000000011', 'c0000001-0000-0000-0000-000000000004', 'Celana', 'pcs', 18000, 72, 4),
    ('t0000001-0000-0000-0000-000000000012', 'c0000001-0000-0000-0000-000000000004', 'Jaket Kulit', 'pcs', 50000, 96, 5)
ON CONFLICT (id) DO NOTHING;

-- Service Templates: Cuci Sepatu
INSERT INTO service_templates (id, category_id, name, pricing_unit, base_price, estimated_duration_hours, sort_order) VALUES
    ('t0000001-0000-0000-0000-000000000013', 'c0000001-0000-0000-0000-000000000005', 'Sneakers', 'pair', 35000, 72, 1),
    ('t0000001-0000-0000-0000-000000000014', 'c0000001-0000-0000-0000-000000000005', 'Sepatu Kulit', 'pair', 45000, 72, 2),
    ('t0000001-0000-0000-0000-000000000015', 'c0000001-0000-0000-0000-000000000005', 'Sandal', 'pair', 20000, 48, 3)
ON CONFLICT (id) DO NOTHING;

-- Service Templates: Cuci Karpet & Gorden
INSERT INTO service_templates (id, category_id, name, pricing_unit, base_price, estimated_duration_hours, sort_order) VALUES
    ('t0000001-0000-0000-0000-000000000016', 'c0000001-0000-0000-0000-000000000006', 'Karpet Kecil (< 2m²)', 'pcs', 30000, 72, 1),
    ('t0000001-0000-0000-0000-000000000017', 'c0000001-0000-0000-0000-000000000006', 'Karpet Besar (≥ 2m²)', 'sqm', 25000, 96, 2),
    ('t0000001-0000-0000-0000-000000000018', 'c0000001-0000-0000-0000-000000000006', 'Gorden', 'meter', 10000, 72, 3)
ON CONFLICT (id) DO NOTHING;

-- Service Templates: Cuci Bed Cover & Selimut
INSERT INTO service_templates (id, category_id, name, pricing_unit, base_price, estimated_duration_hours, sort_order) VALUES
    ('t0000001-0000-0000-0000-000000000019', 'c0000001-0000-0000-0000-000000000007', 'Bed Cover Single', 'pcs', 25000, 72, 1),
    ('t0000001-0000-0000-0000-000000000020', 'c0000001-0000-0000-0000-000000000007', 'Bed Cover Double/King', 'pcs', 35000, 72, 2),
    ('t0000001-0000-0000-0000-000000000021', 'c0000001-0000-0000-0000-000000000007', 'Selimut', 'pcs', 20000, 72, 3),
    ('t0000001-0000-0000-0000-000000000022', 'c0000001-0000-0000-0000-000000000007', 'Bantal / Guling', 'pcs', 15000, 48, 4)
ON CONFLICT (id) DO NOTHING;

-- =============================================
-- Seed: Default payment methods for existing tenant
-- =============================================
-- Note: Payment methods are tenant-scoped, so we seed for the existing "Laundry Express" tenant
-- New tenants should get default payment methods created during registration

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DELETE FROM service_templates WHERE id LIKE 't0000001-0000-0000-0000-%';
DELETE FROM service_categories WHERE id LIKE 'c0000001-0000-0000-0000-%';

-- +goose StatementEnd

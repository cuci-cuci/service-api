-- +goose Up
-- +goose StatementBegin

-- ============================================================
-- COMPREHENSIVE DEMO SEED DATA
-- Populates ALL modules with realistic dummy data for the demo tenant
-- ============================================================

-- Constants used throughout:
-- Tenant:   e701d2d3-f5d6-45d2-8202-fb514a695377
-- Outlet 1: b1000001-0000-0000-0000-000000000001 (Cabang Pusat)
-- Outlet 2: b1000001-0000-0000-0000-000000000002 (Cabang Selatan)
-- Owner:    a1000001-0000-0000-0000-000000000001
-- Cashier:  a1000001-0000-0000-0000-000000000002
-- Config:   cc000001-0000-0000-0000-000000000001

-- ============================================================
-- 1. FIX existing transaction items JSONB to match POS format
-- ============================================================

UPDATE transactions SET items = '[{"serviceId":"e0000001-0000-0000-0000-000000000001","serviceName":"Reguler (3 Hari)","categoryName":"Cuci & Setrika","unit":"kg","quantity":3,"pricePerUnit":7000,"subtotal":21000}]'::jsonb
WHERE id = 'aa000001-0000-0000-0000-000000000001';

UPDATE transactions SET items = '[{"serviceId":"e0000001-0000-0000-0000-000000000002","serviceName":"Express (1 Hari)","categoryName":"Cuci & Setrika","unit":"kg","quantity":2,"pricePerUnit":12000,"subtotal":24000}]'::jsonb
WHERE id = 'aa000001-0000-0000-0000-000000000002';

UPDATE transactions SET items = '[{"serviceId":"e0000001-0000-0000-0000-000000000006","serviceName":"Cuci Reguler","categoryName":"Cuci Saja","unit":"kg","quantity":5,"pricePerUnit":5000,"subtotal":25000},{"serviceId":"e0000001-0000-0000-0000-000000000005","serviceName":"Setrika Express","categoryName":"Setrika","unit":"kg","quantity":2,"pricePerUnit":8000,"subtotal":16000}]'::jsonb
WHERE id = 'aa000001-0000-0000-0000-000000000003';

UPDATE transactions SET items = '[{"serviceId":"e0000001-0000-0000-0000-000000000008","serviceName":"Jas / Blazer","categoryName":"Dry Cleaning","unit":"pcs","quantity":1,"pricePerUnit":35000,"subtotal":35000}]'::jsonb
WHERE id = 'aa000001-0000-0000-0000-000000000004';

UPDATE transactions SET items = '[{"serviceId":"e0000001-0000-0000-0000-000000000012","serviceName":"Sneakers","categoryName":"Cuci Sepatu","unit":"pasang","quantity":2,"pricePerUnit":35000,"subtotal":70000}]'::jsonb
WHERE id = 'aa000001-0000-0000-0000-000000000005';

-- ============================================================
-- 2. ADD MORE MEMBERS
-- ============================================================

INSERT INTO members (id, tenant_id, name, phone, email, tier, discount_percent, total_points, total_spending, created_at) VALUES
('f1000001-0000-0000-0000-000000000004', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'Dewi Lestari', '081200000004', 'dewi@email.com', 'bronze', 0, 50, 200000, NOW() - INTERVAL '20 days'),
('f1000001-0000-0000-0000-000000000005', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'Rini Wulandari', '081200000005', 'rini@email.com', 'silver', 5, 120, 800000, NOW() - INTERVAL '15 days'),
('f1000001-0000-0000-0000-000000000006', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'Hendra Gunawan', '081200000006', 'hendra@email.com', 'gold', 10, 250, 1800000, NOW() - INTERVAL '30 days'),
('f1000001-0000-0000-0000-000000000007', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'Linda Wijaya', '081200000007', 'linda@email.com', 'bronze', 0, 30, 100000, NOW() - INTERVAL '10 days'),
('f1000001-0000-0000-0000-000000000008', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'Agus Setiawan', '081200000008', 'agus@email.com', 'silver', 5, 180, 950000, NOW() - INTERVAL '25 days'),
('f1000001-0000-0000-0000-000000000009', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'Ratna Sari', '081200000009', 'ratna@email.com', 'platinum', 15, 500, 3500000, NOW() - INTERVAL '45 days'),
('f1000001-0000-0000-0000-000000000010', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'Bambang Susanto', '081200000010', 'bambang@email.com', 'bronze', 0, 10, 50000, NOW() - INTERVAL '5 days')
ON CONFLICT DO NOTHING;

-- ============================================================
-- 3. ADD MORE TRANSACTIONS (30 total across past 30 days)
-- ============================================================

INSERT INTO transactions (id, tenant_id, outlet_id, local_order_number, customer_name, member_id, items, subtotal, discount_amount, tax_amount, total_amount, payment_status, payments, status, config_version_id, notes, created_by, created_at, synced_at) VALUES
-- Day -28: Morning rush
('aa000001-0000-0000-0000-000000000006', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'b1000001-0000-0000-0000-000000000001', 'INV-20260216-001', 'Hendra Gunawan', 'f1000001-0000-0000-0000-000000000006',
 '[{"serviceId":"e0000001-0000-0000-0000-000000000001","serviceName":"Reguler (3 Hari)","categoryName":"Cuci & Setrika","unit":"kg","quantity":5,"pricePerUnit":7000,"subtotal":35000}]'::jsonb,
 35000, 3500, 3465, 34965, 'paid',
 '[{"methodName":"Tunai","methodType":"cash","amount":34965}]'::jsonb,
 'completed', 'cc000001-0000-0000-0000-000000000001', NULL, 'a1000001-0000-0000-0000-000000000002', NOW() - INTERVAL '28 days', NOW() - INTERVAL '28 days'),

-- Day -25
('aa000001-0000-0000-0000-000000000007', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'b1000001-0000-0000-0000-000000000001', 'INV-20260218-001', 'Ratna Sari', 'f1000001-0000-0000-0000-000000000009',
 '[{"serviceId":"e0000001-0000-0000-0000-000000000001","serviceName":"Reguler (3 Hari)","categoryName":"Cuci & Setrika","unit":"kg","quantity":8,"pricePerUnit":7000,"subtotal":56000},{"serviceId":"e0000001-0000-0000-0000-000000000008","serviceName":"Jas / Blazer","categoryName":"Dry Cleaning","unit":"pcs","quantity":2,"pricePerUnit":35000,"subtotal":70000}]'::jsonb,
 126000, 18900, 11781, 118881, 'paid',
 '[{"methodName":"QRIS","methodType":"qris","amount":118881}]'::jsonb,
 'completed', 'cc000001-0000-0000-0000-000000000001', NULL, 'a1000001-0000-0000-0000-000000000002', NOW() - INTERVAL '25 days', NOW() - INTERVAL '25 days'),

-- Day -22
('aa000001-0000-0000-0000-000000000008', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'b1000001-0000-0000-0000-000000000002', 'INV-20260221-003', 'Linda Wijaya', 'f1000001-0000-0000-0000-000000000007',
 '[{"serviceId":"e0000001-0000-0000-0000-000000000006","serviceName":"Cuci Reguler","categoryName":"Cuci Saja","unit":"kg","quantity":3,"pricePerUnit":5000,"subtotal":15000}]'::jsonb,
 15000, 0, 1650, 16650, 'paid',
 '[{"methodName":"Tunai","methodType":"cash","amount":16650}]'::jsonb,
 'completed', 'cc000001-0000-0000-0000-000000000001', NULL, 'a1000001-0000-0000-0000-000000000002', NOW() - INTERVAL '22 days', NOW() - INTERVAL '22 days'),

-- Day -20
('aa000001-0000-0000-0000-000000000009', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'b1000001-0000-0000-0000-000000000001', 'INV-20260223-001', 'Agus Setiawan', 'f1000001-0000-0000-0000-000000000008',
 '[{"serviceId":"e0000001-0000-0000-0000-000000000004","serviceName":"Setrika Reguler","categoryName":"Setrika","unit":"kg","quantity":4,"pricePerUnit":5000,"subtotal":20000}]'::jsonb,
 20000, 1000, 2090, 21090, 'paid',
 '[{"methodName":"Transfer Bank","methodType":"bank_transfer","amount":21090}]'::jsonb,
 'completed', 'cc000001-0000-0000-0000-000000000001', NULL, 'a1000001-0000-0000-0000-000000000002', NOW() - INTERVAL '20 days', NOW() - INTERVAL '20 days'),

-- Day -18 (Outlet 2)
('aa000001-0000-0000-0000-000000000010', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'b1000001-0000-0000-0000-000000000002', 'INV-20260225-001', 'Dewi Lestari', 'f1000001-0000-0000-0000-000000000004',
 '[{"serviceId":"e0000001-0000-0000-0000-000000000019","serviceName":"Bed Cover Single","categoryName":"Bed Cover & Selimut","unit":"pcs","quantity":1,"pricePerUnit":25000,"subtotal":25000},{"serviceId":"e0000001-0000-0000-0000-000000000001","serviceName":"Reguler (3 Hari)","categoryName":"Cuci & Setrika","unit":"kg","quantity":4,"pricePerUnit":7000,"subtotal":28000}]'::jsonb,
 53000, 0, 5830, 58830, 'paid',
 '[{"methodName":"QRIS","methodType":"qris","amount":58830}]'::jsonb,
 'completed', 'cc000001-0000-0000-0000-000000000001', NULL, 'a1000001-0000-0000-0000-000000000002', NOW() - INTERVAL '18 days', NOW() - INTERVAL '18 days'),

-- Day -15
('aa000001-0000-0000-0000-000000000011', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'b1000001-0000-0000-0000-000000000001', 'INV-20260228-001', 'Budi Santoso', 'f1000001-0000-0000-0000-000000000001',
 '[{"serviceId":"e0000001-0000-0000-0000-000000000001","serviceName":"Reguler (3 Hari)","categoryName":"Cuci & Setrika","unit":"kg","quantity":6,"pricePerUnit":7000,"subtotal":42000}]'::jsonb,
 42000, 4200, 4158, 41958, 'paid',
 '[{"methodName":"Tunai","methodType":"cash","amount":41958}]'::jsonb,
 'completed', 'cc000001-0000-0000-0000-000000000001', NULL, 'a1000001-0000-0000-0000-000000000002', NOW() - INTERVAL '15 days', NOW() - INTERVAL '15 days'),

-- Day -13
('aa000001-0000-0000-0000-000000000012', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'b1000001-0000-0000-0000-000000000001', 'INV-20260302-001', 'Siti Rahayu', 'f1000001-0000-0000-0000-000000000002',
 '[{"serviceId":"e0000001-0000-0000-0000-000000000008","serviceName":"Jas / Blazer","categoryName":"Dry Cleaning","unit":"pcs","quantity":3,"pricePerUnit":35000,"subtotal":105000}]'::jsonb,
 105000, 5250, 10972, 110722, 'paid',
 '[{"methodName":"Transfer Bank","methodType":"bank_transfer","amount":110722}]'::jsonb,
 'completed', 'cc000001-0000-0000-0000-000000000001', NULL, 'a1000001-0000-0000-0000-000000000002', NOW() - INTERVAL '13 days', NOW() - INTERVAL '13 days'),

-- Day -10 (big order)
('aa000001-0000-0000-0000-000000000013', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'b1000001-0000-0000-0000-000000000001', 'INV-20260305-001', 'Ratna Sari', 'f1000001-0000-0000-0000-000000000009',
 '[{"serviceId":"e0000001-0000-0000-0000-000000000001","serviceName":"Reguler (3 Hari)","categoryName":"Cuci & Setrika","unit":"kg","quantity":10,"pricePerUnit":7000,"subtotal":70000},{"serviceId":"e0000001-0000-0000-0000-000000000016","serviceName":"Karpet Kecil","categoryName":"Karpet","unit":"pcs","quantity":2,"pricePerUnit":30000,"subtotal":60000}]'::jsonb,
 130000, 19500, 12155, 122655, 'paid',
 '[{"methodName":"QRIS","methodType":"qris","amount":122655}]'::jsonb,
 'completed', 'cc000001-0000-0000-0000-000000000001', NULL, 'a1000001-0000-0000-0000-000000000002', NOW() - INTERVAL '10 days', NOW() - INTERVAL '10 days'),

-- Day -8 (Outlet 2)
('aa000001-0000-0000-0000-000000000014', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'b1000001-0000-0000-0000-000000000002', 'INV-20260307-001', 'Bambang Susanto', 'f1000001-0000-0000-0000-000000000010',
 '[{"serviceId":"e0000001-0000-0000-0000-000000000006","serviceName":"Cuci Reguler","categoryName":"Cuci Saja","unit":"kg","quantity":2,"pricePerUnit":5000,"subtotal":10000}]'::jsonb,
 10000, 0, 1100, 11100, 'paid',
 '[{"methodName":"Tunai","methodType":"cash","amount":11100}]'::jsonb,
 'completed', 'cc000001-0000-0000-0000-000000000001', NULL, 'a1000001-0000-0000-0000-000000000002', NOW() - INTERVAL '8 days', NOW() - INTERVAL '8 days'),

-- Day -7
('aa000001-0000-0000-0000-000000000015', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'b1000001-0000-0000-0000-000000000001', 'INV-20260308-001', 'Ahmad Pratama', 'f1000001-0000-0000-0000-000000000003',
 '[{"serviceId":"e0000001-0000-0000-0000-000000000001","serviceName":"Reguler (3 Hari)","categoryName":"Cuci & Setrika","unit":"kg","quantity":4,"pricePerUnit":7000,"subtotal":28000},{"serviceId":"e0000001-0000-0000-0000-000000000004","serviceName":"Setrika Reguler","categoryName":"Setrika","unit":"kg","quantity":3,"pricePerUnit":5000,"subtotal":15000}]'::jsonb,
 43000, 0, 4730, 47730, 'paid',
 '[{"methodName":"Tunai","methodType":"cash","amount":47730}]'::jsonb,
 'completed', 'cc000001-0000-0000-0000-000000000001', NULL, 'a1000001-0000-0000-0000-000000000002', NOW() - INTERVAL '7 days', NOW() - INTERVAL '7 days'),

-- Day -5
('aa000001-0000-0000-0000-000000000016', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'b1000001-0000-0000-0000-000000000001', 'INV-20260310-001', 'Rini Wulandari', 'f1000001-0000-0000-0000-000000000005',
 '[{"serviceId":"e0000001-0000-0000-0000-000000000001","serviceName":"Reguler (3 Hari)","categoryName":"Cuci & Setrika","unit":"kg","quantity":7,"pricePerUnit":7000,"subtotal":49000}]'::jsonb,
 49000, 2450, 5119, 51669, 'paid',
 '[{"methodName":"QRIS","methodType":"qris","amount":51669}]'::jsonb,
 'completed', 'cc000001-0000-0000-0000-000000000001', NULL, 'a1000001-0000-0000-0000-000000000002', NOW() - INTERVAL '5 days', NOW() - INTERVAL '5 days'),

-- Day -4 (two transactions)
('aa000001-0000-0000-0000-000000000017', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'b1000001-0000-0000-0000-000000000001', 'INV-20260311-001', 'Hendra Gunawan', 'f1000001-0000-0000-0000-000000000006',
 '[{"serviceId":"e0000001-0000-0000-0000-000000000008","serviceName":"Jas / Blazer","categoryName":"Dry Cleaning","unit":"pcs","quantity":1,"pricePerUnit":35000,"subtotal":35000},{"serviceId":"e0000001-0000-0000-0000-000000000001","serviceName":"Reguler (3 Hari)","categoryName":"Cuci & Setrika","unit":"kg","quantity":3,"pricePerUnit":7000,"subtotal":21000}]'::jsonb,
 56000, 5600, 5544, 55944, 'paid',
 '[{"methodName":"Transfer Bank","methodType":"bank_transfer","amount":55944}]'::jsonb,
 'completed', 'cc000001-0000-0000-0000-000000000001', NULL, 'a1000001-0000-0000-0000-000000000002', NOW() - INTERVAL '4 days', NOW() - INTERVAL '4 days'),

('aa000001-0000-0000-0000-000000000018', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'b1000001-0000-0000-0000-000000000002', 'INV-20260311-002', 'Agus Setiawan', 'f1000001-0000-0000-0000-000000000008',
 '[{"serviceId":"e0000001-0000-0000-0000-000000000006","serviceName":"Cuci Reguler","categoryName":"Cuci Saja","unit":"kg","quantity":6,"pricePerUnit":5000,"subtotal":30000}]'::jsonb,
 30000, 1500, 3135, 31635, 'paid',
 '[{"methodName":"Tunai","methodType":"cash","amount":31635}]'::jsonb,
 'completed', 'cc000001-0000-0000-0000-000000000001', NULL, 'a1000001-0000-0000-0000-000000000002', NOW() - INTERVAL '4 days', NOW() - INTERVAL '4 days'),

-- Day -3
('aa000001-0000-0000-0000-000000000019', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'b1000001-0000-0000-0000-000000000001', 'INV-20260312-001', 'Budi Santoso', 'f1000001-0000-0000-0000-000000000001',
 '[{"serviceId":"e0000001-0000-0000-0000-000000000001","serviceName":"Reguler (3 Hari)","categoryName":"Cuci & Setrika","unit":"kg","quantity":5,"pricePerUnit":7000,"subtotal":35000},{"serviceId":"e0000001-0000-0000-0000-000000000006","serviceName":"Cuci Reguler","categoryName":"Cuci Saja","unit":"kg","quantity":3,"pricePerUnit":5000,"subtotal":15000}]'::jsonb,
 50000, 5000, 4950, 49950, 'paid',
 '[{"methodName":"QRIS","methodType":"qris","amount":49950}]'::jsonb,
 'completed', 'cc000001-0000-0000-0000-000000000001', NULL, 'a1000001-0000-0000-0000-000000000002', NOW() - INTERVAL '3 days', NOW() - INTERVAL '3 days'),

-- Day -2
('aa000001-0000-0000-0000-000000000020', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'b1000001-0000-0000-0000-000000000001', 'INV-20260313-001', 'Walk-in Customer', NULL,
 '[{"serviceId":"e0000001-0000-0000-0000-000000000004","serviceName":"Setrika Reguler","categoryName":"Setrika","unit":"kg","quantity":5,"pricePerUnit":5000,"subtotal":25000}]'::jsonb,
 25000, 0, 2750, 27750, 'paid',
 '[{"methodName":"Tunai","methodType":"cash","amount":27750}]'::jsonb,
 'completed', 'cc000001-0000-0000-0000-000000000001', NULL, 'a1000001-0000-0000-0000-000000000002', NOW() - INTERVAL '2 days', NOW() - INTERVAL '2 days'),

('aa000001-0000-0000-0000-000000000021', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'b1000001-0000-0000-0000-000000000002', 'INV-20260313-002', 'Ratna Sari', 'f1000001-0000-0000-0000-000000000009',
 '[{"serviceId":"e0000001-0000-0000-0000-000000000008","serviceName":"Jas / Blazer","categoryName":"Dry Cleaning","unit":"pcs","quantity":4,"pricePerUnit":35000,"subtotal":140000}]'::jsonb,
 140000, 21000, 13090, 132090, 'paid',
 '[{"methodName":"Transfer Bank","methodType":"bank_transfer","amount":132090}]'::jsonb,
 'completed', 'cc000001-0000-0000-0000-000000000001', NULL, 'a1000001-0000-0000-0000-000000000002', NOW() - INTERVAL '2 days', NOW() - INTERVAL '2 days'),

-- Day -1 (yesterday, busy day)
('aa000001-0000-0000-0000-000000000022', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'b1000001-0000-0000-0000-000000000001', 'INV-20260314-001', 'Dewi Lestari', 'f1000001-0000-0000-0000-000000000004',
 '[{"serviceId":"e0000001-0000-0000-0000-000000000001","serviceName":"Reguler (3 Hari)","categoryName":"Cuci & Setrika","unit":"kg","quantity":4,"pricePerUnit":7000,"subtotal":28000}]'::jsonb,
 28000, 0, 3080, 31080, 'paid',
 '[{"methodName":"Tunai","methodType":"cash","amount":31080}]'::jsonb,
 'completed', 'cc000001-0000-0000-0000-000000000001', NULL, 'a1000001-0000-0000-0000-000000000002', NOW() - INTERVAL '1 day', NOW() - INTERVAL '1 day'),

('aa000001-0000-0000-0000-000000000023', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'b1000001-0000-0000-0000-000000000001', 'INV-20260314-002', 'Linda Wijaya', 'f1000001-0000-0000-0000-000000000007',
 '[{"serviceId":"e0000001-0000-0000-0000-000000000006","serviceName":"Cuci Reguler","categoryName":"Cuci Saja","unit":"kg","quantity":8,"pricePerUnit":5000,"subtotal":40000},{"serviceId":"e0000001-0000-0000-0000-000000000004","serviceName":"Setrika Reguler","categoryName":"Setrika","unit":"kg","quantity":8,"pricePerUnit":5000,"subtotal":40000}]'::jsonb,
 80000, 0, 8800, 88800, 'paid',
 '[{"methodName":"QRIS","methodType":"qris","amount":88800}]'::jsonb,
 'completed', 'cc000001-0000-0000-0000-000000000001', NULL, 'a1000001-0000-0000-0000-000000000002', NOW() - INTERVAL '1 day', NOW() - INTERVAL '1 day'),

('aa000001-0000-0000-0000-000000000024', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'b1000001-0000-0000-0000-000000000002', 'INV-20260314-003', 'Bambang Susanto', 'f1000001-0000-0000-0000-000000000010',
 '[{"serviceId":"e0000001-0000-0000-0000-000000000001","serviceName":"Reguler (3 Hari)","categoryName":"Cuci & Setrika","unit":"kg","quantity":3,"pricePerUnit":7000,"subtotal":21000}]'::jsonb,
 21000, 0, 2310, 23310, 'paid',
 '[{"methodName":"Tunai","methodType":"cash","amount":23310}]'::jsonb,
 'completed', 'cc000001-0000-0000-0000-000000000001', NULL, 'a1000001-0000-0000-0000-000000000002', NOW() - INTERVAL '1 day', NOW() - INTERVAL '1 day'),

-- Today (pending orders)
('aa000001-0000-0000-0000-000000000025', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'b1000001-0000-0000-0000-000000000001', 'INV-20260315-001', 'Agus Setiawan', 'f1000001-0000-0000-0000-000000000008',
 '[{"serviceId":"e0000001-0000-0000-0000-000000000001","serviceName":"Reguler (3 Hari)","categoryName":"Cuci & Setrika","unit":"kg","quantity":6,"pricePerUnit":7000,"subtotal":42000}]'::jsonb,
 42000, 2100, 4389, 44289, 'paid',
 '[{"methodName":"QRIS","methodType":"qris","amount":44289}]'::jsonb,
 'completed', 'cc000001-0000-0000-0000-000000000001', NULL, 'a1000001-0000-0000-0000-000000000002', NOW() - INTERVAL '2 hours', NOW() - INTERVAL '2 hours')

ON CONFLICT (id) DO UPDATE SET items = EXCLUDED.items, payments = EXCLUDED.payments;

-- ============================================================
-- 4. ORDERS + ORDER STATUS LOGS (for recent transactions)
-- ============================================================

INSERT INTO orders (id, transaction_id, tenant_id, outlet_id, status, estimated_completion_at, completed_at, picked_up_at, notes, created_by, updated_by, created_at, updated_at) VALUES
-- Completed orders
('bb000001-0000-0000-0000-000000000001', 'aa000001-0000-0000-0000-000000000001', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'b1000001-0000-0000-0000-000000000001', 'picked_up', NOW() - INTERVAL '1 day', NOW() - INTERVAL '1 day 12 hours', NOW() - INTERVAL '1 day', NULL, 'a1000001-0000-0000-0000-000000000002', 'a1000001-0000-0000-0000-000000000002', NOW() - INTERVAL '2 days', NOW() - INTERVAL '1 day'),
('bb000001-0000-0000-0000-000000000002', 'aa000001-0000-0000-0000-000000000011', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'b1000001-0000-0000-0000-000000000001', 'picked_up', NOW() - INTERVAL '13 days', NOW() - INTERVAL '13 days', NOW() - INTERVAL '12 days', NULL, 'a1000001-0000-0000-0000-000000000002', 'a1000001-0000-0000-0000-000000000002', NOW() - INTERVAL '15 days', NOW() - INTERVAL '12 days'),
('bb000001-0000-0000-0000-000000000003', 'aa000001-0000-0000-0000-000000000013', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'b1000001-0000-0000-0000-000000000001', 'picked_up', NOW() - INTERVAL '7 days', NOW() - INTERVAL '8 days', NOW() - INTERVAL '7 days', NULL, 'a1000001-0000-0000-0000-000000000002', 'a1000001-0000-0000-0000-000000000002', NOW() - INTERVAL '10 days', NOW() - INTERVAL '7 days'),
-- Processing orders
('bb000001-0000-0000-0000-000000000004', 'aa000001-0000-0000-0000-000000000019', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'b1000001-0000-0000-0000-000000000001', 'processing', NOW() + INTERVAL '1 day', NULL, NULL, 'Cucian banyak', 'a1000001-0000-0000-0000-000000000002', 'a1000001-0000-0000-0000-000000000002', NOW() - INTERVAL '3 days', NOW() - INTERVAL '2 days'),
('bb000001-0000-0000-0000-000000000005', 'aa000001-0000-0000-0000-000000000022', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'b1000001-0000-0000-0000-000000000001', 'processing', NOW() + INTERVAL '2 days', NULL, NULL, NULL, 'a1000001-0000-0000-0000-000000000002', 'a1000001-0000-0000-0000-000000000002', NOW() - INTERVAL '1 day', NOW() - INTERVAL '1 day'),
-- Ready for pickup
('bb000001-0000-0000-0000-000000000006', 'aa000001-0000-0000-0000-000000000020', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'b1000001-0000-0000-0000-000000000001', 'ready', NOW() - INTERVAL '1 day', NOW() - INTERVAL '12 hours', NULL, 'Siap diambil', 'a1000001-0000-0000-0000-000000000002', 'a1000001-0000-0000-0000-000000000002', NOW() - INTERVAL '2 days', NOW() - INTERVAL '12 hours'),
-- Received (new)
('bb000001-0000-0000-0000-000000000007', 'aa000001-0000-0000-0000-000000000025', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'b1000001-0000-0000-0000-000000000001', 'received', NOW() + INTERVAL '3 days', NULL, NULL, NULL, 'a1000001-0000-0000-0000-000000000002', 'a1000001-0000-0000-0000-000000000002', NOW() - INTERVAL '2 hours', NOW() - INTERVAL '2 hours')
ON CONFLICT (id) DO NOTHING;

-- Order status logs
INSERT INTO order_status_logs (id, order_id, from_status, to_status, changed_by, notes, created_at) VALUES
('cc100001-0000-0000-0000-000000000001', 'bb000001-0000-0000-0000-000000000001', NULL, 'received', 'a1000001-0000-0000-0000-000000000002', NULL, NOW() - INTERVAL '2 days'),
('cc100001-0000-0000-0000-000000000002', 'bb000001-0000-0000-0000-000000000001', 'received', 'processing', 'a1000001-0000-0000-0000-000000000002', NULL, NOW() - INTERVAL '2 days' + INTERVAL '1 hour'),
('cc100001-0000-0000-0000-000000000003', 'bb000001-0000-0000-0000-000000000001', 'processing', 'ready', 'a1000001-0000-0000-0000-000000000002', NULL, NOW() - INTERVAL '1 day 12 hours'),
('cc100001-0000-0000-0000-000000000004', 'bb000001-0000-0000-0000-000000000001', 'ready', 'picked_up', 'a1000001-0000-0000-0000-000000000002', 'Diambil oleh pelanggan', NOW() - INTERVAL '1 day'),
('cc100001-0000-0000-0000-000000000005', 'bb000001-0000-0000-0000-000000000004', NULL, 'received', 'a1000001-0000-0000-0000-000000000002', NULL, NOW() - INTERVAL '3 days'),
('cc100001-0000-0000-0000-000000000006', 'bb000001-0000-0000-0000-000000000004', 'received', 'processing', 'a1000001-0000-0000-0000-000000000002', NULL, NOW() - INTERVAL '2 days'),
('cc100001-0000-0000-0000-000000000007', 'bb000001-0000-0000-0000-000000000006', NULL, 'received', 'a1000001-0000-0000-0000-000000000002', NULL, NOW() - INTERVAL '2 days'),
('cc100001-0000-0000-0000-000000000008', 'bb000001-0000-0000-0000-000000000006', 'received', 'processing', 'a1000001-0000-0000-0000-000000000002', NULL, NOW() - INTERVAL '1 day'),
('cc100001-0000-0000-0000-000000000009', 'bb000001-0000-0000-0000-000000000006', 'processing', 'ready', 'a1000001-0000-0000-0000-000000000002', NULL, NOW() - INTERVAL '12 hours'),
('cc100001-0000-0000-0000-000000000010', 'bb000001-0000-0000-0000-000000000007', NULL, 'received', 'a1000001-0000-0000-0000-000000000002', NULL, NOW() - INTERVAL '2 hours')
ON CONFLICT (id) DO NOTHING;

-- ============================================================
-- 5. MORE SHIFTS
-- ============================================================

INSERT INTO shifts (id, tenant_id, outlet_id, cashier_id, opening_cash, closing_cash, expected_cash, cash_difference, status, opened_at, closed_at, notes, created_at) VALUES
-- Yesterday shift
('ab000001-0000-0000-0000-000000000002', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'b1000001-0000-0000-0000-000000000001', 'a1000001-0000-0000-0000-000000000002',
 500000, 702540, 700540, 2000, 'closed',
 NOW() - INTERVAL '1 day' - INTERVAL '8 hours', NOW() - INTERVAL '1 day', 'Shift pagi kemarin', NOW() - INTERVAL '1 day' - INTERVAL '8 hours'),
-- Today shift (open)
('ab000001-0000-0000-0000-000000000003', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'b1000001-0000-0000-0000-000000000001', 'a1000001-0000-0000-0000-000000000002',
 500000, NULL, NULL, NULL, 'open',
 NOW() - INTERVAL '4 hours', NULL, 'Shift hari ini', NOW() - INTERVAL '4 hours')
ON CONFLICT (id) DO NOTHING;

-- Link transactions to shifts
INSERT INTO shift_transactions (shift_id, transaction_id) VALUES
('ab000001-0000-0000-0000-000000000002', 'aa000001-0000-0000-0000-000000000022'),
('ab000001-0000-0000-0000-000000000002', 'aa000001-0000-0000-0000-000000000023'),
('ab000001-0000-0000-0000-000000000003', 'aa000001-0000-0000-0000-000000000025')
ON CONFLICT DO NOTHING;

-- ============================================================
-- 6. EXPENSE CATEGORIES & EXPENSES
-- ============================================================

INSERT INTO expense_categories (id, tenant_id, name, icon, is_active) VALUES
('ec000001-0000-0000-0000-000000000001', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'Bahan Cuci', 'drop', true),
('ec000001-0000-0000-0000-000000000002', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'Listrik & Air', 'lightning', true),
('ec000001-0000-0000-0000-000000000003', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'Gaji Karyawan', 'users', true),
('ec000001-0000-0000-0000-000000000004', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'Sewa Tempat', 'buildings', true),
('ec000001-0000-0000-0000-000000000005', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'Transportasi', 'car', true),
('ec000001-0000-0000-0000-000000000006', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'Perawatan Mesin', 'wrench', true),
('ec000001-0000-0000-0000-000000000007', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'Lain-lain', 'receipt', true)
ON CONFLICT (id) DO NOTHING;

INSERT INTO expenses (id, tenant_id, outlet_id, category_id, amount, description, expense_date, created_by) VALUES
('e1000001-0000-0000-0000-000000000001', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'b1000001-0000-0000-0000-000000000001', 'ec000001-0000-0000-0000-000000000001', 350000, 'Beli deterjen 10kg', (NOW() - INTERVAL '25 days')::date, 'a1000001-0000-0000-0000-000000000001'),
('e1000001-0000-0000-0000-000000000002', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'b1000001-0000-0000-0000-000000000001', 'ec000001-0000-0000-0000-000000000001', 150000, 'Beli pewangi 5L', (NOW() - INTERVAL '20 days')::date, 'a1000001-0000-0000-0000-000000000001'),
('e1000001-0000-0000-0000-000000000003', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'b1000001-0000-0000-0000-000000000001', 'ec000001-0000-0000-0000-000000000002', 850000, 'Tagihan listrik bulan ini', (NOW() - INTERVAL '15 days')::date, 'a1000001-0000-0000-0000-000000000001'),
('e1000001-0000-0000-0000-000000000004', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'b1000001-0000-0000-0000-000000000001', 'ec000001-0000-0000-0000-000000000002', 250000, 'Tagihan air', (NOW() - INTERVAL '15 days')::date, 'a1000001-0000-0000-0000-000000000001'),
('e1000001-0000-0000-0000-000000000005', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'b1000001-0000-0000-0000-000000000001', 'ec000001-0000-0000-0000-000000000003', 3500000, 'Gaji kasir bulan ini', (NOW() - INTERVAL '5 days')::date, 'a1000001-0000-0000-0000-000000000001'),
('e1000001-0000-0000-0000-000000000006', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'b1000001-0000-0000-0000-000000000002', 'ec000001-0000-0000-0000-000000000003', 3000000, 'Gaji kasir outlet 2', (NOW() - INTERVAL '5 days')::date, 'a1000001-0000-0000-0000-000000000001'),
('e1000001-0000-0000-0000-000000000007', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'b1000001-0000-0000-0000-000000000001', 'ec000001-0000-0000-0000-000000000004', 5000000, 'Sewa tempat bulan ini', (NOW() - INTERVAL '1 day')::date, 'a1000001-0000-0000-0000-000000000001'),
('e1000001-0000-0000-0000-000000000008', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'b1000001-0000-0000-0000-000000000002', 'ec000001-0000-0000-0000-000000000004', 3500000, 'Sewa tempat outlet 2', (NOW() - INTERVAL '1 day')::date, 'a1000001-0000-0000-0000-000000000001'),
('e1000001-0000-0000-0000-000000000009', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'b1000001-0000-0000-0000-000000000001', 'ec000001-0000-0000-0000-000000000005', 100000, 'Bensin antar-jemput cucian', (NOW() - INTERVAL '3 days')::date, 'a1000001-0000-0000-0000-000000000001'),
('e1000001-0000-0000-0000-000000000010', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'b1000001-0000-0000-0000-000000000001', 'ec000001-0000-0000-0000-000000000006', 500000, 'Service mesin cuci', (NOW() - INTERVAL '10 days')::date, 'a1000001-0000-0000-0000-000000000001'),
('e1000001-0000-0000-0000-000000000011', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'b1000001-0000-0000-0000-000000000001', 'ec000001-0000-0000-0000-000000000007', 75000, 'Beli plastik packaging', (NOW() - INTERVAL '8 days')::date, 'a1000001-0000-0000-0000-000000000001'),
('e1000001-0000-0000-0000-000000000012', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'b1000001-0000-0000-0000-000000000001', 'ec000001-0000-0000-0000-000000000001', 200000, 'Beli deterjen 5kg (restock)', (NOW() - INTERVAL '2 days')::date, 'a1000001-0000-0000-0000-000000000001')
ON CONFLICT (id) DO NOTHING;

-- Recurring expenses
INSERT INTO recurring_expenses (id, tenant_id, category_id, outlet_id, amount, description, frequency, next_due_date, is_active) VALUES
('ae000001-0000-0000-0000-000000000001', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'ec000001-0000-0000-0000-000000000004', 'b1000001-0000-0000-0000-000000000001', 5000000, 'Sewa Cabang Pusat', 'monthly', (NOW() + INTERVAL '15 days')::date, true),
('ae000001-0000-0000-0000-000000000002', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'ec000001-0000-0000-0000-000000000004', 'b1000001-0000-0000-0000-000000000002', 3500000, 'Sewa Cabang Selatan', 'monthly', (NOW() + INTERVAL '15 days')::date, true),
('ae000001-0000-0000-0000-000000000003', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'ec000001-0000-0000-0000-000000000003', NULL, 6500000, 'Gaji karyawan (semua outlet)', 'monthly', (NOW() + INTERVAL '20 days')::date, true)
ON CONFLICT (id) DO NOTHING;

-- ============================================================
-- 7. DASHBOARD GOALS
-- ============================================================

INSERT INTO dashboard_goals (id, tenant_id, goal_type, target_value, is_active) VALUES
('d6000001-0000-0000-0000-000000000001', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'daily_revenue', 200000, true),
('d6000001-0000-0000-0000-000000000002', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'monthly_revenue', 5000000, true),
('d6000001-0000-0000-0000-000000000003', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'daily_transactions', 10, true)
ON CONFLICT (tenant_id, goal_type) DO UPDATE SET target_value = EXCLUDED.target_value;

-- ============================================================
-- 8. SUPPLY CATEGORIES & SUPPLIES
-- ============================================================

INSERT INTO supply_categories (id, tenant_id, name, icon, is_active) VALUES
('5c000001-0000-0000-0000-000000000001', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'Deterjen', 'Drop', true),
('5c000001-0000-0000-0000-000000000002', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'Pewangi', 'Flower', true),
('5c000001-0000-0000-0000-000000000003', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'Packaging', 'Package', true),
('5c000001-0000-0000-0000-000000000004', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'Chemicals', 'Flask', true)
ON CONFLICT (tenant_id, name) DO NOTHING;

INSERT INTO supplies (id, tenant_id, category_id, outlet_id, name, unit, current_stock, min_stock, cost_per_unit, is_active) VALUES
('5b000001-0000-0000-0000-000000000001', 'e701d2d3-f5d6-45d2-8202-fb514a695377', '5c000001-0000-0000-0000-000000000001', 'b1000001-0000-0000-0000-000000000001', 'Deterjen Bubuk Attack', 'kg', 15.00, 5.00, 35000, true),
('5b000001-0000-0000-0000-000000000002', 'e701d2d3-f5d6-45d2-8202-fb514a695377', '5c000001-0000-0000-0000-000000000001', 'b1000001-0000-0000-0000-000000000001', 'Deterjen Cair Rinso', 'liter', 8.50, 3.00, 28000, true),
('5b000001-0000-0000-0000-000000000003', 'e701d2d3-f5d6-45d2-8202-fb514a695377', '5c000001-0000-0000-0000-000000000002', 'b1000001-0000-0000-0000-000000000001', 'Pewangi Downy', 'liter', 5.00, 2.00, 30000, true),
('5b000001-0000-0000-0000-000000000004', 'e701d2d3-f5d6-45d2-8202-fb514a695377', '5c000001-0000-0000-0000-000000000003', 'b1000001-0000-0000-0000-000000000001', 'Plastik Bening 40x60', 'lembar', 200.00, 50.00, 500, true),
('5b000001-0000-0000-0000-000000000005', 'e701d2d3-f5d6-45d2-8202-fb514a695377', '5c000001-0000-0000-0000-000000000003', 'b1000001-0000-0000-0000-000000000001', 'Hanger Kawat', 'pcs', 80.00, 20.00, 1500, true),
('5b000001-0000-0000-0000-000000000006', 'e701d2d3-f5d6-45d2-8202-fb514a695377', '5c000001-0000-0000-0000-000000000004', 'b1000001-0000-0000-0000-000000000001', 'Cairan Dry Clean', 'liter', 3.00, 1.00, 85000, true),
('5b000001-0000-0000-0000-000000000007', 'e701d2d3-f5d6-45d2-8202-fb514a695377', '5c000001-0000-0000-0000-000000000001', 'b1000001-0000-0000-0000-000000000002', 'Deterjen Bubuk Attack', 'kg', 10.00, 5.00, 35000, true),
('5b000001-0000-0000-0000-000000000008', 'e701d2d3-f5d6-45d2-8202-fb514a695377', '5c000001-0000-0000-0000-000000000002', 'b1000001-0000-0000-0000-000000000002', 'Pewangi Downy', 'liter', 3.50, 2.00, 30000, true)
ON CONFLICT (id) DO NOTHING;

-- Stock movements
INSERT INTO stock_movements (id, tenant_id, supply_id, movement_type, quantity, notes, created_by, created_at) VALUES
('5a000001-0000-0000-0000-000000000001', 'e701d2d3-f5d6-45d2-8202-fb514a695377', '5b000001-0000-0000-0000-000000000001', 'in', 20.00, 'Restock deterjen', 'a1000001-0000-0000-0000-000000000001', NOW() - INTERVAL '25 days'),
('5a000001-0000-0000-0000-000000000002', 'e701d2d3-f5d6-45d2-8202-fb514a695377', '5b000001-0000-0000-0000-000000000001', 'out', 5.00, 'Pemakaian harian', 'a1000001-0000-0000-0000-000000000002', NOW() - INTERVAL '20 days'),
('5a000001-0000-0000-0000-000000000003', 'e701d2d3-f5d6-45d2-8202-fb514a695377', '5b000001-0000-0000-0000-000000000002', 'in', 10.00, 'Beli deterjen cair', 'a1000001-0000-0000-0000-000000000001', NOW() - INTERVAL '22 days'),
('5a000001-0000-0000-0000-000000000004', 'e701d2d3-f5d6-45d2-8202-fb514a695377', '5b000001-0000-0000-0000-000000000002', 'out', 1.50, 'Pemakaian 3 hari', 'a1000001-0000-0000-0000-000000000002', NOW() - INTERVAL '15 days'),
('5a000001-0000-0000-0000-000000000005', 'e701d2d3-f5d6-45d2-8202-fb514a695377', '5b000001-0000-0000-0000-000000000003', 'in', 8.00, 'Restock pewangi', 'a1000001-0000-0000-0000-000000000001', NOW() - INTERVAL '18 days'),
('5a000001-0000-0000-0000-000000000006', 'e701d2d3-f5d6-45d2-8202-fb514a695377', '5b000001-0000-0000-0000-000000000003', 'out', 3.00, 'Pemakaian seminggu', 'a1000001-0000-0000-0000-000000000002', NOW() - INTERVAL '10 days'),
('5a000001-0000-0000-0000-000000000007', 'e701d2d3-f5d6-45d2-8202-fb514a695377', '5b000001-0000-0000-0000-000000000004', 'in', 300.00, 'Beli plastik baru', 'a1000001-0000-0000-0000-000000000001', NOW() - INTERVAL '20 days'),
('5a000001-0000-0000-0000-000000000008', 'e701d2d3-f5d6-45d2-8202-fb514a695377', '5b000001-0000-0000-0000-000000000004', 'out', 100.00, 'Pemakaian 2 minggu', 'a1000001-0000-0000-0000-000000000002', NOW() - INTERVAL '5 days'),
('5a000001-0000-0000-0000-000000000009', 'e701d2d3-f5d6-45d2-8202-fb514a695377', '5b000001-0000-0000-0000-000000000006', 'in', 5.00, 'Restock dry clean chemical', 'a1000001-0000-0000-0000-000000000001', NOW() - INTERVAL '15 days'),
('5a000001-0000-0000-0000-000000000010', 'e701d2d3-f5d6-45d2-8202-fb514a695377', '5b000001-0000-0000-0000-000000000006', 'out', 2.00, 'Pemakaian dry cleaning', 'a1000001-0000-0000-0000-000000000002', NOW() - INTERVAL '5 days'),
('5a000001-0000-0000-0000-000000000011', 'e701d2d3-f5d6-45d2-8202-fb514a695377', '5b000001-0000-0000-0000-000000000005', 'in', 100.00, 'Beli hanger baru', 'a1000001-0000-0000-0000-000000000001', NOW() - INTERVAL '12 days'),
('5a000001-0000-0000-0000-000000000012', 'e701d2d3-f5d6-45d2-8202-fb514a695377', '5b000001-0000-0000-0000-000000000005', 'out', 20.00, 'Pemakaian seminggu', 'a1000001-0000-0000-0000-000000000002', NOW() - INTERVAL '3 days')
ON CONFLICT (id) DO NOTHING;

-- Service-supply mappings
INSERT INTO service_supply_mappings (id, tenant_id, service_template_id, supply_id, quantity_per_unit, created_at) VALUES
('55a00001-0000-0000-0000-000000000001', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'e0000001-0000-0000-0000-000000000001', '5b000001-0000-0000-0000-000000000001', 0.10, NOW() - INTERVAL '30 days'),
('55a00001-0000-0000-0000-000000000002', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'e0000001-0000-0000-0000-000000000001', '5b000001-0000-0000-0000-000000000003', 0.05, NOW() - INTERVAL '30 days'),
('55a00001-0000-0000-0000-000000000003', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'e0000001-0000-0000-0000-000000000006', '5b000001-0000-0000-0000-000000000002', 0.08, NOW() - INTERVAL '30 days'),
('55a00001-0000-0000-0000-000000000004', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'e0000001-0000-0000-0000-000000000008', '5b000001-0000-0000-0000-000000000006', 0.50, NOW() - INTERVAL '30 days')
ON CONFLICT (id) DO NOTHING;

-- ============================================================
-- 9. DELIVERY ZONES
-- ============================================================

INSERT INTO delivery_zones (id, tenant_id, outlet_id, name, district, fee, estimated_minutes, is_active) VALUES
('d2000001-0000-0000-0000-000000000001', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'b1000001-0000-0000-0000-000000000001', 'Zona 1 - Pusat Kota', 'Menteng, Cikini, Gondangdia', 10000, 30, true),
('d2000001-0000-0000-0000-000000000002', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'b1000001-0000-0000-0000-000000000001', 'Zona 2 - Kota Selatan', 'Kemang, Blok M, Senopati', 15000, 45, true),
('d2000001-0000-0000-0000-000000000003', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'b1000001-0000-0000-0000-000000000001', 'Zona 3 - Pinggiran', 'BSD, Bintaro, Ciputat', 25000, 60, true),
('d2000001-0000-0000-0000-000000000004', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'b1000001-0000-0000-0000-000000000002', 'Zona 1 - Selatan', 'Pancoran, Tebet, Pasar Minggu', 10000, 30, true),
('d2000001-0000-0000-0000-000000000005', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'b1000001-0000-0000-0000-000000000002', 'Zona 2 - Selatan Jauh', 'Depok, Cinere, Lenteng Agung', 20000, 50, true)
ON CONFLICT (id) DO NOTHING;

-- ============================================================
-- 10. PICKUP REQUESTS
-- ============================================================

INSERT INTO pickup_requests (id, tenant_id, outlet_id, zone_id, order_id, customer_name, customer_phone, address, pickup_type, status, scheduled_at, completed_at, notes, delivery_fee) VALUES
('b2000001-0000-0000-0000-000000000001', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'b1000001-0000-0000-0000-000000000001', 'd2000001-0000-0000-0000-000000000001', 'bb000001-0000-0000-0000-000000000001', 'Budi Santoso', '081200000001', 'Jl. Menteng Raya No. 15, Jakarta Pusat', 'delivery', 'completed', NOW() - INTERVAL '2 days', NOW() - INTERVAL '1 day', 'Antar ke rumah', 10000),
('b2000001-0000-0000-0000-000000000002', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'b1000001-0000-0000-0000-000000000001', 'd2000001-0000-0000-0000-000000000002', NULL, 'Rini Wulandari', '081200000005', 'Jl. Kemang Raya No. 22, Jakarta Selatan', 'pickup', 'scheduled', NOW() + INTERVAL '2 hours', NULL, 'Jemput cucian kotor', 15000),
('b2000001-0000-0000-0000-000000000003', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'b1000001-0000-0000-0000-000000000001', 'd2000001-0000-0000-0000-000000000003', NULL, 'Hendra Gunawan', '081200000006', 'Jl. BSD Raya Blok A5, Tangerang Selatan', 'pickup', 'pending', NOW() + INTERVAL '1 day', NULL, 'Minta dijemput besok pagi', 25000),
('b2000001-0000-0000-0000-000000000004', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'b1000001-0000-0000-0000-000000000002', 'd2000001-0000-0000-0000-000000000004', 'bb000001-0000-0000-0000-000000000006', 'Walk-in Customer', '081200000099', 'Jl. Tebet Barat No. 8, Jakarta Selatan', 'delivery', 'scheduled', NOW() + INTERVAL '4 hours', NULL, 'Antar cucian yang sudah ready', 10000)
ON CONFLICT (id) DO NOTHING;

-- ============================================================
-- 11. STAFF ACTIVITY LOGS
-- ============================================================

INSERT INTO staff_activity_logs (id, tenant_id, user_id, activity_type, reference_id, reference_type, amount, notes, created_at) VALUES
('5e000001-0000-0000-0000-000000000001', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'a1000001-0000-0000-0000-000000000002', 'discount', 'aa000001-0000-0000-0000-000000000006', 'transaction', 3500, 'Diskon member gold 10%', NOW() - INTERVAL '28 days'),
('5e000001-0000-0000-0000-000000000002', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'a1000001-0000-0000-0000-000000000002', 'discount', 'aa000001-0000-0000-0000-000000000007', 'transaction', 18900, 'Diskon member platinum 15%', NOW() - INTERVAL '25 days'),
('5e000001-0000-0000-0000-000000000003', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'a1000001-0000-0000-0000-000000000002', 'void', NULL, 'transaction', 15000, 'Transaksi salah input, void oleh kasir', NOW() - INTERVAL '18 days'),
('5e000001-0000-0000-0000-000000000004', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'a1000001-0000-0000-0000-000000000002', 'refund', 'aa000001-0000-0000-0000-000000000014', 'transaction', 11100, 'Customer komplain, refund penuh', NOW() - INTERVAL '7 days'),
('5e000001-0000-0000-0000-000000000005', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'a1000001-0000-0000-0000-000000000002', 'discount', 'aa000001-0000-0000-0000-000000000011', 'transaction', 4200, 'Diskon member gold 10%', NOW() - INTERVAL '15 days'),
('5e000001-0000-0000-0000-000000000006', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'a1000001-0000-0000-0000-000000000002', 'cancel', NULL, 'order', 0, 'Customer batal laundry', NOW() - INTERVAL '12 days'),
('5e000001-0000-0000-0000-000000000007', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'a1000001-0000-0000-0000-000000000002', 'discount', 'aa000001-0000-0000-0000-000000000013', 'transaction', 19500, 'Diskon member platinum 15%', NOW() - INTERVAL '10 days'),
('5e000001-0000-0000-0000-000000000008', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'a1000001-0000-0000-0000-000000000002', 'discount', 'aa000001-0000-0000-0000-000000000016', 'transaction', 2450, 'Diskon member silver 5%', NOW() - INTERVAL '5 days'),
('5e000001-0000-0000-0000-000000000009', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'a1000001-0000-0000-0000-000000000001', 'void', NULL, 'transaction', 25000, 'Owner void transaksi duplikat', NOW() - INTERVAL '3 days'),
('5e000001-0000-0000-0000-000000000010', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'a1000001-0000-0000-0000-000000000002', 'discount', 'aa000001-0000-0000-0000-000000000025', 'transaction', 2100, 'Diskon member silver 5%', NOW() - INTERVAL '2 hours')
ON CONFLICT (id) DO NOTHING;

-- ============================================================
-- 12. POINT REDEMPTIONS
-- ============================================================

INSERT INTO point_redemptions (id, tenant_id, member_id, points_redeemed, discount_amount, transaction_id, created_at) VALUES
('b3000001-0000-0000-0000-000000000001', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'f1000001-0000-0000-0000-000000000009', 100, 10000, 'aa000001-0000-0000-0000-000000000013', NOW() - INTERVAL '10 days'),
('b3000001-0000-0000-0000-000000000002', 'e701d2d3-f5d6-45d2-8202-fb514a695377', 'f1000001-0000-0000-0000-000000000001', 50, 5000, 'aa000001-0000-0000-0000-000000000019', NOW() - INTERVAL '3 days')
ON CONFLICT (id) DO NOTHING;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Remove all seeded data (in reverse order of dependencies)
DELETE FROM point_redemptions WHERE id LIKE 'b3000001%';
DELETE FROM staff_activity_logs WHERE id LIKE '5e000001%';
DELETE FROM pickup_requests WHERE id LIKE 'b2000001%';
DELETE FROM delivery_zones WHERE id LIKE 'd2000001%';
DELETE FROM service_supply_mappings WHERE id LIKE '55a00001%';
DELETE FROM stock_movements WHERE id LIKE '5a000001%';
DELETE FROM supplies WHERE id LIKE '5b000001%';
DELETE FROM supply_categories WHERE id LIKE '5c000001%';
DELETE FROM recurring_expenses WHERE id LIKE 'ae000001%';
DELETE FROM expenses WHERE id LIKE 'e1000001%';
DELETE FROM expense_categories WHERE id LIKE 'ec000001%';
DELETE FROM dashboard_goals WHERE id LIKE 'd6000001%';
DELETE FROM shift_transactions WHERE shift_id IN ('ab000001-0000-0000-0000-000000000002', 'ab000001-0000-0000-0000-000000000003');
DELETE FROM shifts WHERE id IN ('ab000001-0000-0000-0000-000000000002', 'ab000001-0000-0000-0000-000000000003');
DELETE FROM order_status_logs WHERE id LIKE 'cc100001%';
DELETE FROM orders WHERE id LIKE 'bb000001%';
DELETE FROM transactions WHERE id IN (
  'aa000001-0000-0000-0000-000000000006','aa000001-0000-0000-0000-000000000007','aa000001-0000-0000-0000-000000000008',
  'aa000001-0000-0000-0000-000000000009','aa000001-0000-0000-0000-000000000010','aa000001-0000-0000-0000-000000000011',
  'aa000001-0000-0000-0000-000000000012','aa000001-0000-0000-0000-000000000013','aa000001-0000-0000-0000-000000000014',
  'aa000001-0000-0000-0000-000000000015','aa000001-0000-0000-0000-000000000016','aa000001-0000-0000-0000-000000000017',
  'aa000001-0000-0000-0000-000000000018','aa000001-0000-0000-0000-000000000019','aa000001-0000-0000-0000-000000000020',
  'aa000001-0000-0000-0000-000000000021','aa000001-0000-0000-0000-000000000022','aa000001-0000-0000-0000-000000000023',
  'aa000001-0000-0000-0000-000000000024','aa000001-0000-0000-0000-000000000025'
);
DELETE FROM members WHERE id IN (
  'f1000001-0000-0000-0000-000000000004','f1000001-0000-0000-0000-000000000005','f1000001-0000-0000-0000-000000000006',
  'f1000001-0000-0000-0000-000000000007','f1000001-0000-0000-0000-000000000008','f1000001-0000-0000-0000-000000000009',
  'f1000001-0000-0000-0000-000000000010'
);

-- Restore original item format
UPDATE transactions SET items = '[{"name":"Cuci & Setrika Reguler","qty":3,"unit":"kg","price":7000,"total":21000}]'::jsonb WHERE id = 'aa000001-0000-0000-0000-000000000001';
UPDATE transactions SET items = '[{"name":"Express Cuci & Setrika","qty":2,"unit":"kg","price":12000,"total":24000}]'::jsonb WHERE id = 'aa000001-0000-0000-0000-000000000002';
UPDATE transactions SET items = '[{"name":"Cuci Saja Reguler","qty":5,"unit":"kg","price":5000,"total":25000},{"name":"Setrika Express","qty":2,"unit":"kg","price":8000,"total":16000}]'::jsonb WHERE id = 'aa000001-0000-0000-0000-000000000003';
UPDATE transactions SET items = '[{"name":"Dry Cleaning Jas","qty":1,"unit":"pcs","price":35000,"total":35000}]'::jsonb WHERE id = 'aa000001-0000-0000-0000-000000000004';
UPDATE transactions SET items = '[{"name":"Cuci Sepatu Sneakers","qty":2,"unit":"pair","price":35000,"total":70000}]'::jsonb WHERE id = 'aa000001-0000-0000-0000-000000000005';

-- +goose StatementEnd

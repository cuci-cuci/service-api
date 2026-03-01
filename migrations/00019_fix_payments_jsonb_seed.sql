-- +goose Up
-- +goose StatementBegin

-- Fix demo seed transaction payments JSONB to match POS protocol.
-- POS sends: {"id":"uuid","methodId":"uuid","methodName":"Tunai","methodType":"cash","amount":N}
-- Old seed used: {"method":"cash","amount":N} — wrong keys.

UPDATE transactions SET payments = '[{"methodName":"Tunai","methodType":"cash","amount":23310}]'::jsonb
WHERE id = 'aa000001-0000-0000-0000-000000000001';

UPDATE transactions SET payments = '[{"methodName":"QRIS","methodType":"qris","amount":26640}]'::jsonb
WHERE id = 'aa000001-0000-0000-0000-000000000002';

UPDATE transactions SET payments = '[{"methodName":"Tunai","methodType":"cash","amount":45510}]'::jsonb
WHERE id = 'aa000001-0000-0000-0000-000000000003';

UPDATE transactions SET payments = '[{"methodName":"Transfer Bank","methodType":"bank_transfer","amount":38850}]'::jsonb
WHERE id = 'aa000001-0000-0000-0000-000000000004';

UPDATE transactions SET payments = '[{"methodName":"QRIS","methodType":"qris","amount":77700}]'::jsonb
WHERE id = 'aa000001-0000-0000-0000-000000000005';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

UPDATE transactions SET payments = '[{"method":"cash","amount":23310}]'::jsonb
WHERE id = 'aa000001-0000-0000-0000-000000000001';

UPDATE transactions SET payments = '[{"method":"qris","amount":26640}]'::jsonb
WHERE id = 'aa000001-0000-0000-0000-000000000002';

UPDATE transactions SET payments = '[{"method":"cash","amount":45510}]'::jsonb
WHERE id = 'aa000001-0000-0000-0000-000000000003';

UPDATE transactions SET payments = '[{"method":"bank_transfer","amount":38850}]'::jsonb
WHERE id = 'aa000001-0000-0000-0000-000000000004';

UPDATE transactions SET payments = '[{"method":"qris","amount":77700}]'::jsonb
WHERE id = 'aa000001-0000-0000-0000-000000000005';

-- +goose StatementEnd

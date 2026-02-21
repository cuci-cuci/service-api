-- +goose Up
-- +goose StatementBegin

-- Seed superadmin user
-- Password: admin123456 (bcrypt hash with cost 10)
INSERT INTO users (id, email, name, password_hash, role, tenant_id, is_active, created_at, updated_at)
VALUES (
    'a0000000-0000-0000-0000-000000000001',
    'admin@laundry.app',
    'Super Admin',
    '$2y$10$/yVI6BRPIWhSL4JQgzhifeIF30fR11JeoTPewWHvlcwc.C/dzlKHK',
    'superadmin',
    NULL,
    true,
    NOW(),
    NOW()
) ON CONFLICT (email) DO NOTHING;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DELETE FROM users WHERE email = 'admin@laundry.app';

-- +goose StatementEnd

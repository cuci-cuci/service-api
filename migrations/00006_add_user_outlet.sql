-- +goose Up
ALTER TABLE users ADD COLUMN outlet_id UUID REFERENCES outlets(id);
CREATE INDEX idx_users_outlet_id ON users(outlet_id);

-- +goose Down
DROP INDEX IF EXISTS idx_users_outlet_id;
ALTER TABLE users DROP COLUMN IF EXISTS outlet_id;

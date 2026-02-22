-- +goose Up
ALTER TABLE members ADD COLUMN total_spending BIGINT NOT NULL DEFAULT 0;
ALTER TABLE transactions ADD COLUMN member_id UUID REFERENCES members(id);
CREATE INDEX idx_transactions_member_id ON transactions(member_id);

-- +goose Down
DROP INDEX IF EXISTS idx_transactions_member_id;
ALTER TABLE transactions DROP COLUMN IF EXISTS member_id;
ALTER TABLE members DROP COLUMN IF EXISTS total_spending;

-- +goose Up
ALTER TABLE members
  ADD COLUMN IF NOT EXISTS referral_code VARCHAR(8) UNIQUE,
  ADD COLUMN IF NOT EXISTS referred_by_member_id UUID REFERENCES members(id) ON DELETE SET NULL,
  ADD COLUMN IF NOT EXISTS has_first_transaction BOOLEAN NOT NULL DEFAULT false;

-- Backfill referral codes for existing members
UPDATE members SET referral_code = UPPER(SUBSTRING(MD5(id::text), 1, 8))
WHERE referral_code IS NULL;

CREATE INDEX IF NOT EXISTS idx_members_referral_code ON members(referral_code);
CREATE INDEX IF NOT EXISTS idx_members_referred_by ON members(referred_by_member_id);

-- +goose Down
DROP INDEX IF EXISTS idx_members_referred_by;
DROP INDEX IF EXISTS idx_members_referral_code;
ALTER TABLE members
  DROP COLUMN IF EXISTS has_first_transaction,
  DROP COLUMN IF EXISTS referred_by_member_id,
  DROP COLUMN IF EXISTS referral_code;

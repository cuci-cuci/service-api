-- +goose Up
ALTER TABLE tenant_notification_settings
  ADD COLUMN IF NOT EXISTS daily_summary_enabled BOOLEAN NOT NULL DEFAULT false,
  ADD COLUMN IF NOT EXISTS daily_summary_time TEXT NOT NULL DEFAULT '20:00',
  ADD COLUMN IF NOT EXISTS owner_phone TEXT,
  ADD COLUMN IF NOT EXISTS daily_summary_last_sent_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE tenant_notification_settings
  DROP COLUMN IF EXISTS daily_summary_last_sent_at,
  DROP COLUMN IF EXISTS owner_phone,
  DROP COLUMN IF EXISTS daily_summary_time,
  DROP COLUMN IF EXISTS daily_summary_enabled;

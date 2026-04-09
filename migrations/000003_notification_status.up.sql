ALTER TABLE notification_log
  ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'pending';

ALTER TABLE notification_log
  DROP CONSTRAINT IF EXISTS notification_log_status_check;

ALTER TABLE notification_log
  ADD CONSTRAINT notification_log_status_check
  CHECK (status IN ('pending', 'sent', 'failed'));

ALTER TABLE notification_log
  ADD COLUMN IF NOT EXISTS last_error TEXT NULL;

ALTER TABLE notification_log
  ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

CREATE INDEX IF NOT EXISTS notification_log_pending_idx
  ON notification_log (status);


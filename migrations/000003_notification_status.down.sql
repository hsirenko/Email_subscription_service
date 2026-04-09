DROP INDEX IF EXISTS notification_log_pending_idx;

ALTER TABLE notification_log DROP CONSTRAINT IF EXISTS notification_log_status_check;
ALTER TABLE notification_log DROP COLUMN IF EXISTS updated_at;
ALTER TABLE notification_log DROP COLUMN IF EXISTS last_error;
ALTER TABLE notification_log DROP COLUMN IF EXISTS status;


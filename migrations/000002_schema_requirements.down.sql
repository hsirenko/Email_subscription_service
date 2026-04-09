DROP TABLE IF EXISTS notification_log;

DROP INDEX IF EXISTS subscriptions_email_status_idx;
DROP INDEX IF EXISTS subscriptions_email_repo_active_uq;

ALTER TABLE subscriptions
  ADD COLUMN IF NOT EXISTS confirmed BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS active BOOLEAN NOT NULL DEFAULT TRUE,
  ADD COLUMN IF NOT EXISTS last_seen_tag TEXT NOT NULL DEFAULT '';

-- Best-effort restore of flags from status
UPDATE subscriptions SET
  active = (status <> 'unsubscribed'),
  confirmed = (status = 'active');

ALTER TABLE subscriptions DROP CONSTRAINT IF EXISTS subscriptions_status_check;

ALTER TABLE subscriptions DROP COLUMN IF EXISTS status;
ALTER TABLE subscriptions DROP COLUMN IF EXISTS repo_owner;
ALTER TABLE subscriptions DROP COLUMN IF EXISTS repo_name;

CREATE UNIQUE INDEX IF NOT EXISTS subscriptions_email_repo_uq
  ON subscriptions (email, repo_full_name);

DROP TABLE IF EXISTS repo_state;
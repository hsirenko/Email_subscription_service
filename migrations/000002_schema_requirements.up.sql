-- Per-repo release (one row per GitHub repo)
CREATE TABLE IF NOT EXISTS repo_state (
  repo_full_name TEXT PRIMARY KEY,
  last_seen_tag    TEXT NOT NULL DEFAULT '',
  last_checked_at  TIMESTAMPTZ NULL
);

-- Backfill repo_state from existing subscriptions (if any)
INSERT INTO repo_state (repo_full_name, last_seen_tag)
SELECT DISTINCT ON (repo_full_name)
  repo_full_name,
  last_seen_tag
FROM subscriptions
ORDER BY repo_full_name, id
ON CONFLICT (repo_full_name) DO NOTHING;

-- Add normalized repo columns + status
ALTER TABLE subscriptions
  ADD COLUMN IF NOT EXISTS repo_owner TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS repo_name  TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'pending';

-- Backfill owner/name from repo_full_name (assumes owner/repo with a single slash)
UPDATE subscriptions
SET
  repo_owner = split_part(repo_full_name, '/', 1),
  repo_name  = split_part(repo_full_name, '/', 2)
WHERE repo_owner = '' OR repo_name = '';

-- Backfill status from legacy flags
UPDATE subscriptions
SET status = CASE
  WHEN active IS FALSE THEN 'unsubscribed'
  WHEN confirmed IS TRUE  THEN 'active'
  ELSE 'pending'
END;

ALTER TABLE subscriptions
  DROP CONSTRAINT IF EXISTS subscriptions_status_check;

ALTER TABLE subscriptions
  ADD CONSTRAINT subscriptions_status_check
  CHECK (status IN ('pending', 'active', 'unsubscribed'));

-- Drop legacy columns (optional: keep until app deployed if you need zero-downtime)
ALTER TABLE subscriptions DROP COLUMN IF EXISTS confirmed;
ALTER TABLE subscriptions DROP COLUMN IF EXISTS active;
ALTER TABLE subscriptions DROP COLUMN IF EXISTS last_seen_tag;

-- Dedupe notifications
CREATE TABLE IF NOT EXISTS notification_log (
  id BIGSERIAL PRIMARY KEY,
  subscription_id BIGINT NOT NULL REFERENCES subscriptions (id) ON DELETE CASCADE,
  release_tag TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (subscription_id, release_tag)
);

CREATE INDEX IF NOT EXISTS notification_log_subscription_id_idx
  ON notification_log (subscription_id);

-- Replace old unique index: allow resubscribe after unsubscribed
DROP INDEX IF EXISTS subscriptions_email_repo_uq;

CREATE UNIQUE INDEX IF NOT EXISTS subscriptions_email_repo_active_uq
  ON subscriptions (email, repo_full_name)
  WHERE status IN ('pending', 'active');

-- Helpful for listing by email
CREATE INDEX IF NOT EXISTS subscriptions_email_status_idx
  ON subscriptions (email, status);
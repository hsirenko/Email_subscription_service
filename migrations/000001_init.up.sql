CREATE TABLE IF NOT EXISTS subscriptions (
  id BIGSERIAL PRIMARY KEY,

  email TEXT NOT NULL,
  repo_full_name TEXT NOT NULL,

  confirm_token TEXT NOT NULL UNIQUE,
  unsubscribe_token TEXT NOT NULL UNIQUE,

  confirmed BOOLEAN NOT NULL DEFAULT FALSE,
  active BOOLEAN NOT NULL DEFAULT TRUE,

  last_seen_tag TEXT NOT NULL DEFAULT '',

  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS subscriptions_email_repo_uq
  ON subscriptions (email, repo_full_name);

CREATE INDEX IF NOT EXISTS subscriptions_email_active_idx
  ON subscriptions (email, active);
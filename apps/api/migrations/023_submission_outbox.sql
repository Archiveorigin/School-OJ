CREATE TABLE IF NOT EXISTS submission_outbox (
  id BIGSERIAL PRIMARY KEY,
  submission_id BIGINT NOT NULL REFERENCES submissions(id) ON DELETE CASCADE,
  event_version INTEGER NOT NULL DEFAULT 1,
  payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  attempts INTEGER NOT NULL DEFAULT 0,
  available_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  locked_until TIMESTAMPTZ,
  published_at TIMESTAMPTZ,
  last_error TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_submission_outbox_pending
  ON submission_outbox(available_at, id)
  WHERE published_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_submission_outbox_submission
  ON submission_outbox(submission_id, id);

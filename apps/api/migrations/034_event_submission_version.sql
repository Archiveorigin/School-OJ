-- Business versions count successful submissions. review_revision remains the
-- independent optimistic-lock token for every mutation, including attachments.
ALTER TABLE exams ADD COLUMN IF NOT EXISTS submission_version INTEGER NOT NULL DEFAULT 0;
ALTER TABLE team_contests ADD COLUMN IF NOT EXISTS submission_version INTEGER NOT NULL DEFAULT 0;
ALTER TABLE event_review_logs ADD COLUMN IF NOT EXISTS submission_version INTEGER NOT NULL DEFAULT 0;
WITH versions AS (
  SELECT id, count(*) FILTER (WHERE action='submit') OVER (
    PARTITION BY event_kind,event_id ORDER BY id ROWS UNBOUNDED PRECEDING
  ) AS version FROM event_review_logs
)
UPDATE event_review_logs l SET submission_version=v.version FROM versions v WHERE l.id=v.id;
UPDATE exams e SET submission_version=(SELECT count(*) FROM event_review_logs l WHERE l.event_kind='exam' AND l.event_id=e.id AND l.action='submit');
UPDATE team_contests e SET submission_version=(SELECT count(*) FROM event_review_logs l WHERE l.event_kind='contest' AND l.event_id=e.id AND l.action='submit');

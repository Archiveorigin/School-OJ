ALTER TABLE team_contests
  ADD COLUMN IF NOT EXISTS state VARCHAR(16) NOT NULL DEFAULT 'draft',
  ADD COLUMN IF NOT EXISTS published_at TIMESTAMPTZ;

UPDATE team_contests
SET state = CASE
  WHEN starts_at IS NULL THEN 'draft'
  WHEN starts_at > NOW() THEN 'published'
  WHEN starts_at + (duration_minutes * INTERVAL '1 minute') > NOW() THEN 'running'
  ELSE 'closed'
END;

UPDATE team_contests
SET published_at = COALESCE(published_at, created_at)
WHERE state <> 'draft';

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'team_contests_state_check') THEN
    ALTER TABLE team_contests
      ADD CONSTRAINT team_contests_state_check
      CHECK (state IN ('draft', 'published', 'running', 'closed'));
  END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_team_contests_state ON team_contests(state);
CREATE INDEX IF NOT EXISTS idx_team_contests_published_at ON team_contests(published_at);

CREATE TABLE IF NOT EXISTS team_contest_participants (
  id BIGSERIAL PRIMARY KEY,
  contest_id BIGINT NOT NULL REFERENCES team_contests(id) ON DELETE CASCADE,
  user_id BIGINT NOT NULL REFERENCES users(id),
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(contest_id, user_id)
);

INSERT INTO team_contest_participants(contest_id, user_id)
SELECT contests.id, memberships.user_id
FROM team_contests contests
JOIN team_memberships memberships ON memberships.team_id = contests.team_id
WHERE contests.state <> 'draft'
ON CONFLICT (contest_id, user_id) DO NOTHING;

CREATE INDEX IF NOT EXISTS idx_team_contest_participants_contest_id ON team_contest_participants(contest_id);
CREATE INDEX IF NOT EXISTS idx_team_contest_participants_user_id ON team_contest_participants(user_id);

CREATE INDEX IF NOT EXISTS idx_submissions_latest_standalone
  ON submissions(user_id, problem_id, id DESC)
  WHERE assignment_id IS NULL AND exam_id IS NULL AND team_contest_id IS NULL AND problem_set_id IS NULL;
CREATE INDEX IF NOT EXISTS idx_submissions_latest_assignment
  ON submissions(assignment_id, user_id, problem_id, id DESC)
  WHERE assignment_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_submissions_latest_exam
  ON submissions(exam_id, user_id, problem_id, id DESC)
  WHERE exam_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_submissions_latest_contest
  ON submissions(team_contest_id, user_id, problem_id, id DESC)
  WHERE team_contest_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_submissions_latest_problem_set
  ON submissions(problem_set_id, user_id, problem_id, id DESC)
  WHERE problem_set_id IS NOT NULL;

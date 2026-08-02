CREATE TABLE IF NOT EXISTS team_contest_problems (
  id BIGSERIAL PRIMARY KEY,
  contest_id BIGINT NOT NULL REFERENCES team_contests(id) ON DELETE CASCADE,
  problem_id BIGINT NOT NULL REFERENCES problems(id),
  label VARCHAR(16),
  sort_order INTEGER NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(contest_id, problem_id)
);

ALTER TABLE submissions
  ADD COLUMN IF NOT EXISTS team_contest_id BIGINT REFERENCES team_contests(id) ON DELETE SET NULL;

ALTER TABLE submissions
  ADD COLUMN IF NOT EXISTS problem_set_id BIGINT REFERENCES team_problem_sets(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_submissions_team_contest_id ON submissions(team_contest_id);
CREATE INDEX IF NOT EXISTS idx_submissions_problem_set_id ON submissions(problem_set_id);
CREATE INDEX IF NOT EXISTS idx_team_contest_problems_contest_id ON team_contest_problems(contest_id);
CREATE INDEX IF NOT EXISTS idx_team_contest_problems_problem_id ON team_contest_problems(problem_id);

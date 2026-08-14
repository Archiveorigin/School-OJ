ALTER TABLE team_contests
  ADD COLUMN IF NOT EXISTS gold_award_percent INTEGER NOT NULL DEFAULT 10,
  ADD COLUMN IF NOT EXISTS silver_award_percent INTEGER NOT NULL DEFAULT 10,
  ADD COLUMN IF NOT EXISTS bronze_award_percent INTEGER NOT NULL DEFAULT 10,
  ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

ALTER TABLE team_problem_sets
  ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'team_contests_award_percentages_check'
  ) THEN
    ALTER TABLE team_contests
      ADD CONSTRAINT team_contests_award_percentages_check
      CHECK (
        gold_award_percent BETWEEN 0 AND 100
        AND silver_award_percent BETWEEN 0 AND 100
        AND bronze_award_percent BETWEEN 0 AND 100
        AND gold_award_percent + silver_award_percent + bronze_award_percent <= 100
      );
  END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_team_contests_deleted_at ON team_contests(deleted_at);
CREATE INDEX IF NOT EXISTS idx_team_problem_sets_deleted_at ON team_problem_sets(deleted_at);

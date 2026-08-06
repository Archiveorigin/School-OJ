ALTER TABLE team_contests
  ADD COLUMN IF NOT EXISTS scoring_rule VARCHAR(16) NOT NULL DEFAULT 'penalty';

UPDATE team_contests
SET scoring_rule = 'penalty'
WHERE scoring_rule IS NULL OR scoring_rule NOT IN ('score', 'penalty');

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'team_contests_scoring_rule_check'
  ) THEN
    ALTER TABLE team_contests
      ADD CONSTRAINT team_contests_scoring_rule_check
      CHECK (scoring_rule IN ('score', 'penalty'));
  END IF;
END $$;

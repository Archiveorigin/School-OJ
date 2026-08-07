ALTER TABLE exams
  ADD COLUMN IF NOT EXISTS scoring_rule VARCHAR(16);

UPDATE exams
SET scoring_rule = 'score'
WHERE scoring_rule IS NULL OR scoring_rule NOT IN ('score', 'penalty');

ALTER TABLE exams
  ALTER COLUMN scoring_rule SET DEFAULT 'penalty',
  ALTER COLUMN scoring_rule SET NOT NULL;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'exams_scoring_rule_check'
  ) THEN
    ALTER TABLE exams
      ADD CONSTRAINT exams_scoring_rule_check
      CHECK (scoring_rule IN ('score', 'penalty'));
  END IF;
END $$;

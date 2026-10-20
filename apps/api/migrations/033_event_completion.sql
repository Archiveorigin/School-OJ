-- All previously ended events await explicit administrative completion.
-- Never infer completion from an end time or backfill an approving identity.
DO $$
DECLARE t TEXT;
BEGIN
  FOREACH t IN ARRAY ARRAY['exams','team_contests'] LOOP
    EXECUTE format('ALTER TABLE %I ADD COLUMN IF NOT EXISTS completed_at TIMESTAMPTZ,
      ADD COLUMN IF NOT EXISTS completed_by BIGINT,
      ADD COLUMN IF NOT EXISTS completion_note TEXT NOT NULL DEFAULT ''''',t);
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname=t||'_completion_pair' AND conrelid=t::regclass) THEN
      EXECUTE format('ALTER TABLE %I ADD CONSTRAINT %I CHECK ((completed_at IS NULL) = (completed_by IS NULL))',t,t||'_completion_pair');
    END IF;
  END LOOP;
END $$;
CREATE INDEX IF NOT EXISTS idx_exams_completed ON exams(completed_at,id) WHERE deleted_at IS NULL AND completed_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_contests_completed ON team_contests(completed_at,id) WHERE deleted_at IS NULL AND completed_at IS NOT NULL;

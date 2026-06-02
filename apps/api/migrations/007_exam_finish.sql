ALTER TABLE exam_attempts
  ADD COLUMN IF NOT EXISTS finished_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_exam_attempts_finished_at
  ON exam_attempts (finished_at);

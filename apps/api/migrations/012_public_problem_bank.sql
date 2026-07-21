ALTER TABLE prepared_problems ADD COLUMN IF NOT EXISTS published_at TIMESTAMPTZ;

UPDATE prepared_problems
SET published_at = COALESCE(
  (SELECT MIN(class_problems.created_at)
   FROM class_problems
   WHERE class_problems.problem_id = prepared_problems.problem_id),
  now()
)
WHERE published_at IS NULL
  AND EXISTS (
    SELECT 1
    FROM class_problems
    WHERE class_problems.problem_id = prepared_problems.problem_id
  );

CREATE INDEX IF NOT EXISTS idx_prepared_problems_published_at
  ON prepared_problems(published_at);

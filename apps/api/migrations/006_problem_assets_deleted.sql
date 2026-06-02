ALTER TABLE problems ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_problems_deleted_at ON problems(deleted_at);

CREATE UNIQUE INDEX IF NOT EXISTS idx_problems_slug_active ON problems(slug) WHERE deleted_at IS NULL;

ALTER TABLE problems DROP CONSTRAINT IF EXISTS problems_slug_key;
ALTER TABLE problems DROP CONSTRAINT IF EXISTS idx_problems_slug;
DROP INDEX IF EXISTS idx_problems_slug;

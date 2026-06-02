ALTER TABLE class_problems
  ADD COLUMN IF NOT EXISTS release_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_class_problems_release_at
  ON class_problems (release_at);

CREATE TABLE IF NOT EXISTS prepared_problems (
  id BIGSERIAL PRIMARY KEY,
  problem_id BIGINT NOT NULL UNIQUE REFERENCES problems(id) ON DELETE CASCADE,
  owner_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  folder VARCHAR(160),
  difficulty VARCHAR(32),
  source VARCHAR(160),
  notes TEXT,
  archived BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ,
  updated_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_prepared_problems_owner_id
  ON prepared_problems (owner_id);

CREATE INDEX IF NOT EXISTS idx_prepared_problems_folder
  ON prepared_problems (folder);

CREATE INDEX IF NOT EXISTS idx_prepared_problems_difficulty
  ON prepared_problems (difficulty);

CREATE INDEX IF NOT EXISTS idx_prepared_problems_archived
  ON prepared_problems (archived);

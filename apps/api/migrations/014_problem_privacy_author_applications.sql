ALTER TABLE problems
  ADD COLUMN IF NOT EXISTS difficulty VARCHAR(32);

ALTER TABLE users
  DROP CONSTRAINT IF EXISTS users_role_check;

ALTER TABLE users
  ADD CONSTRAINT users_role_check
  CHECK (role IN ('student', 'problem_setter', 'teacher', 'admin'));

CREATE INDEX IF NOT EXISTS idx_problems_difficulty
  ON problems(difficulty);

ALTER TABLE submissions
  ADD COLUMN IF NOT EXISTS is_public BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS idx_submissions_is_public
  ON submissions(is_public);

CREATE TABLE IF NOT EXISTS author_applications (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL REFERENCES users(id),
  motivation TEXT NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'pending',
  review_note TEXT,
  reviewed_by BIGINT REFERENCES users(id),
  reviewed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_author_applications_user_id
  ON author_applications(user_id);

CREATE INDEX IF NOT EXISTS idx_author_applications_status
  ON author_applications(status);

CREATE UNIQUE INDEX IF NOT EXISTS idx_author_applications_pending_user
  ON author_applications(user_id)
  WHERE status = 'pending';

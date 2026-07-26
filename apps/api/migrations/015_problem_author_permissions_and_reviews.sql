ALTER TABLE users
  ADD COLUMN IF NOT EXISTS can_author boolean NOT NULL DEFAULT false;

CREATE INDEX IF NOT EXISTS idx_users_can_author ON users(can_author);

UPDATE users
SET can_author = true
WHERE role = 'problem_setter';

CREATE TABLE IF NOT EXISTS problem_reviews (
  id bigserial PRIMARY KEY,
  problem_id bigint NOT NULL REFERENCES problems(id),
  author_id bigint NOT NULL REFERENCES users(id),
  status varchar(32) NOT NULL DEFAULT 'pending',
  review_note text,
  reviewed_by bigint,
  reviewed_at timestamptz,
  submitted_at timestamptz NOT NULL,
  created_at timestamptz,
  updated_at timestamptz
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_problem_reviews_problem_id ON problem_reviews(problem_id);
CREATE INDEX IF NOT EXISTS idx_problem_reviews_author_id ON problem_reviews(author_id);
CREATE INDEX IF NOT EXISTS idx_problem_reviews_status ON problem_reviews(status);
CREATE INDEX IF NOT EXISTS idx_problem_reviews_reviewed_by ON problem_reviews(reviewed_by);
CREATE INDEX IF NOT EXISTS idx_problem_reviews_submitted_at ON problem_reviews(submitted_at);

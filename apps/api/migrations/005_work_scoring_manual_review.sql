ALTER TABLE assignments ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
ALTER TABLE exams ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_assignments_deleted_at ON assignments(deleted_at);
CREATE INDEX IF NOT EXISTS idx_exams_deleted_at ON exams(deleted_at);

CREATE TABLE IF NOT EXISTS assignment_attempts (
  id BIGSERIAL PRIMARY KEY,
  assignment_id BIGINT NOT NULL REFERENCES assignments(id) ON DELETE CASCADE,
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(assignment_id, user_id)
);

CREATE TABLE IF NOT EXISTS exam_attempts (
  id BIGSERIAL PRIMARY KEY,
  exam_id BIGINT NOT NULL REFERENCES exams(id) ON DELETE CASCADE,
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(exam_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_assignment_attempts_user ON assignment_attempts(user_id);
CREATE INDEX IF NOT EXISTS idx_exam_attempts_user ON exam_attempts(user_id);

ALTER TABLE submissions ADD COLUMN IF NOT EXISTS manual_score INT;
ALTER TABLE submissions ADD COLUMN IF NOT EXISTS manual_graded_by BIGINT REFERENCES users(id);
ALTER TABLE submissions ADD COLUMN IF NOT EXISTS manual_graded_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_submissions_assignment_user ON submissions(assignment_id, user_id);
CREATE INDEX IF NOT EXISTS idx_submissions_exam_user ON submissions(exam_id, user_id);
CREATE INDEX IF NOT EXISTS idx_submissions_manual_graded_by ON submissions(manual_graded_by);

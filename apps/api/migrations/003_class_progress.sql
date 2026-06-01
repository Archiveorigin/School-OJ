CREATE TABLE IF NOT EXISTS class_problems (
  id BIGSERIAL PRIMARY KEY,
  class_id BIGINT NOT NULL REFERENCES classes(id) ON DELETE CASCADE,
  problem_id BIGINT NOT NULL REFERENCES problems(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(class_id, problem_id)
);

CREATE TABLE IF NOT EXISTS problem_progresses (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  problem_id BIGINT NOT NULL REFERENCES problems(id) ON DELETE CASCADE,
  status VARCHAR(32) NOT NULL DEFAULT 'unattempted',
  points INT NOT NULL DEFAULT 0,
  points_awarded BOOLEAN NOT NULL DEFAULT false,
  first_accepted TIMESTAMPTZ,
  last_submitted TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(user_id, problem_id)
);

ALTER TABLE assignments ADD COLUMN IF NOT EXISTS class_id BIGINT REFERENCES classes(id) ON DELETE CASCADE;
ALTER TABLE exams ADD COLUMN IF NOT EXISTS class_id BIGINT REFERENCES classes(id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS idx_class_problems_class ON class_problems(class_id);
CREATE INDEX IF NOT EXISTS idx_problem_progresses_status ON problem_progresses(status);
CREATE INDEX IF NOT EXISTS idx_assignments_class_id ON assignments(class_id);
CREATE INDEX IF NOT EXISTS idx_exams_class_id ON exams(class_id);

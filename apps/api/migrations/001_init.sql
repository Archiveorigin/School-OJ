CREATE TABLE IF NOT EXISTS users (
  id BIGSERIAL PRIMARY KEY,
  email VARCHAR(255) NOT NULL UNIQUE,
  name VARCHAR(120) NOT NULL,
  role VARCHAR(32) NOT NULL CHECK (role IN ('student','teacher','admin')),
  password_hash TEXT NOT NULL,
  student_no VARCHAR(64),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS courses (
  id BIGSERIAL PRIMARY KEY,
  code VARCHAR(64) NOT NULL UNIQUE,
  name VARCHAR(160) NOT NULL,
  term VARCHAR(64),
  teacher_id BIGINT NOT NULL REFERENCES users(id),
  description TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS classes (
  id BIGSERIAL PRIMARY KEY,
  course_id BIGINT NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
  name VARCHAR(120) NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS course_memberships (
  id BIGSERIAL PRIMARY KEY,
  course_id BIGINT NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  role VARCHAR(32) NOT NULL CHECK (role IN ('student','teacher','admin')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(course_id, user_id)
);

CREATE TABLE IF NOT EXISTS class_memberships (
  id BIGSERIAL PRIMARY KEY,
  class_id BIGINT NOT NULL REFERENCES classes(id) ON DELETE CASCADE,
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(class_id, user_id)
);

CREATE TABLE IF NOT EXISTS problems (
  id BIGSERIAL PRIMARY KEY,
  owner_id BIGINT NOT NULL REFERENCES users(id),
  slug VARCHAR(120) NOT NULL UNIQUE,
  title VARCHAR(200) NOT NULL,
  statement TEXT,
  tags JSONB,
  time_limit_ms INT NOT NULL DEFAULT 1000,
  memory_limit_mb INT NOT NULL DEFAULT 256,
  output_limit_kb INT NOT NULL DEFAULT 1024,
  package_object VARCHAR(512) NOT NULL,
  package_checksum VARCHAR(128) NOT NULL,
  manifest JSONB,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS assignments (
  id BIGSERIAL PRIMARY KEY,
  course_id BIGINT NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
  title VARCHAR(200) NOT NULL,
  description TEXT,
  starts_at TIMESTAMPTZ,
  due_at TIMESTAMPTZ,
  settings JSONB,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS assignment_problems (
  id BIGSERIAL PRIMARY KEY,
  assignment_id BIGINT NOT NULL REFERENCES assignments(id) ON DELETE CASCADE,
  problem_id BIGINT NOT NULL REFERENCES problems(id),
  score INT NOT NULL DEFAULT 100,
  sort_order INT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS exams (
  id BIGSERIAL PRIMARY KEY,
  course_id BIGINT NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
  title VARCHAR(200) NOT NULL,
  description TEXT,
  starts_at TIMESTAMPTZ,
  ends_at TIMESTAMPTZ,
  settings JSONB,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS exam_problems (
  id BIGSERIAL PRIMARY KEY,
  exam_id BIGINT NOT NULL REFERENCES exams(id) ON DELETE CASCADE,
  problem_id BIGINT NOT NULL REFERENCES problems(id),
  score INT NOT NULL DEFAULT 100,
  sort_order INT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS submissions (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL REFERENCES users(id),
  problem_id BIGINT NOT NULL REFERENCES problems(id),
  assignment_id BIGINT REFERENCES assignments(id),
  exam_id BIGINT REFERENCES exams(id),
  language VARCHAR(32) NOT NULL,
  source_code TEXT NOT NULL,
  status VARCHAR(32) NOT NULL,
  score INT NOT NULL DEFAULT 0,
  time_ms INT NOT NULL DEFAULT 0,
  memory_kb INT NOT NULL DEFAULT 0,
  message TEXT,
  trace JSONB,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS submission_results (
  id BIGSERIAL PRIMARY KEY,
  submission_id BIGINT NOT NULL REFERENCES submissions(id) ON DELETE CASCADE,
  case_name VARCHAR(200) NOT NULL,
  status VARCHAR(32) NOT NULL,
  time_ms INT NOT NULL DEFAULT 0,
  memory_kb INT NOT NULL DEFAULT 0,
  message TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS plagiarism_jobs (
  id BIGSERIAL PRIMARY KEY,
  course_id BIGINT NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
  assignment_id BIGINT REFERENCES assignments(id),
  exam_id BIGINT REFERENCES exams(id),
  language VARCHAR(32) NOT NULL,
  status VARCHAR(32) NOT NULL,
  report_object VARCHAR(512),
  summary JSONB,
  message TEXT,
  created_by BIGINT NOT NULL REFERENCES users(id),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS audit_logs (
  id BIGSERIAL PRIMARY KEY,
  actor_user_id BIGINT REFERENCES users(id),
  action VARCHAR(160) NOT NULL,
  resource_type VARCHAR(80),
  resource_id VARCHAR(120),
  ip VARCHAR(80),
  user_agent VARCHAR(512),
  meta JSONB,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_submissions_status ON submissions(status);
CREATE INDEX IF NOT EXISTS idx_submissions_problem_user ON submissions(problem_id, user_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs(action);

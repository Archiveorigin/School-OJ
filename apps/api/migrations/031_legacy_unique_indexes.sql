-- Historical SQL used inline UNIQUE constraints; models use named unique
-- indexes. GORM otherwise tries to drop a non-existent uni_<table>_<column>
-- constraint on a fresh SQL installation. Create each replacement first so
-- uniqueness is continuously enforced, including during this transaction.
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users(email);
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_email_key;
CREATE UNIQUE INDEX IF NOT EXISTS idx_courses_code ON courses(code);
ALTER TABLE courses DROP CONSTRAINT IF EXISTS courses_code_key;
CREATE UNIQUE INDEX IF NOT EXISTS idx_login_attempts_email ON login_attempts(email);
ALTER TABLE login_attempts DROP CONSTRAINT IF EXISTS login_attempts_email_key;
CREATE UNIQUE INDEX IF NOT EXISTS idx_teams_slug ON teams(slug);
ALTER TABLE teams DROP CONSTRAINT IF EXISTS teams_slug_key;
CREATE UNIQUE INDEX IF NOT EXISTS idx_prepared_problems_problem_id ON prepared_problems(problem_id);
ALTER TABLE prepared_problems DROP CONSTRAINT IF EXISTS prepared_problems_problem_id_key;

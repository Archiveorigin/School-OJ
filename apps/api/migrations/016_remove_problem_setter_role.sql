ALTER TABLE users
  ADD COLUMN IF NOT EXISTS can_author BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE users
  DROP CONSTRAINT IF EXISTS users_role_check;

UPDATE users
SET can_author = TRUE
WHERE role IN ('problem_setter', 'teacher');

UPDATE users
SET role = 'student'
WHERE role = 'problem_setter';

ALTER TABLE users
  ADD CONSTRAINT users_role_check
  CHECK (role IN ('student', 'teacher', 'admin'));

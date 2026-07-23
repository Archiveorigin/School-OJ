ALTER TABLE courses
  ADD COLUMN IF NOT EXISTS college VARCHAR(160);

CREATE INDEX IF NOT EXISTS idx_courses_college
  ON courses(college);

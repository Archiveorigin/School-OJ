ALTER TABLE courses
  ADD COLUMN IF NOT EXISTS archived BOOLEAN NOT NULL DEFAULT false;

CREATE INDEX IF NOT EXISTS idx_courses_archived
  ON courses(archived);

ALTER TABLE classes
  ADD COLUMN IF NOT EXISTS archived BOOLEAN NOT NULL DEFAULT false;

CREATE INDEX IF NOT EXISTS idx_classes_archived
  ON classes(archived);

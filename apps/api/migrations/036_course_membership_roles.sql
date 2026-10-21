-- SQL-first installations retained the original three-role CHECK even after
-- the application introduced course administrators and assistants. Bring both
-- SQL-first and historical ORM installations to the same validated domain.
ALTER TABLE course_memberships DROP CONSTRAINT IF EXISTS course_memberships_role_check;
ALTER TABLE course_memberships ADD CONSTRAINT course_memberships_role_check
  CHECK (role IN ('student', 'teacher', 'admin', 'course_admin', 'course_assistant'));

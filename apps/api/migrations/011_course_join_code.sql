ALTER TABLE courses ADD COLUMN IF NOT EXISTS join_code VARCHAR(12);

CREATE UNIQUE INDEX IF NOT EXISTS idx_courses_join_code ON courses(join_code)
  WHERE join_code IS NOT NULL AND join_code <> '';

UPDATE courses
SET join_code = 'R' || lpad(upper(to_hex(((id * 2654435761::bigint) % 4294967296::bigint)::bigint)), 8, '0')
WHERE join_code IS NULL OR join_code = '';

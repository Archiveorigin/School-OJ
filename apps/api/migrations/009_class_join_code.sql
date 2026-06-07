ALTER TABLE classes
  ADD COLUMN IF NOT EXISTS join_code VARCHAR(12);

UPDATE classes
SET join_code = 'C' || lpad(upper(to_hex(((id * 2654435761::bigint) % 4294967296::bigint)::bigint)), 8, '0')
WHERE join_code IS NULL OR join_code = '';

CREATE UNIQUE INDEX IF NOT EXISTS idx_classes_join_code
  ON classes(join_code)
  WHERE join_code IS NOT NULL AND join_code <> '';

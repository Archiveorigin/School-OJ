ALTER TABLE problem_change_tickets ADD COLUMN IF NOT EXISTS exam_id BIGINT REFERENCES exams(id) ON DELETE RESTRICT;
ALTER TABLE problem_change_tickets ADD COLUMN IF NOT EXISTS contest_id BIGINT REFERENCES team_contests(id) ON DELETE RESTRICT;
ALTER TABLE problem_change_tickets ADD COLUMN IF NOT EXISTS problem_version_id BIGINT REFERENCES problem_versions(id) ON DELETE RESTRICT;
ALTER TABLE problem_change_tickets ADD COLUMN IF NOT EXISTS draft_title VARCHAR(200) NOT NULL DEFAULT '';
ALTER TABLE problem_change_tickets ADD COLUMN IF NOT EXISTS draft_statement TEXT NOT NULL DEFAULT '';
ALTER TABLE problem_change_tickets ADD COLUMN IF NOT EXISTS is_public BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE problem_change_tickets ADD COLUMN IF NOT EXISTS revision INTEGER NOT NULL DEFAULT 1;
ALTER TABLE problem_change_tickets ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

-- Old installations may have generated different names for the inline checks.
DO $$ DECLARE item RECORD; BEGIN
  FOR item IN SELECT conname FROM pg_constraint
    WHERE conrelid = 'problem_change_tickets'::regclass AND contype = 'c'
      AND pg_get_constraintdef(oid) LIKE '%action%'
  LOOP EXECUTE format('ALTER TABLE problem_change_tickets DROP CONSTRAINT %I', item.conname); END LOOP;
END $$;
ALTER TABLE problem_change_tickets ADD CONSTRAINT ticket_action_check CHECK (action IN ('create', 'replace', 'archive', 'suggest'));
ALTER TABLE problem_change_tickets ADD CONSTRAINT ticket_context_check CHECK (
  NOT (exam_id IS NOT NULL AND contest_id IS NOT NULL)
  AND (action <> 'create' OR (exam_id IS NULL AND contest_id IS NULL))
  AND (action = 'create' OR problem_id IS NOT NULL OR exam_id IS NOT NULL OR contest_id IS NOT NULL)
  AND (action <> 'archive' OR problem_id IS NOT NULL)
  AND (NOT is_public OR (exam_id IS NULL AND contest_id IS NULL AND action <> 'create'))
  AND revision > 0
);
DROP INDEX IF EXISTS idx_problem_change_tickets_active_problem;
CREATE INDEX IF NOT EXISTS idx_ticket_exam ON problem_change_tickets(exam_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ticket_contest ON problem_change_tickets(contest_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ticket_problem ON problem_change_tickets(problem_id, created_at DESC);

CREATE TABLE IF NOT EXISTS problem_ticket_attachments (
  id BIGSERIAL PRIMARY KEY,
  ticket_id BIGINT NOT NULL REFERENCES problem_change_tickets(id) ON DELETE RESTRICT,
  name VARCHAR(255) NOT NULL,
  object VARCHAR(512) NOT NULL,
  content_type VARCHAR(100) NOT NULL DEFAULT 'application/octet-stream',
  size BIGINT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_ticket_attachment_ticket ON problem_ticket_attachments(ticket_id);
INSERT INTO problem_ticket_attachments(ticket_id, name, object)
SELECT id, attachment_name, attachment_object FROM problem_change_tickets t
WHERE COALESCE(attachment_object, '') <> ''
  AND NOT EXISTS (SELECT 1 FROM problem_ticket_attachments a WHERE a.ticket_id = t.id AND a.object = t.attachment_object);

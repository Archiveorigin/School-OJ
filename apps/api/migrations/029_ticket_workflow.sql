ALTER TABLE problem_change_tickets ADD COLUMN IF NOT EXISTS history JSONB NOT NULL DEFAULT '[]'::jsonb;
DO $$ DECLARE item RECORD; BEGIN
  FOR item IN SELECT conname FROM pg_constraint
    WHERE conrelid = 'problem_change_tickets'::regclass AND contype = 'c'
      AND pg_get_constraintdef(oid) LIKE '%status%'
  LOOP EXECUTE format('ALTER TABLE problem_change_tickets DROP CONSTRAINT %I', item.conname); END LOOP;
END $$;
ALTER TABLE problem_change_tickets ADD CONSTRAINT ticket_status_check
  CHECK (status IN ('pending', 'awaiting_changes', 'processing', 'completed', 'rejected', 'cancelled'));

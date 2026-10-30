ALTER TABLE event_attachments ADD COLUMN IF NOT EXISTS announcement_id BIGINT;
DO $$ BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid='event_attachments'::regclass AND conname='event_attachments_announcement_id_fkey') THEN
  ALTER TABLE event_attachments ADD CONSTRAINT event_attachments_announcement_id_fkey FOREIGN KEY(announcement_id) REFERENCES event_announcements(id) ON DELETE CASCADE;
 END IF;
END $$;
CREATE INDEX IF NOT EXISTS idx_event_attachment_announcement ON event_attachments(announcement_id);

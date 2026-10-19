ALTER TABLE problems ADD COLUMN IF NOT EXISTS event_kind VARCHAR(16) NOT NULL DEFAULT '';
ALTER TABLE problems ADD COLUMN IF NOT EXISTS event_id BIGINT;
DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname='problems_event_scope' AND conrelid='problems'::regclass) THEN ALTER TABLE problems ADD CONSTRAINT problems_event_scope CHECK
  ((event_kind = '' AND event_id IS NULL) OR (event_kind IN ('exam','contest') AND event_id IS NOT NULL)); END IF; END $$;
CREATE INDEX IF NOT EXISTS idx_problem_event ON problems(event_kind,event_id);

DO $$
DECLARE t TEXT;
BEGIN
  FOREACH t IN ARRAY ARRAY['exams','team_contests'] LOOP
    EXECUTE format('ALTER TABLE %I ADD COLUMN IF NOT EXISTS review_status VARCHAR(16) NOT NULL DEFAULT ''approved'',
      ADD COLUMN IF NOT EXISTS review_revision INTEGER NOT NULL DEFAULT 1,
      ADD COLUMN IF NOT EXISTS review_note TEXT NOT NULL DEFAULT '''',
      ADD COLUMN IF NOT EXISTS review_submitted_at TIMESTAMPTZ,
      ADD COLUMN IF NOT EXISTS reviewed_at TIMESTAMPTZ,
      ADD COLUMN IF NOT EXISTS reviewed_by BIGINT',t);
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname=t||'_review_state' AND conrelid=t::regclass) THEN
    EXECUTE format('ALTER TABLE %I ADD CONSTRAINT %I CHECK (review_status IN (''draft'',''pending'',''approved'',''rejected'') AND review_revision > 0)',t,t||'_review_state'); END IF;
  END LOOP;
END $$;
UPDATE exams SET review_status='pending', review_submitted_at=CURRENT_TIMESTAMP,
  review_note='升级后的赛前整场审核：请管理员确认题目、时间及附件'
WHERE deleted_at IS NULL AND starts_at > CURRENT_TIMESTAMP;
UPDATE team_contests SET review_status='draft' WHERE state='draft';
UPDATE team_contests SET review_status='pending', review_submitted_at=CURRENT_TIMESTAMP,
  review_note='升级后的赛前整场审核：请管理员确认题目、时间及附件'
WHERE deleted_at IS NULL AND state='published' AND starts_at > CURRENT_TIMESTAMP;

-- Retire the old exam-deadline publication schedule for future/cancelled
-- papers. Preserve an independent assignment schedule for the same class.
WITH scheduled AS (
  SELECT cp.id,
    (SELECT min(a.due_at) FROM assignments a
      JOIN assignment_problems ap ON ap.assignment_id=a.id
      WHERE a.deleted_at IS NULL AND a.class_id=cp.class_id
        AND ap.problem_id=cp.problem_id AND a.due_at>CURRENT_TIMESTAMP) AS next_release
  FROM class_problems cp
  WHERE cp.release_at>CURRENT_TIMESTAMP AND EXISTS (
    SELECT 1 FROM exams e JOIN exam_problems ep ON ep.exam_id=e.id
    WHERE e.class_id=cp.class_id AND ep.problem_id=cp.problem_id
      AND e.ends_at=cp.release_at
      AND (e.starts_at>CURRENT_TIMESTAMP OR e.deleted_at IS NOT NULL)
  )
), removed AS (
  DELETE FROM class_problems cp USING scheduled s
  WHERE cp.id=s.id AND s.next_release IS NULL RETURNING cp.id
)
UPDATE class_problems cp SET release_at=s.next_release
FROM scheduled s WHERE cp.id=s.id AND s.next_release IS NOT NULL;

CREATE TABLE IF NOT EXISTS event_review_logs (
  id BIGSERIAL PRIMARY KEY, event_kind VARCHAR(16) NOT NULL CHECK(event_kind IN ('exam','contest')),
  event_id BIGINT NOT NULL, revision INTEGER NOT NULL, action VARCHAR(16) NOT NULL,
  actor_id BIGINT NOT NULL, note TEXT NOT NULL, snapshot JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_event_review_log ON event_review_logs(event_kind,event_id);
CREATE TABLE IF NOT EXISTS event_attachments (
  id BIGSERIAL PRIMARY KEY, event_kind VARCHAR(16) NOT NULL CHECK(event_kind IN ('exam','contest')),
  event_id BIGINT NOT NULL, name VARCHAR(255) NOT NULL, object VARCHAR(512) NOT NULL,
  size BIGINT NOT NULL CHECK(size>0 AND size<=33554432), checksum VARCHAR(64) NOT NULL,
  created_by BIGINT NOT NULL, created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_event_attachment ON event_attachments(event_kind,event_id);
CREATE INDEX IF NOT EXISTS idx_exams_archive ON exams(ends_at,id) WHERE deleted_at IS NULL AND review_status='approved';
CREATE INDEX IF NOT EXISTS idx_contests_archive ON team_contests(starts_at,id) WHERE deleted_at IS NULL AND review_status='approved' AND state<>'draft';

ALTER TABLE exams ADD COLUMN IF NOT EXISTS participants_frozen BOOLEAN NOT NULL DEFAULT FALSE;
CREATE TABLE IF NOT EXISTS exam_participants (
  id BIGSERIAL PRIMARY KEY, exam_id BIGINT NOT NULL, user_id BIGINT NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_exam_participant ON exam_participants(exam_id,user_id);
INSERT INTO exam_participants(exam_id,user_id)
SELECT e.id,u.id FROM exams e JOIN users u ON u.role='student' AND u.account_deleted=false
WHERE e.deleted_at IS NULL AND e.review_status='approved' AND (
  (e.class_id IS NOT NULL AND EXISTS(SELECT 1 FROM class_memberships m WHERE m.user_id=u.id AND m.class_id=e.class_id)) OR
  (e.class_id IS NULL AND EXISTS(SELECT 1 FROM course_memberships m WHERE m.user_id=u.id AND m.course_id=e.course_id AND m.role='student')) OR
  EXISTS(SELECT 1 FROM exam_attempts a WHERE a.exam_id=e.id AND a.user_id=u.id) OR
  EXISTS(SELECT 1 FROM submissions s WHERE s.exam_id=e.id AND s.user_id=u.id)
) ON CONFLICT DO NOTHING;
UPDATE exams SET participants_frozen=true WHERE deleted_at IS NULL AND review_status='approved';

ALTER TABLE problems ADD COLUMN IF NOT EXISTS archived_at TIMESTAMPTZ;
CREATE INDEX IF NOT EXISTS idx_problems_archived_at ON problems(archived_at);

CREATE TABLE IF NOT EXISTS problem_versions (
  id BIGSERIAL PRIMARY KEY,
  problem_id BIGINT NOT NULL REFERENCES problems(id) ON DELETE RESTRICT,
  version INTEGER NOT NULL,
  title VARCHAR(200) NOT NULL,
  statement TEXT NOT NULL DEFAULT '',
  tags JSONB NOT NULL DEFAULT '{}'::jsonb,
  difficulty VARCHAR(32) NOT NULL DEFAULT '入门',
  time_limit_ms INTEGER NOT NULL DEFAULT 1000,
  memory_limit_mb INTEGER NOT NULL DEFAULT 256,
  output_limit_kb INTEGER NOT NULL DEFAULT 1024,
  package_object VARCHAR(512) NOT NULL,
  package_checksum VARCHAR(128) NOT NULL,
  manifest JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_by BIGINT NOT NULL REFERENCES users(id),
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(problem_id, version)
);

CREATE INDEX IF NOT EXISTS idx_problem_versions_problem_created
  ON problem_versions(problem_id, version DESC);

INSERT INTO problem_versions (
  problem_id, version, title, statement, tags, difficulty,
  time_limit_ms, memory_limit_mb, output_limit_kb,
  package_object, package_checksum, manifest, created_by, created_at
)
SELECT
  p.id, 1, p.title, COALESCE(p.statement, ''), COALESCE(p.tags, '{}'::jsonb),
  COALESCE(NULLIF(p.difficulty, ''), '入门'), p.time_limit_ms, p.memory_limit_mb,
  p.output_limit_kb, p.package_object, p.package_checksum,
  COALESCE(p.manifest, '{}'::jsonb), p.owner_id, p.created_at
FROM problems p
WHERE NOT EXISTS (
  SELECT 1 FROM problem_versions pv WHERE pv.problem_id = p.id
);

ALTER TABLE problems ADD COLUMN IF NOT EXISTS current_version_id BIGINT;
UPDATE problems p
SET current_version_id = pv.id
FROM problem_versions pv
WHERE pv.problem_id = p.id AND pv.version = 1 AND p.current_version_id IS NULL;
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conname = 'problems_current_version_fk' AND conrelid = 'problems'::regclass
  ) THEN
    ALTER TABLE problems ADD CONSTRAINT problems_current_version_fk
      FOREIGN KEY (current_version_id) REFERENCES problem_versions(id) ON DELETE RESTRICT;
  END IF;
END $$;

ALTER TABLE exam_problems ADD COLUMN IF NOT EXISTS problem_version_id BIGINT;
UPDATE exam_problems ep SET problem_version_id = p.current_version_id
FROM problems p WHERE p.id = ep.problem_id AND ep.problem_version_id IS NULL;
ALTER TABLE exam_problems ALTER COLUMN problem_version_id SET NOT NULL;

ALTER TABLE team_contest_problems ADD COLUMN IF NOT EXISTS problem_version_id BIGINT;
UPDATE team_contest_problems tcp SET problem_version_id = p.current_version_id
FROM problems p WHERE p.id = tcp.problem_id AND tcp.problem_version_id IS NULL;
ALTER TABLE team_contest_problems ALTER COLUMN problem_version_id SET NOT NULL;

ALTER TABLE submissions ADD COLUMN IF NOT EXISTS problem_version_id BIGINT;
UPDATE submissions s SET problem_version_id = COALESCE(
  (SELECT ep.problem_version_id FROM exam_problems ep
   WHERE ep.problem_id = s.problem_id AND ep.exam_id = s.exam_id LIMIT 1),
  (SELECT tcp.problem_version_id FROM team_contest_problems tcp
   WHERE tcp.problem_id = s.problem_id AND tcp.contest_id = s.team_contest_id LIMIT 1),
  (SELECT p.current_version_id FROM problems p WHERE p.id = s.problem_id)
)
WHERE s.problem_version_id IS NULL;
ALTER TABLE submissions ALTER COLUMN problem_version_id SET NOT NULL;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conname = 'exam_problems_problem_version_fk' AND conrelid = 'exam_problems'::regclass
  ) THEN
    ALTER TABLE exam_problems ADD CONSTRAINT exam_problems_problem_version_fk
      FOREIGN KEY (problem_version_id) REFERENCES problem_versions(id) ON DELETE RESTRICT;
  END IF;
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conname = 'team_contest_problems_problem_version_fk' AND conrelid = 'team_contest_problems'::regclass
  ) THEN
    ALTER TABLE team_contest_problems ADD CONSTRAINT team_contest_problems_problem_version_fk
      FOREIGN KEY (problem_version_id) REFERENCES problem_versions(id) ON DELETE RESTRICT;
  END IF;
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conname = 'submissions_problem_version_fk' AND conrelid = 'submissions'::regclass
  ) THEN
    ALTER TABLE submissions ADD CONSTRAINT submissions_problem_version_fk
      FOREIGN KEY (problem_version_id) REFERENCES problem_versions(id) ON DELETE RESTRICT;
  END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_exam_problems_problem_version ON exam_problems(problem_version_id);
CREATE INDEX IF NOT EXISTS idx_team_contest_problems_problem_version ON team_contest_problems(problem_version_id);
CREATE INDEX IF NOT EXISTS idx_submissions_problem_version ON submissions(problem_version_id);

CREATE TABLE IF NOT EXISTS problem_change_tickets (
  id BIGSERIAL PRIMARY KEY,
  requester_id BIGINT NOT NULL REFERENCES users(id),
  problem_id BIGINT REFERENCES problems(id) ON DELETE RESTRICT,
  action VARCHAR(16) NOT NULL CHECK (action IN ('create', 'replace', 'archive')),
  status VARCHAR(16) NOT NULL DEFAULT 'pending'
    CHECK (status IN ('pending', 'processing', 'completed', 'rejected', 'cancelled')),
  target_scope VARCHAR(24) NOT NULL DEFAULT 'public'
    CHECK (target_scope IN ('public', 'prepared', 'team_problem_set')),
  team_problem_set_id BIGINT REFERENCES team_problem_sets(id) ON DELETE SET NULL,
  description TEXT NOT NULL,
  attachment_object VARCHAR(512),
  attachment_name VARCHAR(255),
  resolution_note TEXT NOT NULL DEFAULT '',
  applied_version_id BIGINT REFERENCES problem_versions(id) ON DELETE RESTRICT,
  processed_by BIGINT REFERENCES users(id),
  processed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CHECK (action = 'create' OR problem_id IS NOT NULL),
  CHECK (
    (target_scope = 'team_problem_set' AND team_problem_set_id IS NOT NULL)
    OR target_scope <> 'team_problem_set'
  )
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_problem_change_tickets_active_problem
  ON problem_change_tickets(problem_id)
  WHERE action IN ('replace', 'archive') AND status IN ('pending', 'processing');
CREATE INDEX IF NOT EXISTS idx_problem_change_tickets_requester
  ON problem_change_tickets(requester_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_problem_change_tickets_status
  ON problem_change_tickets(status, created_at ASC);

INSERT INTO problem_change_tickets (
  requester_id, problem_id, action, status, target_scope, description,
  resolution_note, processed_by, processed_at, created_at, updated_at,
  applied_version_id
)
SELECT
  pr.author_id, pr.problem_id, 'create',
  CASE pr.status
    WHEN 'approved' THEN 'completed'
    WHEN 'rejected' THEN 'rejected'
    WHEN 'withdrawn' THEN 'cancelled'
    ELSE 'pending'
  END,
  'public', '由旧版题目审批记录迁移', COALESCE(pr.review_note, ''),
  pr.reviewed_by, pr.reviewed_at, pr.created_at, pr.updated_at,
  CASE WHEN pr.status = 'approved' THEN p.current_version_id ELSE NULL END
FROM problem_reviews pr
JOIN problems p ON p.id = pr.problem_id
WHERE NOT EXISTS (
  SELECT 1 FROM problem_change_tickets pct
  WHERE pct.problem_id = pr.problem_id AND pct.description = '由旧版题目审批记录迁移'
);

ALTER TABLE team_contests DROP CONSTRAINT IF EXISTS team_contests_scoring_rule_check;
UPDATE team_contests SET scoring_rule = CASE scoring_rule
  WHEN 'score' THEN 'ioi'
  WHEN 'penalty' THEN 'acm'
  WHEN 'oi' THEN 'oi'
  WHEN 'ioi' THEN 'ioi'
  WHEN 'acm' THEN 'acm'
  ELSE 'acm' END;
ALTER TABLE team_contests ALTER COLUMN scoring_rule SET DEFAULT 'acm';
ALTER TABLE team_contests ADD CONSTRAINT team_contests_scoring_rule_check
  CHECK (scoring_rule IN ('oi', 'ioi', 'acm'));
ALTER TABLE team_contests ADD COLUMN IF NOT EXISTS freeze_enabled BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE team_contests ADD COLUMN IF NOT EXISTS freeze_duration_minutes INTEGER NOT NULL DEFAULT 60;
ALTER TABLE team_contests ADD CONSTRAINT team_contests_freeze_duration_check
  CHECK (freeze_enabled = false OR (freeze_duration_minutes > 0 AND freeze_duration_minutes < duration_minutes));

ALTER TABLE exams DROP CONSTRAINT IF EXISTS exams_scoring_rule_check;
UPDATE exams SET scoring_rule = CASE scoring_rule
  WHEN 'score' THEN 'ioi'
  WHEN 'penalty' THEN 'acm'
  WHEN 'oi' THEN 'oi'
  WHEN 'ioi' THEN 'ioi'
  WHEN 'acm' THEN 'acm'
  ELSE 'acm' END;
ALTER TABLE exams ALTER COLUMN scoring_rule SET DEFAULT 'acm';
ALTER TABLE exams ADD CONSTRAINT exams_scoring_rule_check
  CHECK (scoring_rule IN ('oi', 'ioi', 'acm'));
ALTER TABLE exams ADD COLUMN IF NOT EXISTS freeze_enabled BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE exams ADD COLUMN IF NOT EXISTS freeze_duration_minutes INTEGER NOT NULL DEFAULT 60;
ALTER TABLE exams ADD CONSTRAINT exams_freeze_configuration_check CHECK (
  freeze_enabled = false OR (
    ends_at IS NOT NULL AND starts_at IS NOT NULL
    AND freeze_duration_minutes > 0
    AND freeze_duration_minutes < (EXTRACT(EPOCH FROM (ends_at - starts_at)) / 60)
  )
);

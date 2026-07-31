CREATE TABLE IF NOT EXISTS teams (
  id BIGSERIAL PRIMARY KEY,
  name VARCHAR(120) NOT NULL,
  slug VARCHAR(30) NOT NULL UNIQUE,
  owner_id BIGINT NOT NULL REFERENCES users(id),
  visibility VARCHAR(16) NOT NULL DEFAULT 'private',
  join_mode VARCHAR(16) NOT NULL DEFAULT 'application',
  contest_permission VARCHAR(16) NOT NULL DEFAULT 'admin',
  join_code VARCHAR(24),
  description VARCHAR(140),
  announcement TEXT,
  icon_url TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS team_memberships (
  id BIGSERIAL PRIMARY KEY,
  team_id BIGINT NOT NULL REFERENCES teams(id),
  user_id BIGINT NOT NULL REFERENCES users(id),
  role VARCHAR(16) NOT NULL DEFAULT 'member',
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(team_id, user_id)
);

CREATE TABLE IF NOT EXISTS team_join_applications (
  id BIGSERIAL PRIMARY KEY,
  team_id BIGINT NOT NULL REFERENCES teams(id),
  user_id BIGINT NOT NULL REFERENCES users(id),
  message VARCHAR(300),
  status VARCHAR(16) NOT NULL DEFAULT 'pending',
  reviewed_by BIGINT REFERENCES users(id),
  reviewed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS team_contests (
  id BIGSERIAL PRIMARY KEY,
  team_id BIGINT NOT NULL REFERENCES teams(id),
  title VARCHAR(200) NOT NULL,
  description TEXT,
  starts_at TIMESTAMPTZ,
  duration_minutes INTEGER NOT NULL DEFAULT 120,
  created_by BIGINT NOT NULL REFERENCES users(id),
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS team_problem_sets (
  id BIGSERIAL PRIMARY KEY,
  team_id BIGINT NOT NULL REFERENCES teams(id),
  title VARCHAR(200) NOT NULL,
  description TEXT,
  created_by BIGINT NOT NULL REFERENCES users(id),
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE problems
  ADD COLUMN IF NOT EXISTS team_id BIGINT REFERENCES teams(id);

CREATE TABLE IF NOT EXISTS team_problem_set_problems (
  id BIGSERIAL PRIMARY KEY,
  problem_set_id BIGINT NOT NULL REFERENCES team_problem_sets(id),
  problem_id BIGINT NOT NULL REFERENCES problems(id),
  label VARCHAR(16),
  sort_order INTEGER NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(problem_set_id, problem_id)
);

CREATE TABLE IF NOT EXISTS team_discussions (
  id BIGSERIAL PRIMARY KEY,
  problem_set_id BIGINT NOT NULL REFERENCES team_problem_sets(id),
  problem_id BIGINT REFERENCES problems(id),
  author_id BIGINT NOT NULL REFERENCES users(id),
  content TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

UPDATE problems
SET difficulty = CASE
  WHEN difficulty IN ('入门', '基础', '普及', '提高', '综合', '挑战') THEN difficulty
  WHEN lower(trim(COALESCE(difficulty, ''))) IN ('简单', 'easy') THEN '基础'
  WHEN lower(trim(COALESCE(difficulty, ''))) IN ('中等', 'medium') THEN '普及'
  WHEN lower(trim(COALESCE(difficulty, ''))) IN ('困难', 'hard') THEN '提高'
  WHEN lower(trim(COALESCE(difficulty, ''))) IN ('challenge') OR trim(COALESCE(difficulty, '')) = '挑战' THEN '挑战'
  WHEN COALESCE(tags->'labels', '[]'::jsonb) ? '挑战' THEN '挑战'
  WHEN COALESCE(tags->'labels', '[]'::jsonb) ? '综合' THEN '综合'
  WHEN COALESCE(tags->'labels', '[]'::jsonb) ? '提高' THEN '提高'
  WHEN COALESCE(tags->'labels', '[]'::jsonb) ? '普及' THEN '普及'
  WHEN COALESCE(tags->'labels', '[]'::jsonb) ? '基础' THEN '基础'
  ELSE '入门'
END;

UPDATE prepared_problems
SET difficulty = CASE
  WHEN difficulty IN ('入门', '基础', '普及', '提高', '综合', '挑战') THEN difficulty
  WHEN lower(trim(COALESCE(difficulty, ''))) IN ('简单', 'easy') THEN '基础'
  WHEN lower(trim(COALESCE(difficulty, ''))) IN ('中等', 'medium') THEN '普及'
  WHEN lower(trim(COALESCE(difficulty, ''))) IN ('困难', 'hard') THEN '提高'
  WHEN lower(trim(COALESCE(difficulty, ''))) IN ('挑战', 'challenge') THEN '挑战'
  ELSE '入门'
END;

ALTER TABLE problems DROP CONSTRAINT IF EXISTS problems_difficulty_check;
ALTER TABLE problems ADD CONSTRAINT problems_difficulty_check
  CHECK (difficulty IN ('入门', '基础', '普及', '提高', '综合', '挑战'));

ALTER TABLE prepared_problems DROP CONSTRAINT IF EXISTS prepared_problems_difficulty_check;
ALTER TABLE prepared_problems ADD CONSTRAINT prepared_problems_difficulty_check
  CHECK (difficulty IN ('入门', '基础', '普及', '提高', '综合', '挑战'));

CREATE INDEX IF NOT EXISTS idx_problems_team_id ON problems(team_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_team_join_applications_pending
  ON team_join_applications(team_id, user_id) WHERE status = 'pending';

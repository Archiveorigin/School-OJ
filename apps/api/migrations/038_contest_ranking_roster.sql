ALTER TABLE team_contests ADD COLUMN IF NOT EXISTS ranking_user_ids JSONB;
ALTER TABLE team_contests ADD CONSTRAINT team_contests_ranking_roster_array
  CHECK (ranking_user_ids IS NULL OR ranking_user_ids = 'null'::jsonb OR jsonb_typeof(ranking_user_ids) = 'array');

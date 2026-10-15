-- Status updates and result details always select by submission_id then id.
CREATE INDEX IF NOT EXISTS idx_submission_results_submission_id_id
  ON submission_results(submission_id, id);
-- The team directory displays the latest application for each team/user pair.
CREATE INDEX IF NOT EXISTS idx_team_join_applications_latest
  ON team_join_applications(user_id, team_id, id DESC);

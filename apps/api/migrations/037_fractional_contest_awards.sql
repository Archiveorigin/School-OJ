-- Preserve existing allocations while allowing counts to be converted to
-- fractional percentages. The existing range/sum constraint remains in force.
ALTER TABLE team_contests
  ALTER COLUMN gold_award_percent TYPE NUMERIC(9,6),
  ALTER COLUMN silver_award_percent TYPE NUMERIC(9,6),
  ALTER COLUMN bronze_award_percent TYPE NUMERIC(9,6);

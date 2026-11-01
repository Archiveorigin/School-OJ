-- Hourly retention deletes and the 3-month audit view both filter on created_at.
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at);

CREATE TABLE IF NOT EXISTS exam_activities (
  id BIGSERIAL PRIMARY KEY,
  exam_id BIGINT NOT NULL REFERENCES exams(id),
  user_id BIGINT NOT NULL REFERENCES users(id),
  event_id VARCHAR(100) NOT NULL,
  event VARCHAR(32) NOT NULL,
  client_at TIMESTAMPTZ,
  ip VARCHAR(80),
  meta JSONB,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_exam_activity_event ON exam_activities(exam_id, user_id, event_id);
CREATE INDEX IF NOT EXISTS idx_exam_activities_exam_id ON exam_activities(exam_id);
CREATE INDEX IF NOT EXISTS idx_exam_activities_user_id ON exam_activities(user_id);
CREATE INDEX IF NOT EXISTS idx_exam_activities_event ON exam_activities(event);
CREATE INDEX IF NOT EXISTS idx_exam_activities_created_at ON exam_activities(created_at);

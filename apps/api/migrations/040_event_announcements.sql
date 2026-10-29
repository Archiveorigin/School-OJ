CREATE TABLE event_announcements (
 id BIGSERIAL PRIMARY KEY,
 event_kind VARCHAR(16) NOT NULL CHECK (event_kind IN ('exam','contest')),
 event_id BIGINT NOT NULL,
 problem_id BIGINT,
 body TEXT NOT NULL CHECK (char_length(body) BETWEEN 1 AND 4000),
 created_by BIGINT NOT NULL,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_event_announcements ON event_announcements(event_kind,event_id,id);

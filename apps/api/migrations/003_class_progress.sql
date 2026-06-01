CREATE TABLE IF NOT EXISTS class_problems (
  id BIGSERIAL PRIMARY KEY,
  class_id BIGINT NOT NULL REFERENCES classes(id) ON DELETE CASCADE,
  problem_id BIGINT NOT NULL REFERENCES problems(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(class_id, problem_id)
);

CREATE TABLE IF NOT EXISTS problem_progresses (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  problem_id BIGINT NOT NULL REFERENCES problems(id) ON DELETE CASCADE,
  status VARCHAR(32) NOT NULL DEFAULT 'unattempted',
  points INT NOT NULL DEFAULT 0,
  points_awarded BOOLEAN NOT NULL DEFAULT false,
  first_accepted TIMESTAMPTZ,
  last_submitted TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(user_id, problem_id)
);

ALTER TABLE assignments ADD COLUMN IF NOT EXISTS class_id BIGINT REFERENCES classes(id) ON DELETE CASCADE;
ALTER TABLE exams ADD COLUMN IF NOT EXISTS class_id BIGINT REFERENCES classes(id) ON DELETE CASCADE;

UPDATE assignments
SET class_id = classes.id
FROM classes
WHERE assignments.class_id IS NULL
  AND classes.id = (
    SELECT min(c.id)
    FROM classes c
    WHERE c.course_id = assignments.course_id
  );

UPDATE exams
SET class_id = classes.id
FROM classes
WHERE exams.class_id IS NULL
  AND classes.id = (
    SELECT min(c.id)
    FROM classes c
    WHERE c.course_id = exams.course_id
  );

INSERT INTO class_problems (class_id, problem_id, created_at)
SELECT DISTINCT assignments.class_id, assignment_problems.problem_id, now()
FROM assignments
JOIN assignment_problems ON assignment_problems.assignment_id = assignments.id
WHERE assignments.class_id IS NOT NULL
ON CONFLICT DO NOTHING;

INSERT INTO class_problems (class_id, problem_id, created_at)
SELECT DISTINCT exams.class_id, exam_problems.problem_id, now()
FROM exams
JOIN exam_problems ON exam_problems.exam_id = exams.id
WHERE exams.class_id IS NOT NULL
ON CONFLICT DO NOTHING;

CREATE INDEX IF NOT EXISTS idx_class_problems_class ON class_problems(class_id);
CREATE INDEX IF NOT EXISTS idx_problem_progresses_status ON problem_progresses(status);
CREATE INDEX IF NOT EXISTS idx_assignments_class_id ON assignments(class_id);
CREATE INDEX IF NOT EXISTS idx_exams_class_id ON exams(class_id);

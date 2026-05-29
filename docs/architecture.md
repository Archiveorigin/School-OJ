# Architecture

The system is a monorepo with three runtime applications.

- `apps/api`: Gin HTTP API, GORM models, PostgreSQL persistence, Redis Streams enqueueing, MinIO object storage, JWT RBAC, SSE, audit logs, and JPlag orchestration.
- `apps/worker`: Redis Streams consumer group `judge-workers`; it reads submissions, downloads problem ZIP packages from MinIO, and executes Docker sandbox runs.
- `apps/web`: Vue 3, Vite, TypeScript, Element Plus, and Monaco UI.

Core flow:

1. Teacher uploads a ZIP package containing `problem.yaml` and test files.
2. API validates the package, stores the ZIP in MinIO, and stores manifest metadata in PostgreSQL.
3. Student creates a submission.
4. API writes the submission row and enqueues `submission_id` to Redis Stream `oj.submissions`.
5. Worker consumes the stream, compiles/runs code in Docker sandbox containers, updates `submissions` and `submission_results`.
6. Web subscribes to `/api/submissions/:id/events` by SSE and updates live status.

RBAC:

- `student`: view course material, submit, view own submissions, leaderboard.
- `teacher`: manage courses/classes, problems, assignments, exams, plagiarism jobs.
- `admin`: all teacher privileges plus user management and audit logs.

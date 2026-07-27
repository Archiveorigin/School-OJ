# Architecture

The system is a monorepo with three runtime applications.

- `apps/api`: Gin HTTP API, GORM models, PostgreSQL persistence, Redis Streams enqueueing, MinIO object storage, JWT RBAC, SSE, audit logs, and JPlag orchestration. Route registration lives in `internal/handlers/routes.go`; business handlers remain split by domain as they are refactored out of the legacy server file.
- `apps/worker`: Redis Streams consumer group `judge-workers`; it reads submissions, downloads problem ZIP packages from MinIO, and executes Docker sandbox runs.
- `apps/web`: Vue 3, Vite, TypeScript, Element Plus, and Monaco UI.

Core flow:

1. Teacher uploads a ZIP package containing `problem.yaml` and test files.
2. API validates the package, stores the ZIP in MinIO, and stores manifest metadata in PostgreSQL.
3. Student creates a submission.
4. API writes the submission row and enqueues `submission_id` to Redis Stream `oj.submissions`.
5. Worker acquires a Redis lease, consumes the stream, compiles/runs code in Docker sandbox containers, and commits the verdict, case results, and problem progress in one database transaction.
6. Worker publishes a lightweight Redis notification after status changes. The API fans it out over `/api/submissions/:id/events`; a 30-second reconciliation protects against ephemeral Pub/Sub loss without one-second database polling.
7. Web receives SSE status events and updates live status.

API compatibility:

- `/api/*` remains available to existing clients.
- New integrations should use the versioned `/api/v1/*` path.
- CORS origins and trusted reverse proxies are configured with `CORS_ALLOWED_ORIGINS` and `TRUSTED_PROXIES`.

Persistence ownership:

- API owns submission creation and enqueueing.
- Worker owns transitions from `queued`/`running` to a judge verdict and writes `submission_results` and `problem_progresses`.
- Worker verdict writes are transactional. A per-submission Redis lease prevents concurrent workers from accepting the same job, is renewed during long runs, and expires after a crash so pending messages can be reclaimed.

RBAC:

- `student`: view course material, submit, view own submissions, leaderboard.
- `teacher`: manage courses/classes, problems, assignments, exams, plagiarism jobs.
- `admin`: all teacher privileges plus user management and audit logs.

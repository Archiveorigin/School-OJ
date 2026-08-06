# Architecture

The system is a monorepo with three runtime applications.

- `apps/api`: Gin HTTP API, GORM models, PostgreSQL persistence, Redis Streams enqueueing, MinIO object storage, JWT RBAC, SSE, audit logs, and JPlag orchestration. Route registration lives in `internal/handlers/routes.go`; business handlers remain split by domain as they are refactored out of the legacy server file.
- `apps/worker`: Redis Streams consumer group `judge-workers`; it reads submissions, downloads problem ZIP packages from MinIO, and executes Docker sandbox runs.
- `apps/web`: Vue 3, Vite, TypeScript, Element Plus, and Monaco UI.

Product domains:

- The course domain owns classes, assignments, school exams, and their course-scoped ranking and submission views.
- The team domain owns team membership, contests, contest problem links, problem sets, and team-scoped submissions. Team contests deliberately use `/teams/:slug/contests/:contestId`; school exams remain under `/exams/:examId`, so similarly shaped screens never share route identity or request state.
- The public problem bank remains the canonical problem content store. Team contests and team problem sets reference those problems, but submission creation is performed through the owning team endpoint and carries `team_contest_id` or `problem_set_id`. This keeps records, status, and rankings inside the correct workspace instead of redirecting through the public problem page.
- Team problem sets keep the linked problem ID only as the persistent identity. The web derives `A`, `B`, `C`, … from the link order for list rows, detail navigation, discussions, and submission filters, so public problem codes never leak into the team worksheet presentation.

Core flow:

1. Teacher uploads a ZIP package containing `problem.yaml` and test files.
2. API validates the package, stores the ZIP in MinIO, and stores manifest metadata in PostgreSQL.
3. Student creates a submission.
4. API writes the submission row and enqueues `submission_id` to Redis Stream `oj.submissions`.
5. Worker acquires a Redis lease, consumes the stream, compiles/runs code in Docker sandbox containers, and commits the verdict, case results, and problem progress in one database transaction.
6. Worker publishes a lightweight Redis notification after status changes. The API fans it out over `/api/submissions/:id/events`; a 30-second reconciliation protects against ephemeral Pub/Sub loss without one-second database polling.
7. Web receives SSE status events and updates live status.

Judge measurements follow contest-system semantics per test case. `time_ms` is the cgroup CPU usage of the whole sandbox process group, while a separate wall-clock deadline stops sleeping or blocked programs. `memory_kb` is the cgroup peak allocation, including language runtimes such as Python and the JVM. OOM events and Docker state, rather than error-message substrings, distinguish memory exhaustion from ordinary crashes; an observed time-limit event takes precedence over cleanup signals. The database problem limits displayed to contestants are authoritative, with package manifest limits retained only as a legacy fallback.

Team workspace flow:

1. A team manager creates a contest with an explicit title, start time, and end time, then links problems through `team_contest_problems`.
2. The contest detail endpoint returns the contest window, linked problems, scoped status, and permission flags. The web route renders problems, submission records, and a live ranking without entering the school-exam router.
3. A team member opens a problem from the worksheet list, switches between lettered problems inside the detail component, and submits from the scoped code dialog. The API validates membership and the owning workspace before creating the submission and enqueueing the normal judge job.
4. The worker uses the same sandbox pipeline for all submissions. The API and web use the context columns to expose the result only in the correct contest or problem set.
5. Team members can inspect the worksheet submission stream using username text search plus problem, verdict, and language selectors. Source code is not returned by this list endpoint; only its computed length is exposed.

Shared web components:

- `ProblemSwitcher.vue` provides the compact lettered problem grid used by school exams, team contests, and team problem sets. It owns only selection presentation; each parent retains its own route, API calls, timing rules, and submission action. School exams mount the navigator in the problem metadata sidebar instead of a separate full-width toolbar.
- `ProblemTagSelector.vue` is the single authoring control for problem creation, problem editing, prepared-problem editing, and exam-side Markdown problem creation. It presents algorithm, time, and source tags in a popup and stores the selected values in the existing problem tag payload.
- Public and prepared problem lists reuse `ProblemTagSelector.vue` as a multi-select filter. A problem matches when any selected tag is present (OR semantics); an empty selection leaves the result set unrestricted.
- `TeamList.vue` keeps the team entry page at navigation level: “我的团队”, “发现”, and the create action. Team summaries render as responsive two-column cards while the nested `TeamWorkspace` owns contest, problem-set, member, and settings navigation.

API compatibility:

- `/api/*` remains available to existing clients.
- New integrations should use the versioned `/api/v1/*` path.
- CORS origins and trusted reverse proxies are configured with `CORS_ALLOWED_ORIGINS` and `TRUSTED_PROXIES`.

Persistence ownership:

- API owns submission creation and enqueueing.
- Worker owns transitions from `queued`/`running` to a judge verdict and writes `submission_results` and `problem_progresses`.
- Worker verdict writes are transactional. A per-submission Redis lease prevents concurrent workers from accepting the same job, is renewed during long runs, and expires after a crash so pending messages can be reclaimed.
- API owns team contest/problem-set link validation and submission context assignment. A contest moves through `draft → published → running → closed`; publishing freezes its problem links and scoring rule and snapshots eligible participants.
- Submission editors read the caller's last code from `/submissions/latest`, backed by context-specific composite indexes. Full submission lists are presentation history only and are never used as the source of truth for editor recovery.
- Contest ranking uses a PostgreSQL grouped query per participant/problem and performs only final row shaping in the API. It does not load every raw contest submission into application memory.
- Startup migrations are ordered embedded SQL files recorded in `schema_migrations` with SHA-256 checksums. Existing AutoMigrate-era installations are baselined through migration 019, then upgraded by the same versioned runner.

RBAC:

- `student`: view course material, submit, view own submissions, leaderboard.
- `teacher`: manage courses/classes, problems, assignments, exams, plagiarism jobs.
- `admin`: all teacher privileges plus user management and audit logs.

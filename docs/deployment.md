# Deployment

Local:

```bash
cp .env.example .env
# Set JWT_SECRET to at least 32 random bytes, for example:
# openssl rand -hex 32
docker compose up -d --build
```

Pull judge sandbox images before the first submission:

```bash
./scripts/pull_sandbox_images.sh
docker compose restart worker
```

The compose worker uses `/var/lib/school-oj-worker` as its default host-visible
sandbox root. Keep `OJ_WORK_ROOT` on persistent storage; do not configure it
under `/tmp` because host cleanup can remove the path while the worker is still
running.

Open:

- Web: http://localhost:3000
- API health: http://localhost:8080/healthz
- MinIO console: http://localhost:9001
- Mailpit mailbox: http://localhost:8025

Local startup creates admin, teacher, and student seed users for operator verification. Replace the initial credentials or disable `SEED_DATA` before production use.

Kubernetes:

1. Build and push `school-oj-api`, `school-oj-worker`, and `school-oj-web`.
2. Provision PostgreSQL, Redis, and MinIO, or add StatefulSets for them.
3. Update secrets in `deploy/k8s/school-oj.yaml`.
4. Apply:

```bash
kubectl apply -f deploy/k8s/school-oj.yaml
```

Native JPlag:

Mount a JPlag jar into the API container and set `JPLAG_JAR_PATH`. Without it, the API still creates a report object using a lightweight token-overlap fallback so the workflow remains testable.

Remote compose deployments commonly expose only the web service. In that shape,
use the web entrypoint for smoke checks:

```bash
WEB=http://mc.citprobe.cn:25565 ./scripts/smoke.sh
```

For an in-place application update, fast-forward the deployment checkout, rebuild
the application images, recreate services while removing obsolete Compose
containers, and then verify health:

```bash
git pull --ff-only
docker compose build api worker web
docker compose up -d --remove-orphans
docker container prune -f
curl -fsS http://127.0.0.1:25565/healthz
```

The API startup applies the repository migrations, including the team contest
workspace tables and submission context columns. Keep the API and worker images
on the same revision so newly scoped submissions are consumed by a compatible
worker.

For production, set a strong `JWT_SECRET`, database password, and MinIO
credentials in `.env`; do not reuse the example values. PostgreSQL and Redis are
bound to localhost by default in compose (`POSTGRES_BIND`, `REDIS_BIND`) so they
are not unintentionally exposed on public interfaces. The judge worker also
supports `SUBMISSION_MAX_RETRIES` and `SUBMISSION_RETRY_IDLE_SECONDS` for
recovering pending Redis Stream messages after worker restarts.

`JWT_SECRET` is required by the canonical Compose file; deployment fails during
configuration interpolation when it is missing. `CORS_ALLOWED_ORIGINS` is a
comma-separated browser-origin allowlist. `TRUSTED_PROXIES` is a comma-separated
list of reverse-proxy IPs or CIDRs; do not add untrusted client networks because
rate limits use the trusted client IP.

The API serves both `/api/*` and `/api/v1/*`. Existing web deployments remain
compatible, while new external clients should use `/api/v1`.

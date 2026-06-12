# Deployment

Local:

```bash
docker compose up -d --build
```

Pull judge sandbox images before the first submission:

```bash
./scripts/pull_sandbox_images.sh
docker compose restart worker
```

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

For production, set a strong `JWT_SECRET`, database password, and MinIO
credentials in `.env`; do not reuse the example values. PostgreSQL and Redis are
bound to localhost by default in compose (`POSTGRES_BIND`, `REDIS_BIND`) so they
are not unintentionally exposed on public interfaces. The judge worker also
supports `SUBMISSION_MAX_RETRIES` and `SUBMISSION_RETRY_IDLE_SECONDS` for
recovering pending Redis Stream messages after worker restarts.

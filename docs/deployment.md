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

Default users:

- `admin@school.local` / `password`
- `teacher@school.local` / `password`
- `student@school.local` / `password`

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

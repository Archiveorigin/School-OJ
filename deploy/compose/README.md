# Docker Compose

Run from the repository root:

```bash
docker compose up -d --build
```

Or run the compatibility entrypoint, which includes the canonical root file so
deployment settings cannot drift:

```bash
cd deploy/compose
docker compose up -d --build
```

The root `compose.yaml` is the canonical deployment definition. Docker Compose
2.20 or newer is required when using the compatibility entrypoint.

Before accepting submissions, run `./scripts/pull_sandbox_images.sh` from the
repository root so the host Docker daemon has the judge images. PostgreSQL and
Redis bind to `127.0.0.1` by default; set `POSTGRES_BIND` or `REDIS_BIND`
explicitly only when external access is required and protected.

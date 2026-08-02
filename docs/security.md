# Judge Sandbox Security

The worker invokes Docker with these controls for every compile and run:

- `--network none`
- `--read-only`
- `--tmpfs /tmp:rw,nosuid,nodev,noexec,size=64m`
- `--user 65532:65532`
- `--cap-drop ALL`
- `--security-opt no-new-privileges`
- `--security-opt seccomp=/etc/oj-seccomp.json`
- `--pids-limit`
- `--cpus`
- `--memory` and `--memory-swap` set to the same value
- Go context timeout for wall time
- bounded stdout/stderr buffers for output limit
- `--ipc none`, `--ulimit core=0`, and a small open-file limit
- Docker logging disabled for short-lived sandboxes
- read-only `/work` bind mount while user code runs (compile containers alone receive write access)
- deterministic force-removal of a named sandbox after timeout or output-limit cancellation

Compilation uses a separate sandbox limit of 30 seconds and at least 1024 MB memory, because compiler resource usage is not the same as the submitted program's runtime limit. Test execution still uses the problem's `time_limit_ms`, `memory_limit_mb`, and `output_limit_kb`.

Runtime accounting reads cgroup CPU usage and peak memory for the complete sandbox process group. A separate wall-clock guard terminates blocked or sleeping submissions. Memory-limit classification uses cgroup OOM counters and Docker's OOM state; generic text such as `Killed` or `memory` is not sufficient to produce a memory-limit verdict.

The included seccomp profile blocks network socket calls, mount/module/keyring/BPF/perf operations, ptrace, reboot, swap, and namespace unshare calls.

Sandbox workspaces are owned by UID/GID `65532` with mode `0700`; source files
use mode `0600`. They are not world-writable on the host. Output-limit overflow
cancels the active Docker command immediately, and the worker force-removes the
named container as a final cleanup step.

Because the compose worker talks to the host Docker daemon through `/var/run/docker.sock`, sandbox workspaces are created under `SANDBOX_WORK_ROOT` and that absolute path is bind-mounted into the worker at the same absolute path. The default is `/var/lib/school-oj-worker`, which must remain present while the worker is running. Do not place it under host-managed temporary directories such as `/tmp`; their cleanup can invalidate the bind mount while leaving the worker container running. Override `OJ_WORK_ROOT` before `docker compose up` when a different persistent host path is required.

The host Docker daemon must have the judge images available: `gcc:14-bookworm`, `python:3.12-slim`, and `eclipse-temurin:21-jdk`. Run `./scripts/pull_sandbox_images.sh` before the first submission, especially on networks where automatic pulls are slow or blocked.

The worker validates its work root and seccomp profile at startup, while the
Compose healthcheck also validates Docker daemon access. Missing Docker images,
Docker daemon connectivity, sandbox path/mount failures, and image resolution
failures are retryable infrastructure errors. The worker requeues the
submission with a bounded retry count and also claims idle pending Redis Stream
messages, so a worker crash or restart does not leave submissions permanently
stuck in `queued` or `running`.

Each active judge holds a renewable Redis lease keyed by submission ID. A
duplicate stream delivery remains pending while the lease is active. Verdict,
case-result replacement, and progress updates commit in one PostgreSQL
transaction, so a partial write cannot expose a final status with stale cases.

Problem ZIP packages are validated by both API and worker. Accepted entries are
limited to `problem.yaml`, `tests/*.in`, `tests/*.out`, and supported image
assets under `assets/`; unsafe paths, unsupported files, empty test sets, and
oversized archives are rejected before judging.

Sensitive endpoints use Redis-backed token-bucket limits. Login, registration,
verification-code, password-reset, problem upload, plagiarism, SSE connection,
and submission creation limits work across multiple API instances. Redis
failure fails these protected requests closed with HTTP 503 rather than silently
removing abuse protection.

The API and web entrypoint set anti-framing, MIME-sniffing, referrer, browser
permission, and resource-policy headers. The web entrypoint also uses a
Content-Security-Policy. CORS and trusted proxy ranges are environment-driven;
only list proxies that are directly controlled by the deployment.

Bearer tokens are accepted only in the `Authorization` header. The web client
uses authenticated `fetch` streaming for SSE and authenticated blob requests for
private problem images, so long-lived JWTs are not copied into query strings or
reverse-proxy access logs.

For production, run workers on isolated nodes and treat Docker socket access as privileged infrastructure. The compose setup is deployable for a school lab or staging environment; a hardened production cluster should move sandbox execution to dedicated worker hosts.

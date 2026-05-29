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

Compilation uses a separate sandbox limit of 30 seconds and at least 1024 MB memory, because compiler resource usage is not the same as the submitted program's runtime limit. Test execution still uses the problem's `time_limit_ms`, `memory_limit_mb`, and `output_limit_kb`.

The included seccomp profile blocks network socket calls, mount/module/keyring/BPF/perf operations, ptrace, reboot, swap, and namespace unshare calls.

Because the compose worker talks to the host Docker daemon through `/var/run/docker.sock`, sandbox workspaces are created under `SANDBOX_WORK_ROOT` and that absolute path is bind-mounted into the worker at the same absolute path. The default is `/tmp/school-oj-worker`; override `OJ_WORK_ROOT` before `docker compose up` if that path is not suitable.

The host Docker daemon must have the judge images available: `gcc:14-bookworm`, `python:3.12-slim`, and `eclipse-temurin:21-jdk`. Run `./scripts/pull_sandbox_images.sh` before the first submission, especially on networks where automatic pulls are slow or blocked.

For production, run workers on isolated nodes and treat Docker socket access as privileged infrastructure. The compose setup is deployable for a school lab or staging environment; a hardened production cluster should move sandbox execution to dedicated worker hosts.

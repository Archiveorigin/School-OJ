#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
# Read the worker's single source of truth instead of maintaining floating tags.
sed -nE 's/^[[:space:]]*(GCCImage|PythonImage|JavaImage)[[:space:]]*=[[:space:]]*"([^"]+)".*/\2/p' apps/worker/internal/runner/lifecycle.go |
while IFS= read -r image; do docker image inspect "$image" >/dev/null 2>&1 || docker pull "$image"; done

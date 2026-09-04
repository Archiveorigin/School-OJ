#!/usr/bin/env bash
set -euo pipefail

# Safe host cleanup for low-disk deployments. Volumes are deliberately never
# removed because they contain PostgreSQL, Redis and object-storage data.
max_age="${DOCKER_CACHE_MAX_AGE:-168h}"

echo "Disk usage before cleanup:"
df -h .
docker system df

docker container prune --force
docker image prune --force
docker builder prune --force --filter "until=${max_age}"

echo "Disk usage after cleanup:"
df -h .
docker system df

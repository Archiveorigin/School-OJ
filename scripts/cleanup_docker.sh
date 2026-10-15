#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
project=$(docker compose config --format json | python3 -c 'import json,sys; print(json.load(sys.stdin)["name"])')
echo "Cleaning expired judge sandboxes for $project"
docker compose exec -T worker worker --cleanup
# Remove only this project's stopped one-off containers, never service/data
# containers, volumes, rollback images, or another project's build cache.
mapfile -t containers < <(docker ps -aq \
  --filter "label=com.docker.compose.project=$project" \
  --filter 'label=com.docker.compose.oneoff=True' --filter status=exited)
if [ "${#containers[@]}" -gt 0 ]; then docker rm "${containers[@]}"; fi
docker system df

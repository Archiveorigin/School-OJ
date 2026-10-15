#!/usr/bin/env bash
set -euo pipefail
umask 077
cd "$(dirname "$0")/.."
backup_dir=$(realpath "${1:?Usage: scripts/restore.sh /absolute/backup-directory}")
test -f "$backup_dir/COMPLETE"
(cd "$backup_dir" && sha256sum --check --strict SHA256SUMS)
project="${RESTORE_PROJECT:-school-oj-restore-$(date -u +%Y%m%d%H%M%S)}"
[[ "$project" =~ ^school-oj-restore-[a-z0-9-]+$ ]] || { echo 'Restore is restricted to a new isolated project.' >&2; exit 1; }
if [ -n "$(docker ps -aq --filter "label=com.docker.compose.project=$project")$(docker volume ls -q --filter "label=com.docker.compose.project=$project")" ]; then
  echo 'Target already contains containers or volumes; refusing to overwrite it.' >&2; exit 1
fi
restore_dir=$(mktemp -d "${TMPDIR:-/tmp}/$project.XXXXXX")
# An older backup may predate required credentials (for example Redis auth).
# Allow an explicit, private replacement config for the isolated target.
cp "${RESTORE_ENV_FILE:-$backup_dir/source.env}" "$restore_dir/.env"
cat > "$restore_dir/override.yaml" <<'YAML'
services:
  postgres:
    ports: !reset []
  web:
    ports: !reset []
YAML
compose=(docker compose --env-file "$restore_dir/.env" -p "$project" -f compose.yaml -f "$restore_dir/override.yaml")
"${compose[@]}" config --quiet
# No application service or host port is started during restoration.
"${compose[@]}" up -d --wait postgres
"${compose[@]}" create --no-deps minio redis
helper=$(docker inspect --format '{{.Image}}' "$("${compose[@]}" ps -aq postgres)")
"${compose[@]}" exec -T postgres sh -ec 'pg_restore --exit-on-error --single-transaction --no-owner --no-privileges -U "$POSTGRES_USER" -d "$POSTGRES_DB"' < "$backup_dir/postgres.dump"
for service in minio redis; do
  container=$("${compose[@]}" ps -aq "$service")
  test -n "$container"
  docker run --rm --network none --read-only --user 0 --volumes-from "$container" \
    --mount "type=bind,src=$backup_dir,dst=/backup,readonly" --entrypoint sh "$helper" \
    -ec "test -z \"\$(ls -A /data)\"; tar -xzf /backup/$service.tar.gz -C /data; cd /data; sha256sum -c /backup/$service.files.sha256"
done
"${compose[@]}" up -d --wait minio redis
"${compose[@]}" exec -T postgres sh -ec 'psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "SELECT count(*) AS migrations FROM schema_migrations"'
printf 'Isolated restore verified. Project: %s\nConfig: %s\nProduction was not modified.\n' "$project" "$restore_dir"

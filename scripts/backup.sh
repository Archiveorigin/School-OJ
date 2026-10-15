#!/usr/bin/env bash
set -euo pipefail
umask 077
cd "$(dirname "$0")/.."
backup_root="${BACKUP_ROOT:-/opt/School-OJ-backups}"
mkdir -p "$backup_root"
backup_dir=$(mktemp -d "$backup_root/backup-$(date -u +%Y%m%dT%H%M%SZ)-XXXXXX")
compose=(docker compose)
running=$(${compose[@]} ps --status running --services)
resume() {
  status=$?
  trap - EXIT
  # Restart only services that were running before this backup.
  if [ -n "$running" ]; then
    readarray -t services <<< "$running"
    "${compose[@]} start "${services[@]}" || status=1
  fi
  if [ "$status" -ne 0 ]; then echo "Backup incomplete: $backup_dir" >&2; fi
  exit "$status"
}
trap resume EXIT
"${compose[@]} config --quiet
"${compose[@]} stop -t 30 web api worker
"${compose[@]} exec -T postgres sh -ec 'pg_dump -U "$POSTGRES_USER" -d "$POSTGRES_DB" --format=custom --no-owner --no-privileges' > "$backup_dir/postgres.dump"
"${compose[@]} exec -T postgres pg_restore --list < "$backup_dir/postgres.dump" > "$backup_dir/postgres.list"
# Stop object and queue writers for a consistent filesystem snapshot. PostgreSQL
# remains running; pg_dump has already taken its transaction-consistent snapshot.
"${compose[@]} stop -t 30 minio redis
helper=$(docker inspect --format '{{.Image}}' "$("${compose[@]} ps -aq postgres)")
for service in minio redis; do
  container=$("${compose[@]} ps -aq "$service")
  test -n "$container"
  docker run --rm --network none --read-only --user 0 --volumes-from "$container:ro" \
    --mount "type=bind,src=$backup_dir,dst=/backup" --entrypoint sh "$helper" \
    -ec "cd /data; tar -czf /backup/$service.tar.gz .; find . -type f -exec sha256sum {} \; > /backup/$service.files.sha256"
done
cp .env "$backup_dir/source.env"
cp compose.yaml "$backup_dir/source-compose.yaml"
git rev-parse HEAD > "$backup_dir/commit.txt"
docker inspect --format '{{.Name}} {{.Image}}' $("${compose[@]} ps -aq) > "$backup_dir/images.txt"
(cd "$backup_dir" && sha256sum postgres.dump postgres.list minio.tar.gz redis.tar.gz minio.files.sha256 redis.files.sha256 source.env source-compose.yaml commit.txt images.txt > SHA256SUMS)
touch "$backup_dir/COMPLETE"
printf 'Verified backup: %s\n' "$backup_dir"

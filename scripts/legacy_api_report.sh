#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
lua='local coverage=redis.call("BITCOUNT",KEYS[1]); local total=0; local usage=redis.call("HGETALL",KEYS[2]); for i=1,#usage,2 do if string.sub(usage[i],1,7)=="legacy " then total=total+tonumber(usage[i+1]) end end; return {coverage,total}'
eligible=true
printf 'UTC date\tcovered minutes\tlegacy requests\n'
for offset in $(seq 30 -1 1); do
  day=$(date -u -d "$offset days ago" +%F)
  # Read the password inside the container; do not put it in command arguments.
  result=$(docker compose exec -T redis sh -ec 'export REDISCLI_AUTH="$REDIS_PASSWORD"; exec redis-cli --raw EVAL "$1" 2 "$2" "$3"' sh "$lua" "oj:api:coverage:$day" "oj:api:usage:$day")
  mapfile -t values <<< "$result"
  [[ "${values[0]:-}" =~ ^[0-9]+$ && "${values[1]:-}" =~ ^[0-9]+$ ]] || { echo 'Invalid observation data' >&2; exit 1; }
  printf '%s\t%s\t%s\n' "$day" "${values[0]}" "${values[1]}"
  if [ "${values[0]}" != 1440 ] || [ "${values[1]}" != 0 ]; then eligible=false; fi
done
if [ "$eligible" != true ]; then
  echo 'Legacy API must remain enabled: calls or observation gaps exist.' >&2
  exit 1
fi
echo '30 complete UTC days with no legacy calls verified; review external clients before retirement.'

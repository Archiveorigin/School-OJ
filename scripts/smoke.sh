#!/usr/bin/env sh
set -eu

WEB="${WEB:-http://localhost:${WEB_PORT:-25565}}"
if [ -n "${API_BASE:-}" ]; then
  API_BASE="${API_BASE%/}"
  HEALTH="${HEALTH:-${WEB%/}/healthz}"
elif [ -n "${API:-}" ]; then
  API="${API%/}"
  API_BASE="$API/api"
  HEALTH="${HEALTH:-$API/healthz}"
else
  API_BASE="${WEB%/}/api"
  HEALTH="${HEALTH:-${WEB%/}/healthz}"
fi

curl -fsS "$HEALTH" >/dev/null
curl -fsS "$WEB/" >/dev/null

TOKEN="$(curl -fsS "$API_BASE/auth/login" \
  -H 'content-type: application/json' \
  -d '{"email":"student@school.local","password":"password"}' | sed -n 's/.*"token":"\([^"]*\)".*/\1/p')"

curl -fsS "$API_BASE/problems" -H "authorization: Bearer $TOKEN" >/dev/null
echo "smoke ok"

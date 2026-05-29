#!/usr/bin/env sh
set -eu

API="${API:-http://localhost:8080}"
WEB="${WEB:-http://localhost:3000}"

curl -fsS "$API/healthz" >/dev/null
curl -fsS "$WEB/" >/dev/null

TOKEN="$(curl -fsS "$API/api/auth/login" \
  -H 'content-type: application/json' \
  -d '{"email":"student@school.local","password":"password"}' | sed -n 's/.*"token":"\([^"]*\)".*/\1/p')"

curl -fsS "$API/api/problems" -H "authorization: Bearer $TOKEN" >/dev/null
echo "smoke ok"

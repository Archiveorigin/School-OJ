#!/usr/bin/env sh
set -eu

if [ "${JPLAG_JAR_PATH:-}" = "" ]; then
  echo "Set JPLAG_JAR_PATH=/path/to/jplag.jar before running the API for native JPlag reports." >&2
  exit 1
fi

java -jar "$JPLAG_JAR_PATH" "$@"

#!/usr/bin/env sh
set -eu

IMAGES="
gcc:14-bookworm
python:3.12-slim
eclipse-temurin:21-jdk
"

for image in $IMAGES; do
  echo "Pulling $image"
  docker pull "$image"
done

echo "Sandbox images are ready."

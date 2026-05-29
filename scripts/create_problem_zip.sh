#!/usr/bin/env sh
set -eu

OUT="${1:-/tmp/a-plus-b.zip}"
DIR="$(mktemp -d)"
trap 'rm -rf "$DIR"' EXIT

mkdir -p "$DIR/tests"
cat > "$DIR/problem.yaml" <<'YAML'
slug: a-plus-b
title: A + B Problem
statement: 输入两个整数 a 和 b，输出它们的和。
time_limit_ms: 1000
memory_limit_mb: 128
output_limit_kb: 64
cases:
  - name: sample1
    input: tests/01.in
    output: tests/01.out
    weight: 50
  - name: sample2
    input: tests/02.in
    output: tests/02.out
    weight: 50
YAML
printf '1 2\n' > "$DIR/tests/01.in"
printf '3\n' > "$DIR/tests/01.out"
printf '100 250\n' > "$DIR/tests/02.in"
printf '350\n' > "$DIR/tests/02.out"
(cd "$DIR" && zip -qr "$OUT" .)
echo "$OUT"

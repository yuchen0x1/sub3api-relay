#!/bin/sh
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
cd "$repo_root"

fail() {
  printf 'docker runtime resources test failed: %s\n' "$1" >&2
  exit 1
}

assert_line() {
  file=$1
  line=$2
  sed 's/\r$//' "$file" | grep -Fqx "$line" || fail "$file is missing: $line"
}

assert_count() {
  file=$1
  line=$2
  expected=$3
  actual=$(sed 's/\r$//' "$file" | grep -Fxc "$line" || true)
  [ "$actual" -eq "$expected" ] || fail "$file has $actual occurrences of '$line', expected $expected"
}

test -s backend/resources/model-pricing/model_prices_and_context_window.json || \
  fail 'fallback pricing data is missing or empty'

# v0.3.1 起镜像统一由根 Dockerfile 与 deploy/Dockerfile 构建，goreleaser 已移除。
assert_line Dockerfile 'COPY --from=backend-builder --chown=sub3api:sub3api /app/backend/resources /app/resources'
assert_line deploy/Dockerfile 'COPY --from=backend-builder --chown=sub3api:sub3api /app/backend/resources /app/resources'
assert_count Dockerfile 'COPY --from=backend-builder --chown=sub3api:sub3api /app/backend/resources /app/resources' 1

printf 'docker runtime resources test passed\n'

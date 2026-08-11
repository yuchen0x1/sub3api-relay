#!/bin/sh
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
cd "$repo_root"

check_application_security_opt() {
  file=$1
  count=$(
    awk '
      $0 == "  sub2api:" {
        in_application = 1
        next
      }
      in_application && $0 ~ /^  [A-Za-z0-9_-]+:$/ {
        in_application = 0
      }
      in_application && $0 == "    security_opt:" {
        in_security_opt = 1
        next
      }
      in_application && in_security_opt && $0 == "      - no-new-privileges:true" {
        count++
      }
      END { print count + 0 }
    ' "$file"
  )

  if [ "$count" -ne 1 ]; then
    printf '%s must enable no-new-privileges exactly once for the sub2api service\n' "$file" >&2
    exit 1
  fi
}

check_default_admin_credentials() {
  file=$1
  grep -Fq '      - ADMIN_EMAIL=${ADMIN_EMAIL:-admin}' "$file" || {
    printf '%s must default ADMIN_EMAIL to admin\n' "$file" >&2
    exit 1
  }
  grep -Fq '      - ADMIN_PASSWORD=${ADMIN_PASSWORD:-}' "$file" || {
    printf '%s must not provide a default administrator password\n' "$file" >&2
    exit 1
  }
}

check_loopback_binding() {
  file=$1
  grep -Fq '${BIND_HOST:-127.0.0.1}:${SERVER_PORT:-8080}:8080' "$file" || {
    printf '%s must bind the published application port to loopback by default\n' "$file" >&2
    exit 1
  }
}

for compose_file in \
  deploy/docker-compose.yml \
  deploy/docker-compose.local.yml \
  deploy/docker-compose.standalone.yml \
  deploy/docker-compose.dev.yml
do
  check_application_security_opt "$compose_file"
  check_default_admin_credentials "$compose_file"
  check_loopback_binding "$compose_file"
done

grep -Fxq 'ADMIN_EMAIL=admin' deploy/.env.example
grep -Fxq 'ADMIN_PASSWORD=' deploy/.env.example
grep -Fxq 'BIND_HOST=127.0.0.1' deploy/.env.example

printf 'docker compose security test passed\n'

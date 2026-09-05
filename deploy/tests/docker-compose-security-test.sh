#!/bin/sh
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
cd "$repo_root"

check_application_security_opt() {
  file=$1
  count=$(
    awk '
      $0 == "  sub3api:" {
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
    printf '%s must enable no-new-privileges exactly once for the sub3api service\n' "$file" >&2
    exit 1
  fi
}

check_owner_setup_mode() {
  file=$1
  grep -Fq '      - AUTO_SETUP_OWNER_SETUP=true' "$file" || {
    printf '%s must enable the two-stage site-owner setup\n' "$file" >&2
    exit 1
  }
  if grep -Eq '^[[:space:]]*- ADMIN_(EMAIL|PASSWORD)=' "$file"; then
    printf '%s must not inject a default administrator credential\n' "$file" >&2
    exit 1
  fi
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
  check_owner_setup_mode "$compose_file"
  check_loopback_binding "$compose_file"
done

grep -Fxq 'AUTO_SETUP_OWNER_SETUP=true' deploy/.env.example
if grep -Eq '^ADMIN_(EMAIL|PASSWORD)=' deploy/.env.example; then
  echo 'deploy/.env.example must not ship default administrator credentials' >&2
  exit 1
fi
grep -Fxq 'BIND_HOST=127.0.0.1' deploy/.env.example

printf 'docker compose security test passed\n'

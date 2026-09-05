#!/bin/sh
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
cd "$repo_root"

assert_line() {
  file=$1
  line=$2
  sed 's/\r$//' "$file" | grep -Fqx "$line" || {
    printf '%s is missing: %s\n' "$file" "$line" >&2
    exit 1
  }
}

assert_line deploy/sub3api.service 'Environment=SERVER_HOST=0.0.0.0'
assert_line deploy/sub3api.service 'Environment=SETUP_SERVER_HOST=127.0.0.1'
grep -Fq 'Environment=SETUP_SERVER_HOST=${SETUP_SERVER_HOST}' deploy/install.sh
grep -Fq 'SSH: ssh -L ${SERVER_PORT}:127.0.0.1:${SERVER_PORT}' deploy/install.sh
# The installer deliberately adds connect/read timeouts to checksum downloads;
# assert the fail-closed curl path without coupling the test to argument order.
grep -Fq 'curl -fsSL' deploy/install.sh
if grep -F 'checksum_not_found' deploy/install.sh | grep -Fq 'print_warning'; then
  echo "installer must fail closed when checksums are unavailable" >&2
  exit 1
fi

printf 'installer setup binding checks passed\n'

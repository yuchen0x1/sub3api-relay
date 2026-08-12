#!/bin/bash

set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
script="$repo_root/deploy/docker-deploy.sh"

bash -n "$script"

if grep -Eq 'echo .*\$\{?(POSTGRES_PASSWORD|JWT_SECRET|TOTP_ENCRYPTION_KEY)\}?' "$script"; then
  echo "docker deploy script prints generated secrets" >&2
  exit 1
fi

grep -Fq 'input_device="/dev/tty"' "$script"
grep -Fq 'ADMIN_PASSWORD is required when no terminal is available.' "$script"
grep -Fq 'Generated credentials were written to the owner-readable .env file.' "$script"
grep -Fq 'deploy-checksums.txt' "$script"
grep -Fq 'verify_deployment_asset' "$script"
if grep -Fq 'raw.githubusercontent.com/yuchen0x1/sub3api-relay/main' "$script"; then
  echo "docker deploy script downloads mutable main-branch deployment files" >&2
  exit 1
fi

password=$(ADMIN_PASSWORD='correct-horse-battery-staple' bash -c 'source <(head -n -1 "$1"); read_admin_password' bash "$script")
if [ "$password" != 'correct-horse-battery-staple' ]; then
  echo "read_admin_password emitted output besides the password" >&2
  exit 1
fi

printf 'docker deploy secret handling checks passed\n'

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

if ADMIN_PASSWORD=$'valid-password\nINJECTED=true' bash -c 'source <(head -n -1 "$1"); read_admin_password' bash "$script" >/dev/null 2>&1; then
  echo "read_admin_password accepted a line break" >&2
  exit 1
fi

safe_password=$(ADMIN_PASSWORD='Safe-Pass_123@prod' bash -c 'source <(head -n -1 "$1"); read_admin_password' bash "$script")
if [ "$safe_password" != 'Safe-Pass_123@prod' ]; then
  echo "read_admin_password changed a safe password" >&2
  exit 1
fi

compose_config() {
  if docker compose version >/dev/null 2>&1; then
    docker compose "$@"
    return
  fi
  if command -v cmd.exe >/dev/null 2>&1; then
    cmd.exe /c docker compose "$@"
    return
  fi
  echo "Docker Compose is required for password round-trip checks" >&2
  return 1
}

test_dir=$(mktemp -d "$repo_root/.docker-deploy-password-test.XXXXXX")
trap 'rm -rf "$test_dir"' EXIT
cat > "$test_dir/compose.yml" <<'YAML'
services:
  probe:
    image: alpine:3.20
    environment:
      ADMIN_PASSWORD: ${ADMIN_PASSWORD}
YAML

for password in \
  'password with spaces' \
  'password$HOME' \
  'password${HOME}' \
  'password#comment' \
  'password\backslash' \
  'password\' \
  'password"quote' \
  "password'quote" \
  'mix $HOME # "double" '\''single'\'' \\ tail'; do
  printf 'ADMIN_PASSWORD=placeholder\n' > "$test_dir/.env"
  (
    cd "$test_dir"
    source <(head -n -1 "$script")
    set_env_value ADMIN_PASSWORD "$password"
    compose_config --env-file .env -f compose.yml run --rm --no-deps probe sh -c 'printf %s "$ADMIN_PASSWORD"' > actual
    printf '%s' "$password" > expected
    cmp -s expected actual
  ) || {
    echo "ADMIN_PASSWORD did not round-trip through Compose: $password" >&2
    exit 1
  }
done

printf 'docker deploy secret handling checks passed\n'

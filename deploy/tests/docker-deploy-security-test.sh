#!/bin/bash

set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
script="$repo_root/deploy/docker-deploy.sh"

bash -n "$script"

if grep -Eq 'echo .*\$\{?(POSTGRES_PASSWORD|JWT_SECRET|TOTP_ENCRYPTION_KEY)\}?' "$script"; then
  echo "docker deploy script prints generated secrets" >&2
  exit 1
fi

if grep -Eq 'admin/admin|ADMIN_PASSWORD|ADMIN_EMAIL' "$script"; then
  echo "docker deploy script must not provision a default administrator" >&2
  exit 1
fi
grep -Fq 'Defers administrator creation to the first-run owner setup page' "$script"
grep -Fq 'Infrastructure secrets and a one-time setup token were written' "$script"
grep -Fq 'deploy-checksums.txt' "$script"
grep -Fq 'verify_deployment_asset' "$script"
if grep -Fq 'raw.githubusercontent.com/yuchen0x1/sub3api-relay/main' "$script"; then
  echo "docker deploy script downloads mutable main-branch deployment files" >&2
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
  echo "Docker Compose is required for setup-token round-trip checks" >&2
  return 1
}

test_dir=$(mktemp -d "$repo_root/.docker-deploy-token-test.XXXXXX")
trap 'rm -rf "$test_dir"' EXIT
cat > "$test_dir/compose.yml" <<'YAML'
services:
  probe:
    image: alpine:3.20
    environment:
      SETUP_REMOTE_TOKEN: ${SETUP_REMOTE_TOKEN}
YAML

for token in \
  'password with spaces' \
  'password$HOME' \
  'password${HOME}' \
  'password#comment' \
  'password\backslash' \
  'password\' \
  'password"quote' \
  "password'quote" \
  'mix $HOME # "double" '\''single'\'' \\ tail'; do
  printf 'SETUP_REMOTE_TOKEN=placeholder\n' > "$test_dir/.env"
  (
    cd "$test_dir"
    source <(head -n -1 "$script")
    set_env_value SETUP_REMOTE_TOKEN "$token"
    unset SETUP_REMOTE_TOKEN
    compose_config --env-file .env -f compose.yml run --rm --no-deps probe sh -c 'printf %s "$SETUP_REMOTE_TOKEN"' > actual
    printf '%s' "$token" > expected
    cmp -s expected actual
  ) || {
    echo "SETUP_REMOTE_TOKEN did not round-trip through Compose: $token" >&2
    exit 1
  }
done

printf 'docker deploy secret handling checks passed\n'

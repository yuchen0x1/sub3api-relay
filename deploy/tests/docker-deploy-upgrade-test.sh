#!/bin/bash

set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
script="$repo_root/deploy/docker-deploy.sh"
test_dir=$(mktemp -d)
trap 'rm -rf -- "$test_dir"' EXIT
mkdir -p "$test_dir/bundle" "$test_dir/fresh" "$test_dir/existing" "$test_dir/missing" "$test_dir/spaced" "$test_dir/empty"
cp "$repo_root/deploy/docker-compose.local.yml" "$test_dir/bundle/"
cp "$repo_root/deploy/.env.example" "$test_dir/bundle/sub3api.env.example"
(
  cd "$test_dir/bundle"
  sha256sum docker-compose.local.yml sub3api.env.example > deploy-checksums.txt
  tar -czf "$test_dir/sub3api_9.9.9_deploy.tar.gz" .
)
(
  cd "$test_dir"
  sha256sum sub3api_9.9.9_deploy.tar.gz > checksums.txt
)

prepare_deployment() (
  cd "$test_dir/$1"
  source <(head -n -1 "$script")
  resolve_release_tag() { printf 'v9.9.9'; }
  generate_secret() { printf 'generated-fixture-secret'; }
  download_file() { cp "$test_dir/${1##*/}" "$2"; }
  main > preparation.log
  if grep -Eq 'generated-fixture-secret|existing-(jwt|totp|database|setup)' preparation.log; then
    echo "docker deployment preparation printed infrastructure credentials" >&2
    exit 1
  fi
)

prepare_deployment fresh
for key in JWT_SECRET TOTP_ENCRYPTION_KEY POSTGRES_PASSWORD SETUP_REMOTE_TOKEN; do
  grep -Fqx "$key=\"generated-fixture-secret\"" "$test_dir/fresh/.env"
done
grep -Fqx 'SUB3API_IMAGE="ghcr.io/yuchen0x1/sub3api-relay:9.9.9"' "$test_dir/fresh/.env"
grep -Fqx 'AUTO_SETUP_OWNER_SETUP="true"' "$test_dir/fresh/.env"
grep -Fqx 'SETUP_ALLOW_REMOTE="true"' "$test_dir/fresh/.env"

printf '%s\n' \
  'JWT_SECRET="existing-jwt"' \
  'TOTP_ENCRYPTION_KEY="existing-totp"' \
  'POSTGRES_PASSWORD="existing-database"' \
  'SETUP_REMOTE_TOKEN="existing-setup"' \
  'SETUP_ALLOW_REMOTE=false' \
  'CUSTOM_CONFIGURATION=preserved' > "$test_dir/existing/.env"
cp "$test_dir/existing/.env" "$test_dir/existing/expected.env"
prepare_deployment existing
while IFS= read -r line; do
  grep -Fqx "$line" "$test_dir/existing/.env"
done < "$test_dir/existing/expected.env"
grep -Fqx 'SUB3API_IMAGE="ghcr.io/yuchen0x1/sub3api-relay:9.9.9"' "$test_dir/existing/.env"

printf 'CUSTOM_CONFIGURATION=preserved\n' > "$test_dir/missing/.env"
prepare_deployment missing
for key in JWT_SECRET TOTP_ENCRYPTION_KEY POSTGRES_PASSWORD SETUP_REMOTE_TOKEN; do
  grep -Fqx "$key=\"generated-fixture-secret\"" "$test_dir/missing/.env"
done
grep -Fqx 'CUSTOM_CONFIGURATION=preserved' "$test_dir/missing/.env"
grep -Fqx 'SUB3API_IMAGE="ghcr.io/yuchen0x1/sub3api-relay:9.9.9"' "$test_dir/missing/.env"

printf '%s\n' \
  ' export JWT_SECRET = "existing-jwt"' \
  'TOTP_ENCRYPTION_KEY = "existing-totp"' \
  ' POSTGRES_PASSWORD=existing-database' \
  'export SETUP_REMOTE_TOKEN="existing-setup"' \
  ' SUB3API_IMAGE = old-image' > "$test_dir/spaced/.env"
prepare_deployment spaced
for key in JWT_SECRET TOTP_ENCRYPTION_KEY POSTGRES_PASSWORD SETUP_REMOTE_TOKEN; do
  if grep "$key" "$test_dir/spaced/.env" | grep -Fq 'generated-fixture-secret'; then
    echo "docker deployment rotated existing credentials with whitespace or export syntax" >&2
    exit 1
  fi
done
grep -Fqx 'SUB3API_IMAGE="ghcr.io/yuchen0x1/sub3api-relay:9.9.9"' "$test_dir/spaced/.env"

printf '%s\n' 'JWT_SECRET=""' "TOTP_ENCRYPTION_KEY=''" 'POSTGRES_PASSWORD=   ' 'SETUP_REMOTE_TOKEN= # not configured' > "$test_dir/empty/.env"
prepare_deployment empty
for key in JWT_SECRET TOTP_ENCRYPTION_KEY POSTGRES_PASSWORD SETUP_REMOTE_TOKEN; do
  grep -Fqx "$key=\"generated-fixture-secret\"" "$test_dir/empty/.env"
done

printf 'docker deployment fresh-install and upgrade checks passed\n'

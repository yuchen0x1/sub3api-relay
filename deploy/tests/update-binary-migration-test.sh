#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
work=$(mktemp -d)
trap 'status=$?; if [[ "$status" != 0 && -f "$work/output" ]]; then cat "$work/output" >&2; fi; rm -rf -- "$work"' EXIT
mkdir -p "$work/commands" "$work/assets" "$work/stage"

cat > "$work/commands/curl" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
destination=""
while [[ $# -gt 0 ]]; do
    case "$1" in
        -o) destination=$2; shift 2 ;;
        --connect-timeout|--max-time) shift 2 ;;
        http*) url=$1; shift ;;
        *) shift ;;
    esac
done
if [[ -z "$destination" ]]; then exit 1; fi
cp "$FIXTURE_ASSETS/${url##*/}" "$destination"
SH
cat > "$work/commands/systemctl" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
case "$1" in
    stop) printf 'stopped' > "$FIXTURE_STATE" ;;
    start) printf 'started' > "$FIXTURE_STATE" ;;
    show) printf '\n' ;;
    *) exit 1 ;;
esac
SH
printf '#!/usr/bin/env bash\nexit 0\n' > "$work/commands/sleep"
printf '#!/usr/bin/env bash\nexit 0\n' > "$work/commands/flock"
printf '#!/usr/bin/env bash\nprintf "x86_64\\n"\n' > "$work/commands/uname"
chmod +x "$work/commands/"*

make_binary() {
    printf '#!/usr/bin/env bash\nprintf "Sub3API %s (fixture)\\n"\n' "$1" > "$2"
    chmod +x "$2"
}

for version in 0.5.6 0.5.7 0.5.8; do
    make_binary "$version" "$work/stage/sub3api"
    tar -C "$work/stage" -czf "$work/assets/sub3api_${version}_linux_amd64.tar.gz" sub3api
done
(cd "$work/assets" && sha256sum --text sub3api_*.tar.gz > checksums.txt)

run_update() {
    env PATH="$work/commands:$PATH" SUB3API_ALLOW_NON_ROOT=1 \
        SUB3API_BIN="$work/installation/sub3api" \
        SUB3API_BACKUP_DIR="$work/installation/backups" \
        SUB3API_HEALTH_URL=http://fixture.invalid/health \
        SUB3API_DATABASE_BACKUP_CONFIRMED="$2" \
        FIXTURE_ASSETS="$work/assets" FIXTURE_STATE="$work/state" \
        bash "$repo_root/deploy/update-binary.sh" "$1" > "$work/output" 2>&1
}

reset_installation() {
    rm -rf -- "$work/installation"
    mkdir -p "$work/installation"
    make_binary "$1" "$work/installation/sub3api"
    printf 'unchanged' > "$work/state"
}

reset_installation 0.5.6
if run_update 0.5.7 0; then
    echo "migration upgrade did not require a confirmed database backup" >&2
    exit 1
fi
grep -Fq 'SUB3API_DATABASE_BACKUP_CONFIRMED=1' "$work/output"
[[ "$(cat "$work/state")" == unchanged ]]
"$work/installation/sub3api" -version | grep -Fq '0.5.6'

if run_update 0.5.7 1; then
    echo "failed migration health check should stop the update" >&2
    exit 1
fi
grep -Fq 'Service remains stopped' "$work/output"
[[ "$(cat "$work/state")" == stopped ]]
"$work/installation/sub3api" -version | grep -Fq '0.5.7'
[[ -n "$(find "$work/installation/backups" -type f -name 'sub3api-0.5.6-to-0.5.7-*' -print -quit)" ]]

reset_installation 0.5.7
if run_update 0.5.6 1; then
    echo "binary-only downgrade crossed the credential database boundary" >&2
    exit 1
fi
grep -Fq 'requires restoring the pre-upgrade database' "$work/output"
[[ "$(cat "$work/state")" == unchanged ]]
"$work/installation/sub3api" -version | grep -Fq '0.5.7'

reset_installation 0.5.7
if run_update 0.5.8 0; then
    echo "same-generation failed health check should roll back" >&2
    exit 1
fi
grep -Fq 'restoring previous binary' "$work/output"
[[ "$(cat "$work/state")" == started ]]
"$work/installation/sub3api" -version | grep -Fq '0.5.7'

printf 'binary updater database migration boundary checks passed\n'

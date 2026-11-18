#!/usr/bin/env bash
# A disposable Linux/systemd installation upgrades from a published release.
set -euo pipefail

assets=$(realpath "${1:?release asset directory}")
version=${2:?target version}
prior=0.4.5
work=$(mktemp -d)
unit="sub3api-upgrade-test-$$"
artifact="sub3api_${prior}_linux_amd64.tar.gz"
cleanup() {
    sudo systemctl stop "$unit" >/dev/null 2>&1 || true
    if [[ -n "${http_pid:-}" ]]; then kill "$http_pid" 2>/dev/null || true; fi
    sudo unlink "/run/systemd/system/$unit.service" 2>/dev/null || true
    sudo systemctl daemon-reload
    sudo rm -rf -- "$work"
}
trap cleanup EXIT
mkdir -p "$work/old" "$work/repository/releases" "$work/data"
gh release download "v$prior" -R yuchen0x1/sub3api-relay --dir "$work/old" -p "$artifact" -p checksums.txt
(cd "$work/old" && grep -F "  $artifact" checksums.txt | sha256sum --check --strict -)
tar -xzf "$work/old/$artifact" -C "$work"
ln -s "$assets" "$work/repository/v$version"
printf '{"tag_name":"v%s"}\n' "$version" > "$work/repository/releases/latest"
python3 -m http.server 18172 --bind 127.0.0.1 --directory "$work/repository" >"$work/http.log" 2>&1 &
http_pid=$!
sudo tee "/run/systemd/system/$unit.service" >/dev/null <<UNIT
[Service]
User=$(id -un)
WorkingDirectory=$work
Environment=SERVER_HOST=127.0.0.1 SERVER_PORT=18171 DATA_DIR=$work/data
ExecStart=$work/sub3api
UNIT
sudo systemctl daemon-reload
sudo systemctl start "$unit"
health=http://127.0.0.1:18171/health
curl -fsS --retry 10 --retry-connrefused "$health"
update() {
    sudo env SUB3API_BIN="$work/sub3api" SUB3API_SERVICE="$unit" SUB3API_BACKUP_DIR="$work/backups" \
        SUB3API_RELEASE_API_URL=http://127.0.0.1:18172 \
        SUB3API_RELEASE_DOWNLOAD_URL=http://127.0.0.1:18172 SUB3API_HEALTH_URL="$1" \
        bash deploy/update-binary.sh
}
if update "http://127.0.0.1:1"; then echo "Failed health check should roll back" >&2; exit 1; fi
"$work/sub3api" -version 2>&1 | grep -F "Sub3API $prior ("
curl -fsS --retry 10 --retry-connrefused "$health"
update "$health"
"$work/sub3api" -version 2>&1 | grep -F "Sub3API $version ("
curl -fsS "$health"
update "$health"

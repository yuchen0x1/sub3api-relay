#!/usr/bin/env bash
set -euo pipefail

# =============================================================================
# Sub3API binary upgrade script (v0.1.178+)
#
# Release binaries embed the frontend, so upgrading is:
#   download binary -> verify checksum -> backup -> replace -> restart -> health check
#
# Usage:
#   sudo ./update-binary.sh [vX.Y.Z]
#
# Environment overrides:
#   SUB3API_BIN          Path to the running binary (default: /opt/sub3api/sub3api)
#   SUB3API_SERVICE      systemd unit name       (default: sub3api)
#   SUB3API_BACKUP_DIR   Backup directory        (default: <binary dir>/backups)
#   SUB3API_HEALTH_URL   Health check URL        (default: http://127.0.0.1:8080/health)
#   SUB3API_REPO         GitHub repository       (default: yuchen0x1/sub3api-relay)
#   SUB3API_REQUIRE_ATTESTATION  Require GitHub build provenance via gh (v0.5.0+)
#   SUB3API_DATABASE_BACKUP_CONFIRMED  Set to 1 after a pre-v0.5.7 database/config backup
#   SUB3API_FORCE        Set to 1 to reinstall the currently running version
# =============================================================================

SUB3API_BIN="${SUB3API_BIN:-/opt/sub3api/sub3api}"
SUB3API_SERVICE="${SUB3API_SERVICE:-sub3api}"
SUB3API_REPO="${SUB3API_REPO:-yuchen0x1/sub3api-relay}"
SUB3API_RELEASE_API_URL="${SUB3API_RELEASE_API_URL:-https://api.github.com/repos/${SUB3API_REPO}}"
SUB3API_RELEASE_DOWNLOAD_URL="${SUB3API_RELEASE_DOWNLOAD_URL:-https://github.com/${SUB3API_REPO}/releases/download}"
SUB3API_HEALTH_URL="${SUB3API_HEALTH_URL:-http://127.0.0.1:8080/health}"
SUB3API_BACKUP_DIR="${SUB3API_BACKUP_DIR:-$(dirname "$SUB3API_BIN")/backups}"

print_info() { printf '\033[1;34m[INFO]\033[0m %s\n' "$*"; }
print_ok()   { printf '\033[1;32m[OK]\033[0m %s\n' "$*"; }
print_err()  { printf '\033[1;31m[ERROR]\033[0m %s\n' "$*" >&2; }

uses_legacy_credential_digests() {
    [[ "$1" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || return 1
    printf '%s\n' "$1" | awk -F. '{ exit !(($1 + 0) == 0 && (($2 + 0) < 5 || (($2 + 0) == 5 && ($3 + 0) < 7))) }'
}

if [ "$(id -u)" -ne 0 ] && [ "${SUB3API_ALLOW_NON_ROOT:-}" != "1" ]; then
    print_err "Run as root (or set SUB3API_ALLOW_NON_ROOT=1 for custom layouts)."
    exit 1
fi

for cmd in curl tar sha256sum systemctl flock; do
    if ! command -v "$cmd" >/dev/null 2>&1; then
        print_err "Required command not found: $cmd"
        exit 1
    fi
done

if [[ "${SUB3API_REQUIRE_ATTESTATION:-0}" == 1 ]] && ! command -v gh >/dev/null 2>&1; then
    print_err "SUB3API_REQUIRE_ATTESTATION=1 requires the GitHub CLI (gh)."
    exit 1
fi

resolve_version() {
    if [ -n "${1:-}" ]; then
        printf '%s\n' "$1"
        return
    fi
    print_info "Fetching latest release version..." >&2
    curl -fsSL --connect-timeout 10 --max-time 30 \
        "${SUB3API_RELEASE_API_URL}/releases/latest" |
        sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' |
        head -1
}

normalize_version() {
    printf '%s\n' "${1#v}"
}

detect_goarch() {
    local machine
    machine="$(uname -m)"
    case "$machine" in
        x86_64|amd64) printf 'amd64\n' ;;
        aarch64|arm64) printf 'arm64\n' ;;
        *)
            print_err "Unsupported architecture: $machine"
            exit 1
            ;;
    esac
}

get_current_version() {
    if [ ! -x "$SUB3API_BIN" ]; then
        printf 'not_installed\n'
        return
    fi
    "$SUB3API_BIN" -version 2>&1 |
        grep -oE '[0-9]+\.[0-9]+\.[0-9]+' |
        head -1 || true
}

download_release() {
    local version="$1"
    local arch="$2"
    local asset="sub3api_${version}_linux_${arch}.tar.gz"
    local base="${SUB3API_RELEASE_DOWNLOAD_URL}/v${version}"
    local expected actual

    print_info "Downloading ${asset} ..."
    curl -fsSL --connect-timeout 10 --max-time 300 -o "${TMP_DIR}/${asset}" "${base}/${asset}"

    print_info "Downloading checksums.txt ..."
    curl -fsSL --connect-timeout 10 --max-time 60 -o "${TMP_DIR}/checksums.txt" "${base}/checksums.txt"

    expected="$(awk -v name="${asset}" '$2 == name { print $1 }' "${TMP_DIR}/checksums.txt" | head -1)"
    if [ -z "$expected" ]; then
        print_err "No checksum found for ${asset}"
        exit 1
    fi

    actual="$(sha256sum "${TMP_DIR}/${asset}" | awk '{print $1}')"
    if [ "$expected" != "$actual" ]; then
        print_err "Checksum mismatch for ${asset}"
        print_err "Expected: ${expected}"
        print_err "Actual:   ${actual}"
        exit 1
    fi
    print_ok "Checksum verified"
    if [[ "${SUB3API_REQUIRE_ATTESTATION:-0}" == 1 ]]; then
        gh attestation verify "${TMP_DIR}/${asset}" --repo "$SUB3API_REPO" \
            --signer-workflow "$SUB3API_REPO/.github/workflows/release.yml" --source-ref "refs/tags/v${version}"
    fi

    tar -xzf "${TMP_DIR}/${asset}" -C "${TMP_DIR}"
    if [ ! -f "${TMP_DIR}/sub3api" ]; then
        print_err "Binary not found in release archive"
        exit 1
    fi
    chmod +x "${TMP_DIR}/sub3api"
    local reported
    reported="$("${TMP_DIR}/sub3api" -version 2>&1)"
    if [[ "$reported" != *"Sub3API ${version} ("* ]]; then
        print_err "Archive executable version does not match the release tag"
        exit 1
    fi
}

wait_health() {
    local attempt
    for attempt in 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20 21 22 23 24 25 26 27 28 29 30; do
        if curl -fsS --connect-timeout 3 --max-time 5 "$SUB3API_HEALTH_URL" >/dev/null 2>&1; then
            return 0
        fi
        sleep 1
    done
    return 1
}

restore_binary() {
    local backup="$1"
    local user
    cp -a "$backup" "$STAGED_BIN"
    mv -f "$STAGED_BIN" "$SUB3API_BIN"
    chmod +x "$SUB3API_BIN"
    user="$(systemctl show -p User --value "$SUB3API_SERVICE" 2>/dev/null || true)"
    if [ -n "$user" ] && [ "$user" != "root" ]; then
        chown "$user" "$SUB3API_BIN" 2>/dev/null || true
    fi
}

recover_after_start_failure() {
    local reason="$1"
    systemctl stop "$SUB3API_SERVICE" || true
    if [[ "$ROLLBACK_REQUIRES_DATABASE_RESTORE" == 1 ]]; then
        print_err "$reason; automatic binary rollback is disabled across the v0.5.7 database migration."
        print_err "Service remains stopped. Restore the pre-upgrade database and configuration together with binary backup: $BACKUP_FILE"
        return
    fi
    print_err "$reason; restoring previous binary"
    restore_binary "$BACKUP_FILE"
    systemctl start "$SUB3API_SERVICE" || true
}

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

TAG_VERSION="$(resolve_version "${1:-}")"
if [ -z "$TAG_VERSION" ]; then
    print_err "Could not determine target version"
    exit 1
fi
VERSION="$(normalize_version "$TAG_VERSION")"
if [[ ! "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    print_err "Expected a stable semantic version"
    exit 1
fi
GOARCH="$(detect_goarch)"
CURRENT_VERSION="$(get_current_version)"

if [ "$CURRENT_VERSION" = "not_installed" ]; then
    print_err "$SUB3API_BIN does not exist; use deploy/install.sh for a fresh install."
    exit 1
fi

if [ -z "$CURRENT_VERSION" ]; then
    CURRENT_VERSION="unknown"
fi

if [ "$CURRENT_VERSION" = "$VERSION" ] && [ "${SUB3API_FORCE:-}" != "1" ]; then
    print_ok "Already running ${VERSION}; no upgrade needed (set SUB3API_FORCE=1 to reinstall)."
    exit 0
fi

print_info "Current version: ${CURRENT_VERSION:-unknown}"
print_info "Target version:  ${VERSION}"

# Serialize replacements, including rollback, on the same installation.
exec 9>"${SUB3API_BIN}.update.lock"
flock -n 9 || { print_err "Another update is running"; exit 1; }
CURRENT_VERSION="$(get_current_version)"
if [[ "$CURRENT_VERSION" == "$VERSION" && "${SUB3API_FORCE:-}" != 1 ]]; then exit 0; fi
ROLLBACK_REQUIRES_DATABASE_RESTORE=0
if uses_legacy_credential_digests "$VERSION"; then
    if ! uses_legacy_credential_digests "$CURRENT_VERSION"; then
        print_err "Downgrading before v0.5.7 requires restoring the pre-upgrade database and configuration manually."
        exit 1
    fi
elif ! [[ "$CURRENT_VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || uses_legacy_credential_digests "$CURRENT_VERSION"; then
    ROLLBACK_REQUIRES_DATABASE_RESTORE=1
    if [[ "${SUB3API_DATABASE_BACKUP_CONFIRMED:-0}" != 1 ]]; then
        print_err "Upgrading across v0.5.7 changes credential digests. Stop other instances and back up the database and configuration first."
        print_err "Then rerun with SUB3API_DATABASE_BACKUP_CONFIRMED=1. Failed startup will leave the service stopped for a complete restore."
        exit 1
    fi
fi
download_release "$VERSION" "$GOARCH"
mkdir -p "$SUB3API_BACKUP_DIR"
BACKUP_FILE="${SUB3API_BACKUP_DIR}/sub3api-${CURRENT_VERSION}-to-${VERSION}-$(date +%Y%m%d-%H%M%S)"
cp -a "$SUB3API_BIN" "$BACKUP_FILE"
print_ok "Backup created: ${BACKUP_FILE}"

# Prepare a replacement on the same filesystem before stopping the service.
STAGED_BIN="$(mktemp "${SUB3API_BIN}.new.XXXXXX")"
trap 'rm -rf "$TMP_DIR"; rm -f "$STAGED_BIN"' EXIT
cp --preserve=mode,ownership "$SUB3API_BIN" "$STAGED_BIN"
cat "$TMP_DIR/sub3api" > "$STAGED_BIN"

print_info "Stopping ${SUB3API_SERVICE} ..."
systemctl stop "$SUB3API_SERVICE"

if ! mv -f "$STAGED_BIN" "$SUB3API_BIN"; then
    print_err "Failed to install new binary"
    systemctl start "$SUB3API_SERVICE" || true
    exit 1
fi

print_info "Starting ${SUB3API_SERVICE} ..."
if ! systemctl start "$SUB3API_SERVICE"; then
    recover_after_start_failure "Service failed to start"
    exit 1
fi

if ! wait_health; then
    recover_after_start_failure "Health check failed at ${SUB3API_HEALTH_URL}"
    exit 1
fi

NEW_VERSION="$(get_current_version)"
print_ok "Upgraded to ${NEW_VERSION:-${VERSION}} (${SUB3API_SERVICE} active)"

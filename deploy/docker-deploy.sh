#!/bin/bash
# =============================================================================
# Sub3API Docker Deployment Preparation Script
# =============================================================================
# This script prepares deployment files for Sub3API:
#   - Downloads a version-pinned deployment bundle from GitHub Releases
#   - Verifies the deployment bundle with SHA-256 before writing files
#   - Tag-triggered CI publishes the verified bundle to GitHub Releases
#   - Generates secure secrets (JWT_SECRET, TOTP_ENCRYPTION_KEY, POSTGRES_PASSWORD,
#     SETUP_REMOTE_TOKEN)
#   - Defers administrator creation to the first-run owner setup page
#   - Creates necessary data directories
#
# After running this script, you can start services with:
#   docker-compose up -d
# =============================================================================

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

GITHUB_REPO="yuchen0x1/sub3api-relay"
GITHUB_API_URL="https://api.github.com/repos/${GITHUB_REPO}"
GITHUB_RELEASE_URL="https://github.com/${GITHUB_REPO}/releases/download"
SUB3API_VERSION="${SUB3API_VERSION:-latest}"

# Print colored message
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Generate random secret
generate_secret() {
    openssl rand -hex 32
}

dotenv_quote() {
    local value="$1"
    value=${value//\\/\\\\}
    value=${value//\"/\\\"}
    value=${value//\$/\$\$}
    printf '"%s"' "$value"
}

set_env_value() {
    local key="$1"
    local value="$2"
    local encoded_value
    local tmp_file
    local found=0
    local assignment="^[[:space:]]*(export[[:space:]]+)?${key}[[:space:]]*="
    encoded_value=$(dotenv_quote "$value")
    tmp_file=$(mktemp .env.XXXXXX)
    while IFS= read -r line || [ -n "$line" ]; do
        if [[ "$line" =~ $assignment ]]; then
            printf '%s=%s\n' "$key" "$encoded_value" >> "$tmp_file"
            found=1
        else
            printf '%s\n' "$line" >> "$tmp_file"
        fi
    done < .env
    if [ "$found" -eq 0 ]; then
        printf '%s=%s\n' "$key" "$encoded_value" >> "$tmp_file"
    fi
    mv "$tmp_file" .env
}

env_value_is_set() {
    awk -v key="$1" '
        $0 ~ "^[[:space:]]*(export[[:space:]]+)?" key "[[:space:]]*=" {
            value = $0
            sub(/^[^=]*=[[:space:]]*/, "", value)
            sub(/[[:space:]]+$/, "", value)
        }
        END { exit !(value != "" && value != "\"\"" && value != "\047\047" && value !~ /^#/) }
    ' .env
}

# Check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

download_file() {
    local url="$1"
    local destination="$2"
    if command_exists curl; then
        curl -fsSL "$url" -o "$destination"
    elif command_exists wget; then
        wget -q "$url" -O "$destination"
    else
        print_error "Neither curl nor wget is installed. Please install one of them."
        exit 1
    fi
}

resolve_release_tag() {
    if [ "$SUB3API_VERSION" != "latest" ]; then
        if [[ "$SUB3API_VERSION" =~ ^v?[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
            printf 'v%s' "${SUB3API_VERSION#v}"
            return
        fi
        print_error "SUB3API_VERSION must be latest or a semantic version such as v0.1.182."
        exit 1
    fi

    local release_json
    local tag
    release_json=$(mktemp)
    download_file "${GITHUB_API_URL}/releases/latest" "$release_json"
    tag=$(sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$release_json" | head -n 1)
    rm -f "$release_json"
    if [[ "$tag" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        printf '%s' "$tag"
        return
    fi
    print_error "Unable to resolve the latest stable release tag."
    exit 1
}

verify_deployment_asset() {
    local directory="$1"
    local filename="$2"
    local expected
    local actual
    expected=$(awk -v filename="$filename" '$2 == filename || $2 == "*" filename { print $1; exit }' "$directory/deploy-checksums.txt")
    actual=$(sha256sum "$directory/$filename" | awk '{print $1}')
    if ! printf '%s' "$expected" | grep -Eq '^[0-9a-fA-F]{64}$'; then
        print_error "Missing SHA-256 checksum for $filename."
        exit 1
    fi
    if [ "$expected" != "$actual" ]; then
        print_error "SHA-256 verification failed for $filename."
        exit 1
    fi
}

verify_release_archive() {
    local directory="$1"
    local filename="$2"
    local expected
    local actual
    expected=$(awk -v filename="$filename" '$2 == filename || $2 == "*" filename { print $1; exit }' "$directory/checksums.txt")
    if ! printf '%s' "$expected" | grep -Eq '^[0-9a-fA-F]{64}$'; then
        print_error "Missing release SHA-256 checksum for $filename."
        exit 1
    fi
    actual=$(sha256sum "$directory/$filename" | awk '{print $1}')
    if [ "$expected" != "$actual" ]; then
        print_error "Release SHA-256 verification failed for $filename."
        exit 1
    fi
}

# Main installation function
main() {
    echo ""
    echo "=========================================="
    echo "  Sub3API Deployment Preparation"
    echo "=========================================="
    echo ""

    for dependency in openssl sha256sum awk sed; do
        if ! command_exists "$dependency"; then
            print_error "$dependency is not installed. Please install it first."
            exit 1
        fi
    done

    # Check if deployment already exists
    if [ -f "docker-compose.yml" ] && [ -f ".env" ]; then
        print_warning "Deployment files already exist in current directory."
        read -p "Overwrite existing files? (y/N): " -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            print_info "Cancelled."
            exit 0
        fi
    fi

    local release_tag
    local release_base
    local download_dir
    local existing_env
    local deploy_archive
    release_tag=$(resolve_release_tag)
    release_base="${GITHUB_RELEASE_URL}/${release_tag}"
    download_dir=$(mktemp -d)
    trap "rm -rf -- '$download_dir'" EXIT
    existing_env=""
    if [ -f .env ]; then
        existing_env="$download_dir/existing.env"
        cp .env "$existing_env"
    fi

    print_info "Downloading verified deployment bundle for ${release_tag}..."
    deploy_archive="sub3api_${release_tag#v}_deploy.tar.gz"
    download_file "${release_base}/${deploy_archive}" "$download_dir/${deploy_archive}"
    download_file "${release_base}/checksums.txt" "$download_dir/checksums.txt"
    verify_release_archive "$download_dir" "$deploy_archive"
    mv "$download_dir/${deploy_archive}" "$download_dir/deploy.tar.gz"
    tar -xzf "$download_dir/deploy.tar.gz" -C "$download_dir"
    verify_deployment_asset "$download_dir" docker-compose.local.yml
    verify_deployment_asset "$download_dir" sub3api.env.example
    mv "$download_dir/docker-compose.local.yml" docker-compose.yml
    mv "$download_dir/sub3api.env.example" .env.example
    print_success "Downloaded and verified deployment bundle for ${release_tag}"

    # Generate .env file with auto-generated secrets. Preserve existing
    # infrastructure credentials on upgrades so the database remains usable.
    print_info "Generating secure secrets..."
    echo ""

    # Create .env from .env.example
    if [ -n "$existing_env" ]; then
        cp "$existing_env" .env
    else
        cp .env.example .env
    fi

    # Fresh installs must replace template placeholders as well as empty values.
    # Existing deployments retain credentials that their persisted data uses.
    if [ -z "$existing_env" ] || ! env_value_is_set JWT_SECRET; then set_env_value JWT_SECRET "$(generate_secret)"; fi
    if [ -z "$existing_env" ] || ! env_value_is_set TOTP_ENCRYPTION_KEY; then set_env_value TOTP_ENCRYPTION_KEY "$(generate_secret)"; fi
    if [ -z "$existing_env" ] || ! env_value_is_set POSTGRES_PASSWORD; then set_env_value POSTGRES_PASSWORD "$(generate_secret)"; fi
    if [ -z "$existing_env" ] || ! env_value_is_set SETUP_REMOTE_TOKEN; then set_env_value SETUP_REMOTE_TOKEN "$(generate_secret)"; fi
    if [ -z "$existing_env" ]; then
        set_env_value AUTO_SETUP_OWNER_SETUP "true"
        set_env_value SETUP_ALLOW_REMOTE "true"
    fi
    set_env_value SUB3API_IMAGE "ghcr.io/yuchen0x1/sub3api-relay:${release_tag#v}"

    # Create data directories
    print_info "Creating data directories..."
    mkdir -p data postgres_data redis_data
    print_success "Created data directories"

    # Set secure permissions for .env file (readable/writable only by owner)
    chmod 600 .env
    echo ""

    # Display completion message
    echo "=========================================="
    echo "  Preparation Complete!"
    echo "=========================================="
    echo ""
    echo "Infrastructure secrets and a one-time setup token were written to the owner-readable .env file."
    echo "  Site-owner account:     choose it on the /setup page"
    echo "  Site-owner password:    choose it on the /setup page"
    echo ""
    print_warning "Infrastructure secrets and the one-time setup token are saved in .env."
    print_warning "Keep .env secure and do not share it publicly!"
    echo ""
    echo "Directory structure:"
    echo "  docker-compose.yml        - Docker Compose configuration"
    echo "  .env                      - Environment variables (generated secrets)"
    echo "  .env.example              - Example template (for reference)"
    echo "  data/                     - Application data (will be created on first run)"
    echo "  postgres_data/            - PostgreSQL data"
    echo "  redis_data/               - Redis data"
    echo ""
    echo "Next steps:"
    echo "  1. (Optional) Edit .env to customize configuration"
    echo "  2. Start services:"
    echo "     docker-compose up -d"
    echo ""
    echo "  3. View logs:"
    echo "     docker-compose logs -f sub3api"
    echo ""
    echo "  4. Access Web UI:"
    echo "     http://localhost:8080"
    echo ""
    print_warning "First start prepares PostgreSQL/Redis and then opens the owner setup page."
    print_warning "Open http://localhost:8080/setup (or your reverse-proxy /setup URL) and choose the site-owner credentials."
    print_warning "Remote /setup writes require the one-time token stored in .env."
    echo "  Read it only when needed: grep '^SETUP_REMOTE_TOKEN=' .env"
    echo "  Then open: https://YOUR_DOMAIN/setup#setup_token=<token>"
    print_warning "Prefer an SSH tunnel when possible, and do not paste the setup link into tickets or chat."
    echo ""
}

# Run main function
main "$@"

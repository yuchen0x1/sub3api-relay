#!/bin/bash
# =============================================================================
# Sub3API Docker Deployment Preparation Script
# =============================================================================
# This script prepares deployment files for Sub3API:
#   - Downloads a version-pinned deployment bundle from GitHub Releases
#   - Verifies the deployment bundle with SHA-256 before writing files
#   - Generates secure secrets (JWT_SECRET, TOTP_ENCRYPTION_KEY, POSTGRES_PASSWORD)
#   - Requires an explicit strong initial administrator password
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

GITHUB_REPO="${GITHUB_REPO:-yuchen0x1/sub3api-relay}"
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

read_admin_password() {
    local password="${ADMIN_PASSWORD:-}"
    local confirmation
    local input_device="/dev/stdin"

    if [ -z "$password" ]; then
        if [ -t 0 ]; then
            input_device="/dev/stdin"
        elif [ -r /dev/tty ] && (: < /dev/tty) 2>/dev/null; then
            input_device="/dev/tty"
        else
            print_error "ADMIN_PASSWORD is required when no terminal is available." >&2
            exit 1
        fi
        while true; do
            read -r -s -p "Initial administrator password (8-128 characters): " password < "$input_device"
            echo >&2
            read -r -s -p "Confirm administrator password: " confirmation < "$input_device"
            echo >&2
            if [ "${#password}" -lt 8 ] || [ "${#password}" -gt 128 ]; then
                print_error "Administrator password must be 8-128 characters." >&2
                continue
            fi
            if [ "$password" != "$confirmation" ]; then
                print_error "Administrator passwords do not match." >&2
                continue
            fi
            break
        done
    fi

    if [ "${#password}" -lt 8 ] || [ "${#password}" -gt 128 ]; then
        print_error "ADMIN_PASSWORD must be 8-128 characters." >&2
        exit 1
    fi
    printf '%s' "$password"
}

set_env_value() {
    local key="$1"
    local value="$2"
    local tmp_file
    tmp_file=$(mktemp .env.XXXXXX)
    while IFS= read -r line || [ -n "$line" ]; do
        case "$line" in
            "${key}="*) printf '%s=%s\n' "$key" "$value" >> "$tmp_file" ;;
            *) printf '%s\n' "$line" >> "$tmp_file" ;;
        esac
    done < .env
    mv "$tmp_file" .env
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
        print_error "SUB3API_VERSION must be latest or a semantic version such as v0.1.181."
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
    release_tag=$(resolve_release_tag)
    release_base="${GITHUB_RELEASE_URL}/${release_tag}"
    download_dir=$(mktemp -d)
    trap "rm -rf -- '$download_dir'" EXIT

    print_info "Downloading verified deployment bundle for ${release_tag}..."
    download_file "${release_base}/docker-compose.yml" "$download_dir/docker-compose.yml"
    download_file "${release_base}/sub3api.env.example" "$download_dir/sub3api.env.example"
    download_file "${release_base}/deploy-checksums.txt" "$download_dir/deploy-checksums.txt"
    verify_deployment_asset "$download_dir" docker-compose.yml
    verify_deployment_asset "$download_dir" sub3api.env.example
    mv "$download_dir/docker-compose.yml" docker-compose.yml
    mv "$download_dir/sub3api.env.example" .env.example
    print_success "Downloaded and verified deployment bundle for ${release_tag}"

    # Generate .env file with auto-generated secrets
    print_info "Generating secure secrets..."
    echo ""

    # Generate secrets
    JWT_SECRET=$(generate_secret)
    TOTP_ENCRYPTION_KEY=$(generate_secret)
    POSTGRES_PASSWORD=$(generate_secret)
    ADMIN_PASSWORD=$(read_admin_password)
    # Create .env from .env.example
    cp .env.example .env

    # Update .env with generated secrets.
    set_env_value JWT_SECRET "$JWT_SECRET"
    set_env_value TOTP_ENCRYPTION_KEY "$TOTP_ENCRYPTION_KEY"
    set_env_value POSTGRES_PASSWORD "$POSTGRES_PASSWORD"
    set_env_value ADMIN_PASSWORD "$ADMIN_PASSWORD"

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
    echo "Generated credentials were written to the owner-readable .env file."
    echo "  ADMIN_EMAIL:           admin"
    echo "  ADMIN_PASSWORD:        saved in .env (not printed)"
    echo ""
    print_warning "These credentials have been saved to .env file."
    print_warning "Please keep them secure and do not share publicly!"
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
    print_warning "Initial administrator login: admin"
    print_warning "The administrator password is stored only in the owner-readable .env file."
    echo ""
}

# Run main function
main "$@"

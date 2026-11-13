# Sub3API Deployment Files

This directory contains the official Sub3API deployment files for Linux servers and Apple-silicon Macs.

## Deployment Methods

| Method | Best For | Setup Wizard |
|--------|----------|--------------|
| **Docker Compose** | Quick setup, all-in-one | Infrastructure auto-setup + owner credentials |
| **Apple container** | Native local stack on macOS 26 | Infrastructure auto-setup + owner credentials |
| **Binary Install** | Production servers, systemd | Web-based wizard |

## Files

| File | Description |
|------|-------------|
| `docker-compose.yml` | Docker Compose configuration (named volumes) |
| `docker-compose.local.yml` | Docker Compose configuration (local directories, easy migration) |
| `docker-deploy.sh` | **One-click Docker deployment script (recommended)** |
| `apple-container.sh` | Native Apple `container` lifecycle script |
| `APPLE_CONTAINER.md` | Apple `container` deployment and operations guide |
| `.env.example` | Container environment variables template |
| `DOCKER.md` | Container image and Docker documentation |
| `install.sh` | One-click binary installation script |
| `update-binary.sh` | One-command binary upgrade script (embedded frontend, backup, restart, health check) |
| `BINARY_UPGRADE.md` | Binary upgrade guide: embedded frontend, Caddy full-site reverse proxy, verify/rollback |
| `install-datamanagementd.sh` | datamanagementd 一键安装脚本 |
| `sub3api.service` | Systemd service unit file |
| `sub3api-datamanagementd.service` | datamanagementd systemd service unit file |
| `DATAMANAGEMENTD_CN.md` | datamanagementd 部署与联动说明（中文） |
| `config.example.yaml` | Example configuration file |
| `EDGE_SECURITY.md` | Reverse proxy, CDN/WAF, trusted proxy, and ingress hardening guide |

---

## Apple container Deployment

Apple-silicon Macs running macOS 26 can run the complete Sub3API, PostgreSQL, and Redis stack with Apple `container` 1.1.0 or newer:

```bash
./apple-container.sh init
./apple-container.sh up
./apple-container.sh status
./apple-container.sh logs app -f
```

The script uses Apple named volumes, starts dependencies in order, and performs live readiness checks. It does not provide a continuous restart supervisor; run `./apple-container.sh up` after a host reboot. Docker Compose remains the recommended production deployment path.

See [APPLE_CONTAINER.md](./APPLE_CONTAINER.md) for configuration, upgrades, persistence, networking behavior, and limitations.

---

## Docker Deployment (Recommended)

### Method 1: One-Click Deployment (Recommended)

Use the automated preparation script for the easiest setup:

```bash
# Download and run the preparation script; it resolves and verifies the release bundle
curl -fsSL https://raw.githubusercontent.com/yuchen0x1/sub3api-relay/main/deploy/docker-deploy.sh | bash

# Or download first, then run
curl -fsSLO https://raw.githubusercontent.com/yuchen0x1/sub3api-relay/main/deploy/docker-deploy.sh
chmod +x docker-deploy.sh
./docker-deploy.sh
```

**What the script does:**
- Resolves a fixed GitHub Release and verifies the deployment bundle with SHA-256
- Downloads the release's `docker-compose.yml` and `sub3api.env.example`
- Automatically generates secure secrets (JWT_SECRET, TOTP_ENCRYPTION_KEY, POSTGRES_PASSWORD, SETUP_REMOTE_TOKEN)
- Does not create a default administrator; the first-run `/setup` page asks the site owner to choose credentials
- Creates `.env` with generated secrets and a release-pinned `SUB3API_IMAGE`
- Creates necessary data directories (data/, postgres_data/, redis_data/)
- Stores infrastructure secrets and the one-time setup token in the owner-readable `.env` file

**After running the script:**
```bash
# Start services
docker compose up -d

# View logs
docker compose -f docker-compose.local.yml logs -f sub3api

# The first startup prepares the database/Redis and keeps the owner setup page open.
# Choose the site-owner account and password at:
# http://localhost:8080/setup

# Access Web UI
# http://localhost:8080
```

### Method 2: Manual Deployment

If you prefer manual control:

```bash
# Clone repository
git clone https://github.com/yuchen0x1/sub3api-relay.git
cd sub3api-relay/deploy

# Configure environment
cp .env.example .env
chmod 600 .env
nano .env  # Set POSTGRES_PASSWORD and other required variables

# Generate secure secrets (recommended)
JWT_SECRET=$(openssl rand -hex 32)
TOTP_ENCRYPTION_KEY=$(openssl rand -hex 32)
echo "JWT_SECRET=${JWT_SECRET}" >> .env
echo "TOTP_ENCRYPTION_KEY=${TOTP_ENCRYPTION_KEY}" >> .env

# Create data directories
mkdir -p data postgres_data redis_data

# Start all services using local directory version
docker compose -f docker-compose.local.yml up -d

# View logs
docker compose logs -f sub3api

# Access Web UI
# http://localhost:8080
```

### Deployment Version Comparison

| Version | Data Storage | Migration | Best For |
|---------|-------------|-----------|----------|
| **docker-compose.local.yml** | Local directories (./data, ./postgres_data, ./redis_data) | ✅ Easy (tar entire directory) | Production, need frequent backups/migration |
| **docker-compose.yml** | Named volumes (/var/lib/docker/volumes/) | ⚠️ Requires docker commands | Simple setup, don't need migration |

**Recommendation:** Use `docker-compose.local.yml` (deployed by `docker-deploy.sh`) for easier data management and migration.

### How Auto-Setup Works

When using Docker Compose with `AUTO_SETUP=true` and `AUTO_SETUP_OWNER_SETUP=true`:

1. On first run, the system automatically:
   - Connects to PostgreSQL and Redis
   - Applies database migrations (SQL files in `backend/migrations/*.sql`) and records them in `schema_migrations`
   - Generates JWT secret (if not provided)
   - Writes config.yaml
   - Creates a pending owner-setup marker; no administrator password is generated

2. Open `/setup` and choose the single site-owner login account and password.

3. After submission the service restarts and `/login` becomes available. Only the administrator can enter the backend; public registration remains disabled.
   If the database already contains users but no administrator, the owner setup adds one without changing existing users.

### Database Migration Notes (PostgreSQL)

- Migrations are applied in lexicographic order (e.g. `001_...sql`, `002_...sql`).
- `schema_migrations` tracks applied migrations (filename + checksum).
- Migrations are forward-only; rollback requires a DB backup restore or a manual compensating SQL script.

**Verify `users.allowed_groups` → `user_allowed_groups` backfill**

During the incremental GORM→Ent migration, `users.allowed_groups` (legacy `BIGINT[]`) is being replaced by a normalized join table `user_allowed_groups(user_id, group_id)`.

Run this query to compare the legacy data vs the join table:

```sql
WITH old_pairs AS (
  SELECT DISTINCT u.id AS user_id, x.group_id
  FROM users u
  CROSS JOIN LATERAL unnest(u.allowed_groups) AS x(group_id)
  WHERE u.allowed_groups IS NOT NULL
)
SELECT
  (SELECT COUNT(*) FROM old_pairs)           AS old_pair_count,
  (SELECT COUNT(*) FROM user_allowed_groups) AS new_pair_count;
```

### datamanagementd（数据管理）联动

如需启用管理后台“数据管理”功能，请额外部署宿主机 `datamanagementd`：

- 主进程固定探测 `/tmp/sub3api-datamanagement.sock`
- Docker 场景下需把宿主机 Socket 挂载到容器内同路径
- 详细步骤见：`deploy/DATAMANAGEMENTD_CN.md`

### Commands

For **local directory version** (docker-compose.local.yml):

```bash
# Start services
docker compose -f docker-compose.local.yml up -d

# Stop services
docker compose -f docker-compose.local.yml down

# View logs
docker compose -f docker-compose.local.yml logs -f sub3api

# Restart Sub3API only
docker compose -f docker-compose.local.yml restart sub3api

# Update to a specific release
sed -i.bak 's|^SUB3API_IMAGE=.*|SUB3API_IMAGE=ghcr.io/yuchen0x1/sub3api-relay:0.1.187|' .env
docker compose pull sub3api
docker compose up -d --no-deps sub3api

# Remove all data (caution!)
docker compose -f docker-compose.local.yml down
rm -rf data/ postgres_data/ redis_data/
```

For **named volumes version** (docker-compose.yml):

```bash
# Start services
docker compose up -d

# Stop services
docker compose down

# View logs
docker compose logs -f sub3api

# Restart Sub3API only
docker compose restart sub3api

# Update to a specific release
sed -i.bak 's|^SUB3API_IMAGE=.*|SUB3API_IMAGE=ghcr.io/yuchen0x1/sub3api-relay:0.1.187|' .env
docker compose pull sub3api
docker compose up -d --no-deps sub3api

# Remove all data (caution!)
docker compose down -v
```

### Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `POSTGRES_PASSWORD` | **Yes** | - | PostgreSQL password |
| `JWT_SECRET` | **Recommended** | *(auto-generated)* | JWT secret (fixed for persistent sessions) |
| `TOTP_ENCRYPTION_KEY` | **Recommended** | *(auto-generated)* | TOTP encryption key (fixed for persistent 2FA) |
| `SERVER_PORT` | No | `8080` | Server port |
| `AUTO_SETUP_OWNER_SETUP` | No | `true` in Docker Compose | Defer administrator creation to the web owner setup page |
| `SETUP_ALLOW_REMOTE` | No | `false` | Allow setup writes from a non-loopback request (use an SSH tunnel when possible) |
| `SETUP_REMOTE_TOKEN` | When remote setup is enabled | *(empty)* | One-time setup token, at least 32 random characters |
| `TZ` | No | `Asia/Shanghai` | Timezone |
| `UPDATE_GITHUB_TOKEN` | No | *(empty)* | Optional token for GitHub release/tag checks; use a dedicated token and never expose it to the frontend. Asset downloads remain anonymous. |
| `GEMINI_OAUTH_CLIENT_ID` | No | *(builtin)* | Google OAuth client ID (Gemini OAuth). Leave empty to use the built-in Gemini CLI client. |
| `GEMINI_OAUTH_CLIENT_SECRET` | No | *(builtin)* | Google OAuth client secret (Gemini OAuth). Leave empty to use the built-in Gemini CLI client. |
| `GEMINI_OAUTH_SCOPES` | No | *(default)* | OAuth scopes (Gemini OAuth) |
| `GEMINI_QUOTA_POLICY` | No | *(empty)* | JSON overrides for Gemini local quota simulation (Code Assist only). |

See `.env.example` for all available options.

> **Note:** The `docker-deploy.sh` script automatically generates `JWT_SECRET`, `TOTP_ENCRYPTION_KEY`, `POSTGRES_PASSWORD`, and `SETUP_REMOTE_TOKEN`. It never writes an administrator password. Choose that password once on `/setup`.

> **Existing data:** Existing administrators are never overwritten. The owner setup page is only available while the durable pending marker is present.

> **Network exposure:** Published application ports bind to `127.0.0.1` by default. Put Caddy, Nginx, or another trusted TLS reverse proxy in front of Sub3API. Set `BIND_HOST=0.0.0.0` only when you intentionally need direct network access and have configured the host firewall.

> **Remote setup:** Binary installs keep the first-run wizard on loopback; an SSH tunnel is recommended. If Docker is behind a reverse proxy, enable `SETUP_ALLOW_REMOTE=true`, read the one-time `SETUP_REMOTE_TOKEN` from `.env`, and append it as a `#setup_token=...` URL fragment (the frontend removes it from the address bar and never stores it). The proxy must preserve the `X-Setup-Token` header sent by the setup page and must not log or inject a token for other clients. `SETUP_ALLOW_PRIVATE_TARGETS=true` further expands the setup wizard's network reach and should remain disabled unless strictly required.

### Easy Migration (Local Directory Version)

When using `docker-compose.local.yml`, all data is stored in local directories, making migration simple:

```bash
# On source server: Stop services and create archive
cd /path/to/deployment
docker compose -f docker-compose.local.yml down
cd ..
tar czf sub3api-complete.tar.gz deployment/

# Transfer to new server
scp sub3api-complete.tar.gz user@new-server:/path/to/destination/

# On new server: Extract and start
tar xzf sub3api-complete.tar.gz
cd deployment/
docker compose -f docker-compose.local.yml up -d
```

Your entire deployment (configuration + data) is migrated!

---

## Gemini OAuth Configuration

Sub3API supports three methods to connect to Gemini:

### Method 1: Code Assist OAuth (Recommended for GCP Users)

**No configuration needed** - always uses the built-in Gemini CLI OAuth client (public).

1. Leave `GEMINI_OAUTH_CLIENT_ID` and `GEMINI_OAUTH_CLIENT_SECRET` empty
2. In the Admin UI, create a Gemini OAuth account and select **"Code Assist"** type
3. Complete the OAuth flow in your browser

> Note: Even if you configure `GEMINI_OAUTH_CLIENT_ID` / `GEMINI_OAUTH_CLIENT_SECRET` for AI Studio OAuth,
> Code Assist OAuth will still use the built-in Gemini CLI client.

**Requirements:**
- Google account with access to Google Cloud Platform
- A GCP project (auto-detected or manually specified)

**How to get Project ID (if auto-detection fails):**
1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Click the project dropdown at the top of the page
3. Copy the Project ID (not the project name) from the list
4. Common formats: `my-project-123456` or `cloud-ai-companion-xxxxx`

### Method 2: AI Studio OAuth (For Regular Google Accounts)

Requires your own OAuth client credentials.

**Step 1: Create OAuth Client in Google Cloud Console**

1. Go to [Google Cloud Console - Credentials](https://console.cloud.google.com/apis/credentials)
2. Create a new project or select an existing one
3. **Enable the Generative Language API:**
   - Go to "APIs & Services" → "Library"
   - Search for "Generative Language API"
   - Click "Enable"
4. **Configure OAuth Consent Screen** (if not done):
   - Go to "APIs & Services" → "OAuth consent screen"
   - Choose "External" user type
   - Fill in app name, user support email, developer contact
   - Add scopes: `https://www.googleapis.com/auth/generative-language.retriever` (and optionally `https://www.googleapis.com/auth/cloud-platform`)
   - Add test users (your Google account email)
5. **Create OAuth 2.0 credentials:**
   - Go to "APIs & Services" → "Credentials"
   - Click "Create Credentials" → "OAuth client ID"
   - Application type: **Web application** (or **Desktop app**)
   - Name: e.g., "Sub3API Gemini"
   - Authorized redirect URIs: Add `http://localhost:1455/auth/callback`
6. Copy the **Client ID** and **Client Secret**
7. **⚠️ Publish to Production (IMPORTANT):**
   - Go to "APIs & Services" → "OAuth consent screen"
   - Click "PUBLISH APP" to move from Testing to Production
   - **Testing mode limitations:**
     - Only manually added test users can authenticate (max 100 users)
     - Refresh tokens expire after 7 days
     - Users must be re-added periodically
   - **Production mode:** Any Google user can authenticate, tokens don't expire
   - Note: For sensitive scopes, Google may require verification (demo video, privacy policy)

**Step 2: Configure Environment Variables**

```bash
GEMINI_OAUTH_CLIENT_ID=your-client-id.apps.googleusercontent.com
GEMINI_OAUTH_CLIENT_SECRET=GOCSPX-your-client-secret

# 可选：如需使用 Gemini CLI 内置 OAuth Client（Code Assist / Google One）
# 安全说明：本仓库不会内置该 client_secret，请在运行环境通过环境变量注入。
# GEMINI_CLI_OAUTH_CLIENT_SECRET=GOCSPX-your-built-in-secret
```

**Step 3: Create Account in Admin UI**

1. Create a Gemini OAuth account and select **"AI Studio"** type
2. Complete the OAuth flow
   - After consent, your browser will be redirected to `http://localhost:1455/auth/callback?code=...&state=...`
   - Copy the full callback URL (recommended) or just the `code` and paste it back into the Admin UI

### Method 3: API Key (Simplest)

1. Go to [Google AI Studio](https://aistudio.google.com/app/apikey)
2. Click "Create API key"
3. In Admin UI, create a Gemini **API Key** account
4. Paste your API key (starts with `AIza...`)

### Comparison Table

| Feature | Code Assist OAuth | AI Studio OAuth | API Key |
|---------|-------------------|-----------------|---------|
| Setup Complexity | Easy (no config) | Medium (OAuth client) | Easy |
| GCP Project Required | Yes | No | No |
| Custom OAuth Client | No (built-in) | Yes (required) | N/A |
| Rate Limits | GCP quota | Standard | Standard |
| Best For | GCP developers | Regular users needing OAuth | Quick testing |

---

## Binary Installation

For production servers using systemd.

### One-Line Installation

```bash
curl -fsSL https://raw.githubusercontent.com/yuchen0x1/sub3api-relay/main/deploy/install.sh | sudo bash
```

### Manual Installation

1. Download the latest release from [GitHub Releases](https://github.com/yuchen0x1/sub3api-relay/releases)
2. Extract and copy the binary to `/opt/sub3api/`
3. Copy `sub3api.service` to `/etc/systemd/system/`
4. Run:
   ```bash
   sudo systemctl daemon-reload
   sudo systemctl enable sub3api
   sudo systemctl start sub3api
   ```
5. Create an SSH tunnel with `ssh -L 8080:127.0.0.1:8080 user@YOUR_SERVER_IP`, then open `http://127.0.0.1:8080` locally to complete configuration

> **v0.1.178+:** Release binaries embed the frontend. Upgrading only requires replacing the binary and restarting `sub3api`; do **not** download or extract `sub3api_<version>_web.tar.gz` for a normal upgrade.
>
> **Caddy/nginx:** All site traffic, including `/` and `/manus/`, must be reverse-proxied to the backend. Do not expose a separate static frontend directory with `file_server`; keep any old `/var/www/...` directory only as an offline backup.

### Binary Upgrade (v0.1.178+)

```bash
# Upgrade to the latest release
sudo ./update-binary.sh

# Upgrade to a specific version
sudo ./update-binary.sh v0.1.178

# Non-default install layout (custom binary path / service / health URL)
sudo SUB3API_BIN=/opt/sub3api/sub3api \
     SUB3API_SERVICE=sub3api \
     SUB3API_HEALTH_URL=http://127.0.0.1:8080/health \
     ./update-binary.sh v0.1.178
```

See [BINARY_UPGRADE.md](./BINARY_UPGRADE.md) for the full upgrade, Caddy, verification, and rollback guide.

### Commands

```bash
# Install
sudo ./install.sh

# Upgrade
sudo ./install.sh upgrade

# Uninstall
sudo ./install.sh uninstall
```

### Service Management

```bash
# Start the service
sudo systemctl start sub3api

# Stop the service
sudo systemctl stop sub3api

# Restart the service
sudo systemctl restart sub3api

# Check status
sudo systemctl status sub3api

# View logs
sudo journalctl -u sub3api -f

# Enable auto-start on boot
sudo systemctl enable sub3api
```

### Configuration

#### Server Address and Port

During installation, you will be prompted to configure the server listen address and port. These settings are stored in the systemd service file as environment variables.

To change after installation:

1. Edit the systemd service:
   ```bash
   sudo systemctl edit sub3api
   ```

2. Add or modify:
   ```ini
   [Service]
   Environment=SERVER_HOST=0.0.0.0
   Environment=SERVER_PORT=3000
   ```

3. Reload and restart:
   ```bash
   sudo systemctl daemon-reload
   sudo systemctl restart sub3api
   ```

#### Gemini OAuth Configuration

If you need to use AI Studio OAuth for Gemini accounts, add the OAuth client credentials to the systemd service file:

1. Edit the service file:
   ```bash
   sudo nano /etc/systemd/system/sub3api.service
   ```

2. Add your OAuth credentials in the `[Service]` section (after the existing `Environment=` lines):
   ```ini
   Environment=GEMINI_OAUTH_CLIENT_ID=your-client-id.apps.googleusercontent.com
   Environment=GEMINI_OAUTH_CLIENT_SECRET=GOCSPX-your-client-secret
   ```

   如需使用“内置 Gemini CLI OAuth Client”（Code Assist / Google One），还需要注入：
   ```ini
   Environment=GEMINI_CLI_OAUTH_CLIENT_SECRET=GOCSPX-your-built-in-secret
   ```

3. Reload and restart:
   ```bash
   sudo systemctl daemon-reload
   sudo systemctl restart sub3api
   ```

> **Note:** Code Assist OAuth does not require any configuration - it uses the built-in Gemini CLI client.
> See the [Gemini OAuth Configuration](#gemini-oauth-configuration) section above for detailed setup instructions.

#### Application Configuration

The main config file is at `/etc/sub3api/config.yaml` (created by Setup Wizard).

### Prerequisites

- Linux server (Ubuntu 20.04+, Debian 11+, CentOS 8+, etc.)
- PostgreSQL 14+
- Redis 6+
- systemd

### Directory Structure

```
/opt/sub3api/
├── sub3api              # Main binary
├── sub3api.backup       # Backup (after upgrade)
└── data/                # Runtime data

/etc/sub3api/
└── config.yaml          # Configuration file
```

---

## Troubleshooting

### Docker

For **local directory version**:

```bash
# Check container status
docker compose -f docker-compose.local.yml ps

# View detailed logs
docker compose -f docker-compose.local.yml logs --tail=100 sub3api

# Check database connection
docker compose -f docker-compose.local.yml exec postgres pg_isready

# Check Redis connection
docker compose -f docker-compose.local.yml exec redis redis-cli ping

# Restart all services
docker compose -f docker-compose.local.yml restart

# Check data directories
ls -la data/ postgres_data/ redis_data/
```

For **named volumes version**:

```bash
# Check container status
docker compose ps

# View detailed logs
docker compose logs --tail=100 sub3api

# Check database connection
docker compose exec postgres pg_isready

# Check Redis connection
docker compose exec redis redis-cli ping

# Restart all services
docker compose restart
```

### Binary Install

```bash
# Check service status
sudo systemctl status sub3api

# View recent logs
sudo journalctl -u sub3api -n 50

# Check config file
sudo cat /etc/sub3api/config.yaml

# Check PostgreSQL
sudo systemctl status postgresql

# Check Redis
sudo systemctl status redis
```

### Common Issues

1. **Port already in use**: Change `SERVER_PORT` in `.env` or systemd config
2. **Database connection failed**: Check PostgreSQL is running and credentials are correct
3. **Redis connection failed**: Check Redis is running and password is correct
4. **Permission denied**: Ensure proper file ownership for binary install

---

## TLS Fingerprint Configuration

Sub3API supports TLS fingerprint simulation to make requests appear as if they come from the official Claude CLI (Node.js client).

> **💡 Tip:** Visit **[tls.sub3api.org](https://tls.sub3api.org/)** to get TLS fingerprint information for different devices and browsers.

### Default Behavior

- Built-in `claude_cli_v2` profile simulates Node.js 20.x + OpenSSL 3.x
- JA3 Hash: `1a28e69016765d92e3b381168d68922c`
- JA4: `t13d5911h1_a33745022dd6_1f22a2ca17c4`
- Profile selection: `accountID % profileCount`

### Configuration

```yaml
gateway:
  tls_fingerprint:
    enabled: true  # Global switch
    profiles:
      # Simple profile (uses default cipher suites)
      profile_1:
        name: "Profile 1"

      # Profile with custom cipher suites (use compact array format)
      profile_2:
        name: "Profile 2"
        cipher_suites: [4866, 4867, 4865, 49199, 49195, 49200, 49196]
        curves: [29, 23, 24]
        point_formats: 0

      # Another custom profile
      profile_3:
        name: "Profile 3"
        cipher_suites: [4865, 4866, 4867, 49199, 49200]
        curves: [29, 23, 24, 25]
```

### Profile Fields

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Display name (required) |
| `cipher_suites` | []uint16 | Cipher suites in decimal. Empty = default |
| `curves` | []uint16 | Elliptic curves in decimal. Empty = default |
| `point_formats` | []uint8 | EC point formats. Empty = default |

### Common Values Reference

**Cipher Suites (TLS 1.3):** `4865` (AES_128_GCM), `4866` (AES_256_GCM), `4867` (CHACHA20)

**Cipher Suites (TLS 1.2):** `49195`, `49196`, `49199`, `49200` (ECDHE variants)

**Curves:** `29` (X25519), `23` (P-256), `24` (P-384), `25` (P-521)

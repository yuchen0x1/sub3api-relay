# Sub3API：个人自建 AI API 网关

把分散在各个平台的上游 AI 账号收拢到一道网关后面，统一分发 API Key、调度请求、记录用量、管好运维——这就是 Sub3API 做的事。

## 这个项目是做什么的

Sub3API 由 Go 后端和内嵌的 Vue 管理台组成，可以把多个上游 AI 账号统一调度到 OpenAI、Anthropic/Claude、Gemini、Grok、Manus、Antigravity 等兼容入口，并配套用户、分组、API Key、代理、用量、余额、运维监控、备份、审计、风控和系统配置管理。

适用场景：个人自建、内部工具、小团队网关。生产部署推荐 Docker Compose 或发行版二进制；本地开发可以分别运行 Go 后端与 Vite 前端。

## 功能亮点

**网关与调度**

1. 多平台兼容入口：Anthropic/Claude 风格 `/v1/messages`，OpenAI 风格 `/v1/responses`、`/v1/chat/completions`、`/v1/images/*`，以及 Gemini 原生 `/v1beta/models/*`。
2. 多账号调度：按平台、分组、模型路由、账号状态、代理和并发容量进行调度。
3. API Key 分发：用户可创建和管理自己的 API Key，管理员可调整 Key 所属分组。
4. 图片任务：同步图片接口之外，另提供异步图片任务和批量图片任务入口（异步任务需要对象存储配置）。

**计费与运营**

5. 用量与计费：记录请求、Token、模型、费用、余额和聚合统计，支持按用户、Key、分组查看。
6. 内嵌管理台：内置 Vue Web UI，发行版和 Docker 镜像会把前端构建产物嵌入后端服务。

**安全与运维**

7. 登录方式：用户名密码、TOTP、Passkey，以及 LinuxDo、GitHub、Google、WeChat、OIDC、DingTalk 等可配置登录入口。
8. 运维观测：健康检查、实时流量、并发、错误日志、上游错误、系统日志、告警和仪表盘接口。
9. 数据与备份：数据库备份、S3 目标配置、数据管理代理联动和 PostgreSQL SQL 迁移。
10. 安全机制：CSP、CORS、可信代理、URL allowlist、响应头过滤、幂等、面板限流、API Key ACL 和审计日志。

## 技术选型

- 后端：Go 1.26.6、Gin、Ent、Wire、zap
- 前端：Vue 3、Vite 8、TypeScript、Pinia、Vue Router、Tailwind CSS
- 数据库：PostgreSQL（Docker Compose 默认 `postgres:18-alpine`）
- 缓存：Redis（Docker Compose 默认 `redis:8-alpine`）
- 构建发布：Docker、Docker Compose、GoReleaser、pnpm 11.20.0
- 测试：Go test、golangci-lint、Vitest、vue-tsc

## 快速上手

先按部署方式准备环境：

- **Docker 部署**：Linux/macOS/Windows + Docker Engine 或 Docker Desktop，Docker Compose v2，能访问 `ghcr.io/yuchen0x1/sub3api-relay` 拉取镜像；生产环境建议准备好反向代理和 HTTPS 证书。
- **二进制部署**：Linux 服务器 + systemd，自备 PostgreSQL 与 Redis，Bash 4.0 或更高版本，`curl` 或 `wget`，以及发行版安装脚本检查到的系统依赖。
- **本地开发**：Go 1.26.6、Node.js 24、pnpm 11.20.0、PostgreSQL 与 Redis（或使用 `deploy/docker-compose.dev.yml` 从源码构建本地容器环境）。

### 方式一：Docker Compose

```bash
git clone https://gitee.com/yuchen0x1/personal-ai-api-gateway-sub3api.git
cd sub3api-relay/deploy
cp .env.example .env
chmod 600 .env
# 编辑 .env，至少修改 POSTGRES_PASSWORD、JWT_SECRET、TOTP_ENCRYPTION_KEY；首次启动后在 /setup 设置站长账号和密码
docker compose up -d
docker compose logs -f sub3api
```

启动完成后访问 `http://127.0.0.1:8080`。

默认编排使用 Docker named volumes 保存数据。如果你更看重迁移和备份的便利性，可以改用本地目录版本：

```bash
cd deploy
cp .env.example .env
mkdir -p data postgres_data redis_data
docker compose -f docker-compose.local.yml up -d
```

### 方式二：脚本一键准备部署文件

`deploy/docker-deploy.sh` 会从 GitHub Release 下载经过 SHA-256 校验的部署包，自动生成 `JWT_SECRET`、`TOTP_ENCRYPTION_KEY`、`POSTGRES_PASSWORD`，并创建本地数据目录：

```bash
curl -fsSL https://gitee.com/yuchen0x1/personal-ai-api-gateway-sub3api/raw/main/deploy/docker-deploy.sh | bash
docker compose up -d
```

### 方式三：Linux 二进制安装

```bash
curl -fsSL https://gitee.com/yuchen0x1/personal-ai-api-gateway-sub3api/raw/main/deploy/install.sh | sudo bash
```

安装脚本会注册 systemd 服务。首次运行如果没有配置文件，会启动安装向导；远程服务器建议通过 SSH 隧道访问安装向导，不要把未完成初始化的向导暴露到公网：

```bash
ssh -L 8080:127.0.0.1:8080 user@YOUR_SERVER_IP
```

然后在本机浏览器打开 `http://127.0.0.1:8080`。

## 配置说明

配置支持 YAML 文件和环境变量两种来源。Docker Compose 默认通过环境变量完成初始化，并在 `/app/data` 下持久化运行数据。

配置来源：

- `deploy/.env.example`：Docker/Compose 环境变量模板，覆盖常用部署配置
- `deploy/config.example.yaml`：完整 YAML 配置示例，可复制为 `config.yaml` 后挂载到 `/app/data/config.yaml`
- `CONFIG_FILE`：指定自定义配置文件路径
- `DATA_DIR`：指定运行数据目录；未设置时容器内通常使用 `/app/data`

常用环境变量（完整配置项以 `deploy/.env.example` 和 `deploy/config.example.yaml` 为准）：

- `POSTGRES_PASSWORD`：无默认值，Docker Compose 必填，PostgreSQL 密码
- `POSTGRES_USER`：默认 `sub3api`，PostgreSQL 用户
- `POSTGRES_DB`：默认 `sub3api`，PostgreSQL 数据库名
- `SERVER_PORT`：默认 `8080`，宿主机访问端口；容器内服务仍监听 `8080`
- `BIND_HOST`：默认 `127.0.0.1`，Docker Compose 端口绑定地址
- `AUTO_SETUP_OWNER_SETUP`：Docker 默认开启，首次启动先准备基础设施，再由 `/setup` 设置唯一站长账号和密码
- `SETUP_ALLOW_REMOTE` / `SETUP_REMOTE_TOKEN`：通过反向代理远程初始化时使用；令牌至少 32 位
- `JWT_SECRET`：默认空，建议固定为 `openssl rand -hex 32`，否则重启可能导致会话失效
- `TOTP_ENCRYPTION_KEY`：默认空，建议固定为 `openssl rand -hex 32`，否则已绑定的 TOTP 可能失效
- `RUN_MODE`：默认 `standard`；`standard` 启用完整计费/余额逻辑，`simple` 隐藏部分 SaaS 功能并跳过余额校验
- `TZ`：默认 `Asia/Shanghai`，时区，影响日志、统计和过期时间
- `SECURITY_URL_ALLOWLIST_ENABLED`：默认 `false`，是否启用上游 URL allowlist
- `SECURITY_URL_ALLOWLIST_ALLOW_INSECURE_HTTP`：默认 `true`，allowlist 关闭时是否允许 `http://` URL，生产建议设为 `false`
- `SECURITY_URL_ALLOWLIST_ALLOW_PRIVATE_HOSTS`：默认 `true`，是否允许本地/私有地址作为上游、定价或 CRS 目标，生产建议谨慎关闭
- `GEMINI_OAUTH_CLIENT_ID` / `GEMINI_OAUTH_CLIENT_SECRET`：默认空，Gemini AI Studio OAuth 自建客户端配置
- `GEMINI_CLI_OAUTH_CLIENT_SECRET`：默认空，Gemini CLI 内置 OAuth Client Secret，需要时由运行环境注入

## 接口速查

管理台与面板接口：

- 页面入口：`/`、`/home`、`/login`、`/admin/*`（内嵌 Vue 管理台页面）
- `/setup`：首次初始化向导；`/setup/status`：初始化状态；`/health`：健康检查
- `/api/v1/auth/*`：登录、刷新、OAuth 回调、当前用户
- `/api/v1/user/*`：用户资料、密码、TOTP、Passkey、通知邮箱
- `/api/v1/keys/*`：用户 API Key 管理
- `/api/v1/usage/*`：用户用量和错误记录
- `/api/v1/admin/*`：管理员 API，包括账号、分组、代理、设置、备份、运维、审计、风控、提示词审计等
- `/api/v1/settings/public`：前端公开配置
- `/api/v1/channel-monitors/*`：公开渠道状态接口

AI 网关接口（使用 Sub3API 分发的 API Key）：

```bash
curl http://127.0.0.1:8080/v1/models \
  -H 'Authorization: Bearer sk-...'
```

- Claude 系：`/v1/messages`（Anthropic Messages 兼容入口，也会按分组平台自动路由到对应处理器）、`/v1/messages/count_tokens`（Token 计数）
- OpenAI 系：`/v1/responses`（Responses 兼容入口，也支持 WebSocket 升级）、`/v1/chat/completions`、`/v1/embeddings`（仅 OpenAI 平台分组可用）
- 模型列表：`/v1/models`
- 图片：`/v1/images/generations`、`/v1/images/edits`（OpenAI/Grok 图片入口）；异步任务 `/v1/images/generations/async`、`/v1/images/tasks/{task_id}`（需要对象存储配置）；批量任务 `/v1/images/batches/*`
- 视频：`/v1/videos/*`（Grok 视频生成、编辑、扩展和查询）
- Gemini 原生：`/v1beta/models/*`
- 其他兼容入口：`/responses`、`/models`、`/chat/completions` 等无前缀别名；`/backend-api/codex/*`（Codex 兼容入口）；`/antigravity/*`（Antigravity 专用入口）

更完整的行为以 `backend/internal/server/routes/gateway.go` 和相关专题文档为准。

## 本地开发与构建

### 跑后端

```bash
cd backend
go mod download
make build
./bin/server -version
./bin/server -setup
```

根目录的 `make build` 等价于 `make -C backend build`，只构建后端二进制。`backend/Makefile` 使用 `-tags embed`，如果没有先把前端构建到 `backend/internal/web/dist`，服务仍可作为后端启动，但不会包含最新前端产物。

### 跑前端

```bash
cd frontend
corepack enable
corepack prepare pnpm@11.20.0 --activate
pnpm install --frozen-lockfile
pnpm dev
```

Vite 开发服务器默认端口来自 `VITE_DEV_PORT`（未设置时为 `18082`），后端代理目标来自 `VITE_DEV_PROXY_TARGET`（未设置时为 `http://127.0.0.1:18081`）。

### 从源码构建完整 Docker 镜像

```bash
cd deploy
cp .env.example .env
# 修改 POSTGRES_PASSWORD 等必要变量
docker compose -f docker-compose.dev.yml up --build
```

根目录 `Dockerfile` 会先构建前端，再构建带嵌入式前端的 Go 后端，最终生成运行镜像。

### 测试与发布命令

- 后端构建：`make -C backend build`
- 后端全部测试和 lint：`make -C backend test`
- 后端单元 / 集成 / E2E 测试：`make -C backend test-unit` / `make -C backend test-integration` / `make -C backend test-e2e`
- 前端类型检查：`pnpm --dir frontend typecheck`
- 前端 lint：`pnpm --dir frontend lint:check`
- 前端测试：`pnpm --dir frontend test:run`
- 前端构建：`pnpm --dir frontend build`
- 本地完整镜像构建：`docker build -t sub3api:local .`

发布 Release 要求 tag 格式为 `vX.Y.Z`，并校验 `backend/cmd/server/VERSION` 与 `frontend/package.json` 版本一致，细节见 `docs/RELEASING.md` 和 `.github/workflows/release.yml`。

## 目录结构

```text
.
├── backend/                  # Go 后端、Ent 模型、HTTP 路由、服务层、迁移和后端命令
│   ├── cmd/server/            # 主服务入口
│   ├── cmd/jwtgen/            # JWT 生成辅助命令
│   ├── cmd/profit-preview/    # 收益预览辅助命令
│   ├── ent/                   # Ent schema 和生成代码
│   ├── internal/              # 后端业务实现
│   ├── migrations/            # PostgreSQL SQL 迁移
│   └── resources/             # 内置资源，例如模型定价数据
├── frontend/                  # Vue 管理台源码
│   ├── public/
│   └── src/                   # 路由、页面、组件、API client、状态管理
├── deploy/                    # Docker Compose、systemd、安装/升级脚本和部署文档
├── docs/                      # 支付、异步图片、发布等专题文档
├── assets/                    # 项目静态资产
├── openspec/                  # 变更规格说明
├── skills/                    # 项目相关技能/提示资产
├── Dockerfile                 # 完整镜像构建：前端构建 + 后端嵌入 + runtime
├── Dockerfile.goreleaser      # GoReleaser 镜像构建文件
├── Makefile                   # 根构建入口，当前委托到 backend
└── LICENSE                    # LGPL-3.0 License
```

## 文档索引

部署与运维：

- `deploy/README.md`：Docker、Apple container、二进制安装和常用运维命令
- `deploy/DOCKER.md`：Docker 镜像和容器部署说明
- `deploy/BINARY_UPGRADE.md`：二进制升级、验证和回滚
- `deploy/APPLE_CONTAINER.md`：Apple container 部署方式
- `deploy/EDGE_SECURITY.md`：反向代理、CDN/WAF、可信代理和边缘安全
- `deploy/DATAMANAGEMENTD_CN.md`：数据管理代理部署与联动

功能专题：

- `docs/ASYNC_IMAGE_TASKS.md`：异步图片任务配置、接口和排障
- `docs/PAYMENT.md`、`docs/PAYMENT_CN.md`：支付模块说明，当前可用入口请以实际路由挂载为准
- `docs/ADMIN_PAYMENT_INTEGRATION_API.md`：管理侧支付集成接口说明
- `docs/RELEASING.md`：发布流程说明

其他：

- `backend/migrations/README.md`：数据库迁移规则
- `backend/cmd/cleanup-ingress-reject-logs/README.md`：ingress rejection 日志清理命令

## 上线前安全清单

- [ ] 修改 `POSTGRES_PASSWORD`，固定 `JWT_SECRET` 和 `TOTP_ENCRYPTION_KEY`，并在 `/setup` 设置唯一站长账号和强密码。
- [ ] 默认 Compose 只把服务绑定到 `127.0.0.1`；如改为 `0.0.0.0`，请先配置防火墙、HTTPS 和反向代理。
- [ ] 二进制安装的首次初始化向导只通过本机或 SSH 隧道访问，不要暴露到公网。
- [ ] Nginx 反向代理如果依赖带下划线的请求头（例如 `session_id`），需要在 `http` 块设置 `underscores_in_headers on;`。
- [ ] 需要严格限制上游请求目标时，启用 `SECURITY_URL_ALLOWLIST_ENABLED=true` 并配置允许的上游主机；生产环境建议关闭不必要的 HTTP 和私有地址访问。
- [ ] 账号、代理、备份、S3、审计等敏感操作涉及凭证或数据导出，应启用 TOTP/Passkey 等二次验证能力。
- [ ] OAuth `client_secret`、上游账号 Cookie、API Key、数据库密码不要提交到仓库。

## 常见问题

**`docker compose up` 提示 `POSTGRES_PASSWORD is required`**

先复制并编辑环境变量文件：

```bash
cd deploy
cp .env.example .env
chmod 600 .env
# 修改 POSTGRES_PASSWORD 后再启动
docker compose up -d
```

**修改 `.env` 中的站长密码不生效**

新部署的站长密码只在 `/setup` 提交时写入数据库；已有账号需要在管理台修改密码，或使用项目提供的重置/维护流程处理。

**前端开发服务器连不上后端**

确认后端地址与 Vite 代理一致：

```bash
VITE_DEV_PROXY_TARGET=http://127.0.0.1:8080 pnpm --dir frontend dev
```

**异步图片接口返回 404**

异步图片任务依赖对象存储，需要启用并完整配置 `image_storage`，或在管理台配置异步图片对象存储，详见 `docs/ASYNC_IMAGE_TASKS.md`。

**反向代理后粘性会话异常**

检查代理是否保留必要请求头，Nginx 需要配置：

```nginx
underscores_in_headers on;
```

## 周边生态

围绕 Sub3API 的社区扩展与集成项目，当前均处于正常更新状态：

- **Sub3ApiPay**：自助支付系统，提供支付能力扩展与集成工具；Sub3API 同时提供内置支付功能，详见 `docs/PAYMENT_CN.md`。
- **sub3api-mobile**：移动端管理控制台，支持用户管理、账号管理、监控看板和多后端切换的跨平台应用。

## 参与贡献

- 提交前按影响范围运行测试：后端改动至少运行对应的 `go test`，前端改动至少运行 `pnpm --dir frontend typecheck` 和相关 Vitest。
- 数据库迁移必须新增 SQL 文件，不要修改已发布或可能已应用的迁移文件。
- 涉及部署、环境变量或公开 API 的改动，需要同步更新 README 或 `deploy/`、`docs/` 下对应文档。
- Release 版本需保持 `backend/cmd/server/VERSION` 与 `frontend/package.json` 一致。

## 开源协议与交流

- 本项目基于 LGPL-3.0 协议开源，详见 `LICENSE`。
- QQ 交流群：`1016190748`
- Telegram：`https://t.me/lenmoC`

反馈问题时，建议附上版本号、部署方式、运行平台、相关日志、配置来源和复现步骤，便于快速定位。

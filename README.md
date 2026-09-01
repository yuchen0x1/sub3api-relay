

# Sub3API · AI API 网关与管理平台

> 面向个人自建和内部团队使用的 AI API 中转、账号调度、用量计费与运维管理平台。

[![Go](https://img.shields.io/badge/Go-1.26.6-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![Vue](https://img.shields.io/badge/Vue-3-42B883?logo=vuedotjs&logoColor=white)](https://vuejs.org/)
[![Vite](https://img.shields.io/badge/Vite-8-646CFF?logo=vite&logoColor=white)](https://vite.dev/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-18--alpine-4169E1?logo=postgresql&logoColor=white)](https://www.postgresql.org/)
[![Redis](https://img.shields.io/badge/Redis-8--alpine-DC382D?logo=redis&logoColor=white)](https://redis.io/)
[![Docker](https://img.shields.io/badge/Docker-ready-2496ED?logo=docker&logoColor=white)](https://www.docker.com/)

## 项目简介

Sub3API 是一个 Go 后端 + 嵌入式 Vue 管理台的 AI API 网关。它将多个上游 AI 账号统一调度到 OpenAI、Gemini、Grok、Manus、Antigravity 等兼容入口，并提供用户、分组、API Key、代理、用量、余额、运维监控、备份、审计、风控和系统配置管理。

项目默认适合个人自建、内部工具和小团队网关场景。生产部署推荐使用 Docker Compose 或发行版二进制；本地开发可分别运行 Go 后端与 Vite 前端。

## 核心功能

- **多平台网关**：支持 Claude Messages 协议风格 `/v1/messages`、OpenAI 风格 `/v1/responses`、`/v1/chat/completions`、`/v1/images/*`，以及 Gemini 原生 `/v1beta/models/*`。
- **多账号调度**：支持按平台、分组、模型路由、账号状态、代理和并发容量进行调度。
- **API Key 分发**：用户可创建和管理自己的 API Key；管理员可调整 Key 所属分组。
- **用量与计费**：记录请求、Token、模型、费用、余额和聚合统计，支持按用户、Key、分组查看。
- **管理台**：内置 Vue Web UI，发行版和 Docker 镜像会把前端构建产物嵌入后端服务。
- **安全登录**：支持用户名密码、TOTP、Passkey，以及 LinuxDo、GitHub、Google、WeChat、OIDC、DingTalk 等可配置登录入口。
- **运维能力**：提供健康检查、实时流量、并发、错误日志、上游错误、系统日志、告警和仪表盘接口。
- **数据与备份**：支持数据库备份、S3 目标配置、数据管理代理联动和 PostgreSQL SQL 迁移。
- **图片任务**：支持同步图片接口，也提供异步图片任务和批量图片任务入口；异步任务需要对象存储配置。
- **安全控制**：包含 CSP、CORS、可信代理、URL allowlist、响应头过滤、幂等、面板限流、API Key ACL 和审计日志等机制。

## 技术栈

| 模块 | 技术 |
| --- | --- |
| 后端 | Go 1.26.6, Gin, Ent, Wire, zap |
| 前端 | Vue 3, Vite 8, TypeScript, Pinia, Vue Router, Tailwind CSS |
| 数据库 | PostgreSQL；Docker Compose 默认 `postgres:18-alpine` |
| 缓存 | Redis；Docker Compose 默认 `redis:8-alpine` |
| 构建发布 | Docker, Docker Compose, GoReleaser, pnpm 11.20.0 |
| 测试 | Go test, golangci-lint, Vitest, vue-tsc |

## 系统要求

### Docker 部署

- Linux/macOS/Windows + Docker Engine 或 Docker Desktop
- Docker Compose v2
- 可访问 `ghcr.io/yuchen0x1/sub3api-relay` 拉取镜像
- 生产环境建议准备反向代理和 HTTPS 证书

### 二进制部署

- Linux 服务器，systemd
- PostgreSQL 与 Redis 服务
- Bash 4.0 或更高版本
- `curl` 或 `wget`，以及发行版安装脚本检查到的系统依赖

### 本地开发

- Go 1.26.6
- Node.js 24
- pnpm 11.20.0
- PostgreSQL 与 Redis，或使用 `deploy/docker-compose.dev.yml` 从源码构建本地容器环境

## 快速开始

### Docker Compose

```bash
git clone https://gitee.com/yuchen0x1/personal-ai-api-gateway-sub3api.git
cd sub3api/deploy
cp .env.example .env
chmod 600 .env
# 编辑 .env，至少修改 POSTGRES_PASSWORD、JWT_SECRET、TOTP_ENCRYPTION_KEY、ADMIN_PASSWORD
docker compose up -d
docker compose logs -f sub3api
```

默认访问地址：`http://127.0.0.1:8080`。

### 一键准备 Docker 部署文件

`deploy/docker-deploy.sh` 会从 GitHub Release 下载经过 SHA-256 校验的部署包，生成 `JWT_SECRET`、`TOTP_ENCRYPTION_KEY`、`POSTGRES_PASSWORD`，并创建本地数据目录。

```bash
curl -fsSL https://github.com/yuchen0x1/personal-ai-api-gateway-sub3api/releases/latest/download/docker-deploy.sh | bash
docker compose up -d
```

### Linux 二进制安装

```bash
curl -fsSL https://raw.githubusercontent.com/yuchen0x1/sub3api-relay/main/deploy/install.sh | sudo bash
```

二进制安装会注册 systemd 服务。首次运行如果没有配置文件，会启动安装向导；远程服务器建议通过 SSH 隧道访问安装向导。

```bash
ssh -L 8080:127.0.0.1:8080 user@YOUR_SERVER_IP
```

然后在本机浏览器打开 `http://127.0.0.1:8080`。

## 配置说明

Sub3API 支持 YAML 配置文件和环境变量。Docker Compose 默认通过环境变量完成初始化，并在 `/app/data` 下持久化运行数据。

| 配置来源 | 说明 |
| --- | --- |
| `deploy/.env.example` | Docker/Compose 环境变量模板，覆盖常用部署配置 |
| `deploy/config.example.yaml` | 完整 YAML 配置示例，可复制为 `config.yaml` 后挂载到 `/app/data/config.yaml` |
| `CONFIG_FILE` | 指定自定义配置文件路径 |
| `DATA_DIR` | 指定运行数据目录；未设置时容器内通常使用 `/app/data` |

常用环境变量：

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `POSTGRES_PASSWORD` | 无 | Docker Compose 必填，PostgreSQL 密码 |
| `POSTGRES_USER` | `sub3api` | PostgreSQL 用户 |
| `POSTGRES_DB` | `sub3api` | PostgreSQL 数据库名 |
| `SERVER_PORT` | `8080` | 宿主机访问端口；容器内服务仍监听 `8080` |
| `BIND_HOST` | `127.0.0.1` | Docker Compose 端口绑定地址 |
| `ADMIN_EMAIL` | `admin` | 首次初始化管理员账号 |
| `ADMIN_PASSWORD` | `admin` | 首次初始化管理员密码，生产环境必须修改 |
| `JWT_SECRET` | 空 | 建议固定为 `openssl rand -hex 32`，否则重启可能导致会话失效 |
| `TOTP_ENCRYPTION_KEY` | 空 | 建议固定为 `openssl rand -hex 32`，否则已绑定 TOTP 可能失效 |
| `RUN_MODE` | `standard` | `standard` 启用完整计费/余额逻辑；`simple` 隐藏部分 SaaS 功能并跳过余额校验 |
| `TZ` | `Asia/Shanghai` | 时区，影响日志、统计和过期时间 |

## API 入口速查

### 管理台与面板 API

| 入口 | 说明 |
| --- | --- |
| `/`、`/home`、`/login`、`/admin/*` | 嵌入式 Vue 管理台页面 |
| `/setup` | 首次初始化向导 |
| `/health` | 健康检查 |
| `/api/v1/auth/*` | 登录、刷新、OAuth 回调、当前用户 |
| `/api/v1/user/*` | 用户资料、密码、TOTP、Passkey、通知邮箱 |
| `/api/v1/keys/*` | 用户 API Key 管理 |
| `/api/v1/usage/*` | 用户用量和错误记录 |
| `/api/v1/admin/*` | 管理员 API，包括账号、分组、代理、设置、备份、运维、审计、风控、提示词审计等 |

### AI 网关 API

网关请求使用 Sub3API 分发的 API Key：

```bash
curl http://127.0.0.1:8080/v1/models \
  -H 'Authorization: Bearer sk-...'
```

| 入口 | 说明 |
| --- | --- |
| `/v1/messages` | Claude Messages 协议兼容入口 |
| `/v1/messages/count_tokens` | Token 计数入口 |
| `/v1/models` | 模型列表入口 |
| `/v1/responses` | OpenAI Responses 兼容入口 |
| `/v1/chat/completions` | OpenAI Chat Completions 兼容入口 |
| `/v1/embeddings` | OpenAI embeddings 入口 |
| `/v1/images/*` | 图片生成入口 |
| `/v1beta/models/*` | Gemini 原生 API 兼容入口 |
| `/v1/videos/*` | Grok 视频生成、编辑、扩展和查询入口 |

## 本地开发

### 后端

```bash
cd backend
go mod download
make build
./bin/server -version
./bin/server -setup
```

### 前端

```bash
cd frontend
corepack enable
corepack prepare pnpm@11.20.0 --activate
pnpm install --frozen-lockfile
pnpm dev
```

Vite 开发服务器默认端口来自 `VITE_DEV_PORT`，未设置时为 `18082`；后端代理目标来自 `VITE_DEV_PROXY_TARGET`，未设置时为 `http://127.0.0.1:18081`。

### 从源码构建完整 Docker 镜像

```bash
cd deploy
cp .env.example .env
# 修改 POSTGRES_PASSWORD 等必要变量
docker compose -f docker-compose.dev.yml up --build
```

根目录 `Dockerfile` 会先构建前端，再构建带嵌入式前端的 Go 后端，最终生成运行镜像。

## 常见问题

### `docker compose up` 提示 `POSTGRES_PASSWORD is required`

先复制并编辑环境变量文件：

```bash
cd deploy
cp .env.example .env
chmod 600 .env
# 修改 POSTGRES_PASSWORD 后再启动
docker compose up -d
```

### 登录后修改 `.env` 的 `ADMIN_PASSWORD` 不生效

`ADMIN_PASSWORD` 只在首次创建管理员账号时使用。已有账号需要在管理台修改密码。

### 前端开发服务器连不上后端

确认后端地址与 Vite 代理一致：

```bash
VITE_DEV_PROXY_TARGET=http://127.0.0.1:8080 pnpm --dir frontend dev
```

### 异步图片接口返回 404

异步图片任务依赖对象存储。需要启用并完整配置 `image_storage`，或在管理台配置异步图片对象存储。

### 反向代理后粘性会话异常

检查代理是否保留必要请求头。Nginx 需要配置：

```nginx
underscores_in_headers on;
```

## 贡献指南

- 提交前按影响范围运行测试：后端改动至少运行对应 `go test`，前端改动至少运行 `pnpm --dir frontend typecheck` 和相关 Vitest。
- 数据库迁移必须新增 SQL 文件，不要修改已发布或可能已应用的迁移文件。
- 涉及部署、环境变量或公开 API 的改动，需要同步更新 README 或 `deploy/`、`docs/` 下对应文档。
- Release 版本需要保持 `backend/cmd/server/VERSION` 与 `frontend/package.json` 一致。

## License

本项目使用 LGPL-3.0 License。详见 `LICENSE`。

## 联系方式

- QQ 交流群：`1016190748`
- Telegram：`https://t.me/lenmoC`

反馈问题时建议附上版本号、部署方式、运行平台、相关日志、配置来源和复现步骤。
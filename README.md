<div align="center">

# Sub3API · 个人 AI API 网关

> 面向个人自建与内部使用的纯后端 AI API 中转网关（Go + PostgreSQL + Redis）

[![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![Gin](https://img.shields.io/badge/Gin-00ACD7?logo=go&logoColor=white)](https://gin-gonic.com/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15%2B-4169E1?logo=postgresql&logoColor=white)](https://www.postgresql.org/)
[![Redis](https://img.shields.io/badge/Redis-7%2B-DC382D?logo=redis&logoColor=white)](https://redis.io/)
[![Docker](https://img.shields.io/badge/Docker-ready-2496ED?logo=docker&logoColor=white)](https://www.docker.com/)

</div>

## 📖 项目简介

Sub3API 是面向个人使用的纯后端 AI API 网关，提供兼容主流客户端的 API 中转能力，默认关闭注册、前端界面、支付和商业化分发功能。提供 API 鉴权、上游账号调度、计费记录、负载均衡和请求转发能力，适合个人自建与内部使用。

## 🎯 功能特性

- **多账号管理**：支持 OAuth、API Key 等多种上游账号类型
- **API Key 分发**：为用户生成和管理 API Key
- **精确计费**：Token 级别用量追踪与成本计算
- **智能调度**：智能账号选择，支持粘性会话
- **并发控制**：用户级与账号级并发限制
- **速率限制**：可配置的请求与 Token 速率限制
- **管理 API**：通过后端 API 进行监控与管理
- **外部系统集成**：支持 iframe 嵌入外部系统（如工单等）

## 🧱 技术栈

| 组件 | 技术 |
| --- | --- |
| 后端 | Go 1.26 + Gin + Ent |
| 数据库 | PostgreSQL 15+ |
| 缓存/队列 | Redis 7+ |

## 🚀 快速开始

### 方式一：脚本安装（推荐）

前置条件：Linux 服务器（amd64 / arm64）、PostgreSQL 15+、Redis 7+、Root 权限。

```bash
curl -fsSL https://raw.githubusercontent.com/yuchen0x1/sub3api-relay/main/deploy/install.sh | sudo bash
```

脚本会自动检测架构、下载最新版本并完成安装配置。

### 方式二：Docker Compose（推荐）

```bash
cd deploy
docker compose up -d
docker compose logs -f sub3api
```

### 本地开发（Docker Compose Local）

```bash
cd deploy
docker compose -f docker-compose.local.yml up -d
```

## ⚙️ 默认配置与注意事项

| 配置项 | 默认值 | 说明 |
| --- | --- | --- |
| 管理后台 | `/` | 首次部署会引导设置管理员账号密码 |
| 默认账号 | `admin` | 初始管理员，通过 `.env` 的 `ADMIN_EMAIL` / `ADMIN_PASSWORD` 设置 |
| 服务端口 | `8080` | 通过 `SERVER_PORT` 修改 |
| 数据库 | PostgreSQL | 必填 `POSTGRES_PASSWORD` |
| 密钥 | JWT_SECRET / TOTP_ENCRYPTION_KEY | 部署时自动生成，请妥善保存 |

> **初始化项目说明**：本系统部署时会引导你自行设置管理员账号与密钥，请勿使用示例密码。

### Nginx 反向代理注意事项

通过 Nginx 反代 Sub3API 时，需要在 `http` 块添加：

```nginx
underscores_in_headers on;
```

否则带下划线的请求头（如 `session_id`）会被 Nginx 丢弃，导致粘性会话失效。

## 📬 联系方式

- QQ 交流群：`1016190748`
- Telegram：[https://t.me/lenmoC](https://t.me/lenmoC)

联系或反馈时，请说明你使用的版本、运行平台和具体问题。
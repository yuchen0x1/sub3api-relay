# 安全边界与开源架构（Security Boundary）

> 本文说明 Sub3API 的开源边界：哪些内容可以公开、哪些能力由服务端控制、
> 密钥体系在哪里，以及"复制源码"与"获得运行能力"之间的架构边界。
> 适用于运维、二次开发者和安全审计人员。

## 1. 核心原则

```text
开源代码 ≠ 可复制的完整运行能力
```

- 本仓库**全部代码开源**：实现、Schema、部署模板、SDK、管理台 UI。
- 但任何一次真实运行所依赖的**凭据、数据、用户、配额与风控状态**只存在于
  运营者自己的服务端（PostgreSQL / Redis / Secret 环境变量 / 对象存储），
  从不进入 Git 仓库、Docker 镜像、发布产物或前端。
- Fork 本仓库得到的是"自建版（Community Edition）"：你可以完整运行自己的
  实例，但拿不到**任何其他运营方**的凭据、数据库、用户与商业配置。

本项目本身就是服务端（self-hosted gateway）：终端用户只持有网关签发的
`sk-` API Key；上游 AI 供应商（Anthropic / OpenAI / Gemini 等）的凭据只
存在于服务端数据库，永不下发到客户端。

## 2. 边界划分

### A. 可以开源（本仓库全部公开）

| 类别 | 内容 |
| --- | --- |
| 服务端实现 | `backend/`（Go, Gin, Ent）：网关、调度、配额、计费、风控、审计 |
| 管理台 | `frontend/`（Vue 3）：构建产物嵌入二进制，不含任何密钥 |
| 桌面端 | `desktop/`（Tauri）：仅连接用户自配的服务端 URL，无内嵌凭据 |
| 数据结构 | `backend/ent/schema/`、`backend/migrations/`（纯结构，无数据） |
| 部署模板 | `deploy/*.example*`、`docker-compose*.yml`（仅占位符） |
| 文档 | `README*`、`docs/`、`DEV_GUIDE.md` |

### B. 只存在于运行实例的服务端（任何仓库中都没有）

| 类别 | 位置 |
| --- | --- |
| 上游供应商凭据（OAuth token / session key / api key） | 服务端 DB `accounts.credentials`，**字段级 AES-256-GCM 加密**（见 §3） |
| 网关 API Key（`sk-`） | 服务端 DB，用户级，可吊销/轮换，受配额与 ACL 约束 |
| JWT 签名密钥 | 环境变量 `JWT_SECRET` 或首启随机生成并持久化；公开占位值会被**拒绝启动** |
| 凭据加密密钥 | 环境变量 `TOTP_ENCRYPTION_KEY`（64 位 hex）；未配置则凭据保持明文并告警 |
| 用户数据 / 用量 / 计费 / 支付凭据 | 服务端 DB；管理台 API 对凭据一律脱敏（`has_*` 布尔） |
| 风控/限流/内容审核策略与运行状态 | 服务端 DB + Redis（每用户/Key 配额、QPS、并发、滥用计数、冷却） |

### C. 永不允许出现的位置（CI 有 gitleaks 扫描兜底）

Git 仓库、前端 JS、Docker 镜像、发布二进制、日志、测试文件、README。
测试与文档中只允许出现占位符（`sk-ant-xxx` 等）。

## 3. 凭据字段级加密（at-rest）

上游供应商凭据是本系统最有价值的资产。自 v0.2.2 起：

- 写入路径（创建/更新/凭据替换/token 刷新/部分合并）把 `credentials` 中
  敏感子键（`access_token`、`refresh_token`、`api_key`、`session_key`、
  `cookie`、`private_key` 等，见 `SensitiveCredentialKeys`）以
  `enc:v1:` 信封做 **AES-256-GCM** 加密；JSON 键结构保留，
  因此 `credentials ? 'refresh_token'`、`base_url` 匹配等 SQL 查询不受影响。
- 加密会破坏 JSONB 相等比较（CAS/去重/组身份判定），配套引入两个确定性
  摘要列（迁移 `206_account_credentials_digest.sql`）：
  - `credentials_digest`：规范化明文文档的 SHA-256；
  - `credentials_api_key_digest`：`api_key` 值的 SHA-256。
  守卫逻辑：摘要列为 NULL（加密启用前的明文旧行）→ 自动退回明文直比；
  非空 → 摘要比较。旧行在下次全量凭据写入时自愈为加密存储。
- **启用条件**：显式设置 `TOTP_ENCRYPTION_KEY`（64 位 hex，32 字节）。
  未配置时凭据保持明文（启动时告警）——随机临时密钥跨进程不可解密，
  严禁用于持久化数据。
- 解密失败（如换钥后恢复备份）时保留密文原文并告警：上游调用以显式 401
  暴露问题，绝不静默注入垃圾值。
- 摘要列同时保证加密开关切换前后 CAS 语义一致（摘要永远基于明文计算）。

> 备份提示：数据库备份包含密文。跨实例恢复备份时必须使用与加密时相同的
> `TOTP_ENCRYPTION_KEY`，否则凭据需逐账号重新录入。磁盘级加密（LUKS/云盘
> 加密）仍建议叠加使用。

## 4. 认证与授权链路

```text
终端用户 App ──Bearer sk-xxx──▶ 网关 /v1/*
                                │  API Key 校验（IP ACL / QPS / 配额 / 余额 / 订阅）
                                │  无效凭据滥用限流 → 暂时封禁
                                ▼
                          账号调度器（凭据只在服务端解密使用）
                                ▼
                          上游供应商 API

管理台浏览器 ──JWT(短期)+RefreshToken(轮换+重放检测)──▶ /api/v1/*
                                │  adminAuth 角色校验（user.IsAdmin）
                                │  敏感操作 2FA step-up + 审计日志
                                ▼
                          管理 API（凭据响应一律脱敏）
```

- JWT：HS256，密钥 ≥32 字节，公开占位值拒绝启动；改密/封禁通过
  `token_version` 立即吊销存量 token。
- Refresh Token：仅存 SHA-256 哈希，一次性轮换，重放旧 token 撤销整个
  token family。
- 管理台导出明文凭据的唯一点 `GET /admin/accounts/data` 受 2FA step-up
  与审计日志双重约束。
- 登录/刷新/2FA 端点均有 fail-close 限流；登录有滥用计数封禁。

## 5. 反滥用与配额（服务端强制，客户端无从绕过）

- 每 Key：QPS 限流、5h/1d/7d 用量窗口、配额/余额/订阅前置校验、IP黑白名单。
- 每用户：RPM 限制、并发槽位（Redis sorted-set）。
- 全局：面板限流（认证态按用户、公开路由按 IP）、入口拒绝分类统计、
  内容审核/提示词审计引擎。
- 异常处置链：限流 → 降级（临时不可调度/冷却）→ 暂停（封禁）→ 重新验证。
  无任何"破坏性反制"。

## 6. Fork / 自建边界

```text
Fork 本仓库 → 自建实例（Community Edition）
  ✔ 完整网关/调度/配额/计费能力，用自己的供应商凭据
  ✘ 得不到任何其他运营方的：凭据、数据库、用户、License、支付通道、模型资源
```

"商业能力不可复制"由架构保证：这些能力绑定在**运营者的基础设施与数据**
上（服务端 DB / Redis / Secret Manager / 支付商户号），而不是代码里。
代码混淆不是安全边界；服务端控制才是。

## 7. 环境与 CI/CD

- 环境分离：`development / testing / staging / production` 各自独立的
  Secret、数据库与 Endpoint（`.env.example` / `config.example.yaml` 全部
  为占位符；compose 对必填密钥 fail-fast）。
- CI（`.github/workflows/`）：
  - 所有 workflow `permissions: contents: read` 起步；不使用
    `pull_request_target` 执行 PR 代码；fork PR 拿不到任何 secret。
  - `security-scan.yml`：govulncheck（固定版本）+ **gitleaks 全历史密钥扫描**。
  - Release：tag 触发、双注册表（GHCR/DockerHub）凭据走 GitHub Secrets、
    发布产物附 sha256 校验。
- 发布签名：二进制校验和随 Release 发布；生产部署前请校验 sha256。

## 8. 已知边界与后续路线（诚实清单）

以下能力需要**真实的基础设施**才能实现，当前架构不假装拥有它们：

1. **网关 API Key 哈希存储**：`api_keys.key` 目前为明文唯一索引（O(1)
   查找依赖）。DB 泄露时上游凭据已是密文，但用户 Key 仍可读。改哈希存储
   需要键派生 + 查询迁移，列为下一阶段工作。
2. **集中式 License 服务**：本项目为自托管形态，无"官方授权服务器"。
   若要商业化 SaaS 分发，需独立部署 License 签发服务（非对称签名私钥仅存
   服务端，客户端/自建版只持公钥）——本仓库不包含、也不伪装包含它。
3. **前端 token 存储**：管理台 token 存于 `localStorage`（XSS 可窃）。
   已有 CSP 与会话绑定缓解；迁移到 HttpOnly Cookie 在路线图中。
4. **第三方登录凭据**（`auth_identities` 表）尚未纳入字段级加密，与 §3
   同机制迁移即可。

## 9. 上线安全清单（运维）

- [ ] `JWT_SECRET`：`openssl rand -hex 32`，四环境各不相同
- [ ] `TOTP_ENCRYPTION_KEY`：显式配置并离线备份（凭据加密的前置条件）
- [ ] `POSTGRES_PASSWORD` / `REDIS_PASSWORD`：强随机，不使用示例值
- [ ] 反向代理 + HTTPS（参考 `deploy/EDGE_SECURITY.md` / `Caddyfile`）
- [ ] 初始化管理员不用弱口令（web 初始化已强制拒绝 admin/admin）
- [ ] 数据库/Redis 不暴露公网；只在内网 compose 网络互通
- [ ] 定期审计：管理台「审计日志」+ 服务器日志无凭据输出（构建保证）

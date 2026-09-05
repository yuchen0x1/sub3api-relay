# sub3api 升级说明：整合上游 Wei-Shaw/sub2api v0.2.1（提交 578785ee 系 / tag v0.2.1 = adc26f68）

基线：sub3api 0.1.205（本地提交 `cfeedd5`）。整合方式：文件级、模块级与行为级增量移植，
不整仓覆盖、不改历史迁移；新数据库字段全部 forward-only。

## 已移植内容（按提交）

### 阶段 1：高置信、低破坏
| 提交 | 内容 | 迁移 |
|---|---|---|
| `7f88e7f` | 图片 URL 回填 SSRF 加固：初始/逐跳私网与 DNS 解析校验、重定向上限、20 MiB limit+1、空响应拒绝、真实字节内容嗅探、跨源剥离认证头 | — |
| `cfc8384` | `usage_logs.upstream_request_id`：`x-request-id` 贯通三类写入 SQL、扫描列、管理员 DTO/表格/CSV；计费去重仍用逻辑 RequestID，幂等语义不变 | 203 |
| `af611f7` | `channel_model_pricing.max_reasoning_effort_multiplier`（max 等级计费倍率：渠道配置优先，fable-5.1 默认 3x，其余不配置保持 1x）；reasoning effort 映射源支持 `none` | 204 |
| `43f660b` + `f1e5f1a` | OpenAI 分组固定账号 Codex /models manifest：`groups.codex_models_manifest_config` JSONB（默认关闭），并发拉取、按配置顺序合并、`fallback_to_scheduler` 回退、管理表单 | 205 |
| `64efacb` | 管理端账户列表 `lite=1` 真正瘦身：`dto.AccountListItem`（编译期剔除嵌套 Groups/AccountGroups）+ 独立 ETag | — |

### 阶段 2：运行时稳定性与配置
| 提交 | 内容 |
|---|---|
| `dca6c90` | `SUB3API_CLAUDE_CLI_VERSION` 运维覆盖伪装 CLI 版本（严格三段 semver、不低于内置基线、进程内恒定） |
| `e57fa9b` | Fable 5 系 OAuth 伪装弃用通用 CLI 扩展提示块（上游返回 refusal/零输出） |
| `b6121a5` | 会话槽失败即释放：`UnregisterSession` + `ReleaseAccountSession`，失败请求不再占满 `max_sessions` 空闲窗口 |
| `dcfd442` | `pricing.override_file` 字段级定价覆盖（null 删字段、override-only 模型并入、无效条目 WARN），启动校验，定价 tick 热生效 |
| `1553a31` | OAuth 伪装强制内置 UA 参与 billing header 同步 + 指纹后缀按 body+版本重算；`defer_loading` 工具剥离 `cache_control` |
| `34397db` | Grok 压缩回放解码失败（400/422）同账号重试：保留可见摘要，清除空 reasoning 壳/加密状态/悬空 previous_response_id |
| `ee9c554` | Grok body 优先失败分类器（免费额度/计费/空输出/模型容量/限流/兼容性九类决策）+ 账号×模型软屏蔽 + 团队限流覆盖层；容量型 429 不再误装账号级限流 |
| `97b620f` | Grok 图片计数从共享 Responses handler 传播进结果（此前图片计费永不生效） |
| `7548186` | `security.url_allowlist` 出站 URL 白名单（upstream/pricing/crs 三组 hosts + 私网放行 + insecure http 开关），默认关闭保持历史行为 |

### 阶段 3：GPT-6 Astra（识别/路由/计费核心）
| 提交 | 内容 |
|---|---|
| `e0695bd` + `9d637f1` | `isOpenAIGPT6AstraModel`（gpt-6/gpt-6-astra/gpt-6-astra-*）；Codex 归一化与 transform 别名 gpt-6 → gpt-6-astra；`/v1/models` 默认列表新增；billing 与 pricing 双层静态定价（$10/$50 per MTok、priority x2、272k 长上下文）；Chat Completions 兼容路径 prompt cache key 注入对 Astra 放行 |

## 部署与回滚

1. 迁移 203/204/205 全部幂等、forward-only；升级前照例做数据库备份。
2. 应用顺序：先备份 → 停旧实例 → 执行迁移 → 启新二进制。回滚应用镜像安全（新字段为可空/带默认，旧代码不读写）。
3. 新配置均为可选：`pricing.override_file`、`security.url_allowlist`、`SUB3API_CLAUDE_CLI_VERSION` 不配置即保持旧行为。
4. Codex 固定账号 manifest 与 URL 白名单默认关闭，需管理员显式启用。

## 明确未移植项与理由

| 项 | 理由 |
|---|---|
| 上游 Codex 模型元数据能力文件（428 行）及 manifest 上下文窗口/Fast 档细化 | 与上游 manifest 服务深度耦合（4 个调用点 + UpstreamModelMetadata 管线），需与本地 fork 版 manifest 服务整体对接，单独立任务 |
| `stampGrokQuotaSnapshotForPlan` 的 grok-4.5 计划信号 | 本地 `xai.QuotaSnapshot` 无 `Model` 字段；快照观测已保留 |
| Grok 搜索逐调用计数（searchCount） | 本地搜索计费为按成功请求计（`WebSearchCalls=1`），属 fork 有意设计 |
| URL 白名单对 xai OAuth / CRS / 代理订阅等次要调用点收紧 | 已保持历史 http+https 行为（`allowInsecureHTTP=true`），按域收紧属运维决策 |
| 上游 `identity_service` 指纹无重置入口 | 本地已有 `DeleteFingerprint` 管理员重置，领先上游 |
| 上游工具名动态映射 5 工具门槛 | 本地已改为无门槛（少量非核心工具也需混淆），领先上游 |

## 阶段 4：插件系统评估（结论：暂不移植）

上游插件系统（迁移 229/230）为「管理员上传本地进程二进制插件」：
`sub2api_plugin_installations` 保存 binary_path/binary_sha256/signature_status
（trusted/unsigned 两级）/config_encrypted，状态机 disabled→starting→enabled→error。
评估要点：

- **执行隔离**：上游为同进程内本地执行，无沙箱边界；插件崩溃直接影响网关进程，
  与 sub3api 现有"零第三方进程内代码"的安全姿态冲突。
- **供应链**：unsigned 状态仅是标记不是拦截；二进制来源审计、哈希钉扎与升级回滚
  需要额外的制品存储与审计链（迁移 230 仅 4 行，制品细节在上游代码里）。
- **权限**：插件与网关同权限运行，可触及数据库凭据与上游 token；当前无最小权限模型。
- **资源限制**：未见 CPU/内存/并发上限。
- **结论**：只有当出现明确需求且满足"带签名的独立进程 + 资源限额 + 最小权限凭据代理"
  三项边界时，才单独立项移植；不与网关升级混合。

## 验证状态

- 后端：`go build ./...`、`go test ./...`（含 `-tags=unit`）全部通过。
- 前端：`pnpm typecheck`、`pnpm lint:check`、`vitest run`（1509+ 通过）、`pnpm build` 通过。
- 待办：真实 PostgreSQL 上执行 `make integration`（迁移 203-205 的 schema 断言已写入
  `migrations_schema_integration_test.go`，需集成环境）；关键管理界面浏览器验证。

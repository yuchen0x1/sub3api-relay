# UPGRADE_LOG

## 阶段 0 — 门禁与工程卫生（2026-11-10 12:00–12:25）

| 项 | 文件 | 说明 |
| --- | --- | --- |
| 统一换行符 | `.gitattributes`（新增） | 解决 Windows CRLF 导致 gofmt 大面积误报 |
| 构建上下文治理 | `.gitignore` / `.dockerignore` | 排除 `release-assets/`，上下文 4400MB → 51MB |
| 密钥扫描治理 | `.gitleaks.toml`（新增） | 历史误报 135 → 0；真实密钥仍可检出（已反向验证） |
| 依赖更新自动化 | `.github/dependabot.yml`（新增） | 6 个生态分组；gin 次版本、jsdom 主版本人工处理 |
| 前端门禁 | `.github/workflows/backend-ci.yml` | 新增 frontend job（lint/typecheck/vitest） |
| 密钥扫描接线 | `.github/workflows/security-scan.yml` | 显式传 `GITLEAKS_CONFIG` |
| gofmt 修复 | `user_notification_service.go` | HEAD 缺结尾换行，正是 CI lint 失败原因 |

验证：`go build ./...`、`go vet ./...`、`pnpm lint:check`、`pnpm typecheck`、4 个 YAML 解析全部通过。

### 关键教训

gitleaks 白名单最初写成 `[0-9a-f]{64}` 与 `xxxx[0-9a-f]*`，反向测试证明会把**真实高熵十六进制密钥**一起放行。改为逐个枚举 36 个实际校验和值。

---

## 阶段 1 — 修复真实缺陷（2026-11-10 12:25–13:10）

### 1. simple 模式漏建 Anthropic 默认分组（产品缺陷）

- 文件：`backend/internal/repository/simple_mode_default_groups.go`
- 问题：`requiredByPlatform` 缺少 `service.PlatformAnthropic`，而 `ent.go:94` 的注释明确要求 "anthropic/openai/gemini: 确保存在 `<platform>-default`"。simple 模式下新建实例无法获得 Anthropic 默认分组。
- 修复：补上 Anthropic 条目。
- 附带改进：把 `map[string]int` 改为有序切片。map 遍历顺序随机，导致首次启动时各分组创建顺序不稳定，影响日志、审计与测试断言。

### 2. `RefreshStaticProxyLiveSlot` 接口与实现不匹配（运行时缺陷）

- 文件：`backend/internal/repository/concurrency_cache.go`
- 问题：`service.StaticProxyLiveConcurrencyCache`（`concurrency_service.go:75`）声明了该方法，生产代码 `static_proxy_concurrency.go:138` 会调用，但 `concurrencyCache` 未实现。任何长连接续期都会落到 `"static proxy live concurrency cache is unsupported"`。
- 修复：复用现有 `refreshLiveLeaseScript`，在同一 Lua 脚本内同时续期聚合键与分配键，保持 Redis Cluster 哈希标签一致。
- 新增测试（`concurrency_cache_live_test.go`，miniredis，无需 Docker）：
  - `TestStaticProxyLiveSlotRefresh` — 续期成功、计数不重复、非持有者续期被拒、续期后跨半个 TTL 仍存活
  - `TestStaticProxyLiveSlotRefreshFailsAfterExpiry` — 过期后续期返回 false 且计数归零
  - `TestStaticProxyLiveSlotRefreshRejectsInvalidInput` — 5 种非法入参安全返回
  - 结果：3 个测试 + 5 个子测试全部通过

### 3. 调度快照测试与安全设计冲突（测试过时）

- 文件：`backend/internal/repository/scheduler_cache_integration_test.go`
- 问题：断言快照里保留 `api_key`，但 `scheduler_cache.go:957` 的注释与 `filterSchedulerCredentials` 白名单明确要求 `api_key` 绝不能写入 Redis。
- 修复：改断言为「密钥被剔除」，并注释说明网关会从数据库重新水合凭据。**未修改代码的安全行为。**

### 4. PG18 约束定义文本断言过时（测试脆弱）

- 文件：`backend/internal/repository/migrations_schema_integration_test.go`
- 问题：断言 `"max_reasoning_effort_multiplier IS NULL OR ... > 0"` 精确串，PG18 会规范化为带括号与 `::numeric` 转换的文本。
- 修复：改为断言语义片段（列名、`IS NULL`、`> `），不依赖大版本排版。

### 5. Grok outbox 回滚测试构造错误（测试 bug）

- 文件：`backend/internal/repository/account_repo_integration_test.go`
- 问题：`failAtomicSchedulerOutboxSQLExecutor` 把 `args[len(args)-1]` 置 nil，注释称要让 `event_type` 非空约束失败。但该 SQL 的 `$9`（即最后一个参数）是 `expectedDigest`，`$8` 才是 `event_type`。把摘要置空只会让 CAS 守卫不匹配、UPDATE 影响 0 行，INSERT 无行可插，语句既不报错也不回滚 —— 于是测试看到 `err == nil`。
- 修复：改为按参数语义定位 `args[7]`（`$8` = event_type），并注释说明踩坑过程。
- 结论：**生产 SQL 本身正确**（单条 `WITH ... UPDATE ... INSERT` 天然原子），此前失败纯属测试构造错误。

验证：`go vet -tags=integration ./internal/repository/` 通过；`go test -tags=unit ./internal/repository/` 通过（2.541s）。

### 环境限制（未能在本机验证的部分）

Docker 守护进程未启动，`integration_harness_test.go` 的 `TestMain` 在非 CI 环境检测不到 Docker 时会 `os.Exit(0)` 静默跳过。因此上述集成测试改动**本机无法实跑**，只能保证编译与 vet 通过。需要 CI 或本地启动 Docker 后才算真正验证。

---

## 阶段 2 — 前端测试彻底转绿 + 发现产品缺陷（2026-11-10 12:40–13:40）

### 结果：226 文件 / 1538 用例全部通过，Errors 0，退出码 0（连续 3 次稳定）

此前状态：1538 用例通过但 **108 个 EnvironmentTeardownError**，vitest 退出码 1，CI 无法作为门禁。

### 根因（逐层实证，非"环境抖动"）

1. `AccountsView.vue` 用 `defineAsyncComponent` 声明 13 个弹窗，模板中**始终渲染**（`:show` 只控制可见性）→ `mount()` 立即启动全部动态 import。
2. 实测 `defineAsyncComponent` 的 loader **不会被 `stubs` 阻止**（探针中 `loadCount=1`），`vi.mock` 模块级拦截又会引发 `__isTeleport`/`__isKeepAlive` 等内部标志连锁缺失。
3. 真实弹窗内部调用 `useI18n()`。测试只 `vi.mock('vue-i18n')` 替换了 `useI18n`，**却没有安装 i18n 插件**，真实组件渲染时抛 `Need to install with app.use function`。
4. 该异常发生在测试结束、jsdom 销毁之后 → Vitest 归类为 `EnvironmentTeardownError`。

### 修复

- 新增 `src/__tests__/helpers/test-i18n.ts`：`createTestI18n()` 提供真实 i18n 实例（空 messages，`t(key)` 原样返回 key，与原 mock 行为等价）；`settleAsyncComponents()` 等待二级动态 import 收敛。
- 5 个 `AccountsView.*.spec.ts`：移除 `vue-i18n` 局部 mock，改为 `plugins: [testI18n]`；补充 `SettingRevisionsModal`/`ImportAccountsModal` stub；新增 `trackMount` + `afterEach`（先 settle 再 unmount）。
- `frontend/vitest.config.ts`：保持默认池（实测显式 `pool: 'forks'` 反而引入 4 个错误）。

### 附带发现并修复的真实产品缺陷

**`SettingRevisionsModal` 未注册（SettingsView.vue）**

- 模板第 19 行使用 `<SettingRevisionsModal>`，第 15 行的“修订历史”按钮会设 `showRevisionsModal = true`，但该组件**从未 import 或声明**。
- 后果：点按钮无任何反应；Vue 仅在控制台报 `Failed to resolve component`。
- 修复：补 `defineAsyncComponent(() => import('@/components/admin/settings/SettingRevisionsModal.vue'))`。
- 这是全仓唯一一处真实缺失注册：写了脚本扫描全部 309 个 `.vue` 文件的模板标签与注册表，其余 31 处告警均为 `Teleport`/`KeepAlive`/`RouterLink` 等全局组件，或单行多 import 导致的正则误报。

### window.location 重定义（65 个 vmThreads 失败）

- 新增 `src/__tests__/helpers/test-location.ts`（基于 `history.replaceState`，不触碰属性描述符）。
- 实证结论：`defineProperty` / `delete` + `defineProperty` / `vi.stubGlobal` 三条路径在 `--pool=vmThreads` 下**全部**被 jsdom 拒绝，这是池模式本身的限制，不是测试写法问题。
- 决策：默认池已 0 错误，不为此改写 22 处测试；改为在 vitest 配置中保持默认池并在文档说明，避免有人为性能切池后误判成功能回归。

---

## 阶段 3 — 依赖升级（2026-11-10 15:20–16:55）

### 甲批（已完成并验证）

`go get -u` 逐项升级，全部通过 `go build ./...` + `go test -tags=unit ./...`（53 个包全绿）：

| 模块 | 旧 | 新 |
| --- | --- | --- |
| golang.org/x/image | v0.47.x | v0.48.0 |
| golang.org/x/crypto | v0.57.x | v0.58.1-0.20261009181718 |
| klauspost/compress | v1.20.3 | v1.20.4 |
| andybalholm/brotli | v1.2.x | v1.3.0 |
| tidwall/gjson | v1.20.x | v1.20.2 |
| tidwall/sjson | v1.2.x | v1.2.9 |
| stretchr/testify | v1.11.x | v1.12.1 |
| redis/go-redis/v9 | v9.22.x | v9.23.0 |
| alicebob/miniredis/v2 | v2.39.x | v2.40.0 |
| alitto/pond/v2 | v2.6.x | v2.7.3 |
| coder/websocket | v1.8.x | v1.8.15 |
| shopspring/decimal | v1.5.x | v1.6.0 |
| shirou/gopsutil/v4 | v4.26.9 | v4.26.10 |
| go-webauthn/webauthn | v0.17.4 | v0.18.2 |
| imroc/req/v3 | v3.60.x | v3.61.0 |
| lib/pq | v1.12.x | v1.12.3 |

**需要改代码的唯一破坏性变更：go-webauthn v0.18**

- `protocol.AuthenticationExtensions{...}` 不再被 `webauthn.WithExtensions` 接受，改为 ceremony 校验的 `ExtensionOption`。
- `credProps` 字段被移除，改用专用构造器。
- 修复（`internal/service/passkey.go:206`）：`webauthn.WithExtensions(webauthn.WithExtensionCredProps())`
- 验证：`go test -tags=unit -run 'Passkey|WebAuthn|CredProps|Extension' ./internal/service/` → 5 个 passkey 用例全部 PASS。

**govulncheck**：升级前有 6 个可达漏洞，升级后 `No vulnerabilities found`。

### 乙批（未执行，留待下轮）

`zap 1.24 → 1.28`、`viper 1.18 → 1.21`、`ent 0.14.5 → 0.14.6`、`modernc/sqlite 1.44 → 1.62`、`go-zero 1.9.4`。
原因：`modernc/sqlite` 跨 18 个小版本，其 C 翻译层变更可能影响 SQLite 集成测试行为，而本机 Docker 未启动导致集成测试无法实跑。在**无法验证**的前提下升级属于"未验证即宣称完成"，违反任务约束。已记入 `NEXT_UPGRADE_PLAN.md`。

# CodeQL 安全与质量问题复核（2026-11-19）

范围：Issue #27，以及默认分支 `779b4351f2fcc34b25f772d6abebfa5d402c3cc1` 的全部 **162 条开放 CodeQL 告警**。使用分页 API 完整取数；不是首次分页的 100 条，也不是旧评论中的 41 条日志告警。扫描配置、规则与排除路径均未为本轮修复作改动。

## v0.5.7 补充复核（2026-11-27）

下文计数和“开放迁移项”表保留首次复核的历史基线。本次在默认分支 `2be3bec9a93486d286e7bc403fe6e2da95d23528` 的 Go 分析 `2256804145` 中重新检查仍开放的路径，未缩减扫描规则或扩大排除：

| 告警 | 本次复核证据 |
| --- | --- |
| #43、#44 | Gateway 转发失败的错误在日志构造前经过 `logredact.ErrorSummary`；输出限于 Go 类型、固定分类、三位 HTTP 状态与允许列表 provider code。代理字段为 ID、名称、host、port，未输出用户名、密码或完整 URL。 |
| #45、#46、#47 | OpenAI 普通及 WebSocket 转发错误同样使用 `ErrorSummary`。CodeQL 将 `fmt.Sprintf("type=%T kind=%s", err, kind)` 中的类型格式化误判为输出错误正文；`%T` 不调用 `Error()` 或自定义 Formatter。 |
| #48、#49 | Images 转发失败的 Warn/Error 分支同样只记录上述固定摘要。 |
| #50、#51、#52、#53、#54 | `LegacyPrintf` 在格式化前替换所有 `error` 参数；扫描路径把该参数传播至类型断言失败的另一分支。其他来源是数字 `APIKeyID`、布尔 `APIKeyEnabled` 和已经脱敏的认证头，并非原始 API Key。bootstrap 与结构化 sink 均经回归验证。 |
| #156、#158 | 保活 writer 透明转发 JSON/SSE，生产响应限定 JSON/`+json` 或 `text/event-stream`，并带 `nosniff`。SARIF 将构建上游 multipart 请求的 `multipart.Writer.CreatePart().Write(model)` 与下游 HTTP writer 合流，实际不是同一 receiver；WS bridge 则发送 SSE 帧，不渲染 HTML。新增已提交心跳的 `nosniff` 防御。 |

上述 14 条属于已存在安全边界未被分析器识别的误报，处置须在 GitHub 告警中保留对应理由，不能概括为“全部上游输入可信”。已执行日志脱敏、恶意 HTML/SVG/XHTML 媒体类型、API 字节透明性与心跳响应头定向测试，均通过。

#5、#163、#164 的真实迁移在 v0.5.7 实现：新 Web 会话不再持久写入访问与刷新令牌，同源刷新使用 HttpOnly Cookie；持久凭据新写入使用带版本和域分离的 HMAC，配套旧行双读、严格解密及 CAS 分批回填。跨源 Web 的新访问与刷新令牌仅驻留页面内存，重载需重新登录；未升级客户端与不能解密的旧数据仍有明确边界，见 [安全边界](SECURITY_BOUNDARY.md)。Issue #27 仍须在合并后的扫描及兼容检查确认后再关闭。

## 处置口径

“已修复/加固”包含真实漏洞修复与阻止上游错误回显秘密的预防性加固，不表示 70 条原始数据流全部已证实可利用。该列描述代码工作；默认分支告警是否自动关闭须以合并后的 CodeQL 重扫为准。误报与保留项在 GitHub 上逐条附证据处理，真实风险不冒充误报。

| 处置 | 条数 |
| --- | ---: |
| 已修复/加固 | 70 |
| 误报 | 77 |
| 保留的设计/协议 | 12 |
| 开放迁移项 | 3 |

## 已落实的修复

- 代理 TLS 连接池日志不再输出包含用户名、密码的缓存键；日志端点去掉 query/path/fragment，并避免输出 VMess 编码节点凭据。连接缓存本身仍按完整凭据隔离。
- 本次复核的上游、代理和 OAuth 错误日志改为固定诊断摘要，保留事件、账号、类型、网络分类、HTTP 状态和允许列表错误码；相关任意错误正文、WebSocket 对端 close reason 与相邻原始响应正文不直接入普通运行日志。管理台 Ops 的受限错误详情仍沿用已有结构化清洗与长度限制，不将本轮测试扩大解释为任意内容的完整脱敏证明。
- Markdown 和页面图片使用 `os.Root` 约束访问，并从已检查的同一文件句柄读取，移除检查路径后重新打开的竞态；Markdown 读取保持体积上限。
- WebSocket 可选重放历史设 64 MiB 累计预算，计算条目和数据开销；原生续链保留锚点，必须重放且无完整历史的 HTTP bridge 明确以 1009 关闭，避免部分历史重放。
- 配额时间字段在整数转换前检查范围；JSON/SSE 输出约束媒体类型并设置 nosniff，合法 `application/*+json` 保留。
- 注册密码及一次性挑战仅在当前页面内存传递，sessionStorage 只保留邮箱。刷新验证页后返回注册表单重输密码，保留邮箱；读取旧版缓存时移除秘密字段。
- 自定义 Markdown 先完整清洗、通过 DOM 添加标题 ID，再清洗序列化结果；验证码 DOM ID 使用 Vue 实例 ID，去掉无安全用途的随机数调用。
- 额外发现并修复凭据加密失败后静默写入明文的行为：敏感值加密、序列化或显式密钥初始化失败时，整次凭据写入被拒绝，包含批量部分更新路径。

## 必须保持开放的 3 条真实风险

- **#5：浏览器会话令牌。** refresh token 和 companion access token 仍存 localStorage。同源脚本可读。迁移 HttpOnly Cookie 需同步处理刷新轮换、CSRF、跨域/CORS、OAuth、跨标签页协调及桌面客户端；单改 sessionStorage 无法修复 XSS，改为纯内存会改变刷新/新标签页登录行为。本轮保留认证契约，不将该项关闭。
- **#163、#164：持久凭据摘要。** 无盐 SHA-256 用于加密后的 CAS/去重，不用于用户密码认证。供应商高熵 Key 的指纹合理，但自定义低熵凭据在数据库被读取后可能被离线猜测。应迁移到带版本和域分离的 HMAC，配套稳定密钥、双读匹配、旧行解密回填、部分更新及轮换，不能直接替换导致旧数据比较失效。

上述迁移项保留在 Issue #27 跟踪；本报告不宣称 162 条漏洞全部消除。详见 [安全边界](SECURITY_BOUNDARY.md)。

## 验证

- **通过**：`go test -tags=unit ./...`；`golangci-lint v2.9.0 run --timeout=30m ./...`（与 CI 的 v2.9 系列一致）。
- **通过**：前端 `pnpm run lint:check`、`pnpm run typecheck`、`pnpm run test:run`（231 个文件，1555 个用例）及 `pnpm run build`。
- **通过**：`govulncheck` 未发现可达漏洞；前端生产依赖官方 npm 审计接口通过。原配置镜像不提供审计端点，审计命令单次指定官方 registry，未改项目 registry 或锁文件。
- **通过**：代理日志创建/复用/淘汰、错误摘要、注册密码内存传递/刷新恢复、恶意 Markdown、累计重放预算、媒体类型、加密失败阻断及普通日志正文旁路的定向回归。
- Windows 本地无法创建符号链接的测试会明确跳过；Linux CI 应覆盖这些测试。完整连接级 WebSocket 超限测试未覆盖。本地未启动依赖容器运行 integration 组，需以 GitHub Linux CI 为门禁。
- 本机预装 golangci-lint v2.15 的更高版本额外诊断未用于修改无关代码；最终本地门禁使用与 CI 相同的 v2.9.0，通过后未增加任何全局排除。

## 逐条证据

以下位置是上述基线提交的扫描位置，不保证等于修复后的当前行号。机器可读清单见 [JSON](security-codeql-triage-2026-11-19.json)。

| 编号 | 规则 | 基线位置 | 处置 | 复核依据 |
| ---: | --- | --- | --- | --- |
| #2 | `js/incomplete-multi-character-sanitization` | `frontend/src/views/user/CustomPageView.vue:253` | 已修复/加固 | HTML was already sanitized by DOMPurify, but headings were reconstructed with partial tag regexes after sanitization. The new detached DOM path reads textContent, assigns generated IDs, and sanitizes serialized HTML again. Added malicious attributes/URLs and multiline-heading regression coverage. |
| #3 | `js/clear-text-storage-of-sensitive-data` | `frontend/src/views/admin/UsersView.vue:1182` | 误报 | UsersView persists apiKeyGroup as a numeric filter ID; loading checks typeof number. It is neither an API key nor a credential. Reviewed saved filter structure; dismissed on GitHub with this evidence. |
| #4 | `js/clear-text-storage-of-sensitive-data` | `frontend/src/views/auth/RegisterView.vue:501` | 已修复/加固 | Registration previously wrote the user's password and one-time Turnstile proof to sessionStorage. Only email metadata is now stored; secrets are consumed once from module memory. Reloaded verification requires password re-entry via the registration form and retains the email. Legacy stored secrets are scrubbed; ordinary registration and reload recovery have regression tests. |
| #5 | `js/clear-text-storage-of-sensitive-data` | `frontend/src/api/auth.ts:82` | 开放迁移项 | refresh_token (and the companion access token) remains readable by same-origin JavaScript in localStorage. Replacing storage only with sessionStorage does not fix XSS. Existing refresh rotation, Web Locks, peer reconciliation, OAuth, SSE and desktop flows require a coordinated HttpOnly-cookie/CSRF/CORS migration, or explicit acceptance of losing reload/new-tab login for memory-only storage. Authentication behavior is preserved; this genuine risk remains open. |
| #6 | `js/clear-text-storage-of-sensitive-data` | `frontend/src/utils/oauthAffiliate.ts:38` | 误报 | oauthAffiliate persists a normalized referral attribution code, not an authentication token. Its TTL and consumption lifecycle are enforced; knowing a referral code does not grant user or API access. |
| #7 | `js/clear-text-storage-of-sensitive-data` | `frontend/src/utils/oauthAffiliate.ts:94` | 误报 | The second oauthAffiliate storage path holds the same non-secret referral attribution code, not an OAuth credential. Storage is bounded by expiry and cleared after registration; not sensitive-data cleartext storage. |
| #20 | `js/insecure-randomness` | `frontend/src/components/AliyunCaptchaWidget.vue:94` | 已修复/加固 | Math.random was used only to namespace captcha SDK DOM IDs, never to verify a challenge. Replaced with Vue useId, preserving insecure-context self-hosting support and same-app instance uniqueness. Captcha verification still depends on the SDK/server proof, not this identifier. |
| #24 | `go/incorrect-integer-conversion` | `backend/internal/service/account.go:2437` | 已修复/加固 | ComputeQuotaResetAt now checks the truncated daily hour against 0–23 before narrowing to int, preserving legacy round-toward-zero behavior. |
| #25 | `go/incorrect-integer-conversion` | `backend/internal/service/account.go:2456` | 已修复/加固 | ComputeQuotaResetAt now bounds the weekly hour before narrowing. The neighboring weekday is checked before narrowing too. |
| #26 | `go/incorrect-integer-conversion` | `backend/internal/service/account.go:2488` | 已修复/加固 | NormalizeFixedQuotaWindows checks the daily hour before conversion, using midnight for out-of-range or nonfinite values. |
| #27 | `go/incorrect-integer-conversion` | `backend/internal/service/account.go:2508` | 已修复/加固 | NormalizeFixedQuotaWindows bounds weekday and weekly hour before conversion, preventing architecture-dependent conversion during rollover. |
| #28 | `go/uncontrolled-allocation-size` | `backend/internal/repository/user_notification_repo.go:112` | 误报 | userNotificationRepository.List bounds limit to at most 100 with a positive default before the make. The SQL lookup also limits returned rows to limit+1 and scopes them by user ID. |
| #29 | `go/uncontrolled-allocation-size` | `backend/internal/securityaudit/prompt_event_repository.go:87` | 误报 | PostgreSQLRepository.ListEvents itself normalizes pageSize to 1–100 before allocating, including direct repository callers. |
| #31 | `go/uncontrolled-allocation-size` | `backend/internal/service/admin_user.go:1318` | 误报 | Production call site is the admin Generate handler. Its sole count input passes Gin min=1,max=100 binding before calling GenerateRedeemCodes. |
| #32 | `go/request-forgery` | `backend/internal/pkg/proxyutil/vless.go:99` | 保留的设计/协议 | VLESS handshake host and port come from the administrator-configured proxy record or an administrator's imported subscription entry. The client's tunneled destination cannot override the WebSocket handshake host. Internal proxies are supported. |
| #33 | `go/request-forgery` | `backend/internal/pkg/proxyutil/vmess.go:264` | 保留的设计/协议 | VMess handshake constructs its authority from configured proxy ServerHost/ServerPort. Proxy creation, import, and testing are admin routes. Network policy for the operator's chosen proxy is a deployment decision. |
| #34 | `go/request-forgery` | `backend/internal/securityaudit/prompt_qwen3guard.go:224` | 保留的设计/协议 | The worker chooses ActiveEndpoint from administrative prompt audit configuration. Chunk text is inserted into the request JSON, while ChatCompletionsURL uses only the configured HTTP(S) base and a fixed suffix. Private Guard is a supported deployment. |
| #35 | `go/request-forgery` | `backend/internal/securityaudit/prompt_service.go:251` | 保留的设计/协议 | The probe accepting a candidate BaseURL is behind /admin authentication. NormalizeBaseURL rejects unsupported schemes, userinfo, queries and fragments. Stored secrets are reapplied only on exact normalized URL match. Its dialer intentionally allows internal Guard installations. |
| #36 | `go/request-forgery` | `backend/internal/service/content_moderation.go:1765` | 保留的设计/协议 | Moderation uses the saved risk-center base URL or the admin-only API-key test configuration. End users' prompts and images populate request JSON only. Internal moderation deployments and configured proxy selection are supported. |
| #37 | `go/request-forgery` | `backend/internal/service/crs_sync_service.go:1324` | 保留的设计/协议 | CRS sync and preview are administrator routes. The submitted and validated HTTP(S) authority is the intended remote CRS instance, including private instances, and this login sends the submitted CRS credentials to its fixed login path. |
| #38 | `go/request-forgery` | `backend/internal/service/crs_sync_service.go:1362` | 保留的设计/协议 | CRS export follows the login on the same admin-specified base and appends a fixed export path. The Authorization token comes from that configured CRS login, with no endpoint parameter available to ordinary users. |
| #39 | `go/request-forgery` | `backend/internal/service/openai_codex_models_service.go:469` | 误报 | The flagged client.Do is the OAuth branch, whose requestEndpoint is chatgptCodexModelsURL. Client version is encoded in the query. Admin-configured API-key base URLs take the separate validated httpUpstream.Do branch. |
| #40 | `go/request-forgery` | `backend/internal/service/openai_ws_client.go:128` | 保留的设计/协议 | Gateway callers derive the WebSocket URL from fixed OpenAI endpoints or stored administrator-managed API-key account base URLs, then validate the HTTP base and translate its scheme. Ordinary users may trigger scheduling but cannot supply the dial URL. |
| #41 | `go/request-forgery` | `backend/internal/service/vertex_service_account.go:254` | 误报 | The production exchange caller obtains its key through parseVertexServiceAccountKey and parseVertexServiceAccountJSON, which overwrites imported token_uri unconditionally with https://oauth2.googleapis.com/token before request construction. |
| #42 | `go/clear-text-logging` | `backend/internal/handler/admin/openai_oauth_handler.go:628` | 已修复/加固 | 上游/代理/OAuth错误日志已修复或加固：可能带URL认证信息、API查询密钥或响应体回显。改为固定诊断摘要并保留事件上下文，避免任意错误内容进入日志。 |
| #43 | `go/clear-text-logging` | `backend/internal/handler/gateway_handler.go:545` | 已修复/加固 | 上游/代理/OAuth错误日志已修复或加固：可能带URL认证信息、API查询密钥或响应体回显。改为固定诊断摘要并保留事件上下文，避免任意错误内容进入日志。 |
| #44 | `go/clear-text-logging` | `backend/internal/handler/gateway_handler.go:1057` | 已修复/加固 | 上游/代理/OAuth错误日志已修复或加固：可能带URL认证信息、API查询密钥或响应体回显。改为固定诊断摘要并保留事件上下文，避免任意错误内容进入日志。 |
| #45 | `go/clear-text-logging` | `backend/internal/handler/openai_gateway_handler.go:677` | 已修复/加固 | 上游/代理/OAuth错误日志已修复或加固：可能带URL认证信息、API查询密钥或响应体回显。改为固定诊断摘要并保留事件上下文，避免任意错误内容进入日志。 |
| #46 | `go/clear-text-logging` | `backend/internal/handler/openai_gateway_handler.go:680` | 已修复/加固 | 上游/代理/OAuth错误日志已修复或加固：可能带URL认证信息、API查询密钥或响应体回显。改为固定诊断摘要并保留事件上下文，避免任意错误内容进入日志。 |
| #47 | `go/clear-text-logging` | `backend/internal/handler/openai_gateway_handler.go:2288` | 已修复/加固 | 上游/代理/OAuth错误日志已修复或加固：可能带URL认证信息、API查询密钥或响应体回显。改为固定诊断摘要并保留事件上下文，避免任意错误内容进入日志。 |
| #48 | `go/clear-text-logging` | `backend/internal/handler/openai_images.go:357` | 已修复/加固 | 上游/代理/OAuth错误日志已修复或加固：可能带URL认证信息、API查询密钥或响应体回显。改为固定诊断摘要并保留事件上下文，避免任意错误内容进入日志。 |
| #49 | `go/clear-text-logging` | `backend/internal/handler/openai_images.go:360` | 已修复/加固 | 上游/代理/OAuth错误日志已修复或加固：可能带URL认证信息、API查询密钥或响应体回显。改为固定诊断摘要并保留事件上下文，避免任意错误内容进入日志。 |
| #50 | `go/clear-text-logging` | `backend/internal/pkg/logger/logger.go:490` | 已修复/加固 | 共享入口已加固：LegacyPrintf对error参数在格式化前使用安全摘要，覆盖初始化前log.Print和初始化后各zap级别；保留ID、计数、正常字符串。不是声称每条合并数据流均可利用：Ent查询类型、数字ID、已脱敏authorization/x-api-key分支本身是误报；上游错误对象可能含完整URL/任意响应体，统一阻止其回显。 |
| #51 | `go/clear-text-logging` | `backend/internal/pkg/logger/logger.go:502` | 已修复/加固 | 共享入口已加固：LegacyPrintf对error参数在格式化前使用安全摘要，覆盖初始化前log.Print和初始化后各zap级别；保留ID、计数、正常字符串。不是声称每条合并数据流均可利用：Ent查询类型、数字ID、已脱敏authorization/x-api-key分支本身是误报；上游错误对象可能含完整URL/任意响应体，统一阻止其回显。 |
| #52 | `go/clear-text-logging` | `backend/internal/pkg/logger/logger.go:504` | 已修复/加固 | 共享入口已加固：LegacyPrintf对error参数在格式化前使用安全摘要，覆盖初始化前log.Print和初始化后各zap级别；保留ID、计数、正常字符串。不是声称每条合并数据流均可利用：Ent查询类型、数字ID、已脱敏authorization/x-api-key分支本身是误报；上游错误对象可能含完整URL/任意响应体，统一阻止其回显。 |
| #53 | `go/clear-text-logging` | `backend/internal/pkg/logger/logger.go:506` | 已修复/加固 | 共享入口已加固：LegacyPrintf对error参数在格式化前使用安全摘要，覆盖初始化前log.Print和初始化后各zap级别；保留ID、计数、正常字符串。不是声称每条合并数据流均可利用：Ent查询类型、数字ID、已脱敏authorization/x-api-key分支本身是误报；上游错误对象可能含完整URL/任意响应体，统一阻止其回显。 |
| #54 | `go/clear-text-logging` | `backend/internal/pkg/logger/logger.go:508` | 已修复/加固 | 共享入口已加固：LegacyPrintf对error参数在格式化前使用安全摘要，覆盖初始化前log.Print和初始化后各zap级别；保留ID、计数、正常字符串。不是声称每条合并数据流均可利用：Ent查询类型、数字ID、已脱敏authorization/x-api-key分支本身是误报；上游错误对象可能含完整URL/任意响应体，统一阻止其回显。 |
| #55 | `go/clear-text-logging` | `backend/internal/pkg/websearch/manager.go:118` | 已修复/加固 | 上游/代理/OAuth错误日志已修复或加固：可能带URL认证信息、API查询密钥或响应体回显。改为固定诊断摘要并保留事件上下文，避免任意错误内容进入日志。 |
| #56 | `go/clear-text-logging` | `backend/internal/pkg/websearch/manager.go:124` | 已修复/加固 | 上游/代理/OAuth错误日志已修复或加固：可能带URL认证信息、API查询密钥或响应体回显。改为固定诊断摘要并保留事件上下文，避免任意错误内容进入日志。 |
| #57 | `go/clear-text-logging` | `backend/internal/pkg/websearch/manager.go:128` | 已修复/加固 | 上游/代理/OAuth错误日志已修复或加固：可能带URL认证信息、API查询密钥或响应体回显。改为固定诊断摘要并保留事件上下文，避免任意错误内容进入日志。 |
| #58 | `go/clear-text-logging` | `backend/internal/repository/billing_cache.go:362` | 误报 | 记录的是 int64 数据库 API Key ID（%d），不是 APIKey.Key；Redis错误也不包含API key值。保留诊断ID。 |
| #60 | `go/clear-text-logging` | `backend/internal/repository/http_upstream.go:286` | 已修复/加固 | 上游/代理/OAuth错误日志已修复或加固：可能带URL认证信息、API查询密钥或响应体回显。改为固定诊断摘要并保留事件上下文，避免任意错误内容进入日志。 |
| #61 | `go/clear-text-logging` | `backend/internal/repository/http_upstream.go:516` | 已修复/加固 | 确认泄露并修复：连接缓存键由包含userinfo的代理URL.String构造，创建/复用/淘汰日志直接输出。原proxy字段在VMess场景还会输出含认证节点JSON的base64 Host。现在日志只用允许协议+endpoint，VMess用占位符，保留真实缓存键用于隔离。 |
| #62 | `go/clear-text-logging` | `backend/internal/repository/http_upstream.go:530` | 已修复/加固 | 确认泄露并修复：连接缓存键由包含userinfo的代理URL.String构造，创建/复用/淘汰日志直接输出。原proxy字段在VMess场景还会输出含认证节点JSON的base64 Host。现在日志只用允许协议+endpoint，VMess用占位符，保留真实缓存键用于隔离。 |
| #63 | `go/clear-text-logging` | `backend/internal/repository/http_upstream.go:535` | 已修复/加固 | 确认泄露并修复：连接缓存键由包含userinfo的代理URL.String构造，创建/复用/淘汰日志直接输出。原proxy字段在VMess场景还会输出含认证节点JSON的base64 Host。现在日志只用允许协议+endpoint，VMess用占位符，保留真实缓存键用于隔离。 |
| #64 | `go/clear-text-logging` | `backend/internal/repository/http_upstream.go:559` | 已修复/加固 | 确认泄露并修复：连接缓存键由包含userinfo的代理URL.String构造，创建/复用/淘汰日志直接输出。原proxy字段在VMess场景还会输出含认证节点JSON的base64 Host。现在日志只用允许协议+endpoint，VMess用占位符，保留真实缓存键用于隔离。 |
| #65 | `go/clear-text-logging` | `backend/internal/repository/http_upstream.go:559` | 已修复/加固 | 确认泄露并修复：连接缓存键由包含userinfo的代理URL.String构造，创建/复用/淘汰日志直接输出。原proxy字段在VMess场景还会输出含认证节点JSON的base64 Host。现在日志只用允许协议+endpoint，VMess用占位符，保留真实缓存键用于隔离。 |
| #67 | `go/clear-text-logging` | `backend/internal/service/account_usage_service.go:1186` | 已修复/加固 | 上游/代理/OAuth错误日志已修复或加固：可能带URL认证信息、API查询密钥或响应体回显。改为固定诊断摘要并保留事件上下文，避免任意错误内容进入日志。 |
| #68 | `go/clear-text-logging` | `backend/internal/service/antigravity_gateway_retry.go:229` | 已修复/加固 | 上游/代理/OAuth错误日志已修复或加固：可能带URL认证信息、API查询密钥或响应体回显。改为固定诊断摘要并保留事件上下文，避免任意错误内容进入日志。 |
| #69 | `go/clear-text-logging` | `backend/internal/service/antigravity_oauth_service.go:153` | 已修复/加固 | 上游/代理/OAuth错误日志已修复或加固：可能带URL认证信息、API查询密钥或响应体回显。改为固定诊断摘要并保留事件上下文，避免任意错误内容进入日志。 |
| #70 | `go/clear-text-logging` | `backend/internal/service/antigravity_oauth_service.go:245` | 已修复/加固 | 上游/代理/OAuth错误日志已修复或加固：可能带URL认证信息、API查询密钥或响应体回显。改为固定诊断摘要并保留事件上下文，避免任意错误内容进入日志。 |
| #71 | `go/clear-text-logging` | `backend/internal/service/api_key_service.go:463` | 误报 | APIKey命中是Ent APIKey client/query类型及拦截器元数据，不是Key列值；此处按userID做COUNT并记录查询错误，SARIF路径中的格式化为%T类型名。 |
| #72 | `go/clear-text-logging` | `backend/internal/service/claude_token_provider.go:91` | 已修复/加固 | 上游/代理/OAuth错误日志已修复或加固：可能带URL认证信息、API查询密钥或响应体回显。改为固定诊断摘要并保留事件上下文，避免任意错误内容进入日志。 |
| #73 | `go/clear-text-logging` | `backend/internal/service/content_moderation.go:1071` | 已修复/加固 | 上游/代理/OAuth错误日志已修复或加固：可能带URL认证信息、API查询密钥或响应体回显。改为固定诊断摘要并保留事件上下文，避免任意错误内容进入日志。 |
| #74 | `go/clear-text-logging` | `backend/internal/service/gemini_oauth_service.go:543` | 已修复/加固 | 上游/代理/OAuth错误日志已修复或加固：可能带URL认证信息、API查询密钥或响应体回显。改为固定诊断摘要并保留事件上下文，避免任意错误内容进入日志。 |
| #75 | `go/clear-text-logging` | `backend/internal/service/gemini_oauth_service.go:553` | 已修复/加固 | 上游/代理/OAuth错误日志已修复或加固：可能带URL认证信息、API查询密钥或响应体回显。改为固定诊断摘要并保留事件上下文，避免任意错误内容进入日志。 |
| #76 | `go/clear-text-logging` | `backend/internal/service/gemini_oauth_service.go:600` | 已修复/加固 | 上游/代理/OAuth错误日志已修复或加固：可能带URL认证信息、API查询密钥或响应体回显。改为固定诊断摘要并保留事件上下文，避免任意错误内容进入日志。 |
| #77 | `go/clear-text-logging` | `backend/internal/service/gemini_oauth_service.go:815` | 已修复/加固 | 上游/代理/OAuth错误日志已修复或加固：可能带URL认证信息、API查询密钥或响应体回显。改为固定诊断摘要并保留事件上下文，避免任意错误内容进入日志。 |
| #78 | `go/clear-text-logging` | `backend/internal/service/gemini_token_provider.go:128` | 已修复/加固 | 上游/代理/OAuth错误日志已修复或加固：可能带URL认证信息、API查询密钥或响应体回显。改为固定诊断摘要并保留事件上下文，避免任意错误内容进入日志。 |
| #79 | `go/clear-text-logging` | `backend/internal/service/oauth_refresh_api.go:200` | 误报 | 日志中的cache_key/key是锁及访问令牌缓存的索引，不是缓存值或令牌；由平台+数字账号ID或project_id构成，服务账号使用短SHA-256指纹。 |
| #80 | `go/clear-text-logging` | `backend/internal/service/oauth_refresh_api.go:383` | 误报 | 日志中的cache_key/key是锁及访问令牌缓存的索引，不是缓存值或令牌；由平台+数字账号ID或project_id构成，服务账号使用短SHA-256指纹。 |
| #81 | `go/clear-text-logging` | `backend/internal/service/oauth_refresh_api.go:402` | 误报 | 日志中的cache_key/key是锁及访问令牌缓存的索引，不是缓存值或令牌；由平台+数字账号ID或project_id构成，服务账号使用短SHA-256指纹。 |
| #82 | `go/clear-text-logging` | `backend/internal/service/openai_token_provider.go:181` | 已修复/加固 | 上游/代理/OAuth错误日志已修复或加固：可能带URL认证信息、API查询密钥或响应体回显。改为固定诊断摘要并保留事件上下文，避免任意错误内容进入日志。 |
| #83 | `go/clear-text-logging` | `backend/internal/service/ratelimit_service.go:1683` | 误报 | GetCredential("plan_type")是订阅计划字符串，非api_key/access_token；只记录计划变更元数据。 |
| #84 | `go/clear-text-logging` | `backend/internal/service/ratelimit_service.go:2180` | 误报 | modelKey为请求模型/模型映射冷却索引，记录账号ID、模型、固定reason和reset_at；未记录任何认证密钥。 |
| #85 | `go/clear-text-logging` | `backend/internal/service/ratelimit_service.go:2183` | 误报 | modelKey为请求模型/模型映射冷却索引，记录账号ID、模型、固定reason和reset_at；未记录任何认证密钥。 |
| #86 | `go/clear-text-logging` | `backend/internal/service/subscription_expiry_service.go:135` | 误报 | withAPIKeys/withAPIKey为Ent关系查询字段，不是密钥内容；订阅过期提醒的日志参数是List失败的error。SARIF将查询结构字段整体标为敏感。 |
| #87 | `go/clear-text-logging` | `backend/internal/service/token_cache_invalidator.go:64` | 误报 | 日志中的cache_key/key是锁及访问令牌缓存的索引，不是缓存值或令牌；由平台+数字账号ID或project_id构成，服务账号使用短SHA-256指纹。 |
| #88 | `go/clear-text-logging` | `backend/internal/service/token_cache_invalidator.go:111` | 误报 | latest_version来自_token_version毫秒时间戳，和访问/刷新令牌值无关；保留版本比较诊断。 |
| #89 | `go/clear-text-logging` | `backend/internal/handler/content_moderation_helper.go:51` | 误报 | APIKeyName来自 apiKey.Name 展示名称，与认证密钥 APIKey.Key 独立；命中依赖字段名启发式，未记录密钥值。 |
| #90 | `go/clear-text-logging` | `backend/internal/handler/gateway_handler_chat_completions.go:322` | 已修复/加固 | 上游/代理/OAuth错误日志已修复或加固：可能带URL认证信息、API查询密钥或响应体回显。改为固定诊断摘要并保留事件上下文，避免任意错误内容进入日志。 |
| #91 | `go/clear-text-logging` | `backend/internal/handler/gateway_handler_responses.go:316` | 已修复/加固 | 上游/代理/OAuth错误日志已修复或加固：可能带URL认证信息、API查询密钥或响应体回显。改为固定诊断摘要并保留事件上下文，避免任意错误内容进入日志。 |
| #92 | `go/clear-text-logging` | `backend/internal/handler/gateway_handler.go:2107` | 已修复/加固 | 上游/代理/OAuth错误日志已修复或加固：可能带URL认证信息、API查询密钥或响应体回显。改为固定诊断摘要并保留事件上下文，避免任意错误内容进入日志。 |
| #93 | `go/clear-text-logging` | `backend/internal/handler/gemini_v1beta_handler.go:541` | 已修复/加固 | 上游/代理/OAuth错误日志已修复或加固：可能带URL认证信息、API查询密钥或响应体回显。改为固定诊断摘要并保留事件上下文，避免任意错误内容进入日志。 |
| #94 | `go/clear-text-logging` | `backend/internal/handler/grok_media.go:403` | 已修复/加固 | 上游/代理/OAuth错误日志已修复或加固：可能带URL认证信息、API查询密钥或响应体回显。改为固定诊断摘要并保留事件上下文，避免任意错误内容进入日志。 |
| #95 | `go/clear-text-logging` | `backend/internal/handler/grok_x_search.go:200` | 已修复/加固 | 上游/代理/OAuth错误日志已修复或加固：可能带URL认证信息、API查询密钥或响应体回显。改为固定诊断摘要并保留事件上下文，避免任意错误内容进入日志。 |
| #96 | `go/clear-text-logging` | `backend/internal/handler/openai_chat_completions.go:263` | 已修复/加固 | 上游/代理/OAuth错误日志已修复或加固：可能带URL认证信息、API查询密钥或响应体回显。改为固定诊断摘要并保留事件上下文，避免任意错误内容进入日志。 |
| #97 | `go/clear-text-logging` | `backend/internal/handler/openai_chat_completions.go:338` | 已修复/加固 | 上游/代理/OAuth错误日志已修复或加固：可能带URL认证信息、API查询密钥或响应体回显。改为固定诊断摘要并保留事件上下文，避免任意错误内容进入日志。 |
| #98 | `go/clear-text-logging` | `backend/internal/handler/openai_alpha_search.go:198` | 已修复/加固 | 上游/代理/OAuth错误日志已修复或加固：可能带URL认证信息、API查询密钥或响应体回显。改为固定诊断摘要并保留事件上下文，避免任意错误内容进入日志。 |
| #99 | `go/clear-text-logging` | `backend/internal/handler/openai_gateway_count_tokens.go:197` | 已修复/加固 | 上游/代理/OAuth错误日志已修复或加固：可能带URL认证信息、API查询密钥或响应体回显。改为固定诊断摘要并保留事件上下文，避免任意错误内容进入日志。 |
| #100 | `go/clear-text-logging` | `backend/internal/handler/openai_gateway_handler.go:584` | 已修复/加固 | 上游/代理/OAuth错误日志已修复或加固：可能带URL认证信息、API查询密钥或响应体回显。改为固定诊断摘要并保留事件上下文，避免任意错误内容进入日志。 |
| #101 | `go/clear-text-logging` | `backend/internal/handler/openai_gateway_handler.go:1139` | 已修复/加固 | 上游/代理/OAuth错误日志已修复或加固：可能带URL认证信息、API查询密钥或响应体回显。改为固定诊断摘要并保留事件上下文，避免任意错误内容进入日志。 |
| #102 | `go/clear-text-logging` | `backend/internal/handler/openai_gateway_handler.go:1204` | 已修复/加固 | 上游/代理/OAuth错误日志已修复或加固：可能带URL认证信息、API查询密钥或响应体回显。改为固定诊断摘要并保留事件上下文，避免任意错误内容进入日志。 |
| #103 | `go/clear-text-logging` | `backend/internal/handler/openai_gateway_handler.go:1213` | 已修复/加固 | 上游/代理/OAuth错误日志已修复或加固：可能带URL认证信息、API查询密钥或响应体回显。改为固定诊断摘要并保留事件上下文，避免任意错误内容进入日志。 |
| #104 | `go/clear-text-logging` | `backend/internal/handler/openai_gateway_handler.go:1994` | 已修复/加固 | 上游/代理/OAuth错误日志已修复或加固：可能带URL认证信息、API查询密钥或响应体回显。改为固定诊断摘要并保留事件上下文，避免任意错误内容进入日志。 |
| #105 | `go/clear-text-logging` | `backend/internal/handler/openai_gateway_handler.go:2243` | 已修复/加固 | 上游/代理/OAuth错误日志已修复或加固：可能带URL认证信息、API查询密钥或响应体回显。改为固定诊断摘要并保留事件上下文，避免任意错误内容进入日志。 |
| #106 | `go/clear-text-logging` | `backend/internal/handler/openai_gateway_handler.go:2252` | 已修复/加固 | 上游/代理/OAuth错误日志已修复或加固：可能带URL认证信息、API查询密钥或响应体回显。改为固定诊断摘要并保留事件上下文，避免任意错误内容进入日志。 |
| #107 | `go/clear-text-logging` | `backend/internal/handler/openai_images.go:268` | 已修复/加固 | 上游/代理/OAuth错误日志已修复或加固：可能带URL认证信息、API查询密钥或响应体回显。改为固定诊断摘要并保留事件上下文，避免任意错误内容进入日志。 |
| #108 | `go/clear-text-logging` | `backend/internal/handler/openai_images.go:284` | 已修复/加固 | 上游/代理/OAuth错误日志已修复或加固：可能带URL认证信息、API查询密钥或响应体回显。改为固定诊断摘要并保留事件上下文，避免任意错误内容进入日志。 |
| #109 | `go/clear-text-logging` | `backend/internal/repository/http_upstream.go:1401` | 误报 | 日志中的proxyURL.Host仅在已匹配socks5/socks5h/http分支使用；userinfo及password存储在URL.User，与Host分离。VMess不会进入这些分支。 |
| #110 | `go/clear-text-logging` | `backend/internal/repository/http_upstream.go:1410` | 误报 | 日志中的proxyURL.Host仅在已匹配socks5/socks5h/http分支使用；userinfo及password存储在URL.User，与Host分离。VMess不会进入这些分支。 |
| #111 | `go/clear-text-logging` | `backend/internal/securityaudit/prompt_logging.go:65` | 误报 | Prompt Audit日志中的APIKeyID为int64记录ID，safeAttrs仅接受已允许字段；未传入APIKey.Key。 |
| #112 | `go/clear-text-logging` | `backend/internal/securityaudit/prompt_logging.go:71` | 误报 | Prompt Audit日志中的APIKeyID为int64记录ID，safeAttrs仅接受已允许字段；未传入APIKey.Key。 |
| #113 | `go/clear-text-logging` | `backend/internal/securityaudit/prompt_logging.go:77` | 误报 | Prompt Audit日志中的APIKeyID为int64记录ID，safeAttrs仅接受已允许字段；未传入APIKey.Key。 |
| #114 | `go/clear-text-logging` | `backend/internal/service/batch_image_processor.go:103` | 已修复/加固 | 上游/代理/OAuth错误日志已修复或加固：可能带URL认证信息、API查询密钥或响应体回显。改为固定诊断摘要并保留事件上下文，避免任意错误内容进入日志。 |
| #115 | `go/clear-text-logging` | `backend/internal/service/batch_image_public.go:462` | 已修复/加固 | 上游/代理/OAuth错误日志已修复或加固：可能带URL认证信息、API查询密钥或响应体回显。改为固定诊断摘要并保留事件上下文，避免任意错误内容进入日志。 |
| #116 | `go/clear-text-logging` | `backend/internal/service/batch_image_public.go:469` | 已修复/加固 | 上游/代理/OAuth错误日志已修复或加固：可能带URL认证信息、API查询密钥或响应体回显。改为固定诊断摘要并保留事件上下文，避免任意错误内容进入日志。 |
| #117 | `go/clear-text-logging` | `backend/internal/service/batch_image_worker.go:173` | 已修复/加固 | 上游/代理/OAuth错误日志已修复或加固：可能带URL认证信息、API查询密钥或响应体回显。改为固定诊断摘要并保留事件上下文，避免任意错误内容进入日志。 |
| #118 | `go/clear-text-logging` | `backend/internal/service/content_moderation.go:819` | 误报 | 各命中参数均为 input.APIKeyID int64 的数据库引用，日志保留用户/分组/协议/状态等诊断字段，不输出认证密钥；1071行的原始上游错误单独按#73修复。 |
| #119 | `go/clear-text-logging` | `backend/internal/service/content_moderation.go:829` | 误报 | 各命中参数均为 input.APIKeyID int64 的数据库引用，日志保留用户/分组/协议/状态等诊断字段，不输出认证密钥；1071行的原始上游错误单独按#73修复。 |
| #120 | `go/clear-text-logging` | `backend/internal/service/content_moderation.go:839` | 误报 | 各命中参数均为 input.APIKeyID int64 的数据库引用，日志保留用户/分组/协议/状态等诊断字段，不输出认证密钥；1071行的原始上游错误单独按#73修复。 |
| #121 | `go/clear-text-logging` | `backend/internal/service/content_moderation.go:850` | 误报 | 各命中参数均为 input.APIKeyID int64 的数据库引用，日志保留用户/分组/协议/状态等诊断字段，不输出认证密钥；1071行的原始上游错误单独按#73修复。 |
| #122 | `go/clear-text-logging` | `backend/internal/service/content_moderation.go:872` | 误报 | 各命中参数均为 input.APIKeyID int64 的数据库引用，日志保留用户/分组/协议/状态等诊断字段，不输出认证密钥；1071行的原始上游错误单独按#73修复。 |
| #123 | `go/clear-text-logging` | `backend/internal/service/content_moderation.go:881` | 误报 | 各命中参数均为 input.APIKeyID int64 的数据库引用，日志保留用户/分组/协议/状态等诊断字段，不输出认证密钥；1071行的原始上游错误单独按#73修复。 |
| #124 | `go/clear-text-logging` | `backend/internal/service/content_moderation.go:890` | 误报 | 各命中参数均为 input.APIKeyID int64 的数据库引用，日志保留用户/分组/协议/状态等诊断字段，不输出认证密钥；1071行的原始上游错误单独按#73修复。 |
| #125 | `go/clear-text-logging` | `backend/internal/service/content_moderation.go:902` | 误报 | 各命中参数均为 input.APIKeyID int64 的数据库引用，日志保留用户/分组/协议/状态等诊断字段，不输出认证密钥；1071行的原始上游错误单独按#73修复。 |
| #126 | `go/clear-text-logging` | `backend/internal/service/content_moderation.go:916` | 误报 | 各命中参数均为 input.APIKeyID int64 的数据库引用，日志保留用户/分组/协议/状态等诊断字段，不输出认证密钥；1071行的原始上游错误单独按#73修复。 |
| #127 | `go/clear-text-logging` | `backend/internal/service/content_moderation.go:926` | 误报 | 各命中参数均为 input.APIKeyID int64 的数据库引用，日志保留用户/分组/协议/状态等诊断字段，不输出认证密钥；1071行的原始上游错误单独按#73修复。 |
| #128 | `go/clear-text-logging` | `backend/internal/service/content_moderation.go:939` | 误报 | 各命中参数均为 input.APIKeyID int64 的数据库引用，日志保留用户/分组/协议/状态等诊断字段，不输出认证密钥；1071行的原始上游错误单独按#73修复。 |
| #129 | `go/clear-text-logging` | `backend/internal/service/content_moderation.go:966` | 误报 | 各命中参数均为 input.APIKeyID int64 的数据库引用，日志保留用户/分组/协议/状态等诊断字段，不输出认证密钥；1071行的原始上游错误单独按#73修复。 |
| #130 | `go/clear-text-logging` | `backend/internal/service/content_moderation.go:984` | 误报 | 各命中参数均为 input.APIKeyID int64 的数据库引用，日志保留用户/分组/协议/状态等诊断字段，不输出认证密钥；1071行的原始上游错误单独按#73修复。 |
| #131 | `go/clear-text-logging` | `backend/internal/service/content_moderation.go:1013` | 误报 | 各命中参数均为 input.APIKeyID int64 的数据库引用，日志保留用户/分组/协议/状态等诊断字段，不输出认证密钥；1071行的原始上游错误单独按#73修复。 |
| #132 | `go/clear-text-logging` | `backend/internal/service/content_moderation.go:1026` | 误报 | 各命中参数均为 input.APIKeyID int64 的数据库引用，日志保留用户/分组/协议/状态等诊断字段，不输出认证密钥；1071行的原始上游错误单独按#73修复。 |
| #133 | `go/clear-text-logging` | `backend/internal/service/content_moderation.go:1035` | 误报 | 各命中参数均为 input.APIKeyID int64 的数据库引用，日志保留用户/分组/协议/状态等诊断字段，不输出认证密钥；1071行的原始上游错误单独按#73修复。 |
| #134 | `go/clear-text-logging` | `backend/internal/service/content_moderation.go:1063` | 误报 | 各命中参数均为 input.APIKeyID int64 的数据库引用，日志保留用户/分组/协议/状态等诊断字段，不输出认证密钥；1071行的原始上游错误单独按#73修复。 |
| #135 | `go/clear-text-logging` | `backend/internal/service/content_moderation.go:1094` | 误报 | 各命中参数均为 input.APIKeyID int64 的数据库引用，日志保留用户/分组/协议/状态等诊断字段，不输出认证密钥；1071行的原始上游错误单独按#73修复。 |
| #136 | `go/clear-text-logging` | `backend/internal/service/gateway_scheduling.go:338` | 误报 | in_account_map是map查找的ok布尔值；账号对象中的Proxy.Password不会经该表达式输出。 |
| #137 | `go/clear-text-logging` | `backend/internal/service/openai_live.go:344` | 误报 | fmt.Sprintf("%T", err)只输出Go动态类型，绝不调用Error()或序列化密码；此处已有安全日志设计。 |
| #138 | `go/allocation-size-overflow` | `backend/internal/pkg/apicompat/responses_namespace.go:79` | 误报 | Namespace flattening adds element counts of tools and the map built by visiting their function children. Both are materialized from the bounded request and each is backed by allocations with non-byte elements, which preclude an overflowing sum. |
| #139 | `go/allocation-size-overflow` | `backend/internal/pkg/apicompat/responses_to_anthropic_request.go:355` | 误报 | others and kept partition a single parsed message's existing Anthropic content blocks. Their total cannot exceed len(blocks). |
| #140 | `go/allocation-size-overflow` | `backend/internal/pkg/apicompat/responses_to_anthropic_request.go:355` | 误报 | Both operands at this sink are slices of nontrivial structs derived from the same materialized blocks. At most one output block is retained per original block, so the addition cannot overflow. |
| #141 | `go/allocation-size-overflow` | `backend/internal/service/openai_codex_transform.go:1308` | 误报 | The map conversion adds one reserved metadata key to an already instantiated map[string]string. Request JSON ordinarily decodes to map[string]any and does not take this allocation branch. |
| #142 | `go/allocation-size-overflow` | `backend/internal/service/openai_messages_digest_session.go:22` | 误报 | Digest preallocation is the decoded Anthropic messages slice length plus at most one system entry. The gateway limits body size, and each materialized message is a multi-word struct. |
| #143 | `go/allocation-size-overflow` | `backend/internal/service/openai_model_alias.go:171` | 误报 | The variadic alternates at billing call sites are supplied by server code as a small, fixed list of model candidates. Client model strings do not determine the argument count. |
| #144 | `go/allocation-size-overflow` | `backend/internal/service/openai_responses_lite_tools.go:159` | 误报 | Existing additional_tools and moved namespace tools are derived from the same bounded request. Their sum counts materialized interface slice elements and cannot reach int overflow. |
| #145 | `go/allocation-size-overflow` | `backend/internal/service/openai_responses_lite_tools.go:159` | 误报 | The second operand at the Lite merge is the slice of tools selected to migrate. It is built from actual decoded tool definitions rather than from an unchecked numeric field. |
| #146 | `go/allocation-size-overflow` | `backend/internal/service/openai_responses_tool_schema.go:64` | 误报 | Schema rewrite reserves body length plus a constant per encountered explicit null. Each hit corresponds to a checked offset in the bounded gateway request, and default HTTP and WebSocket limits keep the capacity below the machine int range. |
| #147 | `go/allocation-size-overflow` | `backend/internal/service/openai_ws_forwarder_payload.go:170` | 误报 | Create copies the already-decoded request map and reserves a single additional entry. The body or WebSocket message is independently bounded before decoding. |
| #148 | `go/allocation-size-overflow` | `backend/internal/service/openai_ws_forwarder_payload.go:202` | 误报 | The allocation copies a materialized map[string]string of handshake metadata with one extra field. JSON-derived map[string]any is updated directly, and no remote numeric field supplies the capacity. |
| #149 | `go/allocation-size-overflow` | `backend/internal/service/openai_ws_forwarder_payload.go:629` | 已修复/加固 | Previous replay messages can accumulate indefinitely across individually bounded frames. The optional replay history now checks combined byte and entry memory budget before allocating. Native WS keeps continuation anchors when replay is unavailable. |
| #150 | `go/allocation-size-overflow` | `backend/internal/service/openai_ws_forwarder_payload.go:629` | 已修复/加固 | The second source at the replay allocation is checked against the same shared 64 MiB budget, including both direct input and accumulated output appends. A bridge turn requiring unavailable history gets WebSocket status 1009 instead of replaying partial history. |
| #151 | `go/allocation-size-overflow` | `backend/internal/service/openai_ws_forwarder_support.go:74` | 误报 | Prewarm copies one already-built WS payload map and adds the constant generate field. Its source is bounded before JSON decode and the allocation uses existing entry count plus one. |
| #153 | `go/email-injection` | `backend/internal/service/email_service.go:216` | 误报 | buildSMTPMessage rejects CR/LF in parsed envelope addresses, normalizes line breaks in display name and subject before MIME formatting, and emits the quoted-printable body after the header separator through SMTP DATA dot framing. Existing message tests cover header handling. |
| #154 | `go/reflected-xss` | `backend/internal/handler/ops_error_logger.go:652` | 误报 | The ops writer wraps only gateway HTTP routes, forwards an already prepared API response, and captures a bounded prefix of errors for audit. It does not select HTML rendering or copy request values into headers. Response media type is controlled at the producing route. |
| #155 | `go/reflected-xss` | `backend/internal/server/middleware/server_timing.go:64` | 误报 | The timing middleware forwards the handler's already prepared body and adds one formatted Server-Timing header. A caller's UI-scope flag does not bypass role checks or become HTML response content. |
| #156 | `go/reflected-xss` | `backend/internal/service/openai_compact_sse_keepalive.go:215` | 已修复/加固 | The compact wrapper itself only forwards bytes. A reachable upstream passthrough branch did propagate active Content-Type labels. The output paths now select safe JSON/SSE media types while preserving the API body. |
| #157 | `go/reflected-xss` | `backend/internal/service/openai_gateway_passthrough.go:1146` | 误报 | The pending-line writer is inside the streaming passthrough branch, which sets text/event-stream before writing the raw SSE protocol. The separately reviewed non-streaming media type path is addressed under #156. |
| #158 | `go/reflected-xss` | `backend/internal/service/openai_images_json_keepalive.go:194` | 已修复/加固 | The image JSON keepalive writer forwards data from the successful image response. The producer now constrains media type to JSON variants for non-streaming responses and text/event-stream for streaming responses, overriding untrusted active types. |
| #159 | `go/reflected-xss` | `backend/internal/service/openai_ws_forwarder_v2.go:414` | 误报 | emitStreamMessage emits data-prefixed SSE frames under the reqStream branch with Content-Type explicitly set to text/event-stream. Client data is not inserted into an HTML document. |
| #160 | `go/path-injection` | `backend/internal/handler/page_handler.go:122` | 已修复/加固 | The previous os.Stat was a fresh lookup after resolving symlinks. The handler now resolves and opens within the slug os.Root and checks the metadata of the same open handle. |
| #161 | `go/path-injection` | `backend/internal/handler/page_handler.go:128` | 已修复/加固 | Serving via http.ServeContent on the confined file handle removes the path reopen after validation while preserving streaming, range and modification-time behavior. Markdown reads are rooted and size-bounded as well. |
| #162 | `go/weak-sensitive-data-hashing` | `backend/internal/handler/admin/snapshot_cache.go:155` | 误报 | The admin snapshot ETag is a SHA-256 validator of serialized response bytes. It is neither an authenticator nor a password verifier. |
| #163 | `go/weak-sensitive-data-hashing` | `backend/internal/repository/account_credentials_encryption.go:55` | 开放迁移项 | The persisted credentials_digest covers a canonical credentials document that can include operator-selected low-entropy passwords. If the database is read without the configured encryption key, a known or guessable remainder of the document can enable offline guesses despite GCM. Move to a domain-separated HMAC with a stable deployment key and versioned digest. Coordinate legacy/current dual-match CAS and dedup predicates, decrypt-and-backfill, worker rollout, and key rotation before retiring legacy values. |
| #164 | `go/weak-sensitive-data-hashing` | `backend/internal/repository/account_credentials_encryption.go:59` | 开放迁移项 | credentials_api_key_digest supports encrypted Ollama identity comparison. SHA-256 is adequate for high-entropy API keys against guessing, but imported custom api_key values have no token entropy guarantee and the digest is stored beside ciphertext. A stable, separately domain-separated versioned HMAC can cover such keys. Migrate with dual legacy/current comparison, backfill and rotation in lockstep with existing database consumers. |
| #165 | `go/weak-sensitive-data-hashing` | `backend/internal/repository/scheduler_outbox_repo.go:230` | 误报 | SHA-256 identifies duplicate durable scheduler events by type, account/group IDs and JSON payload. It is an equality checksum with no password verification or authentication step. |
| #166 | `go/weak-sensitive-data-hashing` | `backend/internal/service/openai_codex_models_service.go:548` | 误报 | The hash computes a content ETag for the rewritten Codex models manifest body. No password verification uses it. |
| #167 | `go/weak-sensitive-data-hashing` | `backend/internal/service/openai_codex_models_service.go:693` | 误报 | Sorted outbound headers, including upstream Authorization, are hashed into a process-local models manifest cache key to avoid cross-credential cache reuse. The hash is not an exposed verifier or a stored login password. |
| #168 | `go/weak-sensitive-data-hashing` | `backend/internal/handler/admin/account_data.go:587` | 误报 | Credential identity fields are compared as stable SHA-256 digests in local admin import de-duplication structures. The digest is not emitted or persisted by this routine. |
| #169 | `go/weak-sensitive-data-hashing` | `backend/internal/pkg/proxyutil/shadowsocks.go:162` | 保留的设计/协议 | Legacy Shadowsocks SIP004 AEAD requires EVP_BytesToKey(MD5) for interoperable master-key derivation from its configured password. The result feeds protocol-specific session derivation. Replacing MD5 requires a coordinated protocol change. |
| #170 | `go/weak-sensitive-data-hashing` | `backend/internal/pkg/proxyutil/trojan.go:142` | 保留的设计/协议 | Trojan peers require lower-case hex SHA-224 of the configured password in the TLS handshake payload. The algorithm is mandated by the wire protocol, and changing it breaks all existing Trojan peers. |
| #171 | `go/weak-sensitive-data-hashing` | `backend/internal/service/api_key_auth_cache_impl.go:200` | 保留的设计/协议 | SHA-256 gives stable L1/L2 lookup and invalidation keys for API credentials, not account password storage. Generated keys use randomness, but users may intentionally supply a custom key, so Redis/auth-cache access remains sensitive and weak custom tokens are guessable if that cache is disclosed. |
| #172 | `go/weak-sensitive-data-hashing` | `backend/internal/service/auth_service.go:1869` | 误报 | resolvedTokenVersion fingerprints the user's email and already bcrypt-hashed password to invalidate JWTs after a password change. The SHA-256 output is a change detector, with authentication handled separately. |
| #173 | `go/weak-sensitive-data-hashing` | `backend/internal/service/content_moderation.go:2567` | 保留的设计/协议 | Moderation uses a stable provider-key fingerprint for health state and admin-side key selection/removal rather than authenticating passwords. Provider keys are expected to be machine-generated, but an operator may configure a weak custom value and admins receive the fingerprint. |
| #174 | `go/weak-sensitive-data-hashing` | `backend/internal/service/gateway_claude_oauth_body.go:519` | 误报 | This SHA-256 produces a deterministic session-routing UUID from account and request anchors. It never verifies a password or authenticates a user. |
| #175 | `go/weak-sensitive-data-hashing` | `backend/internal/service/gemini_session.go:63` | 误报 | Gemini builds a cache partition prefix from numeric user/API-key identifiers, IP, normalized user agent, platform and model. The underlying API-key secret and password are not inputs. |
| #176 | `go/weak-sensitive-data-hashing` | `backend/internal/service/notification_email_service.go:859` | 误报 | The helper hashes normalized email addresses/identities for deterministic preference and delivery lookups. Their digests are identifiers rather than anonymization or authentication. Token signing uses HMAC independently. |
| #177 | `go/weak-sensitive-data-hashing` | `backend/internal/service/ollama_cloud_usage.go:1058` | 误报 | The service fingerprints an Ollama API key to group in-memory usage accounting by provider identity. The digest is not accepted as a login or upstream credential. |
| #178 | `go/weak-sensitive-data-hashing` | `backend/internal/service/openai_codex_models_service.go:684` | 误报 | Account IDs, configured proxy URL and request URL are keyed together in the process-local models manifest cache. This prevents response reuse across request configuration and does not establish password security. |
| #179 | `go/weak-sensitive-data-hashing` | `backend/internal/service/openai_sticky_compat.go:46` | 误报 | The SHA-256 digest indexes legacy OpenAI sticky-session entries using session IDs. It maintains rolling cache compatibility and does not prove session authentication. |
| #180 | `go/weak-sensitive-data-hashing` | `backend/internal/service/usage_billing.go:135` | 误报 | The usage operation fingerprint binds account/key IDs, usage/cost fields, model information and optional payload checksum for billing idempotency. It is not a password verifier. |
| #181 | `go/weak-sensitive-data-hashing` | `backend/internal/service/usage_billing.go:211` | 误报 | The batch-image hold fingerprint covers IDs, costs and optional payload checksum for idempotency. No login password enters the fingerprint. |
| #182 | `go/weak-sensitive-data-hashing` | `backend/internal/service/vertex_service_account.go:139` | 误报 | Vertex hashes the service account's client email and key ID to index its token cache. Private-key bytes and passwords are not hashed at this sink. |

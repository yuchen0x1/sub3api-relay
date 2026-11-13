# Sub3API 项目分析与备份可靠性升级

日期：2026-11-13。基线：`main` / `2e6cd61b3`，产品版本 `0.4.2`。

本轮完成项目识别、源码审计、5 个开源项目研究、备份链路升级及本地验证。审计重点是管理台的数据库备份、恢复与存储配置；不代表已逐行审计全部网关、支付、身份认证和第三方协议实现。本文记录发布前的本地验证阶段；该阶段未执行提交、推送、发布、真实数据库恢复或真实对象删除。后续 v0.4.3 发布内容见 [版本说明](RELEASE_NOTES_v0.4.3.md)，远程状态以 GitHub Release 和 Actions 为准。

## 1. 项目功能地图与工作区边界

| 层次 | 已有能力与代码入口 | 本轮核实程度 |
| --- | --- | --- |
| Web 管理台 | Vue 3、TypeScript、Pinia、Vue Router、Tailwind；`frontend/src/views`、`frontend/src/api` | 阅读备份视图、API 客户端、刷新令牌协调、设置与公共组件 |
| 服务入口 | Go 1.26、Gin、Ent、Wire；`backend/cmd/server`、`backend/internal/server/routes` | 核对启动、路由、嵌入前端与权限/二次验证挂载 |
| 核心网关 | 多平台账号、模型路由、并发调度、API Key、计费、用量 | 核对路由与模块，不宣称真实上游端到端验证 |
| 运维 | Ops、告警、审计、备份、自动更新、定时测试 | 深入追踪备份 handler → service → PostgreSQL/S3 repository → UI |
| 持久层 | PostgreSQL、Redis、Ent 与 SQL migrations | 阅读设置仓储契约；本轮无数据库结构变更 |
| 发布 | Docker Compose、GitHub Actions；前端嵌入 Go | 阅读 CI、构建和部署说明；不执行远程发布 |
| 可选桌面外壳 | `desktop/` Tauri | 现有 README 明确受支持发行产品为 Web/服务端/Docker，本轮不扩展桌面发行 |

最初工作区已有 `README.md`、`README_CN.md`、`deploy/README.md` 修改，本轮保留，仅在 README 追加本轮升级入口。依赖版本以 `go.mod` 和 `frontend/package.json` 为准；旧 `DEV_GUIDE.md` 的工具版本、历史 fork 信息不能作为当前部署事实。

已检查到 `AccountService.TestCredentials` 和 `adminServiceImpl.RefreshAccountCredentials` 的占位实现。全仓调用搜索未发现它们是当前界面的实际入口，因此未盲目接入上游接口，也未把它们算作新实现功能。

## 2. 开源研究与许可证

研究使用 GitHub 在线搜索、仓库元数据、许可证及具体源文件；没有复制或移植外部代码。维护状态是查询时观察，不能保证未来维护情况。未运行这些项目的 UI 或性能基准。

| 项目 | 实际阅读来源 | 可借鉴实践 | 许可证及维护观察 |
| --- | --- | --- | --- |
| [LiteLLM](https://github.com/BerriAI/litellm) | [health_check.py](https://github.com/BerriAI/litellm/blob/main/litellm/proxy/health_check.py)、LICENSE、提交记录 | 单项健康检查超时不取消无关任务；健康结果与管理字段有明确边界 | 根许可 MIT，`litellm/enterprise/` 另有许可；观察到 2026-11-13 提交 |
| [New API](https://github.com/QuantumNous/new-api) | [channel-test.go](https://github.com/QuantumNous/new-api/blob/main/controller/channel-test.go)、LICENSE、提交记录 | 有界并行渠道测试、后台任务入队、冲突状态反馈 | AGPL-3.0；观察到 2026-11-13 提交；仅参考理念 |
| [One API](https://github.com/songquanpeng/one-api) | [channel-test.go](https://github.com/songquanpeng/one-api/blob/main/controller/channel-test.go)、LICENSE、提交记录 | 真实适配器探测、耗时及结果记录、后台测试互斥 | MIT；查询默认分支最新提交为 2025-02-21，作为成熟实现参考，不称持续活跃 |
| [Portkey Gateway](https://github.com/Portkey-AI/gateway) | [retryHandler.ts](https://github.com/Portkey-AI/gateway/blob/main/src/handlers/retryHandler.ts)、LICENSE、提交记录 | 超时、有界重试、错误传播，避免无上限等待 | MIT；查询最新提交为 2026-05-25 |
| [Bifrost](https://github.com/maximhq/bifrost) | README、core 目录、[failureclass.go](https://github.com/maximhq/bifrost/blob/dev/core/failureclass.go)、[abandonedstream_test.go](https://github.com/maximhq/bifrost/blob/dev/core/abandonedstream_test.go)、LICENSE | 区分可恢复与凭据错误；回归测试证明无人消费的流不会永久阻塞生产者 | Apache-2.0 根许可；默认分支 `dev`，观察到 2026-11-13 更新；企业能力另论 |

当前仓库根许可为 MulanPSL-2.0。此次独立实现不引入上述许可证的代码义务；未来移植必须按具体文件和分发方式重新评估，不能因为某仓库根目录宽松许可就忽略企业子目录。

| 对比维度 | 当前实际情况 | 参考实践与差距 | 本轮处理 |
| --- | --- | --- | --- |
| 核心功能 | 已有调度、计费、备份恢复 | 成熟网关注重任务真实成功状态；备份有失败误报成功风险 | 核验 dump 最终退出结果、持久化后再接受恢复任务 |
| UI | 已有统一卡片/表格、移动端滚动与深色模式 | 运维需要持续可见的历史结果；原页面主要靠一次性通知 | 表格展示备份/恢复错误与恢复状态 |
| 用户体验 | 轮询、页面隐藏暂停、二次验证均已存在 | 慢请求可能重叠；零值配置被默认值覆盖 | 防重叠、卸载后忽略结果、保留零值、冲突按钮禁用 |
| 技术架构 | service 与 repository 已分层 | 异步任务登记和关闭等待存在竞态 | 统一任务准入及清理，关闭时停止新任务 |
| 自动化 | 定时备份及保留策略已有实现 | 失败任务占用保留份数、手动快照被全局策略误删 | 按记录到期时间清理；份数仅作用于已完成定时备份 |
| 扩展能力 | 大量供应商及运维模块，已有插件式 factory | 无需为一次升级重写网关；备份 JSON 历史不适合无限扩展 | 保留接口与格式，数据库任务表/分页作为后续架构工作 |

## 3. 优先级与实际交付

| 优先级 | 问题/升级 | 收益、难度与风险 | 状态 |
| --- | --- | --- | --- |
| P0 | 修改存储配置会明文保存原有 S3 密钥 | 避免敏感凭据保护退化；小范围修复，保留密钥也重新加密 | 已修复并测试 |
| P0 | 无限制设置变成默认限制、清理误算、删除失败丢失记录 | 降低备份数据丢失与不可追踪风险；策略范围需明确说明 | 已修复并同步三语提示 |
| P0 | 损坏/读取失败的历史可能被覆盖 | 避免原始记录被新记录覆盖；正常缺失配置仍兼容 | 已修复并测试 |
| P0 | 备份与恢复并发；恢复任务持久化失败仍启动 | 避免操作冲突；同一服务进程内互斥，不冒充分布式锁 | 已修复并测试 |
| P0 | psql 恢复参数不完整 | 加入 `--file=-`、`--set=ON_ERROR_STOP=on`、`--no-psqlrc`，配合单事务 | 命令契约测试通过；真实恢复待隔离数据库验证 |
| P1 | pg_dump 退出失败仍可能标记完成 | 上传成功需同时确认 dump/gzip 完成 | 已修复并覆盖同步/异步 |
| P1 | 同秒备份文件名冲突、成功历史超过 100 条丢失 | UUID 写入新 ID/文件名；只限制失败历史，不丢弃有效对象索引 | 已修复并测试 |
| P1 | S3 上传整包读入内存 | 改为临时文件暂存，保留 PutObject 兼容性；需要足够临时磁盘空间 | 已修复，HTTP fixture 验证内容/长度/失败清理 |
| P2 | 状态反馈、轮询与冲突按钮 | 避免慢网络重叠请求、后台完成后按钮卡住 | 已修复，组件与浏览器 fixture 验证 |

恢复命令行为依据 [PostgreSQL psql 官方文档](https://www.postgresql.org/docs/current/app-psql.html)：单事务选项需要配合 `-c` 或 `-f`，SQL 失败时须设置 `ON_ERROR_STOP` 才能可靠停止并返回失败。此处不把命令参数检查当作数据库恢复验收。

## 4. 保留策略、兼容性与部署影响

- `retain_days=0`：新定时备份不设置到期时间。修改该配置不改写已有记录的 `expires_at`。
- `retain_count=0`：定时备份份数不限制；正数只统计未到期、已完成的定时备份。手动备份及失败任务不挤占保留份数。
- 手动备份按自身 `expire_days` 保存；设为 0 的手动备份不会被定时份数策略清理。
- 清理仍在成功执行定时备份之后进行；禁用调度不会在后台独立执行到期清理。
- S3 删除失败时返回失败并保留记录，可重试；不会把未成功删除的文件标记为已删除。
- 历史记录及 API 字段仍兼容，无 schema migration。新记录采用完整 UUID，旧短 ID 仍可读取；表格缩短显示并保留完整 title。
- 到期天数范围为 0–36500，负数、过大值或非法 JSON 会拒绝；无请求体仍使用默认 14 天。
- 暂存使用操作系统临时目录（Unix 可通过 `TMPDIR` 指定），正常成功/失败均关闭并删除。进程被强制终止后可能遗留文件，部署需配置磁盘监控与临时目录清理。此改动是内存占用结构优化，没有伪造峰值内存或吞吐量数据。
- 已由旧逻辑丢弃的记录无法凭空恢复，本轮不扫描真实 S3，也不修改真实数据库。

## 5. 文件清单

| 文件 | 变化 |
| --- | --- |
| `backend/internal/service/backup_service.go` | 保留策略、密钥加密、任务互斥/关闭、记录读写、dump 结果校验、唯一文件名 |
| `backend/internal/handler/admin/backup_handler.go` | 拒绝非法 JSON，请求参数不再静默回落 |
| `backend/internal/repository/backup_pg_dumper.go` | 恢复命令的单事务、遇错停止与禁用用户启动文件 |
| `backend/internal/repository/backup_s3_store.go` | 磁盘暂存上传，避免整包驻留内存 |
| `frontend/src/views/admin/BackupView.vue` | 保留零值、错误展示、互斥按钮、慢请求与隐藏/卸载处理 |
| `frontend/src/i18n/locales/{zh,en,ja}/admin/overview.ts` | 明确保留策略范围与清理触发时机 |
| `backend/internal/service/backup_reliability_test.go` | 数据安全与生命周期回归 |
| `backend/internal/service/oauth_refresh_api_test.go` | 全量回归发现的 Windows 定时器顺序不稳定；改为等待实际取消，保留原有安全断言 |
| `backend/internal/repository/backup_storage_test.go` | 真实本地 HTTP 上传 fixture 与恢复参数契约 |
| `backend/internal/handler/admin/backup_handler_validation_test.go` | 非法输入在触发业务之前被拒绝 |
| `frontend/src/views/admin/__tests__/BackupView.spec.ts` | 6 项组件回归 |
| 本报告、`README.md` | 当前行为、证据及部署影响 |

没有删除源码文件、添加依赖、改变框架或修改远程仓库。

## 6. 验收记录

验证基于 Windows、Go 1.26.11、Node 24.20.0、pnpm 11.20.0；仓库最低 Go 版本为 1.26.10。

| 状态 | 命令/检查 | 结果与边界 |
| --- | --- | --- |
| PASS | `pnpm typecheck` | 退出 0 |
| PASS | `pnpm lint:check` | 全前端检查，退出 0 |
| PASS | `golangci-lint run --allow-parallel-runners --new-from-rev=HEAD --timeout=5m ./internal/service ./internal/handler/admin ./internal/repository` | 0 个新增问题，退出 0。按 HEAD 差异过滤，非全仓历史问题清零；使用工具官方并行选项避开其他 lint 实例的锁，没有终止其他任务 |
| PASS | `pnpm exec vitest run --maxWorkers=4` | 227 文件 / 1544 测试通过，退出 0 |
| PASS | `pnpm build` | 类型构建、Vite、PWA 均通过；日语 chunk 超过 500 kB 警告保留 |
| PASS | `go test -tags=unit ./internal/service -run 'Test(BackupReliability\|BackupService\|StartBackup\|StartRestore\|RecoverStaleRecords\|GracefulShutdown)' -count=1` | 备份专项通过，退出 0；含同步/异步、配置缺失、密钥再加密、错误路径 |
| PASS | `go test ./internal/handler/admin -run TestCreateBackupRejectsMalformedRequest -count=1` | 输入校验通过，退出 0 |
| PASS | `go test ./internal/repository -run 'TestBackup(Restore\|Upload)' -count=1` | 本地 HTTP fixture 收到准确内容与长度；成功、读取失败、上传失败均清除暂存文件；退出 0 |
| PASS | `go build -tags embed -o bin/server-evolution.exe ./cmd/server` | 含真实前端产物的 Windows 服务端构建通过 |
| PASS | `backend/bin/server-evolution.exe --version` | 二进制运行，返回 0.4.2；不是数据库连接或完整服务启动验收 |
| PASS | 隔离首次启动冒烟：`AUTO_SETUP=false SKIP_SETUP=false SERVER_HOST=127.0.0.1 SERVER_PORT=19085`，全新 `DATA_DIR=artifacts/startup-smoke` | 实际二进制成功启动；GET `/health`、`/setup/status`、`/setup` HTML 通过。没有提交初始化表单、连接数据库或执行迁移；测试后停止进程 |
| PARTIAL | `node frontend/.cache/evolution-preview/verify.mjs`（从 frontend 运行时路径为 `.cache/evolution-preview/verify.mjs`） | 真实 BackupView 与样式、隔离 API fixture；1440px / 390px / 深色截图，零值保留、无 pageerror、无整页溢出；不是生产端到端 |
| PASS（首轮 FAIL 已修复） | `go test -tags=unit ./internal/service ./internal/handler/admin -count=1` | 最终 service 与 admin 全部通过，退出 0。首轮仅 OAuth 超时测试因 Sleep 顺序依赖失败；改为等待取消事件，不放宽断言。随后新增的关闭后清理保护再次通过备份专项回归 |
| BLOCKED | Docker/PostgreSQL/Redis/真实 S3 灾备联调 | Docker Desktop 引擎未运行；没有使用真实数据恢复，也没有冒用生产凭据 |
| PARTIAL | 从零安装依赖 | 复用已有 node_modules/Go 模块缓存，未重装或新增依赖；不能据此认定全新机器安装验收通过 |
| NOT APPLICABLE | 数据库迁移、签名、远程 CI、发布 | 无 schema 变化；本地验证阶段未发布；后续发布另见版本说明 |

本地测试日志为 `artifacts/evolution-vitest.log`、`artifacts/evolution-go-tests.log` 及最终复验日志。截图为 `artifacts/backup-desktop.png`、`backup-mobile.png`、`backup-dark.png`，画面明确使用“测试场景”记录；这些隔离产物被 Git 忽略，不写入生产资源。

浏览器工具链限制：内置 Browser 缺少运行时模块，本地 Playwright 默认 Chromium 未安装；使用已有 Edge 的独立临时会话完成隔离视图检查，没有读取用户浏览器会话或凭据。

## 7. 剩余问题与后续路线

1. **P1，多实例一致性**：当前备份互斥及 JSON 历史写锁是进程内锁。多副本需持久任务表、租约与按任务恢复；不能把本轮改动当作多实例保障。价值高、涉及 schema 与部署拓扑，须隔离集成验证。
2. **P1，真实灾备演练**：在一次性数据库/专用 S3 bucket 中验证完整备份、SQL 失败回滚、断流/截断文件、服务中断及升级后恢复。没有这些证据不能给“生产灾备完成”结论。
3. **P1，存储位置与配置版本**：历史记录没有固定 bucket/endpoint 身份；切换 S3 目标前需验证旧备份可访问。后续应为记录保存存储目标引用，避免切换目标造成历史不可达。
4. **P2，备份记录分页与持久模型**：有效历史不再丢弃，但长期不限量的 JSON 数组会增长。宜迁移独立记录表与分页 API，同时保证老记录兼容。
5. **P2，配置/构建文档统一**：旧 `DEV_GUIDE.md`、根 Makefile 与目前嵌入 Web 发行方式有历史差异；后续统一开发入口，禁止使用旧示例凭据部署生产。
6. **P2，前端资源**：构建提示日语资源 chunk 超过 500 kB；应在真实加载性能数据支持下拆分，避免随意重排 chunk 导致已有循环依赖问题复发。
7. **P3，插件或新平台**：已有多供应商及完整管理模块，当前没有证据支持大规模重写或堆砌新的平台入口。

## 8. 完成度评价

不对未验证的全项目给任意百分制评分。以下只评价证据覆盖：

| 维度 | 评价依据 | 状态 |
| --- | --- | --- |
| 功能完整度 | 本轮备份/恢复/保留/错误反馈路径已实现；真实灾备仍缺证据 | PARTIAL |
| UI/UX | 6 项组件回归；桌面/手机/深色隔离页面无脚本错误或整页溢出 | PASS（本轮视图范围） |
| 代码质量 | 类型、前端全量 Lint、Go 新增问题检查、service/admin 全量与备份专项回归 | PASS（已列检查范围） |
| 性能 | 上传从整包内存改为磁盘；轮询不重叠，无真实负载指标 | PARTIAL |
| 安全性 | 保留密钥再次加密、非法输入拒绝、恢复参数修正；非全仓安全审计 | PARTIAL |
| 稳定性 | 失败/取消/重启与并发路径有回归；外部服务异常实测尚缺 | PARTIAL |
| 可维护性 | 复用现有分层与接口、任务入口集中、无新增依赖 | 本轮改善 |
| 扩展性 | 保留旧记录格式；多实例与记录表迁移留有明确后续路线 | PARTIAL |

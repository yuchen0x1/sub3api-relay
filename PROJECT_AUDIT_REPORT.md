# Sub3API 项目审计与本地升级报告

审计日期：2026-11-09（Asia/Shanghai）
工作目录：`C:\Users\lenmo\Desktop\Sub3api`
分支/基线：`main`，HEAD `4ba6d66f7`；审计开始时工作区存在未提交改动。

> 本报告记录本次实际检查范围和结果。项目约 3,600 个 Git 跟踪文件、1,102 个 Go 测试文件、226 个前端测试文件，不能把有限时段的自动检查表述成逐行人工审计。未确认安全的清理项均保留。

## 1. 项目现状

Sub3API 是自托管 AI API 网关与管理平台，提供多上游账号/渠道、统一兼容 API、密钥和用户管理、使用统计、运维与支付等功能。仓库名和 README 的产品定位是 Sub3API / AI API Gateway Platform。

| 项目项 | 识别结果 |
| --- | --- |
| 版本 | 0.4.0；`backend/cmd/server/VERSION` 与前端 `package.json` 一致 |
| 后端 | Go 1.26.10 module；Gin HTTP 服务、Ent ORM；SQLite 支持，部署说明使用 PostgreSQL 与 Redis |
| 前端 | Vue 3、TypeScript、Vite 8、pnpm 11.20.0、Pinia、Vue Router、Vue I18n、Tailwind |
| 桌面/移动 | 仓库含 `desktop/` 与 PWA 能力；本次未进行桌面打包或移动端实机验证 |
| 构建 | Go modules、pnpm、Docker 多阶段构建；前端构建产物嵌入后端 |
| 工作区 | 已有两处用户 Go 格式变更；`release-assets/` 含 36 个未跟踪文件，含 v0.3.1/v0.4.0 发行包与移动管理界面截图；均按用户数据保留 |
| 运行状态 | 未发现本项目服务进程；系统当前监听端口只看到 FileZilla/nginx/系统服务。没有启动项目服务，未占用端口 |

### 架构与主要模块

- `backend/cmd/server`：启动、依赖注入和版本信息。
- `backend/internal/handler`、`server/routes`、`middleware`：管理 API、网关 API、路由和访问控制。
- `backend/internal/service`：业务编排与后台任务。
- `backend/internal/repository`、`ent/schema`、`migrations`：数据库读写、Ent 模型和迁移。
- `backend/internal/pkg`：协议适配、代理、认证、网络和通用实现。
- `frontend/src/api`、`stores`、`views`、`components`、`features`：前端 API、状态、页面、组件与功能模块。
- `deploy`、`Dockerfile`、`.github/workflows`：部署、镜像和 CI/安全扫描。
- `docs`、多语言 README、release notes：产品、部署与版本文档。

总体判断：功能覆盖广，Go 与 Vue 项目边界清楚，后端已有大量单元/集成测试和多条安全 CI。主要维护风险是仓库体量与构建时间、依赖面较大，以及部分依赖 Go 内部/弃用 API 的测试断言和 Windows 本地测试下的不稳定行为。

## 2. 清理记录

本次没有删除文件、目录、依赖或锁文件。清理前和报告生成前跟踪文件约 3,600 个；确认的未跟踪发布资产为 36 个。Go 与前端依赖声明均未改变。

检查到但保留的项目文件：

- `release-assets/` 的跨平台归档、校验和及 PNG 属于用户未跟踪数据/发行资产，不能当作垃圾。
- `frontend/.tmp-vitest.log`、`frontend/*.tsbuildinfo` 被 `.gitignore` 明确忽略；来源和用途没有充分依据证明可删除。
- `backend/internal/web/dist/` 是前端嵌入构建目标；本次构建更新了它，未清理。
- 配置、迁移、数据库、密钥材料及未知用途文件没有删除。

依赖审计只覆盖声明、锁文件和运行入口的静态识别；没有证明任何依赖可安全移除，因此删除依赖数为 0。未执行全量依赖升级。

## 3. GitHub 竞品研究

通过 GitHub REST API 核查仓库元数据和顶层源码结构，时间为 2026-11-09（UTC）。Star 与更新时间会变化，以下数值是核查时的快照，不代表质量排名。重点研究了 README、仓库目录/模块分层、许可证元数据、提交活跃时间；未对每个项目逐个审阅全部 issue、发布说明与源码，因此不据此声称完成人工全仓代码审计。

| 仓库 | Stars | 最近推送 | 许可证元数据 | 结构与可借鉴点 |
| --- | ---: | --- | --- | --- |
| [BerriAI/litellm](https://github.com/BerriAI/litellm) | 62,954 | 2026-11-09 | API 返回 `NOASSERTION`；仓库存在 `LICENSE`，应在复用具体组件前逐项核查 | Python/Rust 核心并行、provider/router、插件、测试、部署、cookbook 分层；适合参考 provider 能力清单与网关扩展边界 |
| [QuantumNous/new-api](https://github.com/QuantumNous/new-api) | 51,608 | 2026-11-09 | AGPL-3.0 | Go 后端按 controller/service/relay/model/router 分层，另有 web、plugins、e2e；与 Sub3API 技术及产品形态接近，可对照管理能力和协议适配边界 |
| [songquanpeng/one-api](https://github.com/songquanpeng/one-api) | 37,364 | 2026-01-09 | MIT | Go 单体服务，controller/middleware/model/relay/router/web 结构；可作较轻量的统一 API 与令牌管理基线，活跃度低于近期持续推送的仓库 |
| [Portkey-AI/gateway](https://github.com/Portkey-AI/gateway) | 13,352 | 2026-05-25 | MIT | TypeScript 网关、插件、部署配置与测试分层；可对照 guardrail/plugin 接口及配置化路由，技术栈不同 |

| 对比维度 | Sub3API 当前证据 | 竞品可参考方向 | 差距/行动判断 |
| --- | --- | --- | --- |
| 核心功能 | 多协议 AI 网关、账号/渠道/组、API key、用量、运维、支付等已有实现与路由 | LiteLLM/New API 的 provider/router 能力整理 | 当前更适合补齐现有 provider 的兼容与验收，不宜仅追求 provider 数量 |
| UI/体验 | Vue 管理台、响应式页面、PWA、多语言；发行资产含移动管理页面截图 | New API 的独立 web 目录；Portkey 的配置与操作入口 | 本次未运行浏览器做页面逐项比较；不据截图或 README 给 UI 质量打分 |
| 架构 | Go handler/service/repository/schema + Vue API/store/view/features | 明确插件、provider、relay 适配边界 | 现有分层完整；后续重构应以具体耦合/重复证据为依据 |
| 性能/稳定性 | 已有缓存、连接池、代理/超时测试；全量测试出现 VMess 与 HTTP/2 测试失败 | 参考网关的池化、失败策略、性能基准 | 应先稳定本地/CI 测试和记录基准，再做性能改造 |
| 扩展能力 | 多 provider 与支付实现，前后端扩展分布于相应模块 | LiteLLM provider/router、New API plugins、Portkey plugins | 先定义适配契约和兼容测试，避免把不同业务职责合并 |
| 安全 | 安全配置审计、密钥加密选项、CI secret/dependency/security workflows | 参考各项目的权限最小化、安全发布和供应链流程 | 本次做静态入口检查，未进行渗透测试或完整依赖漏洞扫描 |
| 文档 | 中、英、日 README、开发指南、部署与 release notes | New API 多语言文档，LiteLLM 架构/贡献文档 | 文档覆盖面较好；避免文档承诺超出实测实现 |
| 测试 | 1,102 个 Go 测试文件、226 个前端测试文件；CI 有 unit/integration/lint 与安全 workflow | 参考竞品 e2e、插件契约与 provider 一致性测试 | 现有测试数量多但本机全量执行并非全绿，优先稳定和 CI 环境对齐 |

## 4. 发现的问题与优先级

### 高优先级

1. **HTTP/2 测试原先依赖 Go 内部实现细节**：测试直接断言 `TLSNextProto["h2"]`，而 Go 1.27 把配置迁移至 `Transport.Protocols`。现已改为兼容 Go 1.26/1.27 的公开配置检查；Go 1.26.10 与 1.27.0 的定向 HTTP/2 测试均通过。该测试改动不是线上 HTTP/2 功能故障的证据。
2. **VMess 端到端回归已定位并修复本地测试与客户端缺陷**：响应认证标记原先读取到临时字节切片，导致 `session.respAuth` 始终未设置；现已保存随机值。fake server 现在先读取并验证客户端请求首块，避免 Windows 下关闭仍有未读数据的 TCP 连接触发 reset，并通过错误通道报告解析失败。客户端 VMess 记录也改为完整写入，处理短写及零写。`TestVMESSDialerEndToEnd -count=50` 在 Go 1.27.0 通过，Go 1.26.10 下重复测试也通过。fake server 属于同仓库测试实现，不等同于独立 V2Ray/Xray 服务端互操作验证。

### 一般问题

- `pnpm lint:check` 初始结果有 1 个错误和 1 个 warning，均位于测试文件；本次清理未使用变量并修正 `this` alias 错误后，相关两文件 ESLint 通过。其他全仓 ESLint 文件仍需复核。
- 前端生产构建通过，但 Rolldown 报告日文语言包 chunk 超过 500 kB，建议按语言动态加载并核对 PWA 离线行为后再拆包。
- `AnnouncementPopup` 测试使用英文 locale 时报告 `common.time.daysAgo`、`common.time.never` 缺失；对应 key 实际在 en common locale 中存在，属于测试 locale/mocking 配置差异，本次未修改。

## 5. 本次本地升级与验证

- `backend/internal/pkg/proxyutil/vmess.go`：修复响应认证标记未保存的缺陷；为 VMess AEAD 请求头及数据记录增加完整写入循环，处理 `net.Conn.Write` 合法短写与零写返回。
- `backend/internal/pkg/proxyutil/protocols_test.go`：fake VMess 服务端完整读取并验证请求首块、将解析错误反馈给测试，同时按完整记录写出响应，避免关闭时未读请求数据导致的 Windows TCP reset。
- `backend/internal/repository/http_upstream_http2_keepalive_test.go`：断言兼容 Go 1.26 的 `TLSNextProto` 与 Go 1.27 的 `Transport.Protocols`，避免依赖单一版本的结构表达。
- `frontend/src/components/keys/__tests__/UseKeyModal.spec.ts`：删除未使用测试常量。
- `frontend/src/utils/__tests__/chartExport.spec.ts`：避免 `this` 局部别名，测试仍验证下载 anchor 的文件名与数据 URL。
- 未新增产品功能、未改 API/数据库 schema/依赖，也未进行 UI 重设计；竞品发现形成后续建议，未复制第三方代码。

验证结果：

| 检查 | 结果 |
| --- | --- |
| 前端 `pnpm build` | 通过；生成后端嵌入式 dist；有大 chunk 警告 |
| 前端 `pnpm typecheck` | 通过 |
| 前端全量 `pnpm test:run` | 失败：1538 个断言通过，但并行执行有 110 个 Vitest unhandled teardown/module load errors；不可标记全绿 |
| 前端全量串行 Vitest | 失败：226 个文件/1538 个断言通过，但仍有 105 个 `EnvironmentTeardownError`（326.73 秒）；并行度不是唯一原因 |
| 前端定向 Vitest（图表导出、公告弹窗） | 2 个文件、19 个测试通过 |
| 前端两处改动文件 ESLint | 通过 |
| 前端全量 `pnpm exec eslint .`（修复后） | 通过；本次修复前的 1 error 与 1 warning 均清除 |
| 后端 `go vet ./...` | 通过 |
| 后端 Go 1.27 定向 VMess e2e | 通过：`TestVMESSDialerEndToEnd -count=50` |
| 后端 Go 1.26.10 定向 VMess 与 HTTP/2 | 通过：VMess 重复测试、HTTP/2 keepalive 定向测试 |
| 后端 Go 1.26.10 全量 unit | 本轮正在运行；最终状态待命令退出后补录 |
| 后端 golangci-lint | 启动后未获得可确认的完成结果；不报告通过 |
| Integration/E2E/数据库回归 | 未运行；未启动 PostgreSQL、Redis 或服务 |
| 端口与进程安全 | 未启动服务；未占用或关闭其他进程 |

## 6. 清理、架构、UI 与依赖结论

- **删除文件/依赖：** 0。清理前后依赖声明不变。
- **迁移/整合：** 0。没有将竞品代码或功能迁入项目。
- **架构：** 未做大范围目录或层级重构；当前后端分层和前端功能目录具备可维护基础。
- **UI/UX：** 没有运行浏览器/人工页面检查，因此未作已升级声明。
- **性能：** 构建产物显示多语言资源 chunk 较大；未做服务负载或内存基准测试。
- **安全：** 检查了安全工作流、配置入口与常见敏感字段搜索；没有进行渗透测试、完整 SBOM 漏洞复核或部署环境验证。

## 7. 建议的后续工作

1. 完成仓库 CI 指定 Go 1.26.10 全量 unit/integration 回归；本轮全量 unit 结果待更新。
2. 以独立 V2Ray/Xray 等兼容节点做 VMess 端到端互操作验证；当前仅通过同仓库 fake server 回归。
3. 用 `pnpm exec vitest run --no-file-parallelism --maxWorkers=1` 复核全量前端测试；若 unhandled teardown 仍发生，再定位隔离/模块加载问题。
4. 修复全仓 ESLint 后重跑 lint；重跑 `golangci-lint run ./...` 并记录完成结果。
5. 对语言包使用动态导入做拆包评估，再以生产构建和 PWA 离线测试验收。
6. 后续审计若考虑删除本地忽略文件或发行资产，先由用户确认具体路径及用途；本轮不触碰这些数据。

## 8. Git 与文件变更记录

本报告记录的审计阶段没有执行 `git add`、`git commit`、`git push`、远程 tag、Release 或上传；后续 v0.4.1 发布动作由本次交付流程完成。

本次新增：`PROJECT_AUDIT_REPORT.md`。
本次修改：两处 VMess 源码/测试文件；两处前端测试文件。
本次删除/移动：无。
开始任务时已存在且保留：`backend/internal/service/balance_notify_service.go`、`backend/internal/service/user_notification_service.go` 的用户格式修改，以及 `release-assets/` 未跟踪资产。

验收前需重点关注：后端 VMess e2e、Go 1.26.10 下 HTTP/2 测试、前端全量 Vitest teardown errors、全仓 ESLint。当前成果是一次有限范围的本地审计与小修复，不应视为整个项目全面升级完成。

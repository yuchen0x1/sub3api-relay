# 平台、发布与维护状态

## 产品与基线

审计起点为 main 的 4a3701e09，正式版 v0.4.5，3,676 个跟踪文件。请求先经过 Gin 的协议路由与认证中间件，再进入账号调度、额度检查、用量记录及 PostgreSQL 持久化。Redis 承担并发和短期状态。后台服务负责监控、定时任务、备份、支付回调与审计。Vue 管理台嵌入 Go 进程，浏览器直接连接部署实例。数据库演进使用 backend/migrations，数据和加密密钥由部署者持有。desktop 中的 React/Tauri 客户端已有账号与静态出口视图，属于实验项目。

| 原有能力 | 所在模块与外部条件 |
|---|---|
| 模型协议转换、路由、限流、调度 | backend/internal/server/routes、handler、service；连接运营者获授权的上游 |
| API Key、用户分组、额度、订阅、账单 | Ent schema、repository、payment；PostgreSQL、Redis，收款依赖商户配置 |
| 注册、密码、OAuth、Passkey、TOTP | backend/internal/service、frontend/src/views/auth；外部身份源及邮件按需配置 |
| 用户控制台及管理员运维 | frontend/src/views/user、admin、features；权限由服务端决定 |
| 告警、通道健康、审计、备份、异步图片任务 | 后端调度器；S3、通知服务等按需配置 |
| 安装、更新、回退 | deploy 脚本、内置管理 API、Workbench 风格的 Web 控制台与 PWA 缓存 |

基线 CI 在 v0.4.5 成功。原流水线未将标签发布与完整 CI 串联，Docker 构建未发布镜像，脚本自动发现版本时可能把日志写入版本变量，PWA 自动激活可能打断编辑。v0.5.0 围绕这四项调整，无数据库迁移。后端设计中现存的 API Key 明文索引与浏览器会话存储见 [安全边界](../SECURITY_BOUNDARY.md)。

## 平台取舍

分类 A 表示现有通道继续维护，B 表示本轮补齐有实际价值的适配，C 表示保留候选，D 表示缺少合理的核心运行条件。服务端能力取决于数据库、缓存、网络和部署权限。客户端只需连接服务端。

| 平台 | 分类与交付 | 使用与维护边界 |
|---|---|---|
| Windows 10、11 | A，amd64/arm64 服务端压缩包及桌面 Web | 运行 Go 进程，外接 PostgreSQL/Redis。安装服务和系统托盘是独立产品需求。Windows 10 和 ARM 设备需单独实测 |
| Linux、Ubuntu、Debian、Fedora、Arch | A，amd64/arm64 压缩包及 systemd 安装、升级脚本 | shell 工具、systemd 和管理员权限；发行版差异集中在依赖和服务管理。Ubuntu 为流水线运行环境；其他发行版需验证 |
| macOS Intel、Apple Silicon | A，darwin 服务端包与 Web | 两种原生架构；macOS ARM 纳入可执行文件验证。桌面安装器涉及签名、公证及独立维护，C |
| FreeBSD amd64 | A，保留已有服务端分发 | 交叉构建；rc.d 与真机运行待验，systemd 脚本仅用于 Linux |
| 桌面 Web、移动 Web、响应式 Web | A，Vue 控制台 | 同源 API 和权限模型。桌面使用完整管理界面，移动端通过已有响应式布局和触控操作 |
| PWA | B，可安装 Web 控制台及延后刷新 | 依赖 HTTPS 或 loopback、浏览器 Service Worker 支持。静态资源可缓存，账户、账单与管理写入需要在线服务 |
| Android 手机、Android 平板、折叠屏 | A，Web/PWA。原生管理客户端 C，本地托管网关 D | 浏览器承担窗口与触控适配。专用 APK 增加签名、商店审核和机型矩阵，后台常驻与数据库维护成本高 |
| iOS、iPadOS | A，Web 与主屏幕访问。原生 C，本地常驻网关 D | 平台管理后台生命周期与安装，需 Safari/真机验收。无 App Store 包 |
| Chrome、Edge、Firefox 扩展 | C | 网关操作集中在独立站点，扩展需要附加站点权限、商店审核与三个分发渠道。浏览器可直接访问 Web |
| Docker、NAS、私有化、云服务器 | B，Linux AMD64/ARM64 镜像、Compose 及持久卷 | 私有主机需可用的容器运行时、存储、备份与 TLS。NAS 型号及厂商界面需实测 |
| CLI | A，服务端版本、初始化、Linux 更新工具 | 复用同一服务端与版本规则，操作系统管理员负责备份和重启 |
| 原生 Windows GUI、Linux AppImage/DEB/RPM/Flatpak | C | 已有自托管服务和远程管理台。为账号管理额外分发 GUI 会引入凭据存储与签名更新责任 |

## 功能差距与选择

| 能力 | Sub3API 基线 | 对照与设计取舍 |
|---|---|---|
| API Key、预算、路由 | 已有按用户、Key、分组的用量与账户调度 | LiteLLM 的预算模型、Bifrost 的预算展示、new-api 的额度服务适合作为跨组预算模拟参考。涉及账务一致性，另行设计 |
| 遥测与排错 | 已有运营面板、错误分类和审计 | Helicone 按会话浏览、Langfuse 按 trace 下钻，可减少跨视图定位。新增跨请求关联要同时评估存储与隐私 |
| 部署可验证性 | 有校验清单，标签作业独立运行，镜像只在 runner 上构建 | 采用 GoReleaser 将工件与版本绑定的思路，复用现存资产名，增加来源证明和原生运行检查 |
| 升级期间编辑体验 | Workbox 自动更新 | 参考 Vite PWA 的 waiting/controlling 事件；保留当前操作，提供刷新和稍后两个明确动作 |
| 告警与恢复 | 有定时备份、手动恢复和健康检查 | 参考 restic 将恢复列为可单独验证的操作，给系统升级增加真实启动与健康失败回退演练 |
| 计量与收款 | 账单、订阅和支付回调由网关持有 | OpenMeter 的事件计量和 Lago 的指标分层可用于日后对账。当前无需引入这些服务 |

## 30 项产品与工程候选

| # | 项目、用户价值 | 依据 | 优先级 / 难度 | 决定 |
|---|---|---|---|---|
| 1 | 标签发布等待完整 CI，阻止测试失败版本上线 | 原 release 工作流 | A / 中 | 实施 |
| 2 | 发布作业与构建权限隔离 | GitHub Actions | A / 中 | 实施 |
| 3 | 构建依赖固定到 Action SHA | GitHub Actions | A / 低 | 实施 |
| 4 | 发布清单及压缩包的来源证明 | GitHub Attest | A / 中 | 实施 |
| 5 | Linux 更新工具校验来源标签 | gh attestation | B / 中 | 实施为可选严格模式 |
| 6 | 自动发现版本输出纯值 | 原 update-binary.sh | A / 低 | 实施 |
| 7 | 升级前核对可执行文件版本 | GoReleaser | A / 低 | 实施 |
| 8 | 防止同时替换运行中的程序 | 原 Linux 升级脚本 | A / 中 | 实施 |
| 9 | 原子替换并保留上一版 | restic 恢复流程、现有备份机制 | A / 中 | 实施 |
| 10 | systemd 健康失败回退演练 | Traefik 健康检查 | A / 中 | 实施 |
| 11 | 分发 AMD64、ARM64 容器 | Compose 默认镜像契约 | A / 中 | 实施 |
| 12 | 对发布镜像运行健康探针 | Caddy、Traefik | A / 低 | 实施 |
| 13 | 在原生 runner 运行打包后的程序 | 现有七种 Go 目标 | B / 中 | 实施四种架构组合 |
| 14 | PWA 更新前保护未保存表单 | Vite PWA、Workbox | A / 中 | 实施 |
| 15 | 更新提示遵从中英日语言设置 | 已有 i18n | A / 低 | 实施 |
| 16 | 页面隐藏和离线时降低检查开销 | Workbox 生命周期 | B / 低 | 实施 |
| 17 | 安全报告私密提交 | GitHub 安全报告入口 | A / 低 | 实施 |
| 18 | 更新失败 Issue 收集架构与失败阶段 | 发行渠道差异 | B / 低 | 实施 |
| 19 | PR 标明迁移与兼容影响 | 当前 SQL 迁移体系 | B / 低 | 实施 |
| 20 | 定期依赖更新与 PR 依赖审查 | Dependabot | A / 低 | 实施 |
| 21 | 月度同类项目活动采样 | 30 个来源与正式版本 | B / 低 | 实施 |
| 22 | 预算保存前预览各组影响 | Bifrost、LiteLLM | C / 中 | 需明确跨组计量与账期 |
| 23 | 按事件回放对账 | OpenMeter、Lago | C / 高 | 需确定账本、不变量和重算协议 |
| 24 | 请求级追踪下钻 | Langfuse、Helicone | C / 中 | 先定义保留期、脱敏及索引成本 |
| 25 | 运营告警规则离线校验 | Prometheus | C / 中 | 待规则格式稳定 |
| 26 | 按监控故障展示恢复时间线 | Uptime Kuma | C / 中 | 需持久化故障事件定义 |
| 27 | 备份恢复演练覆盖真实对象存储 | restic | C / 高 | 需专用恢复目标及访问权限 |
| 28 | 折叠屏细分布局和原生离线编辑 | 现有响应式控制台 | C / 高 | 等真机和操作需求，账单修改必须在线 |
| 29 | 浏览器扩展注入网关密钥 | 本项目服务端凭据边界 | D / 高 | 不适合现有安全模型 |
| 30 | 三端独立的原生商店客户端 | Tauri 实验端、Web 管理台 | C / 高 | 需独立日常任务、签名身份与维护人员 |

## 维护入口

[完整的 30 个项目证据、版本和活动快照](research.md) · [实时 CI](https://github.com/yuchen0x1/sub3api-relay/actions/workflows/backend-ci.yml) · [安全扫描](https://github.com/yuchen0x1/sub3api-relay/actions/workflows/security-scan.yml) · [代码扫描（CodeQL）](https://github.com/yuchen0x1/sub3api-relay/actions/workflows/codeql.yml) · [发布流水线](https://github.com/yuchen0x1/sub3api-relay/actions/workflows/release.yml) · [当前正式版本](https://github.com/yuchen0x1/sub3api-relay/releases/latest)

CodeQL 于 2026-11-19 启用，覆盖 Go 与 JavaScript/TypeScript，测试夹具已通过 `codeql-config.yml` 排除；首批生产代码命中在 [Issue #27](https://github.com/yuchen0x1/sub3api-relay/issues/27) 分诊。

每月采集一次项目公开活动，核对发布资产；大版本前复核功能差距和真机覆盖。计费、真支付、真实账号、物理 NAS、手机及完整生产数据库恢复需独立验收。实验端未纳入原生安装器或自动更新覆盖。

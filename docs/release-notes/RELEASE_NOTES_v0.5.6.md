## 📢 版本概述

本次补丁修复了安全扫描和依赖层面的两个问题：全历史密钥扫描配置在固定版本 gitleaks 下实际未生效，以及 WebSocket 依赖存在已公开的弱随机数安全公告。同时统一了前端依赖覆盖、清理了陈旧审计产物，并同步中英日文发行说明。升级无需数据库迁移或配置变更。

## 🔒 安全与兼容性

- **WebSocket 依赖安全修复**：升级 `github.com/gorilla/websocket` 至包含修复的 `v1.5.4-0.20240701034025-d67f41855da4`，处理 GO-2026-6278 / GHSA-w67g-5rqw-f597。该依赖用于管理端 QPS 实时 WebSocket 等路径；升级后 API 与连接协议不变。
- **恢复全历史密钥扫描信号**：修复 `.gitleaks.toml` 在 gitleaks 8.24.3 下的配置格式问题。此前全局豁免表被静默忽略，每周全历史扫描会报告 136 条已核实的误报并持续失败；修复后同一范围内为 0 条泄漏。
- **依赖覆盖统一**：将前端依赖覆盖统一到 `pnpm-workspace.yaml`，`dompurify` 实际安装版本由 3.4.17 对齐到 3.4.18，并重新生成锁文件。

## 🐛 问题修复

- **发行信息不同步**：README 的“最新更新”恢复为当前实际发行状态；README_CN、README_JA 从 v0.4.4 更新到当前版本。
- **发行状态记录错误**：更新历史中将 v0.5.4 标记为“未发布、变更已并入 v0.5.5”，将 v0.5.5 标记为 2026-11-23 已发布，并移除 `RELEASING.md` 中过期的 v0.5.4 待发布段落。
- **前端测试偶发超时**：`SettingsView` 的异步组件等待增加超时余量，避免全量并行运行时因机器负载偶发失败。
- **陈旧审计产物**：移除未被引用、且仍引用已不存在 `xlsx` 依赖的 `frontend/audit.json`。

## 🔧 改进与优化

- **README 同步流程**：自动同步工作流改为每次从当前默认分支重建维护分支，只管理 README 的发行标记块，避免维护分支落后后 PR 永久冲突。
- **CI 可观测性**：修复后的密钥扫描不会再被历史误报淹没，新出现的真实凭据泄漏会直接体现在每周扫描结果中。

## 📦 下载与安装

| 文件名 | 平台 / 架构 | 说明 |
|---|---|---|
| `sub3api_0.5.6_linux_amd64.tar.gz` | Linux x64 | 服务端二进制压缩包 |
| `sub3api_0.5.6_linux_arm64.tar.gz` | Linux arm64 | 服务端二进制压缩包 |
| `sub3api_0.5.6_darwin_amd64.tar.gz` | macOS Intel | 服务端二进制压缩包 |
| `sub3api_0.5.6_darwin_arm64.tar.gz` | macOS Apple Silicon | 服务端二进制压缩包 |
| `sub3api_0.5.6_windows_amd64.tar.gz` | Windows x64 | 服务端二进制压缩包 |
| `sub3api_0.5.6_windows_arm64.tar.gz` | Windows arm64 | 服务端二进制压缩包 |
| `sub3api_0.5.6_freebsd_amd64.tar.gz` | FreeBSD x64 | 服务端二进制压缩包 |
| `sub3api_0.5.6_web.tar.gz` | 通用 | 可嵌入的 Web 管理台构建产物 |
| `sub3api_0.5.6_deploy.tar.gz` | Linux | 安装、Docker 部署示例与校验清单 |
| `checksums.txt` | 通用 | 所有发行归档的 SHA-256 校验和 |

- 首次安装：参见 [部署说明](https://github.com/yuchen0x1/sub3api-relay/blob/main/deploy/README.md)。
- 已安装用户：管理台会读取最新正式 Release 并提示新版本；也可直接下载对应平台压缩包替换二进制。

## 🔄 升级与数据迁移

- **兼容性**：可直接覆盖升级，无需数据库迁移。
- **配置和数据**：现有配置文件、数据库和加密密钥保持不变；建议升级前按既有流程备份数据库。
- **回退方式**：可重新安装上一版本二进制回退；本版本未改变数据库结构。

## 🧪 验证结果

- Go 1.26.11：`go build ./...`、`go vet ./...`、`go test -tags=unit ./...`、`golangci-lint v2.9.0`（0 issues）通过。
- Go 全量 `go test ./...`：通过。
- 前端：`pnpm typecheck`、`pnpm lint:check`、`pnpm exec vitest run`（232 个文件 / 1558 个用例）、`pnpm build` 通过；`pnpm audit` 未发现已知漏洞。
- 安全扫描：gitleaks 8.24.3 全历史扫描 0 条泄漏；govulncheck v1.7.0 未发现可达漏洞。
- 需要 Docker 的集成测试与发布平台冒烟由 GitHub Actions 在标签发布时执行。

## ⚠️ 已知问题

- 浏览器会话令牌仍按现有安全边界存储在 localStorage，迁移到 HttpOnly Cookie 的架构工作仍在 #27 跟踪；本补丁未改变认证协议。
- `desktop` 下的 Tauri 客户端为实验性脚手架，版本号独立，不在此次发行渠道内。

---

**项目地址：** https://github.com/yuchen0x1/sub3api-relay
**问题反馈：** https://github.com/yuchen0x1/sub3api-relay/issues
**版本下载：** https://github.com/yuchen0x1/sub3api-relay/releases/tag/v0.5.6

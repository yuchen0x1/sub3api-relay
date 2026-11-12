# 自动更新最终上线闭环验收

验收日期：2026-11-12  
仓库：`yuchen0x1/sub3api-relay`  
目标版本：`v0.4.2`  
提交：`0e9ab6ad1f61c09a12cceb1059dbb621e6cdcfe5`

## 结论

项目实际交付形态是 Web 管理台、服务器二进制和 Docker 部署，不是 Windows、Linux 或 macOS 桌面应用。仓库中的 `desktop/` 是可选 Tauri 脚手架，不是当前项目对外发布的桌面产品。

因此，Tauri 桌面签名、桌面安装包、`.sig`、`latest.json` 和旧桌面客户端升级，不属于当前项目正式上线闭环的验收条件。

## 四个状态

| 状态 | 结果 | 证据 |
|---|---|---|
| 代码已接入 | 已确认 | Web 管理台、服务器二进制和 Docker 更新路径已存在；Tauri 代码属于可选脚手架 |
| 发布流水线已配置 | 已确认 | `.github/workflows/release.yml` 校验后端/前端版本并发布服务器、Web 和部署资产 |
| 首次正式版本已成功发布 | 已确认 | GitHub Release `v0.4.2` 已于 2026-11-12 创建并包含通用发布资产 |
| 旧客户端已成功更新到新版本 | 不适用 | 当前项目没有对外发布的桌面客户端；服务器二进制和 Docker 更新需要按部署方式验证 |

## 检查结果

### 1. 版本、标签和 Release

- 本地和远端标签：`v0.4.2`。
- 后端、前端版本与标签版本 `0.4.2` 一致。
- GitHub Release：`Sub3API v0.4.2`，非草稿、非预发布。
- Release 说明文件：`docs/release-notes/RELEASE_NOTES_v0.4.2.md`。

### 2. GitHub Actions

- 通用 `build-and-validate`：成功。
- 早先的桌面矩阵尝试曾因缺失 Tauri 签名 Secret 失败；该任务已经从正式流水线移除。
- 当前正式流水线只负责服务器、Web、部署和校验资产。

### 3. Secrets

当前正式服务器/Web Workflow 使用内置 `GITHUB_TOKEN`。不需要配置 Tauri signing secrets。

本机仍存在 Tauri 私钥，但它不属于当前服务器/Web 发布流程，未输出、未提交。

### 4. Release 资产

`v0.4.2` Release 已包含通用校验文件、后端平台压缩包、Web 包和部署包，下载地址可访问。

桌面 `latest.json` 返回 HTTP 404，这是预期结果，因为当前项目没有发布桌面更新通道。

### 5. 平台发布矩阵

当前 Release 的 Linux、macOS、Windows、FreeBSD 条目是 Go 服务器二进制压缩包，不是桌面安装程序。不能把它们描述为 Windows/Linux/macOS 桌面版。

## 当前剩余工作

1. 按实际部署方式验证服务器二进制升级或 Docker 镜像升级。
2. 如果未来决定正式发布桌面版，再单独恢复桌面矩阵、签名 Secret、`latest.json` 和旧桌面客户端升级测试。

## 最终判断

截至 2026-11-12，对于当前服务器/Web 项目，推送版本标签即可触发正式 Release 构建和发布；不存在需要验证的 Windows/Linux/macOS 桌面自动更新闭环。


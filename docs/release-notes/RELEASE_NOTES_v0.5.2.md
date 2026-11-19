## 📢 版本概述

本次补丁更新后端、管理台和 CI 依赖，包含构建及安全扫描工具的主版本升级。升级无需数据库迁移。

## 🔧 依赖更新

- **后端 Go 依赖**：`aws-sdk-go-v2/credentials` 升级至 1.21.0，`spf13/viper` 升级至 1.21.0，`testcontainers-go/modules/postgres` 升级至 0.45.0。
- **前端依赖**：`vue` 升级至 3.5.45，`dompurify` 升级至 3.4.18，`vite-plugin-checker` 升级至 1.0.0。
- **CI 动作**：`actions/setup-node` 升级至 7.1.0、`pnpm/action-setup` 升级至 6.1.1、`gitleaks/gitleaks-action` 升级至 3.0.0。

## 🔒 安全与兼容性

- 发布提交通过 Go 可达漏洞检查、gitleaks 3.0.0 密钥扫描，以及 Go 和 TypeScript CodeQL 扫描。
- `vite-plugin-checker` 1.0.0 与 `vue` 3.5.45 通过管理台构建、类型检查、Lint 和前端单元测试。

## 📦 下载与安装

| 文件名 | 平台 / 架构 | 说明 |
|---|---|---|
| `sub3api_0.5.2_linux_amd64.tar.gz` | Linux x64 | 服务端二进制 |
| `sub3api_0.5.2_linux_arm64.tar.gz` | Linux ARM64 | 服务端二进制 |
| `sub3api_0.5.2_darwin_amd64.tar.gz` | macOS Intel | 服务端二进制 |
| `sub3api_0.5.2_darwin_arm64.tar.gz` | macOS Apple Silicon | 服务端二进制 |
| `sub3api_0.5.2_windows_amd64.tar.gz` | Windows x64 | 服务端二进制 |
| `sub3api_0.5.2_windows_arm64.tar.gz` | Windows ARM64 | 服务端二进制 |
| `sub3api_0.5.2_freebsd_amd64.tar.gz` | FreeBSD x64 | 服务端二进制 |
| `sub3api_0.5.2_web.tar.gz` | 浏览器 | Web 管理台静态资源 |
| `sub3api_0.5.2_deploy.tar.gz` | Linux / Docker Compose | 安装与部署脚本和示例配置 |
| `checksums.txt` | 全部平台 | 九个归档的 SHA-256 校验值 |

- **首次安装**：按[安装文档](https://github.com/yuchen0x1/sub3api-relay/blob/v0.5.2/README.md)选择服务端二进制或 Docker Compose 部署。
- **已安装用户**：按[升级指南](https://github.com/yuchen0x1/sub3api-relay/blob/v0.5.2/docs/AUTO_UPDATE.md)更新；Docker 镜像 `ghcr.io/yuchen0x1/sub3api-relay:0.5.2` 提供 Linux AMD64 和 ARM64。

## 🔄 升级与数据迁移

- **兼容性**：可直接覆盖升级，无已知不兼容的旧版本范围。
- **配置和数据**：现有数据库、持久卷和部署配置保持不变；升级前请先备份。
- **回退方式**：Linux 二进制更新脚本在健康检查失败时自动恢复原二进制；Docker 部署可用上一版本镜像摘要回退。

## 🧪 验证结果

- 本地通过 `go test ./...`、`go vet ./...`、前端锁文件校验和管理台构建；发布 CI 的 `unit` 标签单元测试与 `integration` 标签集成测试均通过。
- 发布流水线完成九个归档的 SHA-256 校验、Linux 升级与回退演练。Linux x64、Linux ARM64、macOS ARM64 和 Windows x64 二进制通过版本检查，Linux AMD64 与 ARM64 镜像均通过 `/health` 探针。[查看发布流水线](https://github.com/yuchen0x1/sub3api-relay/actions/runs/44510138464)
- FreeBSD x64、macOS Intel 和 Windows ARM64 归档通过构建与校验，未在对应平台执行运行验证。

---

- **项目地址：** [https://github.com/yuchen0x1/sub3api-relay](https://github.com/yuchen0x1/sub3api-relay)
- **问题反馈：** [https://github.com/yuchen0x1/sub3api-relay/issues](https://github.com/yuchen0x1/sub3api-relay/issues)
- **版本下载：** [https://github.com/yuchen0x1/sub3api-relay/releases/tag/v0.5.2](https://github.com/yuchen0x1/sub3api-relay/releases/tag/v0.5.2)

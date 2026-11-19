## 📢 版本概述

本次补丁修复支付订单限额和结账信息错误处理，并完善部分日志字段的脱敏和安全扫描。升级无需数据库迁移。

## 🐛 问题修复

- **并发下单**：同一用户的待支付订单数和每日限额检查在创建订单时串行执行，避免并发请求突破上限。
- **每日支付限额**：待支付订单预占额度，余额充值按实际应付金额计入限额。
- **结账套餐**：套餐查询失败时，结账接口返回错误，页面可区分加载失败和确实没有在售套餐。

## 🔒 安全与兼容性

- 加强代理连接及部分管理端日志的敏感字段处理，并将 Go 和 TypeScript 的 CodeQL 扫描纳入仓库检查。

## 📦 下载与安装

| 文件名 | 平台 / 架构 | 说明 |
|---|---|---|
| `sub3api_0.5.1_linux_amd64.tar.gz` | Linux x64 | 服务端二进制 |
| `sub3api_0.5.1_linux_arm64.tar.gz` | Linux ARM64 | 服务端二进制 |
| `sub3api_0.5.1_darwin_amd64.tar.gz` | macOS Intel | 服务端二进制 |
| `sub3api_0.5.1_darwin_arm64.tar.gz` | macOS Apple Silicon | 服务端二进制 |
| `sub3api_0.5.1_windows_amd64.tar.gz` | Windows x64 | 服务端二进制 |
| `sub3api_0.5.1_windows_arm64.tar.gz` | Windows ARM64 | 服务端二进制 |
| `sub3api_0.5.1_freebsd_amd64.tar.gz` | FreeBSD x64 | 服务端二进制 |
| `sub3api_0.5.1_web.tar.gz` | 浏览器 | Web 管理台静态资源 |
| `sub3api_0.5.1_deploy.tar.gz` | Linux / Docker Compose | 安装与部署脚本和示例配置 |
| `checksums.txt` | 全部平台 | 九个归档的 SHA-256 校验值 |

- **首次安装**：按[安装文档](https://github.com/yuchen0x1/sub3api-relay/blob/v0.5.1/README.md)选择服务端二进制或 Docker Compose 部署。
- **已安装用户**：按[升级指南](https://github.com/yuchen0x1/sub3api-relay/blob/v0.5.1/docs/AUTO_UPDATE.md)更新；Docker 镜像 `ghcr.io/yuchen0x1/sub3api-relay:0.5.1` 提供 Linux AMD64 和 ARM64。

## 🔄 升级与数据迁移

- **兼容性**：可直接覆盖升级，无已知不兼容的旧版本范围。
- **配置和数据**：现有数据库、持久卷和部署配置保持不变；升级前请先备份。
- **回退方式**：Linux 二进制更新脚本在健康检查失败时自动恢复原二进制；Docker 部署可用上一版本镜像摘要回退。

## 🧪 验证结果

- 本地通过 Go 全量测试、支付相关 `unit` 标签测试和 `go vet`。
- 发布流水线完成归档 SHA-256 校验、Linux 升级与回退演练，以及 Linux x64、Linux ARM64、macOS ARM64、Windows x64 的二进制版本检查。Linux AMD64 与 ARM64 镜像均通过 `/health` 探针。[查看发布流水线](https://github.com/yuchen0x1/sub3api-relay/actions/runs/44492428077)
- FreeBSD x64、macOS Intel 和 Windows ARM64 归档通过构建及校验，未在对应平台执行运行验证。

---

- **项目地址：** [https://github.com/yuchen0x1/sub3api-relay](https://github.com/yuchen0x1/sub3api-relay)
- **问题反馈：** [https://github.com/yuchen0x1/sub3api-relay/issues](https://github.com/yuchen0x1/sub3api-relay/issues)
- **版本下载：** [https://github.com/yuchen0x1/sub3api-relay/releases/tag/v0.5.1](https://github.com/yuchen0x1/sub3api-relay/releases/tag/v0.5.1)

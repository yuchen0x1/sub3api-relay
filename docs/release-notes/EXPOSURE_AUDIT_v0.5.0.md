# v0.5.0 逆向暴露面审计

日期：2026-11-18。审计对象是本地构建的 v0.5.0 Web 资源与 Windows amd64 服务端二进制。扫描器只读文件，不执行产物，也不联网。

## 结果

| 目标 | 结果 | 说明 |
|---|---|---|
| `backend/internal/web/dist` | 100 / A | 170 个文件、122 个 JavaScript 资源。未发现 source map 或静态凭据规则命中。 |
| Windows amd64 二进制 | 原始 14 / D | 已使用 `-trimpath -s -w`。扫描器命中 1 个数据库 URI 模板、1 个 RSA 标记、OAuth 回调模板、`entitlement` 字段、PE 未签名和 CFG 标记缺失。 |

## 人工核对

- `postgres://15:04:05-07` 是 Go 时间格式字符串，`postgresql://PGSERVICEFILE` 是 PostgreSQL 客户端环境变量文档字符串，不是连接凭据。
- `-----BEGIN RSA PRIVATE KEY-----` 只命中支付测试用例中使用的示例标记，未在发行源码或构建环境发现完整私钥块。
- `localhost:8085` 与 `127.0.0.1:56121` 是 Antigravity/X OAuth 的本地回调地址，运行时用于本地授权回调，不是部署内网地址。
- `entitlement` 是上游账户状态字段。服务端决定权限、配额和计费，客户端没有最终授权判定。
- Windows 发行二进制当前未签名，CFG 标记缺失。构建来源证明和 SHA-256 校验在 v0.5.0 发布流水线中提供，Authenticode 签名需要后续证书与签名环境。

## 后续处理

Web 资源继续保持不发布 source map。生产环境使用发布资产的 SHA-256，并可设置 `SUB3API_REQUIRE_ATTESTATION=1` 让 Linux 更新脚本验证 GitHub 构建来源。Windows Authenticode、macOS 公证和桌面安装器仍属于独立发布通道，当前没有标记为已完成。

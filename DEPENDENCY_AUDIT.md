# 依赖审计（2026-08-22，2026-08-31 更新）

按升级方案卡 3 要求，逐个检查以下依赖是否被核心路径引用。结论：**三项均被核心路径引用，无需裁剪**。

| 依赖 | 引用位置 | 结论 |
| --- | --- | --- |
| apernet/hysteria/core/v2 | `internal/pkg/proxyutil/dialer.go`、`internal/pkg/proxyurl/parse.go`（Hysteria2 代理拨号与订阅解析） | 已移除（2026-08-31）：上游存在未修复的 OOM 漏洞，运行时依赖与新建入口全部下线，存量 Hysteria2 代理由迁移 197 停用保留 |
| go-webauthn/webauthn | `internal/config/config.go`、`internal/service/passkey.go`、`internal/repository/passkey_repo.go`（Passkey 免密登录） | 保留 |
| alibabacloud-go/captcha | `internal/repository/aliyun_captcha_verifier.go`、`internal/handler/auth_handler.go`、`internal/config/config.go`（阿里云滑块验证） | 保留 |

检查方法：`rg -l <关键词> --glob "*.go" internal`，并人工确认非测试文件的引用点。

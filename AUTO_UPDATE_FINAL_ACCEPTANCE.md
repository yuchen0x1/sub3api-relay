# 自动更新最终上线闭环验收

验收日期：2026-11-12  
仓库：`yuchen0x1/sub3api-relay`  
目标版本：`v0.4.2`  
提交：`0e9ab6ad1f61c09a12cceb1059dbb621e6cdcfe5`

## 结论

当前不能宣布自动更新闭环成功，也不能确认以后每次只推送标签就一定能完成桌面自动更新。

`v0.4.2` 标签已经推送，通用后端 Release 已创建，版本校验和通用构建成功；但是三个桌面发布任务都在 Tauri 更新签名阶段失败，因此 Release 没有 Windows、Linux、macOS 的桌面安装包、`.sig` 和 `latest.json`。旧客户端真实更新也没有发生。

## 四个状态

| 状态 | 结果 | 证据 |
|---|---|---|
| 代码已接入 | 已确认 | Tauri updater 插件、权限、公钥、HTTPS endpoint、前端检查/下载/安装/重启逻辑已在提交 `0e9ab6ad1` 中；本地公钥与配置一致 |
| 发布流水线已配置 | 已确认但凭据未满足 | `.github/workflows/release.yml` 包含版本校验、通用构建、三平台桌面矩阵、签名上传和 `latest.json` 上传 |
| 首次正式版本已成功发布 | 部分完成 | GitHub Release `v0.4.2` 已于 2026-11-12 创建，但只含通用后端/部署资产，不含桌面更新资产 |
| 旧客户端已成功更新到新版本 | 未完成 | 没有可用的 `latest.json`、桌面安装包或 `.sig`，未进行真实下载、验签、安装和重启测试 |

## 检查结果

### 1. 版本、标签和 Release

- 本地和远端标签：`v0.4.2`。
- 后端、前端、桌面 package、Tauri 配置和 Cargo 版本均为 `0.4.2`。
- GitHub Release：`Sub3API v0.4.2`，非草稿、非预发布。
- Release 说明文件：`docs/release-notes/RELEASE_NOTES_v0.4.2.md`。

### 2. GitHub Actions

运行：`43136544509`。

- `build-and-validate`：成功。
- `desktop-release (windows-latest, x86_64-pc-windows-msvc)`：失败。
- `desktop-release (ubuntu-22.04, x86_64-unknown-linux-gnu)`：失败。
- `desktop-release (macos-latest, aarch64-apple-darwin)`：失败。

三个桌面任务均完成了编译和普通打包，但在生成更新签名时失败，日志错误为：

`failed to decode secret key: incorrect updater private key password: Missing comment in secret key`

Actions 日志中的 `TAURI_SIGNING_PRIVATE_KEY` 和 `TAURI_SIGNING_PRIVATE_KEY_PASSWORD` 均为空。仓库 API token 当前也没有写入 Actions Secrets 的权限，因此无法在本地代替仓库管理员配置它们。

### 3. Secrets

Workflow 使用：

- `GITHUB_TOKEN`：由 GitHub 自动提供；通用 Release 上传已证明可用。
- `TAURI_SIGNING_PRIVATE_KEY`：当前运行中为空，必须配置。
- `TAURI_SIGNING_PRIVATE_KEY_PASSWORD`：当前运行中为空。本机密钥可无密码签名；若继续使用该密钥，可以保持为空，但建议按团队密钥策略明确配置并记录。

本机存在 `C:\Users\lenmo\.tauri\sub3api-cockpit.key`，并且本机无密码签名成功；私钥内容未输出、未提交、未写入本文档。其公钥与 `desktop/src-tauri/tauri.conf.json` 中的 updater 公钥完全一致。

### 4. Release 资产和更新元数据

`v0.4.2` 当前 Release 含 9 个通用资产：校验文件、后端平台压缩包、Web 包和部署包；这些下载地址可访问。

缺少：

- Windows MSI/NSIS 安装包及 `.sig`；
- Linux AppImage/deb/rpm 及 `.sig`；
- macOS app/dmg 及 `.sig`；
- `latest.json`。

已直接请求：

- `https://github.com/yuchen0x1/sub3api-relay/releases/download/v0.4.2/checksums.txt`：HTTP 200；
- `https://github.com/yuchen0x1/sub3api-relay/releases/download/v0.4.2/latest.json`：HTTP 404。

因此目前无法验证更新 URL、安装包与签名的对应关系，也无法验证客户端 updater 接口。

### 5. 三平台矩阵

Workflow 当前矩阵为：

- Ubuntu 22.04：`x86_64-unknown-linux-gnu`；
- Windows latest：`x86_64-pc-windows-msvc`；
- macOS latest：`aarch64-apple-darwin`。

三项均能完成应用编译和普通 bundle 生成；签名步骤失败后没有上传桌面 Release 资产。当前矩阵也没有覆盖 Windows ARM64、Linux ARM64 或 macOS Intel 桌面安装包，这些平台不应被宣称为已验证。

## 必须手动完成的操作

1. 在 `yuchen0x1/sub3api-relay` 的 **Settings → Secrets and variables → Actions** 中配置 `TAURI_SIGNING_PRIVATE_KEY`，值为与 `desktop/src-tauri/tauri.conf.json` 公钥配对的完整 Tauri/minisign 私钥。
2. 如果私钥设置了密码，配置 `TAURI_SIGNING_PRIVATE_KEY_PASSWORD`；如果继续使用当前无密码私钥，保持该变量为空即可。
3. 重新运行 `v0.4.2` 的失败 workflow，或在确认 Release 资产策略后重新发布一个新的版本标签。不要用不同私钥覆盖已有客户端公钥。
4. 确认 Release 出现全部桌面安装包、每个安装包对应的 `.sig` 以及 `latest.json`。
5. 在一台安装旧桌面版本的 Windows 电脑上执行真实更新：检查更新、下载、验签、安装、重启，并确认启动后的版本为 `0.4.2`。

## 最终判断

截至 2026-11-12，答案是：**还不能只通过更新版本号并推送 GitHub 标签，就可靠完成桌面自动更新。** 标签推送和通用 Release 发布已经工作；桌面签名 Secrets 缺失导致正式更新渠道尚未发布，旧客户端自动更新闭环尚未真实验证。


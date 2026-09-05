# 二进制部署升级指南（v0.1.178+）

Sub3API 从 `v0.1.178` 开始把前端构建产物直接嵌入后端二进制和 Docker 镜像。服务器升级时只需要替换后端二进制并重启 systemd 服务，前端会随二进制一起更新。

## 为什么会出现“后端更新了、前端还是旧版”

`v0.1.178` 之前采用前后端分离发布：

- GitHub Release 只包含后端二进制。
- 前端单独打包为 `sub3api_<版本>_web.tar.gz`。
- 服务器由 Caddy 直接托管一个静态前端目录（例如 `/var/www/sub3api-v2-aee102038`）。
- 只更新后端、没有同步前端包时，页面仍是旧版，Manus 网关等新功能不会出现。

`v0.1.178` 之后的发布方式：

- GoReleaser 使用 `-tags embed` 构建，前端静态资源内嵌进 `sub3api` 二进制。
- Dockerfile 增加前端构建阶段，镜像同样包含完整前端。
- Release 仍会附带 `sub3api_<版本>_web.tar.gz`，仅用于兼容旧式静态目录部署，常规二进制升级不再需要它。

## 服务器升级步骤（标准）

```bash
# 备份现场
sudo cp /opt/sub3api/sub3api /opt/sub3api/sub3api.backup

# 下载并校验指定版本（示例 v0.1.178）
cd /tmp
curl -fL -O https://github.com/yuchen0x1/sub3api-relay/releases/download/v0.1.178/sub3api_0.1.178_linux_amd64.tar.gz
curl -fL -O https://github.com/yuchen0x1/sub3api-relay/releases/download/v0.1.178/checksums.txt
sha256sum -c <(grep 'sub3api_0.1.178_linux_amd64.tar.gz' checksums.txt)
tar -xzf sub3api_0.1.178_linux_amd64.tar.gz

# 替换二进制并重启
sudo systemctl stop sub3api
sudo cp sub3api /opt/sub3api/sub3api
sudo chmod +x /opt/sub3api/sub3api
sudo chown sub3api:sub3api /opt/sub3api/sub3api
sudo systemctl start sub3api

# 验证
/opt/sub3api/sub3api -version
curl -fsS http://127.0.0.1:8080/health
```

或者使用仓库内置的 `deploy/update-binary.sh`：

```bash
# 首次使用时给脚本加执行权限（也可以 sudo bash deploy/update-binary.sh）
chmod +x deploy/update-binary.sh

# 升级到最新版
sudo ./update-binary.sh

# 升级到指定版本
sudo ./update-binary.sh v0.1.178

# 非默认安装布局（例如二进制在 backend/bin/server、服务名 sub2api）
sudo SUB3API_BIN=/opt/sub2api/backend/bin/server \
     SUB3API_SERVICE=sub2api \
     SUB3API_HEALTH_URL=http://127.0.0.1:3000/health \
     ./update-binary.sh v0.1.178
```

## Caddy 必须全站反代到后端

Caddy 配置原则：站点所有路径（包括 `/` 和 `/manus/`）都必须 `reverse_proxy` 到后端监听端口，不要再把 `/var/www/...` 静态目录用 `file_server` 单独暴露。

正确示例（所有流量都进后端）：

```caddy
your-domain.com {
    reverse_proxy 127.0.0.1:3000
}
```

错误示例（会造成前后端不一致）：

```caddy
your-domain.com {
    root * /var/www/sub3api-v2-aee102038
    file_server
    reverse_proxy 127.0.0.1:3000   # 部分路径仍由旧静态目录响应
}
```

说明：

- 后端内嵌中间件会自动放行 `/api/`、`/v1/`、`/manus/`、`/health` 等接口路径，剩余请求返回内嵌前端页面。
- 旧静态目录（例如 `/var/www/sub3api-v2-aee102038`）建议保留为现场备份，但不要继续挂到 Caddy。
- 修改 Caddy 后必须校验并重载：`sudo caddy validate --config /etc/caddy/Caddyfile && sudo systemctl reload caddy`。

## 升级后的验证清单

- [ ] `/opt/sub3api/sub3api -version` 输出目标版本
- [ ] `systemctl is-active sub3api` 为 `active`
- [ ] `curl -fsS http://127.0.0.1:8080/health` 返回 200
- [ ] 浏览器打开站点，确认出现新版前端与 Manus 网关入口
- [ ] 如果 Caddy 有改动，`caddy validate` 通过且站点 HTTPS 正常

## 回滚

```bash
sudo systemctl stop sub3api
sudo cp /opt/sub3api/sub3api.backup /opt/sub3api/sub3api
sudo chmod +x /opt/sub3api/sub3api
sudo systemctl start sub3api
```

> 注意：不要删除旧静态目录 `/var/www/...` 和旧 `_web.tar.gz` 现场备份，除非确认不再需要回滚到旧式部署。

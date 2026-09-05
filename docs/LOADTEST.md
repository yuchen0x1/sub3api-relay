# 压测基线（loadtest）

工具：`backend/cmd/loadtest`（无依赖 Go 压测器，强制直连、忽略系统代理）。

```bash
cd backend
go build -o bin/loadtest ./cmd/loadtest
bin/loadtest -url http://127.0.0.1:18081/health -d 10s -c 8
bin/loadtest -url http://127.0.0.1:18081/api/v1/admin/dashboard/stats \
  -d 15s -c 8 -header "Authorization: Bearer <token>"
```

常用参数：`-n` 总请求数（默认按 `-d` 时长）、`-c` 并发、`-method`、`-body`、`-header`（可重复）。

## 本机基线（2026-09-02，Windows 开发机 + Docker PG/Redis，embed 构建，c=8）

| 端点 | RPS | avg | p50 | p90 | p99 | 错误 |
| --- | --- | --- | --- | --- | --- | --- |
| GET /health | 50,839 | 142µs | ~0 | 529µs | 1.01ms | 0 |
| GET /（静态 UI） | 23,033 | 171µs | ~0 | 532µs | 722µs | 0 |
| GET /api/v1/admin/dashboard/stats | 1,058 | 7.5ms | 7.5ms | 8.2ms | 9.1ms | 0 |
| GET /api/v1/admin/users?limit=20 | 497 | 16.1ms | 16.0ms | 17.2ms | 18.4ms | 0 |

结论：静态与健康检查路径在数万 RPS 级；管理台 DB 端点在数百至千 RPS、p99 < 20ms，
对单实例自用网关而言余量充足。

## 注意事项

- 本机存在 `http_proxy=127.0.0.1:7897`（无 scheme）。loadtest 已强制 `Proxy: nil` 直连；
  其他工具（curl 加 `--noproxy '*'`、Go HTTP 客户端自定义 Transport）同样需要绕开，
  否则流量会打到本机代理，导致结果失真甚至线程爆炸。
- 中转链路（/v1/chat/completions 等）压测依赖真实上游账号，未纳入默认基线；
  需要时用 `-body` 指定请求 JSON、`-header` 携带 API Key 对目标端点压测。

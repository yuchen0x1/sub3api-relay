# Playwright E2E 测试

针对本地 embed 环境（生产同构构建）的管理台核心流程冒烟测试。

## 前置条件

1. 构建并启动本地环境（见 DEV_GUIDE）：
   - 前端：`pnpm build`（产物进 `backend/internal/web/dist`）
   - 后端：`go build -tags embed -o bin/server-dev.exe ./cmd/server`
   - 启动：`DATA_DIR=<backend 目录> ./bin/server-dev.exe`（默认 127.0.0.1:18081）
2. 数据库中存在 E2E 管理员账号（密码/邮箱经环境变量注入）。

## 运行

```bash
pnpm test:e2e
# 或指定环境与凭据
E2E_BASE_URL=http://127.0.0.1:18081 \
E2E_ADMIN_EMAIL=e2e-admin@local.demo \
E2E_ADMIN_PASSWORD=e2e-admin-local \
pnpm test:e2e
```

## 用例范围

| 文件 | 覆盖 |
| --- | --- |
| `auth.setup.ts` | 管理员登录，保存 storageState 供后续项目复用 |
| `auth.spec.ts` | 登录页渲染、错误口令报错、未登录访问管理台被拦截 |
| `admin-pages.spec.ts` | 管理台 7 个核心路由渲染冒烟：无 pageerror、内容非白屏；settings 页签壳可见 |
| `ops-settings.spec.ts` | 阶段 4 验收点：监控自动禁用联动同平台账号暂停的开关存在 |
| `public.spec.ts` | 未登录可访问公开落地页 |

## 产物

- 失败截图/trace：`e2e/.artifacts/`（已 gitignore）
- 登录态：`e2e/.auth/admin.json`（已 gitignore）

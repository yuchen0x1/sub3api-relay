# Sub3API 项目进化报告

日期：2026-11-12（Asia/Shanghai）

## 1. 项目原始情况

Sub3API 是一个 Go + Vue 3 + Tauri 的 AI API 网关和管理平台，提供多上游账号/渠道调度、统一协议入口、API Key、用量/计费、运维、认证、备份和 PWA/桌面端能力。当前工作分支为 `upgrade/autonomous-2026-11-10`，HEAD 为 `4ba6d66f76e768f202d8b80e9258f5856d0cd508`。工作区在本轮开始前已有未提交自动升级改动，全部保留；没有执行 reset、分支切换、远程推送或发布。

## 2. 竞品研究清单

首轮筛选了 15 个项目，第二轮选出 8 个，深度研究 5 个。完整记录见 [`research.json`](./research.json)。重点参考 LiteLLM、New API、One API、Portkey Gateway 和 Bifrost 的 provider/router 边界、错误归一化、运维信息架构和测试组织方式。没有复制竞品代码或素材；许可证仅作为筛选依据，具体复用前仍需重新核验。

## 3. 功能差距分析

当前最重要的差距不是再增加 provider 数量，而是把现有 provider 的能力矩阵、协议契约和端到端验收做得更明确。P2/P3 后续项包括多实例 DingTalk token 共享缓存、usage group 聚合性能、前端大 chunk 拆分，以及桌面端更完整的组件级错误回归测试。完整矩阵见 [`feature_gap.json`](./feature_gap.json)。

## 4. 实际移植/实现功能

本轮完成了三个小范围、真实可验证的改进：

1. `desktop/src/features/staticExits/components/StaticExitsPage.tsx`：静态出口 toggle/delete 失败现在被捕获，并通过已有错误归一化逻辑显示可理解的错误面板，不再产生 unhandled rejection。
2. `desktop/src/lib/config.ts` 与 `desktop/README.md`：桌面端默认 API 地址统一为 `http://127.0.0.1:18081`，仍支持 `VITE_SUB3API_BASE_URL` 覆盖。
3. `frontend/src/views/admin/__tests__/AccountsView.bulkEdit.spec.ts`：所有 mount 使用既有 `trackMount`，使异步组件卸载清理真正生效，同时消除 lint warning。

## 5. UI 与交互升级

本轮没有进行大规模视觉重构；只补齐了桌面静态出口操作的失败反馈，避免用户点击后无响应。现有项目已经具备多主题、多语言、响应式/PWA 和管理台组件体系，因此下一步应基于浏览器实测和用户流程证据做增量升级，而不是复制竞品界面。

## 6. 架构优化

本轮保持现有 Go handler/service/repository、Vue API/store/view/features 与 Tauri shell 分层。没有引入新依赖，没有跨项目文件操作。研究结论建议后续围绕 provider capability contract、错误分类、健康快照和性能基准做接口级优化。

## 7. 修改文件清单

- `desktop/src/features/staticExits/components/StaticExitsPage.tsx`
- `desktop/src/lib/config.ts`
- `desktop/README.md`
- `frontend/src/views/admin/__tests__/AccountsView.bulkEdit.spec.ts`
- `.evolution/project_identity.json`
- `.evolution/progress.json`
- `.evolution/research.json`
- `.evolution/feature_gap.json`
- `.evolution/implementation_log.json`
- `.evolution/test_report.json`
- `.evolution/final_report.md`

## 8. 新增依赖

无。

## 9. 测试结果

前端 typecheck、lint、226 个测试文件/1538 个测试、目标 AccountsView 7 个测试均通过；桌面端直接调用本地 `tsc`、Vite 和 Vitest 二进制，4 个测试文件/19 个测试通过，生产构建通过。后端本轮无法重跑，因为 Go 不在 PATH；Docker-backed 集成测试无法运行，因为 Docker Desktop Linux daemon 在 2026-11-12 不可用。之前已验证的后端结果和限制保存在 `.autonomous-upgrade/UPGRADE_LOG.md`，本报告没有把未执行项目标为通过。详见 [`test_report.json`](./test_report.json)。

## 10. 未完成事项与残余风险

- 后端 build/unit/vet/govulncheck 需要在 Go 工具链可用环境中重跑。
- PostgreSQL/Redis 集成测试需要启动 Docker daemon 后执行。
- 桌面 package scripts 仍可能因 pnpm 的 ignored build scripts 阻断；直接二进制验证已通过。
- 前端构建仍有大 chunk 警告，尚未做路由级拆包。
- DingTalk token 多实例缓存、usage group 全表聚合、内容违规时 API Key mutation 等既有 P2/P3 项未在本轮改动。
- 未执行浏览器 E2E、真实服务启动、外部上传、发布或桌面安装包验收。

## 11. 项目进化评分（证据化）

| 维度 | 改造前 | 改造后 | 依据 |
| --- | ---: | ---: | --- |
| 功能完整度 /25 | 20 | 21 | 桌面静态出口错误路径补齐；主要网关功能本轮未扩展 |
| UI 与体验 /20 | 15 | 16 | 操作失败有反馈，默认桌面连接可用性对齐 |
| 稳定性 /20 | 15 | 16 | 消除一个未处理 Promise 路径；前端/桌面回归通过 |
| 架构与代码质量 /15 | 12 | 12 | 保持分层，清除一项测试清理失效/ lint warning |
| 性能与安全 /10 | 7 | 7 | 本轮未改变核心热路径；性能项仍待基准 |
| 文档与工程交付 /10 | 7 | 8 | 增加独立 evolution 状态、研究、测试和最终报告 |
| **合计 /100** | **76** | **80** | 仅按本轮有证据的改进计分 |

## 12. 下一阶段建议

1. 在 CI 或 Go 环境执行 backend build/unit/vet/govulncheck，并启动 Docker 跑真实集成测试。
2. 为桌面静态出口页补 React component test，覆盖 toggle/delete rejection、错误面板和 retry/reload 行为。
3. 建立 provider capability matrix 与协议契约测试，优先覆盖 OpenAI、Claude、Gemini、Grok 及图片/视频入口。
4. 针对前端大 chunk 做路由级拆包和构建预算门禁。
5. 以实际数据量和查询计划为依据处理 DingTalk 分布式缓存与 usage summary 聚合性能。

# Design System Themes

Phase 1 主题层（对应 `upgrade-report/02-UI升级方案.md` 2.2.2）。

| 文件 | 职责 |
|------|------|
| `tokens.css` | 语义 Token：颜色 / 阴影 / 圆角 / 字体 / 图表色；`:root` 浅色 + `.dark` 深色 + `prefers-color-scheme` 系统回退 + `prefers-reduced-motion` |
| `dark.css` | 深色模式 UI 层微调（`color-scheme`、body 底色、滚动条），依赖 tokens.css |
| `chart-theme.ts` | Chart.js 调色板单源（`chartColors` / `chartColorAlpha` / `chartPalette`），9 个图表组件已接入并通过 `pnpm build` |

## 使用约定

- 新增颜色一律引用 Token：`var(--color-*)`、`rgb(var(--color-*) / <alpha>)`、Tailwind `token-*` 类；禁止裸 hex。
- 柔和底纹语义色：`--color-success-soft` / `--color-info-soft` / `--color-warning-soft` / `--color-danger-soft`（浅色 50/100 级、深色 900 级）。
- 深色适配：`.dark` 作用域覆盖 Token；组件内分支用 `dark:` Tailwind 变体。
- 主题切换统一走 `useAppStore()`：`setTheme('light'|'dark'|'system')` / `toggleTheme()`；`main.ts` 挂载前调用 `initTheme()`，系统主题变化由 store 内 `matchMedia` 监听实时跟随。

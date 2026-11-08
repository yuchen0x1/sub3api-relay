# Design System Themes

Phase 1 主题层（对应 UI 升级方案 2.2.2）。

| 文件 | 职责 |
|------|------|
| `tokens.css` | 语义 Token：颜色 / 阴影 / 圆角 / 字体 / 图表色；`:root` 浅色 + `.dark` 深色 + `prefers-color-scheme` 系统回退 + `prefers-reduced-motion` |
| `dark.css` | 深色模式 UI 层微调（`color-scheme`、body 底色、滚动条），依赖 tokens.css |
| `accent.ts` | 强调色预设（6 套）：运行时覆写 `<html>` 上的 `--color-primary-*` 与 `--brand-*` |
| `chart-theme.ts` | Chart.js 调色板单源（`chartColors` / `chartColorAlpha` / `chartPalette`），9 个图表组件已接入并通过 `pnpm build` |

## 使用约定

- 新增颜色一律引用 Token：`var(--color-*)`、`rgb(var(--color-*) / <alpha>)`、Tailwind `token-*` 类；禁止裸 hex。
- 柔和底纹语义色：`--color-success-soft` / `--color-info-soft` / `--color-warning-soft` / `--color-danger-soft`（浅色 50/100 级、深色 900 级）。
- 深色适配：`.dark` 作用域覆盖 Token；组件内分支用 `dark:` Tailwind 变体。
- 主题切换统一走 `useAppStore()`：`setTheme('light'|'dark'|'system')` / `toggleTheme()`；`main.ts` 挂载前调用 `initTheme()`，系统主题变化由 store 内 `matchMedia` 监听实时跟随。

## 主题三档与强调色

- 三档主题：`light` / `dark` / `system`。store 会同时在 `<html>` 上写 `light` 或 `dark` class，
  显式选择浅色时不会被 `tokens.css` 里的 `prefers-color-scheme` 回退块覆盖。
- 强调色：`accent.ts` 的 `applyAccent(id)` 在 `<html>` 上写两套变量——
  `--color-primary-{shade}`（十六进制，喂给 Tailwind 工具类）和 `--brand-{shade}`（RGB 三元组，喂给语义 Token 层）。
  `primary` 阶梯色因此必须声明在 `src/style.css` 的 `@theme` 里（而非 `tailwind.config.js`），
  否则工具类会编译成静态色值导致换肤失效。
- 切换入口是 `components/common/AppearanceMenu.vue`（侧栏底部「外观」），面板 Teleport 到 body，
  因为 `.sidebar` 是 `overflow: hidden`，内部浮层会被裁切。
- 图表调色板（`chart-theme.ts`）是固定的分类色序，不跟随强调色：多系列图表需要互相可区分的颜色。


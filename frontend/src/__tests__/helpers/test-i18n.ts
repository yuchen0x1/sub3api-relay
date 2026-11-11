/**
 * 测试用 i18n 装配辅助。
 *
 * 背景：`AccountsView.vue` 用 `defineAsyncComponent` 声明了 13 个弹窗，
 * 且它们在模板里始终渲染（只用 `:show` 控制可见性）。因此 `mount()` 会立刻
 * 启动全部动态 import。这些真实组件内部调用 `useI18n()`：
 *
 * - 若测试只 `vi.mock('vue-i18n')` 替换 `useI18n` 却不安装插件，
 *   真实组件渲染时会抛 `Need to install with app.use function`；
 * - 该异常发生在测试结束、jsdom 环境销毁之后，Vitest 报成
 *   `EnvironmentTeardownError` 并以非零码退出，看起来像"环境抖动"。
 *
 * 正确做法是安装一个真实 i18n 实例：缺少翻译时 vue-i18n 原样返回 key，
 * 与原先 `t: (key) => key` 的 mock 行为一致，同时保证插件链路完整。
 */

import { createI18n } from 'vue-i18n'

export type TestI18n = ReturnType<typeof createTestI18n>

/** 创建空消息的真实 i18n 实例（`t(key)` 原样返回 key）。 */
export function createTestI18n(locale = 'en') {
  return createI18n({
    legacy: false,
    locale,
    fallbackLocale: 'en',
    missingWarn: false,
    fallbackWarn: false,
    messages: { [locale]: {} },
  })
}

/**
 * 等待挂起的异步组件加载与渲染收敛。
 *
 * `defineAsyncComponent` 的 loader 不会被 `stubs` 阻止，因此挂载后必须给它
 * 完成时间，否则 import 会在环境销毁后才 resolve。
 */
export async function settleAsyncComponents(rounds = 3, delayMs = 120): Promise<void> {
  for (let i = 0; i < rounds; i += 1) {
    await new Promise((resolve) => setTimeout(resolve, delayMs))
  }
}

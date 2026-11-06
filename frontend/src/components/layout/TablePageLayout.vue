<template>
  <div class="table-page-layout" :class="{ 'mobile-mode': isMobile }">
    <!-- 固定区域：操作按钮 -->
    <div v-if="$slots.actions" class="layout-section-fixed">
      <slot name="actions" />
    </div>

    <!-- 固定区域：搜索和过滤器 -->
    <div v-if="$slots.filters" class="layout-section-fixed layout-section-filters">
      <slot name="filters" />
    </div>

    <!-- 滚动区域：表格 -->
    <div class="layout-section-scrollable">
      <div class="card table-scroll-container">
        <slot name="table" />
      </div>
    </div>

    <!-- 固定区域：分页器 -->
    <div v-if="$slots.pagination" class="layout-section-fixed">
      <slot name="pagination" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'

const isMobile = ref(false)

const checkMobile = () => {
  isMobile.value = window.innerWidth < 1024
}

onMounted(() => {
  checkMobile()
  window.addEventListener('resize', checkMobile)
})

onUnmounted(() => {
  window.removeEventListener('resize', checkMobile)
})
</script>

<style scoped>
@reference "../../style.css";
/* 桌面端：Flexbox 布局 */
.table-page-layout {
  @apply flex flex-col gap-4;
  height: calc(100vh - 64px - 4rem); /* 减去 header + lg:p-8 的上下padding */
}

.layout-section-fixed {
  @apply flex-shrink-0;
}

/* 筛选/动作工具栏：统一 HUD 面板语言 */
.layout-section-filters {
  @apply rounded-xl border px-4 py-3;
  border-color: var(--glass-border);
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.04), transparent 60%),
    var(--glass-bg);
  box-shadow: var(--shadow-card);
  backdrop-filter: blur(14px) saturate(140%);
  -webkit-backdrop-filter: blur(14px) saturate(140%);
}

.layout-section-scrollable {
  @apply flex-1 min-h-0 flex flex-col;
}

/* 表格滚动容器 - 增强版表体滚动方案 */
.table-scroll-container {
  @apply flex flex-col overflow-hidden h-full rounded-xl border;
  border-color: var(--glass-border-strong);
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.035), transparent 42%),
    var(--glass-bg-strong);
  box-shadow: var(--shadow-card);
}

.table-scroll-container :deep(.table-wrapper) {
  @apply flex-1 overflow-x-auto overflow-y-auto;
  /* 确保横向滚动条显示在最底部 */
  scrollbar-gutter: stable;
}

.table-scroll-container :deep(table) {
  @apply w-full;
  min-width: max-content; /* 关键：确保表格宽度根据内容撑开，从而触发横向滚动 */
  display: table; /* 使用标准 table 布局以支持 sticky 列 */
}

.table-scroll-container :deep(thead) {
  @apply backdrop-blur-sm;
  background: color-mix(in srgb, var(--glass-bg-strong) 82%, transparent);
}

.table-scroll-container :deep(tbody) {
  /* 保持默认 table-row-group 显示，不使用 block */
}

.table-scroll-container :deep(th) {
  @apply whitespace-nowrap px-4 py-3 text-left font-mono text-[11px] font-semibold uppercase tracking-[0.08em] text-gray-500 dark:text-dark-400;
  border-bottom: 1px solid var(--glass-border-strong);
}

.table-scroll-container :deep(td) {
  @apply px-4 py-3 text-sm text-gray-700 dark:text-dark-200;
  border-bottom: 1px solid color-mix(in srgb, var(--glass-border) 60%, transparent);
}

/* 移动端：恢复正常滚动 */
.table-page-layout.mobile-mode .table-scroll-container {
  @apply h-auto overflow-visible border-none shadow-none bg-transparent;
}

.table-page-layout.mobile-mode .layout-section-filters {
  @apply border-none shadow-none bg-transparent p-0 dark:bg-transparent;
}

.table-page-layout.mobile-mode .layout-section-scrollable {
  @apply flex-none min-h-fit;
}

.table-page-layout.mobile-mode .table-scroll-container :deep(table) {
  @apply flex-none;
  display: table;
  min-width: 100%;
}
</style>

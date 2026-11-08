<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'

const props = withDefaults(
  defineProps<{
    /** 是否在可横向滚动时显示一次性的滑动提示 */
    hint?: boolean
    /** 追加到滚动视口上的样式类（例如 max-h-80、rounded-lg、border） */
    viewportClass?: string
  }>(),
  { hint: true, viewportClass: '' }
)

const emit = defineEmits<{
  scroll: [event: Event]
}>()

const { t } = useI18n()

const HINT_STORAGE_KEY = 'table-scroll-hint-dismissed'

const viewportRef = ref<HTMLElement | null>(null)
const scrollLeft = ref(0)
const scrollWidth = ref(0)
const clientWidth = ref(0)
const hintDismissed = ref(false)

const isOverflowing = computed(() => scrollWidth.value > clientWidth.value + 4)
const hintVisible = computed(() => props.hint && !hintDismissed.value && isOverflowing.value)
const canScrollLeft = computed(() => scrollLeft.value > 1)
const canScrollRight = computed(() => scrollLeft.value + clientWidth.value < scrollWidth.value - 1)

let resizeObserver: ResizeObserver | null = null

function syncMetrics(): void {
  const el = viewportRef.value
  if (!el) return
  scrollLeft.value = el.scrollLeft
  scrollWidth.value = el.scrollWidth
  clientWidth.value = el.clientWidth
}

function dismissHint(): void {
  if (hintDismissed.value) return
  hintDismissed.value = true
  try {
    sessionStorage.setItem(HINT_STORAGE_KEY, '1')
  } catch {
    // sessionStorage 不可用时仅本次挂载有效
  }
}

function onScroll(event: Event): void {
  syncMetrics()
  if (scrollLeft.value > 4) dismissHint()
  emit('scroll', event)
}

onMounted(() => {
  try {
    hintDismissed.value = sessionStorage.getItem(HINT_STORAGE_KEY) === '1'
  } catch {
    hintDismissed.value = false
  }

  syncMetrics()

  const el = viewportRef.value
  if (el && typeof ResizeObserver !== 'undefined') {
    resizeObserver = new ResizeObserver(() => syncMetrics())
    resizeObserver.observe(el)
    if (el.firstElementChild) resizeObserver.observe(el.firstElementChild)
  }
  window.addEventListener('resize', syncMetrics)
})

onBeforeUnmount(() => {
  resizeObserver?.disconnect()
  resizeObserver = null
  window.removeEventListener('resize', syncMetrics)
})
</script>

<template>
  <div class="table-scroll">
    <div
      ref="viewportRef"
      class="table-scroll__viewport"
      :class="viewportClass"
      @scroll="onScroll"
    >
      <slot />
    </div>

    <div
      class="table-scroll__shadow table-scroll__shadow--left"
      :class="{ 'table-scroll__shadow--visible': canScrollLeft }"
      aria-hidden="true"
    ></div>
    <div
      class="table-scroll__shadow table-scroll__shadow--right"
      :class="{ 'table-scroll__shadow--visible': canScrollRight }"
      aria-hidden="true"
    ></div>

    <div v-if="hintVisible" class="table-scroll__hint" data-test="table-scroll-hint">
      <Icon name="arrowsUpDown" size="xs" class="flex-shrink-0" />
      <span>{{ t('common.scrollHorizontally') }}</span>
      <button
        type="button"
        class="table-scroll__hint-close"
        :aria-label="t('common.close')"
        data-test="table-scroll-hint-close"
        @click="dismissHint"
      >
        <Icon name="x" size="xs" />
      </button>
    </div>
  </div>
</template>

<style scoped>
.table-scroll {
  position: relative;
  display: flex;
  flex-direction: column;
}

.table-scroll__viewport {
  flex: 1 1 auto;
  min-height: 0;
  overflow: auto;
  border-radius: inherit;
  -webkit-overflow-scrolling: touch;
}

.table-scroll__shadow {
  position: absolute;
  top: 0;
  bottom: 0;
  width: 1.25rem;
  z-index: 2;
  border-radius: inherit;
  pointer-events: none;
  opacity: 0;
  transition: opacity 0.18s ease;
}

.table-scroll__shadow--visible {
  opacity: 1;
}

.table-scroll__shadow--left {
  left: 0;
  background: linear-gradient(to right, rgb(15 23 42 / 0.14), transparent);
}

.table-scroll__shadow--right {
  right: 0;
  background: linear-gradient(to left, rgb(15 23 42 / 0.14), transparent);
}

:global(.dark) .table-scroll__shadow--left {
  background: linear-gradient(to right, rgb(2 6 16 / 0.72), transparent);
}

:global(.dark) .table-scroll__shadow--right {
  background: linear-gradient(to left, rgb(2 6 16 / 0.72), transparent);
}

.table-scroll__hint {
  position: absolute;
  top: 0.5rem;
  right: 0.5rem;
  display: none;
  align-items: center;
  gap: 0.35rem;
  padding: 0.25rem 0.5rem;
  border-radius: 9999px;
  font-size: 0.6875rem;
  line-height: 1rem;
  white-space: nowrap;
  pointer-events: none;
  z-index: 20;
  color: rgb(var(--color-text-secondary));
  background: rgb(var(--color-surface) / 0.92);
  border: 1px solid rgb(var(--color-border));
  box-shadow: var(--shadow-card);
  backdrop-filter: blur(8px);
}

.table-scroll__hint-close {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  pointer-events: auto;
  color: rgb(var(--color-text-muted));
  transition: color 0.15s ease;
}

.table-scroll__hint-close:hover {
  color: rgb(var(--color-text));
}

@media (max-width: 767px) {
  .table-scroll__hint {
    display: flex;
  }
}
</style>

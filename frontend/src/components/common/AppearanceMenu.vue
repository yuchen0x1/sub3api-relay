<template>
  <div ref="triggerRef" class="relative">
    <button
      type="button"
      :class="triggerClass"
      :title="t('nav.appearance')"
      :aria-haspopup="'menu'"
      :aria-expanded="isOpen"
      @click="toggle"
    >
      <Icon name="sparkles" size="md" class="h-5 w-5 flex-shrink-0" />
      <span
        v-if="variant === 'sidebar'"
        class="label"
        :class="{ 'label-collapsed': collapsed }"
        :aria-hidden="collapsed ? 'true' : 'false'"
      >{{ t('nav.appearance') }}</span>
    </button>

    <Teleport to="body">
      <transition name="appearance-dropdown">
        <div
          v-if="isOpen"
          ref="panelRef"
          role="menu"
          :style="panelStyle"
          class="appearance-panel fixed z-[100] rounded-xl border border-white/10 bg-[#0a1120]/95 p-3 shadow-2xl backdrop-blur-xl"
        >
          <p class="mb-2 font-mono text-[10px] font-semibold uppercase tracking-[0.14em] text-slate-500">
            {{ t('nav.themeMode') }}
          </p>
          <div class="grid grid-cols-3 gap-1.5">
            <button
              v-for="option in themeOptions"
              :key="option.value"
              type="button"
              class="theme-option"
              :class="{ 'theme-option-active': appStore.themeMode === option.value }"
              :aria-pressed="appStore.themeMode === option.value"
              @click="appStore.setTheme(option.value)"
            >
              <Icon :name="option.icon" size="sm" class="h-4 w-4" />
              <span class="text-xs">{{ t(option.labelKey) }}</span>
            </button>
          </div>

          <p class="mb-2 mt-4 font-mono text-[10px] font-semibold uppercase tracking-[0.14em] text-slate-500">
            {{ t('nav.accentColor') }}
          </p>
          <div class="grid grid-cols-6 gap-1.5">
            <button
              v-for="preset in accentPresets"
              :key="preset.id"
              type="button"
              class="swatch"
              :class="{ 'swatch-active': appStore.accentId === preset.id }"
              :style="{ backgroundColor: preset.swatch }"
              :title="t(preset.labelKey)"
              :aria-label="t(preset.labelKey)"
              :aria-pressed="appStore.accentId === preset.id"
              @click="appStore.setAccent(preset.id)"
            >
              <Icon
                v-if="appStore.accentId === preset.id"
                name="check"
                size="xs"
                class="h-3 w-3 text-black/70"
              />
            </button>
          </div>
        </div>
      </transition>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores'
import { accentPresets } from '@/design-system/themes/accent'

const { t } = useI18n()
const appStore = useAppStore()

const props = withDefaults(defineProps<{ collapsed?: boolean; variant?: 'sidebar' | 'inline' }>(), {
  collapsed: false,
  variant: 'sidebar'
})

const triggerClass = computed(() => {
  if (props.variant === 'inline') {
    return 'rounded-lg p-2 text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-700 dark:text-dark-400 dark:hover:bg-dark-800 dark:hover:text-white'
  }
  return ['sidebar-link mb-2 w-full', props.collapsed ? 'link-collapsed' : '']
})

const isOpen = ref(false)
const triggerRef = ref<HTMLElement | null>(null)
const panelRef = ref<HTMLElement | null>(null)
const panelStyle = ref<Record<string, string>>({})

const themeOptions = [
  { value: 'light' as const, labelKey: 'nav.themeLight', icon: 'sun' as const },
  { value: 'dark' as const, labelKey: 'nav.themeDark', icon: 'moon' as const },
  { value: 'system' as const, labelKey: 'nav.themeSystem', icon: 'cpu' as const }
]

const PANEL_WIDTH = 248
const GAP = 8

function updatePosition(): void {
  const trigger = triggerRef.value
  if (!trigger) return
  const rect = trigger.getBoundingClientRect()
  const panelHeight = panelRef.value?.offsetHeight ?? 220
  const viewportWidth = window.innerWidth
  const viewportHeight = window.innerHeight

  // Inside the sidebar the panel flies out to the right of the rail so it does
  // not cover the navigation items; elsewhere it stays aligned to the trigger.
  const sidebar = trigger.closest('.sidebar')
  const anchorLeft = sidebar ? sidebar.getBoundingClientRect().right + GAP : rect.right - PANEL_WIDTH

  let left = anchorLeft
  let top = rect.bottom + GAP
  if (top + panelHeight > viewportHeight - GAP) {
    top = rect.top - GAP - panelHeight
  }
  if (top < GAP) {
    top = Math.min(rect.bottom + GAP, Math.max(GAP, viewportHeight - panelHeight - GAP))
  }
  if (left + PANEL_WIDTH > viewportWidth - GAP) {
    left = Math.max(GAP, rect.right - PANEL_WIDTH)
  }
  left = Math.min(Math.max(GAP, left), Math.max(GAP, viewportWidth - PANEL_WIDTH - GAP))

  panelStyle.value = {
    left: `${left}px`,
    top: `${top}px`,
    width: `${PANEL_WIDTH}px`
  }
}

function open(): void {
  isOpen.value = true
  void nextTick(updatePosition)
}

function toggle(): void {
  if (isOpen.value) {
    isOpen.value = false
    return
  }
  open()
}

function close(): void {
  isOpen.value = false
}

function handleClickOutside(event: MouseEvent): void {
  if (!isOpen.value) return
  const target = event.target as Node
  if (triggerRef.value?.contains(target) || panelRef.value?.contains(target)) return
  close()
}

function handleKeydown(event: KeyboardEvent): void {
  if (event.key === 'Escape' && isOpen.value) {
    close()
  }
}

onMounted(() => {
  document.addEventListener('click', handleClickOutside)
  document.addEventListener('keydown', handleKeydown)
  window.addEventListener('resize', close)
  window.addEventListener('scroll', close, true)
})

onBeforeUnmount(() => {
  document.removeEventListener('click', handleClickOutside)
  document.removeEventListener('keydown', handleKeydown)
  window.removeEventListener('resize', close)
  window.removeEventListener('scroll', close, true)
})
</script>

<style scoped>
.label {
  display: block;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  transition:
    max-width 0.2s ease,
    opacity 0.12s ease,
    transform 0.12s ease;
  max-width: 12rem;
}

.label-collapsed {
  max-width: 0;
  opacity: 0;
  transform: translateX(-4px);
  pointer-events: none;
}

.link-collapsed {
  gap: 0;
  padding-left: 0.875rem;
  padding-right: 0.875rem;
}

.appearance-panel {
  color: #e2e8f0;
}

.theme-option {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.25rem;
  padding: 0.5rem 0.25rem;
  border: 1px solid rgba(148, 178, 214, 0.14);
  border-radius: 0.5rem;
  background: rgba(148, 178, 214, 0.06);
  color: #94a3b8;
  font-weight: 500;
  transition: all 0.15s ease;
}

.theme-option:hover {
  background: rgba(148, 178, 214, 0.12);
  color: #e2e8f0;
}

.theme-option-active {
  border-color: rgb(var(--brand-400) / 0.45);
  background: rgb(var(--brand-400) / 0.14);
  color: rgb(var(--brand-300));
}

.swatch {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 1.5rem;
  border: 2px solid transparent;
  border-radius: 0.375rem;
  transition: transform 0.15s ease;
}

.swatch:hover {
  transform: translateY(-1px);
}

.swatch-active {
  border-color: rgba(255, 255, 255, 0.85);
  box-shadow: 0 0 0 2px rgba(4, 7, 15, 0.6);
}

.appearance-dropdown-enter-active,
.appearance-dropdown-leave-active {
  transition: all 0.15s ease;
}

.appearance-dropdown-enter-from,
.appearance-dropdown-leave-to {
  opacity: 0;
  transform: scale(0.96) translateY(-4px);
}
</style>

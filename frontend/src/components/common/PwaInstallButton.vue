<template>
  <button
    v-if="canInstall"
    type="button"
    class="inline-flex items-center gap-2 transition-colors"
    :class="buttonClass"
    :title="variant === 'sidebar' && collapsed ? t('nav.installApp') : undefined"
    @click="handleInstall"
  >
    <Icon name="download" :size="variant === 'sidebar' ? 'md' : 'sm'" class="flex-shrink-0" />
    <span
      class="sidebar-label"
      :class="{ 'sidebar-label-collapsed': variant === 'sidebar' && collapsed }"
      :aria-hidden="variant === 'sidebar' && collapsed ? 'true' : 'false'"
      >{{ t('nav.installApp') }}</span
    >
  </button>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { usePwaInstall } from '@/composables/usePwaInstall'

const props = withDefaults(
  defineProps<{
    variant?: 'sidebar' | 'inline'
    collapsed?: boolean
  }>(),
  { variant: 'inline', collapsed: false }
)

const { t } = useI18n()
const { canInstall, promptInstall } = usePwaInstall()

const buttonClass = computed(() =>
  props.variant === 'sidebar'
    ? 'sidebar-link mb-2 w-full'
    : 'rounded-lg border border-gray-200 bg-white px-3 py-1.5 text-xs font-medium text-gray-700 shadow-sm hover:bg-gray-50 dark:border-dark-600 dark:bg-dark-800 dark:text-gray-300 dark:hover:bg-dark-700'
)

async function handleInstall(): Promise<void> {
  await promptInstall()
}
</script>

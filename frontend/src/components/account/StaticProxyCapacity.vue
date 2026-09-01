<template>
  <div class="flex flex-col gap-0.5" :aria-label="t('admin.accounts.capacity.staticProxy.label')">
    <div class="flex items-center gap-1">
      <CapacityBadge
        data-test="static-capacity-total"
        :color-class="totalClass"
        :tooltip="totalTooltip"
        :current="capacity.current"
        :max="capacity.max"
      >
        <svg class="h-2.5 w-2.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2" aria-hidden="true">
          <path stroke-linecap="round" stroke-linejoin="round" d="M3.75 6A2.25 2.25 0 016 3.75h2.25A2.25 2.25 0 0110.5 6v2.25a2.25 2.25 0 01-2.25 2.25H6a2.25 2.25 0 01-2.25-2.25V6zM3.75 15.75A2.25 2.25 0 016 13.5h2.25a2.25 2.25 0 012.25 2.25V18a2.25 2.25 0 01-2.25 2.25H6A2.25 2.25 0 013.75 18v-2.25zM13.5 6a2.25 2.25 0 012.25-2.25H18A2.25 2.25 0 0120.25 6v2.25A2.25 2.25 0 0118 10.5h-2.25a2.25 2.25 0 01-2.25-2.25V6zM13.5 15.75a2.25 2.25 0 012.25-2.25H18a2.25 2.25 0 012.25 2.25V18A2.25 2.25 0 0118 20.25h-2.25A2.25 2.25 0 0113.5 18v-2.25z" />
        </svg>
      </CapacityBadge>
    </div>

    <div
      v-for="exit in capacity.exits"
      :key="exit.id"
      class="flex min-w-0 items-center gap-1 pl-3"
    >
      <span class="min-w-0 truncate text-[10px] text-gray-500 dark:text-gray-400">{{ exit.name }}</span>
      <CapacityBadge
        :data-test="`static-capacity-exit-${exit.id}`"
        :color-class="exitClass(exit)"
        :tooltip="exitTooltip(exit)"
        :current="exit.current"
        :max="exit.max"
      >
        <svg class="h-2.5 w-2.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2" aria-hidden="true">
          <path stroke-linecap="round" stroke-linejoin="round" d="M12 3v18m9-9H3" />
        </svg>
      </CapacityBadge>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import CapacityBadge from '@/components/account/CapacityBadge.vue'
import type { StaticIPCapacity, StaticProxyExitCapacity } from '@/types'

const props = defineProps<{
  capacity: StaticIPCapacity
}>()

const { t } = useI18n()

const capacityClass = (current: number, max: number) => {
  if (max > 0 && current >= max) return 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400'
  // Near-full (>=80%) gets its own amber tier between active and full.
  if (max > 0 && current / max >= 0.8) return 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400'
  if (current > 0) return 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400'
  return 'bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-400'
}

const totalClass = computed(() => capacityClass(props.capacity.current, props.capacity.max))

const totalTooltip = computed(() =>
  t('admin.accounts.capacity.staticProxy.total', {
    current: props.capacity.current,
    max: props.capacity.max
  })
)

const exitClass = (exit: StaticProxyExitCapacity) => {
  switch (exit.status.toLowerCase()) {
    case 'disabled':
      return 'bg-gray-100 text-gray-500 opacity-60 dark:bg-gray-800 dark:text-gray-500'
    case 'unhealthy':
      return 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400'
    default:
      // Healthy exits: idle shows green, then yellow/amber/red as load rises.
      if (exit.current === 0) {
        return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400'
      }
      return capacityClass(exit.current, exit.max)
  }
}

const exitTooltip = (exit: StaticProxyExitCapacity) =>
  t('admin.accounts.capacity.staticProxy.exit', {
    name: exit.name,
    host: exit.host,
    status: exit.status,
    current: exit.current,
    max: exit.max
  })
</script>

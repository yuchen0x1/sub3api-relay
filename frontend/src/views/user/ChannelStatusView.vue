<template>
  <main class="min-h-screen bg-gray-50 px-4 py-8 dark:bg-dark-950 sm:px-6 lg:px-8">
    <section class="mx-auto max-w-7xl">
      <header class="mb-8 text-center">
        <p class="text-xs font-semibold uppercase tracking-[0.28em] text-primary-500">
          Sub3API
        </p>
        <h1 class="mt-2 text-3xl font-bold text-gray-900 dark:text-gray-100">
          {{ t('channelStatus.title') }}
        </h1>
      </header>

      <MonitorCardGrid
        :items="items"
        window="7d"
        :countdown-seconds="countdown"
        :loading="loading"
        :detail-cache="{}"
        :interactive="false"
      />
    </section>
  </main>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { list as listChannelMonitorViews, type UserMonitorView } from '@/api/channelMonitor'
import MonitorCardGrid from '@/components/user/monitor/MonitorCardGrid.vue'
import { DEFAULT_INTERVAL_SECONDS } from '@/constants/channelMonitor'
import { useAutoRefresh } from '@/composables/useAutoRefresh'

const { t } = useI18n()
const items = ref<UserMonitorView[]>([])
const loading = ref(false)

let abortController: AbortController | null = null

async function reload(silent = false) {
  abortController?.abort()
  const controller = new AbortController()
  abortController = controller
  if (!silent) loading.value = true

  try {
    const response = await listChannelMonitorViews({ signal: controller.signal })
    if (controller.signal.aborted || abortController !== controller) return
    items.value = response.items || []
  } catch (error: unknown) {
    const requestError = error as { name?: string; code?: string }
    if (requestError?.name === 'AbortError' || requestError?.code === 'ERR_CANCELED') return
    if (abortController === controller) items.value = []
  } finally {
    if (abortController === controller) {
      loading.value = false
      autoRefresh.resetCountdown()
      abortController = null
    }
  }
}

const autoRefresh = useAutoRefresh({
  storageKey: 'public-channel-status-auto-refresh',
  intervals: [DEFAULT_INTERVAL_SECONDS] as const,
  defaultInterval: DEFAULT_INTERVAL_SECONDS,
  onRefresh: () => reload(true),
  shouldPause: () => document.hidden || loading.value,
})
const countdown = autoRefresh.countdown

onMounted(() => {
  void reload(false)
  autoRefresh.setEnabled(true)
})

onBeforeUnmount(() => {
  abortController?.abort()
})
</script>

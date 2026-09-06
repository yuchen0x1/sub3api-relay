<template>
  <BaseDialog :show="show" :title="t('usage.bodyLog.title')" width="extra-wide" @close="close">
    <div class="space-y-4">
      <div v-if="loading" class="py-8 text-center text-sm text-gray-500">{{ t('common.loading') }}</div>
      <template v-else-if="detail">
        <div class="grid grid-cols-2 gap-3 text-xs text-gray-600 dark:text-gray-300 md:grid-cols-4">
          <div><span class="text-gray-400">ID:</span> {{ detail.id }}</div>
          <div class="truncate"><span class="text-gray-400">Request ID:</span> {{ detail.request_id }}</div>
          <div><span class="text-gray-400">{{ t('usage.bodyLog.path') }}:</span> {{ detail.method }} {{ detail.path }}</div>
          <div><span class="text-gray-400">Model:</span> {{ detail.model || '-' }}</div>
        </div>
        <pre class="max-h-[50vh] overflow-auto rounded-lg bg-gray-100 p-3 text-xs leading-5 text-gray-800 dark:bg-gray-900 dark:text-gray-100">{{ formattedBody }}</pre>
        <div v-if="replayResult">
          <div class="mb-1 flex items-center gap-2 text-xs">
            <span
              class="inline-flex rounded px-1.5 py-0.5 font-mono font-medium"
              :class="replayResult.status < 400 ? 'bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-300' : 'bg-red-100 text-red-700 dark:bg-red-900/40 dark:text-red-300'"
            >
              HTTP {{ replayResult.status }}
            </span>
            <span class="text-gray-500">{{ replayResult.elapsed_ms }}ms</span>
          </div>
          <pre class="max-h-[30vh] overflow-auto rounded-lg bg-gray-100 p-3 text-xs leading-5 text-gray-800 dark:bg-dark-700 dark:text-gray-200">{{ replayResult.body || '-' }}</pre>
        </div>
      </template>
      <div v-else class="py-8 text-center text-sm text-gray-500">
        <p>{{ t('usage.bodyLog.notRecorded') }}</p>
        <p class="mt-2 text-xs text-gray-400 dark:text-dark-500">{{ t('usage.bodyLog.notRecordedHint') }}</p>
      </div>
    </div>
    <template #footer>
      <div class="flex justify-end gap-2">
        <button class="btn btn-secondary" @click="close">{{ t('common.close') }}</button>
        <button
          v-if="detail"
          class="btn btn-primary"
          :disabled="replaying"
          @click="handleReplay"
        >
          {{ replaying ? t('usage.bodyLog.replaying') : t('usage.bodyLog.replay') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import { useAppStore } from '@/stores/app'
import { adminAPI } from '@/api/admin'
import type { GatewayBodyLogDetail, GatewayBodyLogReplayResult } from '@/api/admin/bodylogs'

const props = defineProps<{ show: boolean; requestId: string }>()
const emit = defineEmits<{ close: [] }>()

const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(false)
const detail = ref<GatewayBodyLogDetail | null>(null)
const replaying = ref(false)
const replayResult = ref<GatewayBodyLogReplayResult | null>(null)

const formattedBody = computed(() => {
  if (!detail.value?.body) return ''
  try {
    return JSON.stringify(JSON.parse(detail.value.body), null, 2)
  } catch {
    return detail.value.body
  }
})

const load = async () => {
  if (!props.requestId) return
  loading.value = true
  detail.value = null
  replayResult.value = null
  try {
    const item = await adminAPI.bodyLogs.getByRequestId(props.requestId)
    if (item) {
      detail.value = await adminAPI.bodyLogs.getById(item.id)
    }
  } catch (err: any) {
    appStore.showError(err?.response?.data?.detail || err?.message || t('common.loadFailed'))
  } finally {
    loading.value = false
  }
}

const handleReplay = async () => {
  if (!detail.value) return
  replaying.value = true
  try {
    replayResult.value = await adminAPI.bodyLogs.replay(detail.value.id)
  } catch (err: any) {
    appStore.showError(err?.response?.data?.detail || err?.message || t('usage.bodyLog.replayFailed'))
  } finally {
    replaying.value = false
  }
}

const close = () => emit('close')

watch(
  () => props.show,
  (visible) => {
    if (visible) load()
  }
)
</script>

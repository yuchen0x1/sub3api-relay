<template>
  <BaseDialog :show="show" :title="t('admin.accounts.importChannels.title')" width="wide" @close="handleClose">
    <!-- Step 1: paste payload -->
    <div v-if="step === 1" class="space-y-4">
      <p class="text-sm text-gray-500 dark:text-gray-400">
        {{ t('admin.accounts.importChannels.description') }}
      </p>
      <div>
        <label class="input-label">{{ t('admin.accounts.importChannels.payloadLabel') }}</label>
        <textarea
          v-model="payloadText"
          data-testid="import-payload-input"
          class="input font-mono text-xs"
          rows="10"
          :placeholder="t('admin.accounts.importChannels.payloadPlaceholder')"
        ></textarea>
      </div>
      <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
        <div>
          <label class="input-label">{{ t('admin.accounts.importChannels.targetGroup') }}</label>
          <Select
            v-model="targetGroupId"
            :options="groupOptions"
            :placeholder="t('admin.accounts.importChannels.noGroup')"
          />
        </div>
        <div>
          <label class="input-label">{{ t('admin.accounts.importChannels.targetProxy') }}</label>
          <Select
            v-model="targetProxyId"
            :options="proxyOptions"
            :placeholder="t('admin.accounts.importChannels.noProxy')"
          />
        </div>
      </div>
      <div v-if="parseError" class="text-xs text-red-500">{{ parseError }}</div>
      <div class="flex items-center justify-end gap-3 border-t border-gray-200 pt-4 dark:border-dark-600">
        <button type="button" class="btn btn-secondary" @click="handleClose">
          {{ t('common.cancel') }}
        </button>
        <button
          type="button"
          class="btn btn-primary"
          data-testid="import-parse-button"
          :disabled="!payloadText.trim()"
          @click="handleParse"
        >
          {{ t('admin.accounts.importChannels.parse') }}
        </button>
      </div>
    </div>

    <!-- Step 2: preview mapping -->
    <div v-else-if="step === 2" class="space-y-4">
      <div class="flex flex-wrap items-center gap-4 text-sm">
        <span class="text-emerald-600 dark:text-emerald-400">
          {{ t('admin.accounts.importChannels.willCreate', { count: parsed?.accounts.length ?? 0 }) }}
        </span>
        <span v-if="parsed && parsed.skipped.length > 0" class="text-amber-600 dark:text-amber-400">
          {{ t('admin.accounts.importChannels.willSkip', { count: parsed.skipped.length }) }}
        </span>
      </div>

      <div v-if="parsed && parsed.skipped.length > 0" class="max-h-32 overflow-y-auto rounded-lg border border-amber-300 bg-amber-50 p-3 text-xs dark:border-amber-700 dark:bg-amber-900/20">
        <div v-for="item in parsed.skipped" :key="`${item.name}-${item.reason}`" class="text-amber-700 dark:text-amber-300">
          {{ item.name }}: {{ item.reason }}
        </div>
      </div>

      <div class="max-h-72 overflow-y-auto rounded-lg border border-gray-200 dark:border-dark-600">
        <table class="w-full text-left text-xs">
          <thead class="bg-gray-50 text-gray-500 dark:bg-dark-800 dark:text-gray-400">
            <tr>
              <th class="px-3 py-2">{{ t('admin.accounts.importChannels.colName') }}</th>
              <th class="px-3 py-2">{{ t('admin.accounts.importChannels.colPlatform') }}</th>
              <th class="px-3 py-2">{{ t('admin.accounts.baseUrl') }}</th>
              <th class="px-3 py-2">{{ t('admin.accounts.apiKeyRequired') }}</th>
              <th class="px-3 py-2">{{ t('admin.accounts.importChannels.colSourceStatus') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="(account, idx) in parsed?.accounts ?? []"
              :key="`${account.name}-${idx}`"
              class="border-t border-gray-100 dark:border-dark-700"
            >
              <td class="px-3 py-2 font-medium">{{ account.name }}</td>
              <td class="px-3 py-2">
                {{ account.platform === 'openai' ? 'OpenAI' : 'Gemini' }}
              </td>
              <td class="px-3 py-2 font-mono">
                {{ account.baseUrl || t('admin.accounts.importChannels.providerDefault') }}
              </td>
              <td class="px-3 py-2 font-mono">{{ maskKey(account.apiKey) }}</td>
              <td class="px-3 py-2">
                <span v-if="account.sourceStatus !== 1" class="text-amber-600 dark:text-amber-400">
                  {{ t('admin.accounts.importChannels.sourceDisabled') }}
                </span>
                <span v-else>{{ t('admin.accounts.importChannels.sourceEnabled') }}</span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <div class="flex items-center justify-between gap-3 border-t border-gray-200 pt-4 dark:border-dark-600">
        <button type="button" class="btn btn-secondary" :disabled="importing" @click="step = 1">
          {{ t('common.back') }}
        </button>
        <button
          type="button"
          class="btn btn-primary"
          data-testid="import-submit-button"
          :disabled="importing || (parsed?.accounts.length ?? 0) === 0"
          @click="handleImport"
        >
          {{ importing ? t('admin.accounts.importChannels.importing') : t('admin.accounts.importChannels.submit', { count: parsed?.accounts.length ?? 0 }) }}
        </button>
      </div>
    </div>

    <!-- Step 3: result -->
    <div v-else class="space-y-4">
      <div class="flex flex-col items-center gap-2 py-6 text-center">
        <Icon :name="failedCount > 0 ? 'exclamationTriangle' : 'check'" size="lg" :class="failedCount > 0 ? 'text-amber-500' : 'text-emerald-500'" />
        <div class="text-sm text-gray-600 dark:text-gray-300">
          {{ t('admin.accounts.importChannels.resultSummary', { created: createdCount, failed: failedCount }) }}
        </div>
      </div>
      <div v-if="errors.length > 0" class="max-h-48 overflow-y-auto rounded-lg border border-red-200 bg-red-50 p-3 text-xs dark:border-red-800 dark:bg-red-900/20">
        <div v-for="(err, idx) in errors" :key="idx" class="text-red-600 dark:text-red-400">
          {{ err }}
        </div>
      </div>
      <div class="flex justify-end border-t border-gray-200 pt-4 dark:border-dark-600">
        <button type="button" class="btn btn-primary" @click="handleClose">
          {{ t('common.close') }}
        </button>
      </div>
    </div>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'
import { adminAPI } from '@/api/admin'
import type { Proxy, AdminGroup, CreateAccountRequest } from '@/types'
import { parseImportPayload, type ParsedImportResult } from '@/utils/oneapiImport'

const props = defineProps({
  show: { type: Boolean, default: false },
  proxies: { type: Array as () => Proxy[], default: () => [] },
  groups: { type: Array as () => AdminGroup[], default: () => [] }
})

const emit = defineEmits(['close', 'imported'])

const { t } = useI18n()

const step = ref<1 | 2 | 3>(1)
const payloadText = ref('')
const parseError = ref('')
const parsed = ref<ParsedImportResult | null>(null)
const targetGroupId = ref<number | ''>('')
const targetProxyId = ref<number | ''>('')
const importing = ref(false)
const createdCount = ref(0)
const failedCount = ref(0)
const errors = ref<string[]>([])

const groupOptions = computed(() =>
  props.groups.map((g) => ({ value: g.id, label: g.name }))
)
const proxyOptions = computed(() =>
  props.proxies.map((p) => ({ value: p.id, label: p.name }))
)

watch(
  () => props.show,
  (visible) => {
    if (visible) resetState()
  }
)

function resetState() {
  step.value = 1
  payloadText.value = ''
  parseError.value = ''
  parsed.value = null
  targetGroupId.value = ''
  targetProxyId.value = ''
  importing.value = false
  createdCount.value = 0
  failedCount.value = 0
  errors.value = []
}

function handleClose() {
  if (importing.value) return
  emit('close')
}

function maskKey(key: string): string {
  if (key.length <= 10) return '••••'
  return `${key.slice(0, 6)}••••${key.slice(-4)}`
}

function handleParse() {
  parseError.value = ''
  const result = parseImportPayload(payloadText.value)
  if (result.accounts.length === 0 && result.skipped.length === 0) {
    parseError.value = t('admin.accounts.importChannels.parseEmpty')
    return
  }
  parsed.value = result
  step.value = 2
}

async function handleImport() {
  if (!parsed.value || importing.value) return
  importing.value = true
  createdCount.value = 0
  failedCount.value = 0
  errors.value = []

  const requests: CreateAccountRequest[] = parsed.value.accounts.map((draft) => {
    const credentials: Record<string, unknown> = { api_key: draft.apiKey }
    if (draft.baseUrl) credentials.base_url = draft.baseUrl
    return {
      name: draft.name,
      platform: draft.platform,
      type: draft.type,
      credentials,
      proxy_id: typeof targetProxyId.value === 'number' ? targetProxyId.value : undefined,
      group_ids: typeof targetGroupId.value === 'number' ? [targetGroupId.value] : undefined
    } as CreateAccountRequest
  })

  // Chunk the requests so one oversized paste cannot blow the request limit.
  const chunkSize = 20
  for (let i = 0; i < requests.length; i += chunkSize) {
    try {
      const result = await adminAPI.accounts.batchCreate(requests.slice(i, i + chunkSize))
      createdCount.value += result.success
      failedCount.value += result.failed
      for (const item of result.results) {
        if (!item.success && item.error) {
          errors.value.push(item.error)
        }
      }
    } catch (error) {
      failedCount.value += Math.min(chunkSize, requests.length - i)
      errors.value.push(error instanceof Error ? error.message : String(error))
    }
  }

  importing.value = false
  step.value = 3
  if (createdCount.value > 0) {
    emit('imported', createdCount.value)
  }
}
</script>

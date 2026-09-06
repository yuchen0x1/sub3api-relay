<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="flex flex-wrap items-center gap-3">
          <div class="w-full sm:w-56">
            <select v-model="filters.accountId" class="input" @change="load(1)">
              <option :value="null">{{ t('admin.staticExitsPage.allAccounts') }}</option>
              <option v-for="account in accounts" :key="account.id" :value="account.id">
                {{ account.name }}
              </option>
            </select>
          </div>
          <div class="w-full sm:w-40">
            <select v-model="filters.enabled" class="input" @change="load(1)">
              <option :value="null">{{ t('admin.staticExitsPage.allStates') }}</option>
              <option :value="true">{{ t('admin.accounts.staticExits.enabled') }}</option>
              <option :value="false">{{ t('admin.accounts.staticExits.disabled') }}</option>
            </select>
          </div>
          <div class="flex flex-1 flex-wrap items-center justify-end gap-2">
            <button @click="load()" :disabled="loading" class="btn btn-secondary" :title="t('common.refresh')">
              <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
            </button>
          </div>
        </div>
      </template>

      <template #table>
      <!-- 容量汇总 -->
      <div class="mb-4 grid grid-cols-2 gap-3 lg:grid-cols-4" data-test="static-exits-summary">
        <div class="rounded-lg border border-gray-200 p-4 dark:border-gray-700">
          <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.staticExitsPage.totalExits') }}</p>
          <p class="mt-1 text-2xl font-semibold text-gray-900 dark:text-gray-100">{{ summary.assignment_count }}</p>
        </div>
        <div class="rounded-lg border border-gray-200 p-4 dark:border-gray-700">
          <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.staticExitsPage.enabledExits') }}</p>
          <p class="mt-1 text-2xl font-semibold text-gray-900 dark:text-gray-100">{{ summary.enabled_count }}</p>
        </div>
        <div class="rounded-lg border border-gray-200 p-4 dark:border-gray-700">
          <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.staticExitsPage.enabledCapacity') }}</p>
          <p class="mt-1 text-2xl font-semibold text-gray-900 dark:text-gray-100">{{ summary.enabled_capacity }}</p>
        </div>
        <div class="rounded-lg border border-gray-200 p-4 dark:border-gray-700">
          <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.staticExitsPage.capacityPerExit') }}</p>
          <p class="mt-1 text-2xl font-semibold text-gray-900 dark:text-gray-100">{{ summary.capacity_per_assignment }}</p>
        </div>
      </div>

      <!-- 新建出口 -->
      <div class="mb-4 rounded-lg border border-gray-200 p-4 dark:border-gray-700" data-test="static-exit-create">
        <div class="mb-3 flex flex-wrap items-center justify-between gap-2">
          <h3 class="text-sm font-medium text-gray-900 dark:text-gray-100">{{ t('admin.staticExitsPage.createTitle') }}</h3>
          <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.accounts.staticExits.hint') }}</p>
        </div>
        <div class="flex flex-wrap items-center gap-2">
          <select v-model="createForm.accountId" class="input h-9 w-56">
            <option :value="null" disabled>{{ t('admin.staticExitsPage.selectAccount') }}</option>
            <option v-for="account in accounts" :key="account.id" :value="account.id">
              {{ account.name }}
            </option>
          </select>
          <select v-model="createForm.proxyId" class="input h-9 w-56">
            <option :value="null" disabled>{{ t('admin.accounts.staticExits.selectProxy') }}</option>
            <option v-for="proxy in proxies" :key="proxy.id" :value="proxy.id">
              {{ proxyLabel(proxy) }}
            </option>
          </select>
          <input
            v-model="createForm.name"
            type="text"
            class="input h-9 w-44"
            :placeholder="t('admin.accounts.staticExits.namePlaceholder')"
          />
          <button
            type="button"
            class="btn btn-primary h-9"
            :disabled="createForm.accountId === null || createForm.proxyId === null || creating"
            data-test="static-exit-create-submit"
            @click="handleCreate"
          >
            {{ t('admin.accounts.staticExits.add') }}
          </button>
        </div>
      </div>

      <!-- 出口列表 -->
      <div class="overflow-x-auto rounded-lg border border-gray-200 dark:border-gray-700">
        <table class="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
          <thead class="bg-gray-50 dark:bg-dark-800/60">
            <tr>
              <th class="px-3 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-400">#</th>
              <th class="px-3 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('admin.staticExitsPage.exitName') }}</th>
              <th class="px-3 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('admin.staticExitsPage.account') }}</th>
              <th class="px-3 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('admin.staticExitsPage.proxy') }}</th>
              <th class="px-3 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('admin.staticExitsPage.health') }}</th>
              <th class="px-3 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('admin.staticExitsPage.failures') }}</th>
              <th class="px-3 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('admin.staticExitsPage.enabledState') }}</th>
              <th class="px-3 py-2 text-right text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('admin.staticExitsPage.actions') }}</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-100 bg-white dark:divide-dark-800 dark:bg-dark-900">
            <tr v-if="loading">
              <td colspan="8" class="px-3 py-6 text-center text-sm text-gray-500">
                <LoadingSpinner size="sm" color="gray" />
              </td>
            </tr>
            <tr v-else-if="exits.length === 0">
              <td colspan="8" class="px-3 py-6 text-center text-sm text-gray-500 dark:text-gray-400" data-test="static-exits-empty">
                {{ t('admin.accounts.staticExits.empty') }}
              </td>
            </tr>
            <tr v-for="exit in exits" :key="exit.id" :data-test="`static-exit-row-${exit.id}`">
              <td class="px-3 py-2 text-xs text-gray-500 dark:text-gray-400">{{ exit.position }}</td>
              <td class="px-3 py-2 text-sm text-gray-900 dark:text-gray-100">{{ exit.name || '-' }}</td>
              <td class="px-3 py-2 text-sm text-gray-700 dark:text-gray-300">
                {{ accountName(exit.account_id) }}
              </td>
              <td class="px-3 py-2 text-sm text-gray-700 dark:text-gray-300">
                <template v-if="exit.proxy">{{ proxyLabelFromRef(exit.proxy) }}</template>
                <template v-else>-</template>
              </td>
              <td class="px-3 py-2">
                <span class="rounded px-1.5 py-0.5 text-[10px] font-medium" :class="healthClass(exit.health_status)">
                  {{ healthLabel(exit.health_status) }}
                </span>
              </td>
              <td class="px-3 py-2 text-xs text-gray-500 dark:text-gray-400">
                {{ exit.failure_count }}
                <span v-if="exit.last_failure_reason" class="ml-1 text-red-500" :title="exit.last_failure_reason">!</span>
              </td>
              <td class="px-3 py-2">
                <button
                  type="button"
                  class="relative inline-flex h-5 w-10 flex-shrink-0 cursor-pointer rounded-full transition-colors duration-200 ease-in-out focus:outline-none"
                  :class="exit.enabled ? 'bg-indigo-600' : 'bg-gray-200 dark:bg-gray-700'"
                  :data-test="`static-exit-toggle-${exit.id}`"
                  @click="handleToggle(exit)"
                >
                  <span
                    :class="[
                      'pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out',
                      exit.enabled ? 'translate-x-5' : 'translate-x-0'
                    ]"
                  />
                </button>
              </td>
              <td class="px-3 py-2 text-right">
                <button
                  type="button"
                  class="h-7 rounded px-2 text-xs font-medium"
                  :class="confirmingDelete === exit.id
                    ? 'bg-red-600 text-white'
                    : 'text-red-600 hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-900/30'"
                  :data-test="`static-exit-delete-${exit.id}`"
                  @click="handleDelete(exit)"
                >
                  {{ confirmingDelete === exit.id ? t('admin.accounts.staticExits.confirmDelete') : t('admin.accounts.staticExits.delete') }}
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      </template>

      <template #pagination>
      <!-- 分页 -->
      <div v-if="pages > 1" class="mt-3 flex items-center justify-between text-sm text-gray-600 dark:text-gray-400">
        <span>{{ t('admin.staticExitsPage.pageIndicator', { page, pages }) }}</span>
        <div class="flex gap-2">
          <button class="btn btn-secondary h-8 px-3 text-xs" :disabled="page <= 1 || loading" @click="load(page - 1)">
            {{ t('admin.staticExitsPage.prevPage') }}
          </button>
          <button class="btn btn-secondary h-8 px-3 text-xs" :disabled="page >= pages || loading" @click="load(page + 1)">
            {{ t('admin.staticExitsPage.nextPage') }}
          </button>
        </div>
      </div>
      </template>
    </TablePageLayout>
  </AppLayout>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { adminAPI } from '@/api/admin'
import type { Account, Proxy, StaticProxyExit, StaticProxyExitProxyRef, StaticProxyExitCapacitySummary } from '@/types'

const { t } = useI18n()
const appStore = useAppStore()

const CAPACITY_PER_EXIT = 10

const exits = ref<StaticProxyExit[]>([])
const accounts = ref<Account[]>([])
const proxies = ref<Proxy[]>([])
const loading = ref(false)
const creating = ref(false)
const confirmingDelete = ref<number | null>(null)
const page = ref(1)
const pages = ref(1)

const filters = reactive<{ accountId: number | null; enabled: boolean | null }>({
  accountId: null,
  enabled: null
})

const createForm = reactive<{ accountId: number | null; proxyId: number | null; name: string }>({
  accountId: null,
  proxyId: null,
  name: ''
})

const summary = reactive<StaticProxyExitCapacitySummary>({
  account_id: null,
  assignment_count: 0,
  enabled_count: 0,
  disabled_count: 0,
  capacity_per_assignment: CAPACITY_PER_EXIT,
  total_capacity: 0,
  enabled_capacity: 0
})

const proxyLabel = (proxy: Proxy) =>
  `${proxy.name || proxy.host} (${proxy.protocol} · ${proxy.host}:${proxy.port})`

const proxyLabelFromRef = (proxy: StaticProxyExitProxyRef) =>
  `${proxy.name || proxy.host} (${proxy.protocol} · ${proxy.host}:${proxy.port})`

const accountName = (accountId: number) =>
  accounts.value.find((account) => account.id === accountId)?.name ?? `#${accountId}`

const healthLabel = (status: string) => {
  switch ((status || '').toLowerCase()) {
    case 'healthy':
      return t('admin.accounts.staticExits.healthHealthy')
    case 'unhealthy':
      return t('admin.accounts.staticExits.healthUnhealthy')
    case 'degraded':
      return t('admin.accounts.staticExits.healthDegraded')
    default:
      return t('admin.accounts.staticExits.healthUnknown')
  }
}

const healthClass = (status: string) => {
  switch ((status || '').toLowerCase()) {
    case 'healthy':
      return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400'
    case 'unhealthy':
      return 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400'
    case 'degraded':
      return 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400'
    default:
      return 'bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-400'
  }
}

const loadReferences = async () => {
  try {
    const [accountPage, proxyPage] = await Promise.all([
      adminAPI.accounts.list(1, 500),
      adminAPI.proxies.list(1, 500)
    ])
    accounts.value = accountPage.items ?? []
    proxies.value = proxyPage.items ?? []
  } catch (error) {
    appStore.showError(t('admin.staticExitsPage.loadReferencesFailed') + ': ' + extractApiErrorMessage(error))
  }
}

const load = async (targetPage?: number) => {
  loading.value = true
  if (targetPage) page.value = targetPage
  try {
    const [data, capacity] = await Promise.all([
      adminAPI.staticProxyExits.list({
        account_id: filters.accountId ?? undefined,
        enabled: filters.enabled ?? undefined,
        page: page.value,
        page_size: 50
      }),
      adminAPI.staticProxyExits.capacitySummary(filters.accountId ?? undefined)
    ])
    exits.value = data.items ?? []
    pages.value = data.pages ?? 1
    Object.assign(summary, capacity)
  } catch (error) {
    appStore.showError(t('admin.accounts.staticExits.loadFailed') + ': ' + extractApiErrorMessage(error))
  } finally {
    loading.value = false
  }
}

const handleCreate = async () => {
  if (createForm.accountId === null || createForm.proxyId === null || creating.value) return
  creating.value = true
  try {
    await adminAPI.staticProxyExits.create({
      account_id: createForm.accountId,
      proxy_id: createForm.proxyId,
      name: createForm.name.trim() || undefined,
      enabled: true
    })
    appStore.showSuccess(t('admin.accounts.staticExits.add'))
    createForm.proxyId = null
    createForm.name = ''
    await load()
  } catch (error) {
    appStore.showError(t('admin.accounts.staticExits.addFailed') + ': ' + extractApiErrorMessage(error))
  } finally {
    creating.value = false
  }
}

const handleToggle = async (exit: StaticProxyExit) => {
  try {
    const updated = await adminAPI.staticProxyExits.toggle(exit.id, !exit.enabled)
    exits.value = exits.value.map((item) => (item.id === updated.id ? updated : item))
    await load()
  } catch (error) {
    appStore.showError(t('admin.accounts.staticExits.updateFailed') + ': ' + extractApiErrorMessage(error))
  }
}

const handleDelete = async (exit: StaticProxyExit) => {
  if (confirmingDelete.value !== exit.id) {
    confirmingDelete.value = exit.id
    window.setTimeout(() => {
      if (confirmingDelete.value === exit.id) confirmingDelete.value = null
    }, 3000)
    return
  }
  confirmingDelete.value = null
  try {
    await adminAPI.staticProxyExits.remove(exit.id)
    await load()
  } catch (error) {
    appStore.showError(t('admin.accounts.staticExits.deleteFailed') + ': ' + extractApiErrorMessage(error))
  }
}

onMounted(async () => {
  await loadReferences()
  await load()
})
</script>

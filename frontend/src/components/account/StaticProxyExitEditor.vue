<template>
  <div class="rounded-lg border border-gray-200 p-3 dark:border-gray-700" data-test="static-exit-editor">
    <div class="mb-2 flex flex-wrap items-center justify-between gap-2">
      <label class="input-label mb-0">{{ t('admin.accounts.staticExits.title') }}</label>
      <span
        v-if="exits.length > 0"
        class="text-xs font-medium text-gray-600 dark:text-gray-400"
        data-test="static-exit-capacity"
      >
        {{ t('admin.accounts.staticExits.capacity', { enabled: enabledCapacity, total: totalCapacity }) }}
      </span>
    </div>
    <p class="input-hint mb-2">{{ t('admin.accounts.staticExits.hint') }}</p>

    <div v-if="loading" class="flex justify-center py-2">
      <LoadingSpinner size="sm" color="gray" />
    </div>

    <template v-else>
      <div v-if="exits.length === 0" class="mb-2 text-xs text-gray-500 dark:text-gray-400" data-test="static-exit-empty">
        {{ t('admin.accounts.staticExits.empty') }}
      </div>

      <ul v-else class="mb-2 flex flex-col gap-1.5">
        <li
          v-for="(exit, index) in exits"
          :key="exit.id"
          class="flex flex-wrap items-center gap-2 rounded-md bg-gray-50 px-2 py-1.5 dark:bg-gray-800/60"
          :data-test="`static-exit-row-${exit.id}`"
        >
          <span class="w-5 text-center text-xs text-gray-500 dark:text-gray-400">{{ index + 1 }}</span>

          <input
            v-model="exit.name"
            type="text"
            class="input h-7 w-32 flex-1 px-2 py-0 text-xs"
            :placeholder="t('admin.accounts.staticExits.namePlaceholder')"
            :disabled="busy(exit.id)"
            @change="saveName(exit)"
          />

          <select
            :value="exit.proxy_id"
            class="input h-7 w-40 flex-1 px-1 py-0 text-xs"
            :disabled="busy(exit.id)"
            @change="saveProxy(exit, $event)"
          >
            <option v-for="proxy in availableProxies(exit.proxy_id)" :key="proxy.id" :value="proxy.id">
              {{ proxyLabel(proxy) }}
            </option>
          </select>

          <span
            class="rounded px-1.5 py-0.5 text-[10px] font-medium"
            :class="healthClass(exit.health_status)"
          >
            {{ healthLabel(exit.health_status) }}
          </span>

          <button
            type="button"
            class="relative inline-flex h-5 w-10 flex-shrink-0 cursor-pointer rounded-full transition-colors duration-200 ease-in-out focus:outline-none"
            :class="exit.enabled ? 'bg-indigo-600' : 'bg-gray-200 dark:bg-gray-700'"
            :disabled="busy(exit.id)"
            :data-test="`static-exit-toggle-${exit.id}`"
            @click="toggleExit(exit)"
          >
            <span
              :class="[
                'pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out',
                exit.enabled ? 'translate-x-5' : 'translate-x-0'
              ]"
            />
          </button>

          <button
            type="button"
            class="h-7 flex-shrink-0 rounded px-2 text-xs font-medium"
            :class="confirmingDelete === exit.id
              ? 'bg-red-600 text-white'
              : 'text-red-600 hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-900/30'"
            :disabled="busy(exit.id)"
            :data-test="`static-exit-delete-${exit.id}`"
            @click="deleteExit(exit)"
          >
            {{ confirmingDelete === exit.id ? t('admin.accounts.staticExits.confirmDelete') : t('admin.accounts.staticExits.delete') }}
          </button>
        </li>
      </ul>

      <div class="flex items-center gap-2">
        <select v-model="newExitProxyId" class="input h-8 flex-1 px-2 py-0 text-xs" data-test="static-exit-add-proxy">
          <option :value="null" disabled>{{ t('admin.accounts.staticExits.selectProxy') }}</option>
          <option v-for="proxy in addableProxies" :key="proxy.id" :value="proxy.id">
            {{ proxyLabel(proxy) }}
          </option>
        </select>
        <input
          v-model="newExitName"
          type="text"
          class="input h-8 w-40 px-2 py-0 text-xs"
          :placeholder="t('admin.accounts.staticExits.namePlaceholder')"
          data-test="static-exit-add-name"
        />
        <button
          type="button"
          class="h-8 flex-shrink-0 rounded-md bg-indigo-600 px-3 text-xs font-medium text-white hover:bg-indigo-700 disabled:cursor-not-allowed disabled:opacity-50"
          :disabled="newExitProxyId === null || adding"
          data-test="static-exit-add"
          @click="addExit"
        >
          {{ t('admin.accounts.staticExits.add') }}
        </button>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import staticProxyExitsAPI from '@/api/admin/staticProxyExits'
import type { Proxy, StaticProxyExit } from '@/types'

const props = defineProps<{
  accountId: number
  proxies: Proxy[]
}>()

const { t } = useI18n()
const appStore = useAppStore()

const exits = ref<StaticProxyExit[]>([])
const loading = ref(true)
const adding = ref(false)
const busyIds = ref<Set<number>>(new Set())
const confirmingDelete = ref<number | null>(null)
const newExitProxyId = ref<number | null>(null)
const newExitName = ref('')

const CAPACITY_PER_EXIT = 10

const busy = (id: number) => busyIds.value.has(id)

const markBusy = (id: number, value: boolean) => {
  const next = new Set(busyIds.value)
  if (value) {
    next.add(id)
  } else {
    next.delete(id)
  }
  busyIds.value = next
}

const enabledExits = computed(() => exits.value.filter((exit) => exit.enabled))
const enabledCapacity = computed(() => enabledExits.value.length * CAPACITY_PER_EXIT)
const totalCapacity = computed(() => exits.value.length * CAPACITY_PER_EXIT)

const addableProxies = computed(() => {
  const assigned = new Set(exits.value.map((exit) => exit.proxy_id))
  return props.proxies.filter((proxy) => !assigned.has(proxy.id))
})

const availableProxies = (currentProxyId: number) => {
  const assigned = new Set(exits.value.map((exit) => exit.proxy_id))
  return props.proxies.filter((proxy) => proxy.id === currentProxyId || !assigned.has(proxy.id))
}

const proxyLabel = (proxy: Proxy) =>
  `${proxy.name || proxy.host} (${proxy.protocol} · ${proxy.host}:${proxy.port})`

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

const load = async () => {
  loading.value = true
  try {
    exits.value = await staticProxyExitsAPI.listByAccount(props.accountId)
  } catch (error) {
    appStore.showError(t('admin.accounts.staticExits.loadFailed') + ': ' + extractApiErrorMessage(error))
  } finally {
    loading.value = false
  }
}

const addExit = async () => {
  if (newExitProxyId.value === null || adding.value) return
  adding.value = true
  try {
    const exit = await staticProxyExitsAPI.create({
      account_id: props.accountId,
      proxy_id: newExitProxyId.value,
      name: newExitName.value.trim() || undefined,
      position: exits.value.length + 1,
      enabled: true
    })
    exits.value = [...exits.value, exit]
    newExitProxyId.value = null
    newExitName.value = ''
  } catch (error) {
    appStore.showError(t('admin.accounts.staticExits.addFailed') + ': ' + extractApiErrorMessage(error))
  } finally {
    adding.value = false
  }
}

const saveName = async (exit: StaticProxyExit) => {
  const name = exit.name.trim()
  markBusy(exit.id, true)
  try {
    await staticProxyExitsAPI.update(exit.id, { name: name || undefined })
  } catch (error) {
    appStore.showError(t('admin.accounts.staticExits.updateFailed') + ': ' + extractApiErrorMessage(error))
    await reloadQuiet()
  } finally {
    markBusy(exit.id, false)
  }
}

const saveProxy = async (exit: StaticProxyExit, event: Event) => {
  const proxyId = Number((event.target as HTMLSelectElement).value)
  if (!proxyId || proxyId === exit.proxy_id) return
  markBusy(exit.id, true)
  try {
    const updated = await staticProxyExitsAPI.update(exit.id, { proxy_id: proxyId })
    replaceExit(updated)
  } catch (error) {
    appStore.showError(t('admin.accounts.staticExits.updateFailed') + ': ' + extractApiErrorMessage(error))
    await reloadQuiet()
  } finally {
    markBusy(exit.id, false)
  }
}

const toggleExit = async (exit: StaticProxyExit) => {
  markBusy(exit.id, true)
  try {
    const updated = await staticProxyExitsAPI.toggle(exit.id, !exit.enabled)
    replaceExit(updated)
  } catch (error) {
    appStore.showError(t('admin.accounts.staticExits.updateFailed') + ': ' + extractApiErrorMessage(error))
  } finally {
    markBusy(exit.id, false)
  }
}

const deleteExit = async (exit: StaticProxyExit) => {
  if (confirmingDelete.value !== exit.id) {
    confirmingDelete.value = exit.id
    window.setTimeout(() => {
      if (confirmingDelete.value === exit.id) confirmingDelete.value = null
    }, 3000)
    return
  }
  confirmingDelete.value = null
  markBusy(exit.id, true)
  try {
    await staticProxyExitsAPI.remove(exit.id)
    exits.value = exits.value.filter((item) => item.id !== exit.id)
  } catch (error) {
    appStore.showError(t('admin.accounts.staticExits.deleteFailed') + ': ' + extractApiErrorMessage(error))
  } finally {
    markBusy(exit.id, false)
  }
}

const replaceExit = (updated: StaticProxyExit) => {
  exits.value = exits.value.map((exit) => (exit.id === updated.id ? updated : exit))
}

const reloadQuiet = async () => {
  try {
    exits.value = await staticProxyExitsAPI.listByAccount(props.accountId)
  } catch {
    // Keep the current rows on reload failure; the API error was already shown.
  }
}

onMounted(load)
</script>

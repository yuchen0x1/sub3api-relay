<template>
  <AppLayout>
    <div class="mx-auto w-full max-w-4xl px-1 pb-10">
      <!-- Page header -->
      <div class="mb-6 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 class="text-xl font-bold text-gray-900 dark:text-white">
            {{ t('notifications.title') }}
          </h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
            {{ t('notifications.description') }}
          </p>
        </div>
        <button
          v-if="unreadCount > 0"
          @click="markAllAsRead"
          :disabled="loading"
          class="inline-flex items-center gap-2 self-start rounded-xl bg-blue-600 px-4 py-2 text-sm font-medium text-white shadow-lg shadow-blue-500/30 transition-all hover:bg-blue-700 disabled:opacity-50 dark:bg-blue-500 dark:hover:bg-blue-600"
        >
          <Icon name="check" size="sm" />
          {{ t('notifications.markAllRead') }}
        </button>
      </div>

      <!-- Type filter tabs -->
      <div class="mb-4 flex flex-wrap items-center gap-2">
        <button
          v-for="tab in typeTabs"
          :key="tab.value"
          @click="selectType(tab.value)"
          class="rounded-lg px-3 py-1.5 text-sm font-medium transition-colors"
          :class="activeType === tab.value
            ? 'bg-blue-600 text-white shadow shadow-blue-500/20'
            : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-dark-700 dark:text-gray-300 dark:hover:bg-dark-600'"
        >
          {{ tab.label }}
          <span
            v-if="tab.value === 'all' && total > 0"
            class="ml-1 text-xs opacity-70"
          >{{ total }}</span>
        </button>
      </div>

      <!-- List -->
      <div class="overflow-hidden rounded-2xl border border-gray-100 bg-white shadow-card dark:border-dark-700 dark:bg-dark-800">
        <!-- Loading skeleton -->
        <div v-if="loading" class="divide-y divide-gray-100 dark:divide-dark-700">
          <div v-for="i in 6" :key="i" class="flex items-center gap-4 px-5 py-4">
            <div class="h-10 w-10 animate-pulse rounded-xl bg-gray-100 dark:bg-dark-700"></div>
            <div class="flex-1 space-y-2">
              <div class="h-3 w-3/4 animate-pulse rounded bg-gray-100 dark:bg-dark-700"></div>
              <div class="h-2.5 w-1/3 animate-pulse rounded bg-gray-100 dark:bg-dark-700"></div>
            </div>
          </div>
        </div>

        <!-- Empty state -->
        <div v-else-if="items.length === 0" class="flex flex-col items-center justify-center py-16">
          <div class="mb-4 flex h-20 w-20 items-center justify-center rounded-full bg-gray-100 dark:bg-dark-700">
            <Icon name="inbox" size="xl" class="text-gray-400 dark:text-gray-500" />
          </div>
          <p class="text-sm font-medium text-gray-900 dark:text-white">{{ t('notifications.empty') }}</p>
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t('notifications.emptyDescription') }}</p>
        </div>

        <!-- Items -->
        <div v-else class="divide-y divide-gray-100 dark:divide-dark-700">
          <div
            v-for="item in items"
            :key="item.id"
            class="group relative flex cursor-pointer items-start gap-4 px-5 py-4 transition-colors hover:bg-gray-50 dark:hover:bg-dark-700/30"
            :class="{ 'bg-blue-50/40 dark:bg-blue-900/10': !item.read_at }"
            @click="openItem(item)"
          >
            <!-- Unread bar -->
            <div
              v-if="!item.read_at"
              class="absolute left-0 top-0 h-full w-1 bg-blue-500"
            ></div>

            <!-- Type icon -->
            <div
              class="mt-0.5 flex h-11 w-11 flex-shrink-0 items-center justify-center rounded-xl"
              :class="typeBadgeClass(item.type)"
            >
              <Icon :name="typeIcon(item.type)" size="md" />
            </div>

            <!-- Content -->
            <div class="min-w-0 flex-1">
              <div class="flex items-start justify-between gap-3">
                <p class="text-sm font-semibold text-gray-900 dark:text-white">{{ item.title }}</p>
                <span class="mt-0.5 flex-shrink-0 rounded-md px-1.5 py-0.5 text-xs font-medium" :class="typeTagClass(item.type)">
                  {{ t(`notifications.types.${item.type}`) }}
                </span>
              </div>
              <p v-if="item.body" class="mt-1 line-clamp-2 text-sm text-gray-600 dark:text-gray-400">
                {{ item.body }}
              </p>
              <div class="mt-1.5 flex items-center gap-3 text-xs text-gray-500 dark:text-gray-400">
                <time>{{ formatRelativeTime(item.created_at) }}</time>
                <span v-if="item.read_at" class="inline-flex items-center gap-1">
                  <Icon name="check" size="xs" />
                  {{ t('notifications.read') }}
                </span>
              </div>
            </div>
          </div>
        </div>

        <!-- Load more -->
        <div v-if="hasMore && !loading" class="border-t border-gray-100 px-5 py-3 text-center dark:border-dark-700">
          <button
            @click="store.loadMore()"
            class="text-sm font-medium text-blue-600 hover:underline dark:text-blue-400"
          >
            {{ t('notifications.loadMore') }}
          </button>
        </div>
        <div v-else-if="items.length > 0 && !hasMore && !loading" class="border-t border-gray-100 px-5 py-3 text-center text-xs text-gray-400 dark:border-dark-700">
          {{ t('notifications.noMore') }}
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { storeToRefs } from 'pinia'
import AppLayout from '@/components/layout/AppLayout.vue'
import { useNotificationStore } from '@/stores/notifications'
import { useAppStore } from '@/stores'
import { formatRelativeTime } from '@/utils/format'
import type { UserNotificationType } from '@/types'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()
const appStore = useAppStore()
const store = useNotificationStore()

const { notifications, loading, total, hasMore } = storeToRefs(store)
const activeType = ref<UserNotificationType | 'all'>('all')

const items = computed(() => notifications.value)
const unreadCount = computed(() => store.unreadCount)

const typeTabs = computed(() => {
  const values: Array<'all' | UserNotificationType> = ['all', 'system', 'balance_low', 'account_quota', 'payment_received', 'key_rotated']
  return values.map((v) => ({ value: v, label: t(`notifications.types.${v}`) }))
})

function selectType(type: UserNotificationType | 'all') {
  activeType.value = type
  store.fetchNotifications(true, {
    reset: true,
    type: type === 'all' ? undefined : type
  })
}

function typeIcon(type: UserNotificationType): 'dollar' | 'infoCircle' | 'check' | 'key' | 'bell' {
  switch (type) {
    case 'balance_low': return 'dollar'
    case 'account_quota': return 'infoCircle'
    case 'payment_received': return 'check'
    case 'key_rotated': return 'key'
    default: return 'bell'
  }
}

function typeBadgeClass(type: UserNotificationType): string {
  switch (type) {
    case 'balance_low': return 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400'
    case 'account_quota': return 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400'
    case 'payment_received': return 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400'
    case 'key_rotated': return 'bg-indigo-100 text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-400'
    default: return 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-400'
  }
}

function typeTagClass(type: UserNotificationType): string {
  switch (type) {
    case 'balance_low': return 'bg-amber-50 text-amber-700 dark:bg-amber-900/20 dark:text-amber-400'
    case 'account_quota': return 'bg-red-50 text-red-700 dark:bg-red-900/20 dark:text-red-400'
    case 'payment_received': return 'bg-green-50 text-green-700 dark:bg-green-900/20 dark:text-green-400'
    case 'key_rotated': return 'bg-indigo-50 text-indigo-700 dark:bg-indigo-900/20 dark:text-indigo-400'
    default: return 'bg-gray-50 text-gray-600 dark:bg-dark-700 dark:text-gray-400'
  }
}

async function openItem(item: { id: number; read_at?: string }) {
  if (!item.read_at) {
    await store.markAsRead(item.id)
  }
}

async function markAllAsRead() {
  try {
    await store.markAllAsRead()
    appStore.showSuccess(t('notifications.allMarkedAsRead'))
  } catch (err: any) {
    appStore.showError(err?.message || t('common.unknownError'))
  }
}

onMounted(() => {
  store.fetchNotifications(true, { reset: true })
})
</script>
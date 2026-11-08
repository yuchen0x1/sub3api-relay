<template>
  <div class="relative">
    <!-- 通知铃铛按钮 -->
    <button
      @click="toggleOpen"
      class="relative flex h-9 w-9 items-center justify-center rounded-lg text-gray-600 transition-all hover:bg-gray-100 hover:scale-105 dark:text-gray-400 dark:hover:bg-dark-800"
      :class="{ 'text-blue-600 dark:text-blue-400': unreadCount > 0 }"
      :aria-label="t('notifications.title')"
    >
      <Icon name="bell" size="md" />
      <!-- 未读角标 -->
      <span
        v-if="unreadCount > 0"
        class="absolute right-1 top-1 flex h-2 w-2"
      >
        <span class="absolute inline-flex h-full w-full animate-ping rounded-full bg-red-500 opacity-75"></span>
        <span class="relative inline-flex h-2 w-2 rounded-full bg-red-500"></span>
      </span>
    </button>

    <!-- 下拉面板 -->
    <Teleport to="body">
      <Transition name="dropdown-fade">
        <div
          v-if="isOpen"
          class="fixed inset-0 z-[90]"
          @click="close"
        ></div>
      </Transition>
    </Teleport>

    <Transition name="dropdown-fade">
      <div
        v-if="isOpen"
        class="notification-dropdown absolute right-0 z-[95] mt-2 w-80 overflow-hidden rounded-2xl border border-gray-100 bg-white shadow-2xl ring-1 ring-black/5 dark:border-dark-700 dark:bg-dark-800 dark:ring-white/10 sm:w-96"
      >
        <!-- Header -->
        <div class="flex items-center justify-between border-b border-gray-100 bg-gray-50/60 px-4 py-3 dark:border-dark-700 dark:bg-dark-900/40">
          <h3 class="text-sm font-semibold text-gray-900 dark:text-white">
            {{ t('notifications.title') }}
            <span v-if="unreadCount > 0" class="ml-1 text-xs font-normal text-blue-600 dark:text-blue-400">
              ({{ unreadCount }})
            </span>
          </h3>
          <div class="flex items-center gap-1">
            <button
              v-if="unreadCount > 0"
              @click="markAllAsRead"
              :disabled="loading"
              class="rounded-md px-2 py-1 text-xs font-medium text-blue-600 transition-colors hover:bg-blue-50 disabled:opacity-50 dark:text-blue-400 dark:hover:bg-blue-900/30"
            >
              {{ t('notifications.markAllRead') }}
            </button>
            <router-link
              :to="{ path: '/notifications' }"
              @click="close"
              class="rounded-md px-2 py-1 text-xs font-medium text-gray-500 transition-colors hover:bg-gray-100 dark:text-gray-400 dark:hover:bg-dark-700"
            >
              {{ t('announcements.viewAll') }}
            </router-link>
          </div>
        </div>

        <!-- Body -->
        <div class="max-h-[60vh] overflow-y-auto">
          <div v-if="loading" class="flex items-center justify-center py-10">
            <div class="h-6 w-6 animate-spin rounded-full border-2 border-gray-200 border-t-blue-600 dark:border-dark-600 dark:border-t-blue-400"></div>
          </div>

          <template v-else-if="items.length > 0">
            <div
              v-for="item in items"
              :key="item.id"
              class="group relative flex cursor-pointer items-start gap-3 border-b border-gray-100 px-4 py-3 transition-colors hover:bg-gray-50 dark:border-dark-700 dark:hover:bg-dark-700/30"
              :class="{ 'bg-blue-50/30 dark:bg-blue-900/5': !item.read_at }"
              @click="openItem(item)"
            >
              <!-- Type icon -->
              <div
                class="mt-0.5 flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-lg"
                :class="typeBadgeClass(item.type)"
              >
                <Icon :name="typeIcon(item.type)" size="sm" />
              </div>

              <!-- Content -->
              <div class="min-w-0 flex-1">
                <div class="flex items-start justify-between gap-2">
                  <p class="line-clamp-2 text-sm font-medium text-gray-900 dark:text-white">
                    {{ item.title }}
                  </p>
                  <span
                    v-if="!item.read_at"
                    class="mt-1 flex h-2 w-2 flex-shrink-0 rounded-full bg-blue-500"
                  ></span>
                </div>
                <time class="mt-0.5 block text-xs text-gray-500 dark:text-gray-400">
                  {{ formatRelativeTime(item.created_at) }}
                </time>
              </div>
            </div>
          </template>

          <div v-else class="flex flex-col items-center justify-center py-10">
            <div class="mb-2 flex h-14 w-14 items-center justify-center rounded-full bg-gray-100 dark:bg-dark-700">
              <Icon name="inbox" size="xl" class="text-gray-400 dark:text-gray-500" />
            </div>
            <p class="text-sm font-medium text-gray-900 dark:text-white">{{ t('notifications.empty') }}</p>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t('notifications.emptyDescription') }}</p>
          </div>
        </div>
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { useI18n } from 'vue-i18n'
import { storeToRefs } from 'pinia'
import { useNotificationStore } from '@/stores/notifications'
import { useAppStore } from '@/stores'
import { formatRelativeTime } from '@/utils/format'
import type { UserNotification, UserNotificationType } from '@/types'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()
const appStore = useAppStore()
const store = useNotificationStore()

const { notifications, loading } = storeToRefs(store)
const items = computed(() => notifications.value.slice(0, 20))
const unreadCount = computed(() => store.unreadCount)

const isOpen = ref(false)

function toggleOpen() {
  if (isOpen.value) {
    close()
  } else {
    isOpen.value = true
    store.fetchNotifications()
  }
}

function close() {
  isOpen.value = false
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

async function openItem(item: UserNotification) {
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

function handleEscape(e: KeyboardEvent) {
  if (e.key === 'Escape' && isOpen.value) {
    close()
  }
}

onMounted(() => {
  document.addEventListener('keydown', handleEscape)
})

onBeforeUnmount(() => {
  document.removeEventListener('keydown', handleEscape)
})
</script>

<style scoped>
.dropdown-fade-enter-active {
  transition: opacity 0.2s ease, transform 0.2s ease;
}

.dropdown-fade-leave-active {
  transition: opacity 0.15s ease;
}

.dropdown-fade-enter-from,
.dropdown-fade-leave-to {
  opacity: 0;
}

.dropdown-fade-enter-from {
  transform: translateY(-6px);
}
</style>
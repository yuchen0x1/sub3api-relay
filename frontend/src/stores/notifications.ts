import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { notificationsAPI } from '@/api'
import type { UserNotification, UserNotificationType } from '@/types'

const THROTTLE_MS = 60 * 1000 // 1 minute

export const useNotificationStore = defineStore('notifications', () => {
  // State
  const notifications = ref<UserNotification[]>([])
  const loading = ref(false)
  const total = ref(0)
  const nextAfterId = ref(0)
  const hasMore = ref(false)
  const lastFetchTime = ref(0)

  // Getters
  const unreadCount = computed(() => notifications.value.filter((n) => !n.read_at).length)

  // Actions
  async function fetchNotifications(force = false, opts: { reset?: boolean; type?: string } = {}) {
    const now = Date.now()
    if (!force && !opts.reset && lastFetchTime.value > 0 && now - lastFetchTime.value < THROTTLE_MS) {
      return
    }

    if (opts.reset) {
      lastFetchTime.value = now
      loading.value = true
      try {
        const res = await notificationsAPI.list({ limit: 30, type: opts.type })
        notifications.value = res.items
        total.value = res.total
        nextAfterId.value = res.next_after_id
        hasMore.value = res.next_after_id > 0
      } catch (err: any) {
        lastFetchTime.value = 0
        console.error('Failed to fetch notifications:', err)
      } finally {
        loading.value = false
      }
      return
    }

    // Normal (throttled) fetch for the bell preview — first page only
    lastFetchTime.value = now
    loading.value = true
    try {
      const res = await notificationsAPI.list({ limit: 20 })
      notifications.value = res.items
      total.value = res.total
      nextAfterId.value = res.next_after_id
      hasMore.value = res.next_after_id > 0
    } catch (err: any) {
      lastFetchTime.value = 0
      console.error('Failed to fetch notifications:', err)
    } finally {
      loading.value = false
    }
  }

  async function loadMore() {
    if (!hasMore.value || loading.value) return
    loading.value = true
    try {
      const res = await notificationsAPI.list({ after_id: nextAfterId.value, limit: 30 })
      notifications.value.push(...res.items)
      nextAfterId.value = res.next_after_id
      hasMore.value = res.next_after_id > 0
    } catch (err: any) {
      console.error('Failed to load more notifications:', err)
    } finally {
      loading.value = false
    }
  }

  async function markAsRead(id: number) {
    try {
      await notificationsAPI.markRead(id)
      const n = notifications.value.find((x) => x.id === id)
      if (n) n.read_at = new Date().toISOString()
    } catch (err: any) {
      console.error('Failed to mark notification as read:', err)
    }
  }

  async function markAllAsRead() {
    try {
      await notificationsAPI.markAllRead()
      notifications.value.forEach((n) => {
        if (!n.read_at) n.read_at = new Date().toISOString()
      })
    } catch (err: any) {
      console.error('Failed to mark all notifications as read:', err)
      throw err
    }
  }

  function filterByType(type: UserNotificationType | 'all') {
    if (type === 'all') return notifications.value
    return notifications.value.filter((n) => n.type === type)
  }

  function reset() {
    notifications.value = []
    total.value = 0
    nextAfterId.value = 0
    hasMore.value = false
    lastFetchTime.value = 0
    loading.value = false
  }

  return {
    // State
    notifications,
    loading,
    total,
    hasMore,
    // Getters
    unreadCount,
    // Actions
    fetchNotifications,
    loadMore,
    markAsRead,
    markAllAsRead,
    filterByType,
    reset,
  }
})
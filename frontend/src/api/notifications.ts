/**
 * User Notifications API endpoints (站内通知中心)
 */

import { apiClient } from './client'
import type { UserNotificationList } from '@/types'

export interface NotificationsQuery {
  unread_only?: boolean
  type?: string
  after_id?: number
  limit?: number
}

export async function list(query: NotificationsQuery = {}): Promise<UserNotificationList> {
  const params: Record<string, string | number> = {}
  if (query.unread_only) params.unread_only = 1
  if (query.type) params.type = query.type
  if (query.after_id) params.after_id = query.after_id
  if (query.limit) params.limit = query.limit
  const { data } = await apiClient.get<UserNotificationList>('/notifications', { params })
  return data
}

export async function unreadCount(): Promise<number> {
  const { data } = await apiClient.get<{ count: number }>('/notifications/unread-count')
  return data.count
}

export async function markRead(id: number): Promise<{ message: string }> {
  const { data } = await apiClient.post<{ message: string }>(`/notifications/${id}/read`)
  return data
}

export async function markAllRead(): Promise<{ message: string }> {
  const { data } = await apiClient.post<{ message: string }>('/notifications/read-all')
  return data
}

const notificationsAPI = {
  list,
  unreadCount,
  markRead,
  markAllRead
}

export default notificationsAPI
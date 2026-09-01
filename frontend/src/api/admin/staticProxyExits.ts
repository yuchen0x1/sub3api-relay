/**
 * Admin Static Proxy Exit API endpoints
 * Manages account-owned static proxy exit assignments.
 * Each enabled exit provides 10 request concurrency slots.
 */

import { apiClient } from '../client'
import type {
  StaticProxyExit,
  StaticProxyExitCreateRequest,
  StaticProxyExitUpdateRequest,
  StaticProxyExitCapacitySummary,
  PaginatedResponse
} from '@/types'

/**
 * List static proxy exits, optionally filtered by account.
 */
export async function list(params?: {
  account_id?: number
  enabled?: boolean
  page?: number
  page_size?: number
}): Promise<PaginatedResponse<StaticProxyExit>> {
  const { data } = await apiClient.get<PaginatedResponse<StaticProxyExit>>(
    '/admin/static-proxy-exits',
    { params }
  )
  return data
}

/**
 * List all static proxy exits for one account.
 */
export async function listByAccount(accountId: number, pageSize = 100): Promise<StaticProxyExit[]> {
  const data = await list({ account_id: accountId, page: 1, page_size: pageSize })
  return data.items ?? []
}

/**
 * Get one static proxy exit by ID.
 */
export async function getById(id: number): Promise<StaticProxyExit> {
  const { data } = await apiClient.get<StaticProxyExit>(`/admin/static-proxy-exits/${id}`)
  return data
}

/**
 * Create a static proxy exit assignment for an account.
 */
export async function create(payload: StaticProxyExitCreateRequest): Promise<StaticProxyExit> {
  const { data } = await apiClient.post<StaticProxyExit>('/admin/static-proxy-exits', payload)
  return data
}

/**
 * Update a static proxy exit assignment.
 */
export async function update(id: number, payload: StaticProxyExitUpdateRequest): Promise<StaticProxyExit> {
  const { data } = await apiClient.put<StaticProxyExit>(`/admin/static-proxy-exits/${id}`, payload)
  return data
}

/**
 * Enable/disable a static proxy exit. When enabled is omitted the backend
 * inverts the current value.
 */
export async function toggle(id: number, enabled?: boolean): Promise<StaticProxyExit> {
  const { data } = await apiClient.post<StaticProxyExit>(
    `/admin/static-proxy-exits/${id}/toggle`,
    enabled === undefined ? {} : { enabled }
  )
  return data
}

/**
 * Delete a static proxy exit assignment.
 */
export async function remove(id: number): Promise<void> {
  await apiClient.delete(`/admin/static-proxy-exits/${id}`)
}

/**
 * Capacity summary for the whole system or one account.
 */
export async function capacitySummary(accountId?: number): Promise<StaticProxyExitCapacitySummary> {
  const { data } = await apiClient.get<StaticProxyExitCapacitySummary>(
    '/admin/static-proxy-exits/capacity',
    { params: accountId ? { account_id: accountId } : undefined }
  )
  return data
}

export default {
  list,
  listByAccount,
  getById,
  create,
  update,
  toggle,
  remove,
  capacitySummary
}

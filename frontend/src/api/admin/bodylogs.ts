/**
 * Gateway body log admin endpoints (record/replay debug feature).
 */

import { apiClient } from '../client'

export interface GatewayBodyLogItem {
  id: number
  request_id: string
  method: string
  path: string
  api_key_id: number
  user_id: number
  model: string
  created_at: string
}

export interface GatewayBodyLogDetail extends GatewayBodyLogItem {
  body: string
}

export interface GatewayBodyLogReplayResult {
  status: number
  body: string
  elapsed_ms: number
}

export async function getByRequestId(requestId: string): Promise<GatewayBodyLogItem | null> {
  const { data } = await apiClient.get<{ items: GatewayBodyLogItem[]; total: number }>(
    '/admin/gateway-body-logs',
    { params: { request_id: requestId, limit: 1 } }
  )
  return data.items?.[0] ?? null
}

export async function getById(id: number): Promise<GatewayBodyLogDetail> {
  const { data } = await apiClient.get<GatewayBodyLogDetail>(`/admin/gateway-body-logs/${id}`)
  return data
}

export async function replay(id: number): Promise<GatewayBodyLogReplayResult> {
  const { data } = await apiClient.post<GatewayBodyLogReplayResult>(
    `/admin/gateway-body-logs/${id}/replay`
  )
  return data
}

const bodyLogsAPI = { getByRequestId, getById, replay }
export default bodyLogsAPI

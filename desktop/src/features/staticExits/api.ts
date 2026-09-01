import { apiClient } from "../../lib/http/apiClient"
import {
  normalizeAccountOptions,
  normalizeProxyOptions,
  normalizeStaticProxyCapacitySummary,
  normalizeStaticProxyExit,
  normalizeStaticProxyExitsResponse,
} from "./normalization"
import {
  CreateStaticProxyExitInput,
  PickerOption,
  ProxyPickerOption,
  StaticExitsSnapshot,
  StaticProxyExit,
  StaticProxyExitCapacitySummary,
  UpdateStaticProxyExitInput,
} from "./types"

const EXITS_BASE = "/api/v1/admin/static-proxy-exits"

export interface ListStaticProxyExitsOptions {
  page?: number
  pageSize?: number
  accountId?: string
  signal?: AbortSignal
}

export async function listStaticProxyExits(options: ListStaticProxyExitsOptions = {}): Promise<StaticExitsSnapshot> {
  const payload = await apiClient.get<unknown>(EXITS_BASE, {
    query: {
      page: options.page ?? 1,
      page_size: options.pageSize ?? 100,
      sort_by: "position",
      sort_order: "asc",
      account_id: options.accountId,
    },
    signal: options.signal,
  })
  return normalizeStaticProxyExitsResponse(payload)
}

export async function getStaticProxyCapacitySummary(
  options: { accountId?: string; signal?: AbortSignal } = {},
): Promise<StaticProxyExitCapacitySummary> {
  const payload = await apiClient.get<unknown>(`${EXITS_BASE}/capacity`, {
    query: { account_id: options.accountId },
    signal: options.signal,
  })
  return normalizeStaticProxyCapacitySummary(payload)
}

export async function createStaticProxyExit(
  input: CreateStaticProxyExitInput,
  signal?: AbortSignal,
): Promise<StaticProxyExit> {
  const payload = await apiClient.post<unknown>(EXITS_BASE, {
    body: {
      account_id: input.accountId,
      proxy_id: input.proxyId,
      name: input.name ?? "",
      position: input.position ?? 0,
      enabled: input.enabled ?? true,
    },
    signal,
  })
  return normalizeStaticProxyExit(payload) as StaticProxyExit
}

export async function updateStaticProxyExit(
  id: string,
  input: UpdateStaticProxyExitInput,
  signal?: AbortSignal,
): Promise<StaticProxyExit> {
  const payload = await apiClient.put<unknown>(`${EXITS_BASE}/${id}`, {
    body: {
      account_id: input.accountId,
      proxy_id: input.proxyId,
      name: input.name,
      position: input.position,
      enabled: input.enabled,
    },
    signal,
  })
  return normalizeStaticProxyExit(payload) as StaticProxyExit
}

export async function toggleStaticProxyExit(
  id: string,
  enabled: boolean,
  signal?: AbortSignal,
): Promise<StaticProxyExit> {
  const payload = await apiClient.post<unknown>(`${EXITS_BASE}/${id}/toggle`, {
    body: { enabled },
    signal,
  })
  return normalizeStaticProxyExit(payload) as StaticProxyExit
}

export async function deleteStaticProxyExit(id: string, signal?: AbortSignal): Promise<void> {
  await apiClient.del<unknown>(`${EXITS_BASE}/${id}`, { signal })
}

export async function listProxyOptions(signal?: AbortSignal): Promise<ProxyPickerOption[]> {
  const payload = await apiClient.get<unknown>("/api/v1/admin/proxies", {
    query: { page: 1, page_size: 200, sort_by: "id", sort_order: "asc" },
    signal,
  })
  return normalizeProxyOptions(payload)
}

export async function listAccountOptions(signal?: AbortSignal): Promise<PickerOption[]> {
  const payload = await apiClient.get<unknown>("/api/v1/admin/accounts", {
    query: { page: 1, page_size: 200, sort_by: "created_at", sort_order: "desc" },
    signal,
  })
  return normalizeAccountOptions(payload)
}

import { finiteNumber } from "../accounts/capacity"
import {
  PickerOption,
  ProxyPickerOption,
  StaticExitsSnapshot,
  StaticProxyExit,
  StaticProxyExitCapacitySummary,
  StaticProxyExitProxy,
} from "./types"

type JsonRecord = Record<string, unknown>

function isRecord(value: unknown): value is JsonRecord {
  return typeof value === "object" && value !== null && !Array.isArray(value)
}

function firstValue(record: JsonRecord, keys: string[]): unknown {
  for (const key of keys) {
    if (record[key] !== undefined && record[key] !== null) return record[key]
  }
  return undefined
}

function stringValue(value: unknown): string {
  return typeof value === "string" ? value.trim() : ""
}

function numberValue(value: unknown): number {
  return finiteNumber(value) ?? 0
}

function unwrapEnvelope(payload: unknown): unknown {
  if (isRecord(payload) && "code" in payload && isRecord(payload.data)) return payload.data
  return payload
}

function listFromPayload(payload: unknown, keys: string[]): { items: unknown[]; metadata: JsonRecord } {
  const data = unwrapEnvelope(payload)
  if (Array.isArray(data)) return { items: data, metadata: {} }
  if (!isRecord(data)) return { items: [], metadata: {} }
  for (const key of [...keys, "items", "results"]) {
    const candidate = firstValue(data, [key])
    if (Array.isArray(candidate)) return { items: candidate, metadata: data }
  }
  return { items: [], metadata: data }
}

function normalizeProxyRef(value: unknown): StaticProxyExitProxy | null {
  if (!isRecord(value)) return null
  const id = firstValue(value, ["id"])
  if (id === undefined || id === null) return null
  return {
    id: String(id),
    name: stringValue(firstValue(value, ["name"])) || `Proxy ${id}`,
    protocol: stringValue(firstValue(value, ["protocol"])) || "unknown",
    host: stringValue(firstValue(value, ["host", "address", "ip"])),
    port: finiteNumber(firstValue(value, ["port"])),
    status: stringValue(firstValue(value, ["status"])) || "unknown",
  }
}

export function normalizeStaticProxyExit(value: unknown, index = 0): StaticProxyExit | null {
  const data = unwrapEnvelope(value)
  if (!isRecord(data)) return null
  const rawId = firstValue(data, ["id", "exit_id"])
  if (rawId === undefined || rawId === null) return null
  return {
    id: String(rawId),
    accountId: String(firstValue(data, ["account_id", "accountId"]) ?? ""),
    proxyId: String(firstValue(data, ["proxy_id", "proxyId"]) ?? ""),
    name: stringValue(firstValue(data, ["name"])) || `Exit ${rawId}`,
    position: numberValue(firstValue(data, ["position"])),
    enabled: firstValue(data, ["enabled"]) === true,
    capacity: numberValue(firstValue(data, ["capacity"])),
    healthStatus: stringValue(firstValue(data, ["health_status", "healthStatus"])) || "unknown",
    failureCount: numberValue(firstValue(data, ["failure_count", "failureCount"])),
    lastFailureReason: stringValue(firstValue(data, ["last_failure_reason", "lastFailureReason"])),
    proxy: normalizeProxyRef(firstValue(data, ["proxy"])),
  }
}

export function normalizeStaticProxyExitsResponse(payload: unknown): StaticExitsSnapshot {
  const { items, metadata } = listFromPayload(payload, ["exits"])
  const exits = items.flatMap((item, index) => {
    const exit = normalizeStaticProxyExit(item, index)
    return exit ? [exit] : []
  })

  return {
    exits,
    total: finiteNumber(firstValue(metadata, ["total", "count"])) ?? exits.length,
    page: finiteNumber(firstValue(metadata, ["page", "current_page"])),
    pages: finiteNumber(firstValue(metadata, ["pages", "total_pages"])),
  }
}

export function normalizeStaticProxyCapacitySummary(payload: unknown): StaticProxyExitCapacitySummary {
  const data = unwrapEnvelope(payload)
  const record = isRecord(data) ? data : {}
  const accountId = firstValue(record, ["account_id", "accountId"])
  return {
    accountId: accountId === undefined || accountId === null ? null : String(accountId),
    assignmentCount: numberValue(firstValue(record, ["assignment_count", "assignmentCount"])),
    enabledCount: numberValue(firstValue(record, ["enabled_count", "enabledCount"])),
    disabledCount: numberValue(firstValue(record, ["disabled_count", "disabledCount"])),
    capacityPerAssignment: numberValue(firstValue(record, ["capacity_per_assignment", "capacityPerAssignment"])),
    totalCapacity: numberValue(firstValue(record, ["total_capacity", "totalCapacity"])),
    enabledCapacity: numberValue(firstValue(record, ["enabled_capacity", "enabledCapacity"])),
  }
}

function proxyOptionLabel(record: JsonRecord): string {
  const name = stringValue(firstValue(record, ["name"]))
  const protocol = stringValue(firstValue(record, ["protocol"])) || "proxy"
  const host = stringValue(firstValue(record, ["host", "address", "ip"])) || "unknown-host"
  const port = finiteNumber(firstValue(record, ["port"]))
  const endpoint = port === null ? host : `${host}:${port}`
  const identity = name ? `${name} — ` : ""
  return `${identity}${protocol}://${endpoint}`
}

export function normalizeProxyOptions(payload: unknown): ProxyPickerOption[] {
  const { items } = listFromPayload(payload, ["proxies"])
  return items.flatMap((item) => {
    if (!isRecord(item)) return []
    const id = firstValue(item, ["id"])
    if (id === undefined || id === null) return []
    return [{
      id: String(id),
      label: proxyOptionLabel(item),
      host: stringValue(firstValue(item, ["host", "address", "ip"])),
      port: finiteNumber(firstValue(item, ["port"])),
      protocol: stringValue(firstValue(item, ["protocol"])) || "unknown",
    }]
  })
}

export function normalizeAccountOptions(payload: unknown): PickerOption[] {
  const { items } = listFromPayload(payload, ["accounts"])
  return items.flatMap((item) => {
    if (!isRecord(item)) return []
    const id = firstValue(item, ["id", "account_id", "accountId"])
    if (id === undefined || id === null) return []
    const name = stringValue(firstValue(item, ["name", "email", "account_name"]))
    return [{ id: String(id), label: name || `Account ${id}` }]
  })
}

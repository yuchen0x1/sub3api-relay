import { calculateCapacity, finiteNumber } from "./capacity"
import { AccountCapacity, AccountsSnapshot, CapacityMetric, NormalizedAccount, StaticIpCapacity } from "./types"

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

function metricFromValue(value: unknown, fallback: JsonRecord = {}): CapacityMetric | null {
  if (isRecord(value)) {
    const used = firstValue(value, ["used", "current", "in_use", "current_concurrency"])
    const limit = firstValue(value, ["limit", "max", "capacity", "total", "max_capacity"])
    return calculateCapacity(used, limit)
  }
  return calculateCapacity(value, firstValue(fallback, ["capacity", "limit", "max_capacity"]))
}

function aggregateCapacity(record: JsonRecord): CapacityMetric | null {
  const nested = firstValue(record, ["aggregate_capacity", "aggregateCapacity", "capacity"])
  if (nested !== undefined) {
    if (!isRecord(nested)) {
      const nestedLimit = finiteNumber(nested)
      if (nestedLimit !== null) {
        return calculateCapacity(
          firstValue(record, ["current_concurrency", "concurrency_used", "aggregate_used", "capacity_used", "used_capacity"]),
          nestedLimit,
        )
      }
    }
    const nestedMetric = metricFromValue(nested, record)
    if (nestedMetric) return nestedMetric
  }

  return calculateCapacity(
    firstValue(record, ["current_concurrency", "concurrency_used", "aggregate_used", "capacity_used", "used_capacity"]),
    firstValue(record, ["concurrency", "concurrency_limit", "capacity_limit", "max_capacity"]),
  )
}

function staticIpLabel(value: JsonRecord, fallback: string): string {
  const label = firstValue(value, ["ip", "address", "static_ip", "staticIp", "name"])
  return typeof label === "string" && label.trim() ? label : fallback
}

function staticIpMetric(value: unknown): CapacityMetric | null {
  if (!isRecord(value)) return null
  return metricFromValue(value)
}

function perStaticIpCapacity(record: JsonRecord): StaticIpCapacity[] {
  const source = firstValue(record, ["per_static_ip_capacity", "perStaticIpCapacity", "static_ip_capacity", "staticIpCapacity", "static_ip_capacities"])
  if (Array.isArray(source)) {
    return source.flatMap((entry, index) => {
      if (!isRecord(entry)) return []
      const capacity = staticIpMetric(entry)
      return capacity ? [{ ip: staticIpLabel(entry, `Static IP ${index + 1}`), capacity }] : []
    })
  }

  if (isRecord(source)) {
    const directMetric = staticIpMetric(source)
    if (directMetric) return [{ ip: staticIpLabel(source, "Static IP"), capacity: directMetric }]

    return Object.entries(source).flatMap(([ip, value]) => {
      const capacity = staticIpMetric(value)
      return capacity ? [{ ip, capacity }] : []
    })
  }

  return []
}

function normalizeAccount(value: unknown, index: number): NormalizedAccount | null {
  if (!isRecord(value)) return null
  const rawId = firstValue(value, ["id", "account_id", "accountId"])
  const id = rawId === undefined || rawId === null ? `unknown-${index + 1}` : String(rawId)
  const nameValue = firstValue(value, ["name", "email", "account_name", "accountName"])
  const capacity: AccountCapacity = {
    aggregate: aggregateCapacity(value),
    perStaticIp: perStaticIpCapacity(value),
  }

  return {
    id,
    name: typeof nameValue === "string" && nameValue.trim() ? nameValue : `Account ${id}`,
    platform: String(firstValue(value, ["platform", "provider"]) ?? "unknown"),
    type: String(firstValue(value, ["type", "account_type"]) ?? "unknown"),
    status: String(firstValue(value, ["status"]) ?? "unknown"),
    capacity,
  }
}

function itemsFromPayload(payload: unknown): { items: unknown[]; metadata: JsonRecord } {
  if (Array.isArray(payload)) return { items: payload, metadata: {} }
  if (!isRecord(payload)) return { items: [], metadata: {} }

  const items = firstValue(payload, ["items", "accounts", "results"])
  if (Array.isArray(items)) return { items, metadata: payload }
  if (isRecord(payload.data)) return itemsFromPayload(payload.data)
  return { items: [], metadata: payload }
}

export function normalizeAccountsResponse(payload: unknown): AccountsSnapshot {
  let data = payload
  if (isRecord(payload) && "code" in payload) data = payload.data

  const { items, metadata } = itemsFromPayload(data)
  const accounts = items.flatMap((item, index) => {
    const account = normalizeAccount(item, index)
    return account ? [account] : []
  })

  return {
    accounts,
    total: finiteNumber(firstValue(metadata, ["total", "count"])) ?? (accounts.length ? accounts.length : 0),
    page: finiteNumber(firstValue(metadata, ["page", "current_page"])),
    pages: finiteNumber(firstValue(metadata, ["pages", "total_pages"])),
  }
}

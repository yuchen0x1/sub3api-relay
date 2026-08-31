import { CapacityMetric } from "./types"

export function finiteNumber(value: unknown): number | null {
  if (typeof value === "number" && Number.isFinite(value)) return value
  if (typeof value === "string" && value.trim() !== "") {
    const parsed = Number(value)
    return Number.isFinite(parsed) ? parsed : null
  }
  return null
}

export function calculateCapacity(usedValue: unknown, limitValue: unknown): CapacityMetric | null {
  const usedNumber = finiteNumber(usedValue)
  const limitNumber = finiteNumber(limitValue)
  if (usedNumber === null && limitNumber === null) return null

  const used = Math.max(0, usedNumber ?? 0)
  const limit = limitNumber === null || limitNumber < 0 ? null : Math.max(0, limitNumber)
  const available = limit === null ? null : Math.max(0, limit - used)
  const utilization = limit === null ? null : limit === 0 ? (used > 0 ? 1 : 0) : used / limit

  return { used, limit, available, utilization }
}

const UPDATE_INTERVAL_KEY = "sub3api.desktop.update-check-hours"
const DEFAULT_UPDATE_INTERVAL_HOURS = 6
const ALLOWED_UPDATE_INTERVALS = [0, 6, 24]

export function getUpdateStorage(): Storage | undefined {
  try {
    return typeof window === "undefined" ? undefined : window.localStorage
  } catch {
    return undefined
  }
}

export function readUpdateInterval(storage: Pick<Storage, "getItem"> | undefined): number {
  if (!storage) return DEFAULT_UPDATE_INTERVAL_HOURS
  try {
    const raw = storage.getItem(UPDATE_INTERVAL_KEY)
    if (raw === null) return DEFAULT_UPDATE_INTERVAL_HOURS
    if (!raw.trim()) return DEFAULT_UPDATE_INTERVAL_HOURS
    const value = Number(raw)
    return ALLOWED_UPDATE_INTERVALS.includes(value) ? value : DEFAULT_UPDATE_INTERVAL_HOURS
  } catch {
    return DEFAULT_UPDATE_INTERVAL_HOURS
  }
}

export function saveUpdateInterval(storage: Pick<Storage, "setItem"> | undefined, hours: number): void {
  if (!storage || !ALLOWED_UPDATE_INTERVALS.includes(hours)) return
  try {
    storage.setItem(UPDATE_INTERVAL_KEY, String(hours))
  } catch {
    // Keep the selected interval for this run if persistent storage is unavailable.
  }
}

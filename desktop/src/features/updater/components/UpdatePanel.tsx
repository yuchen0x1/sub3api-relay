import { useCallback, useEffect, useState } from "react"
import type { DownloadEvent, Update } from "@tauri-apps/plugin-updater"
import { checkForDesktopUpdate, installDesktopUpdate, isTauriRuntime } from "../api"

type CheckState = "idle" | "checking" | "available" | "current" | "downloading" | "error"

const UPDATE_INTERVAL_KEY = "sub3api.desktop.update-check-hours"
const UPDATE_INTERVALS = [
  { value: 0, label: "Startup only" },
  { value: 6, label: "Every 6 hours" },
  { value: 24, label: "Every 24 hours" },
] as const

function readUpdateInterval(): number {
  if (typeof window === "undefined") return 6
  const value = Number(window.localStorage.getItem(UPDATE_INTERVAL_KEY))
  return UPDATE_INTERVALS.some((option) => option.value === value) ? value : 6
}

function formatBytes(value: number): string {
  if (!Number.isFinite(value) || value <= 0) return ""
  const units = ["B", "KB", "MB", "GB"]
  let amount = value
  let unit = 0
  while (amount >= 1024 && unit < units.length - 1) {
    amount /= 1024
    unit += 1
  }
  return `${amount >= 10 || unit === 0 ? amount.toFixed(0) : amount.toFixed(1)} ${units[unit]}`
}

export function UpdatePanel() {
  const [state, setState] = useState<CheckState>("idle")
  const [update, setUpdate] = useState<Update | null>(null)
  const [downloaded, setDownloaded] = useState(0)
  const [contentLength, setContentLength] = useState<number | undefined>()
  const [error, setError] = useState<string | null>(null)
  const [intervalHours, setIntervalHours] = useState(readUpdateInterval)

  const checkUpdate = useCallback(async () => {
    if (!isTauriRuntime()) return
    setState("checking")
    setError(null)
    try {
      const next = await checkForDesktopUpdate()
      setUpdate(next)
      setState(next ? "available" : "current")
    } catch (cause) {
      setState("error")
      setError(cause instanceof Error ? cause.message : "Unable to check for updates.")
    }
  }, [])

  useEffect(() => {
    if (!isTauriRuntime()) return
    const timer = window.setTimeout(() => void checkUpdate(), 2500)
    return () => window.clearTimeout(timer)
  }, [checkUpdate])

  useEffect(() => {
    if (!isTauriRuntime() || intervalHours <= 0) return
    const timer = window.setInterval(() => void checkUpdate(), intervalHours * 60 * 60 * 1000)
    return () => window.clearInterval(timer)
  }, [checkUpdate, intervalHours])

  useEffect(() => {
    return () => {
      if (update) void update.close().catch(() => undefined)
    }
  }, [update])

  async function install() {
    if (!update || state === "downloading") return
    setState("downloading")
    setDownloaded(0)
    setError(null)
    try {
      await installDesktopUpdate(update, (event) => {
        if (event.event === "Started") {
          setContentLength(event.data.contentLength)
        } else if (event.event === "Progress") {
          setDownloaded((current) => current + event.data.chunkLength)
        }
      })
    } catch (cause) {
      setState("error")
      setError(cause instanceof Error ? cause.message : "Unable to install the update.")
    }
  }

  function deferUpdate() {
    setUpdate(null)
    setError(null)
    setState("idle")
  }

  if (!isTauriRuntime()) return null

  const progress = contentLength && contentLength > 0
    ? Math.min(100, Math.round((downloaded / contentLength) * 100))
    : undefined

  return (
    <section className="update-panel" aria-live="polite">
      <div className="update-heading">
        <div><p className="eyebrow">Software</p><h2>Updates</h2></div>
        <div className="update-actions">
          <label className="update-interval">Auto-check
            <select value={intervalHours} onChange={(event) => {
              const next = Number(event.target.value)
              setIntervalHours(next)
              window.localStorage.setItem(UPDATE_INTERVAL_KEY, String(next))
            }}>
              {UPDATE_INTERVALS.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}
            </select>
          </label>
          <button className="button button-quiet" type="button" onClick={() => void checkUpdate()} disabled={state === "checking" || state === "downloading"}>
            {state === "checking" ? "Checking..." : "Check now"}
          </button>
        </div>
      </div>
      {state === "available" && update ? (
        <div className="update-card update-card-available">
          <div><strong>Version {update.version} is available.</strong><p>{update.body || "A signed update is ready to install."}</p>{update.date ? <p>Published {new Date(update.date).toLocaleString()}</p> : null}</div>
          <div className="update-card-actions"><button className="button button-quiet" type="button" onClick={deferUpdate}>Later</button><button className="button button-primary" type="button" onClick={() => void install()}>Download and install</button></div>
        </div>
      ) : state === "downloading" ? (
        <div className="update-card"><strong>Downloading version {update?.version}...</strong><div className="progress-track"><span style={{ width: `${progress ?? 0}%` }} /></div><p>{progress === undefined ? formatBytes(downloaded) : `${progress}% · ${formatBytes(downloaded)}${contentLength ? ` / ${formatBytes(contentLength)}` : ""}`}</p></div>
      ) : state === "error" ? (
        <div className="update-card update-card-error"><strong>Update check failed.</strong><p>{error}</p></div>
      ) : (
        <p className="update-status">{state === "current" ? "You are running the latest signed desktop release." : "Checks run against the signed GitHub Release channel."}</p>
      )}
    </section>
  )
}

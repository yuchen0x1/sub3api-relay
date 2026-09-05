import { FormEvent, useCallback, useEffect, useMemo, useRef, useState } from "react"
import { ApiError } from "../../../lib/http/apiClient"
import {
  createStaticProxyExit,
  deleteStaticProxyExit,
  getStaticProxyCapacitySummary,
  listAccountOptions,
  listProxyOptions,
  listStaticProxyExits,
  toggleStaticProxyExit,
} from "../api"
import {
  PickerOption,
  ProxyPickerOption,
  StaticExitsSnapshot,
  StaticProxyExit,
  StaticProxyExitCapacitySummary,
} from "../types"

const DELETE_CONFIRM_TIMEOUT_MS = 3000

function formatNumber(value: number): string {
  return new Intl.NumberFormat("en-US", { maximumFractionDigits: 0 }).format(value)
}

function healthTagClass(status: string): string {
  if (status === "healthy") return "tag tag-success"
  if (status === "unhealthy") return "tag tag-danger"
  return "tag tag-warning"
}

function proxyLabel(exit: StaticProxyExit): string {
  const proxy = exit.proxy
  if (!proxy) return `Proxy #${exit.proxyId || "unknown"}`
  const endpoint = proxy.port === null ? proxy.host : `${proxy.host}:${proxy.port}`
  return `${proxy.protocol}://${endpoint}`
}

function exitRowKey(exit: StaticProxyExit): string {
  return exit.id
}

function errorCopy(error: unknown): { title: string; detail: string } {
  if (error instanceof ApiError && error.status === 401) return { title: "Authentication required", detail: "Load an admin API key or bearer JWT above, then retry." }
  if (error instanceof ApiError && error.status === 403) return { title: "Admin access denied", detail: "The current session does not have permission to manage static proxy exits." }
  if (error instanceof ApiError) return { title: "Static exits request failed", detail: error.message }
  return { title: "Unable to reach Sub3API", detail: "Check the endpoint and network connection, then retry." }
}

interface ExitRowProps {
  exit: StaticProxyExit
  busy: boolean
  confirmDelete: boolean
  onToggle: (exit: StaticProxyExit) => void
  onRequestDelete: (exit: StaticProxyExit) => void
}

function ExitRow({ exit, busy, confirmDelete, onToggle, onRequestDelete }: ExitRowProps) {
  return (
    <tr>
      <td>
        <div className="account-identity">
          <span className="account-avatar" aria-hidden="true">#{exit.position || exit.id.slice(-2)}</span>
          <div>
            <div className="account-name">{exit.name}</div>
            <div className="account-id">Exit ID {exit.id}{exit.lastFailureReason ? ` — ${exit.lastFailureReason}` : ""}</div>
          </div>
        </div>
      </td>
      <td><span className="account-id">Account {exit.accountId || "unassigned"}</span></td>
      <td><span className="capacity-value">{proxyLabel(exit)}</span></td>
      <td><span className={`tag ${exit.enabled ? "tag-success" : ""}`}>{exit.enabled ? `${formatNumber(exit.capacity)} slots` : "paused"}</span></td>
      <td><span className={healthTagClass(exit.healthStatus)}>{exit.healthStatus}</span></td>
      <td>
        <div className="row-actions">
          <button className="button button-quiet" type="button" disabled={busy} onClick={() => onToggle(exit)}>{exit.enabled ? "Disable" : "Enable"}</button>
          <button className={`button ${confirmDelete ? "button-danger" : "button-quiet"}`} type="button" disabled={busy} onClick={() => onRequestDelete(exit)}>{confirmDelete ? "Confirm delete" : "Delete"}</button>
        </div>
      </td>
    </tr>
  )
}

export function StaticExitsPage({ enabled }: { enabled: boolean }) {
  const [snapshot, setSnapshot] = useState<StaticExitsSnapshot | null>(null)
  const [summary, setSummary] = useState<StaticProxyExitCapacitySummary | null>(null)
  const [accounts, setAccounts] = useState<PickerOption[]>([])
  const [proxies, setProxies] = useState<ProxyPickerOption[]>([])
  const [loading, setLoading] = useState(enabled)
  const [error, setError] = useState<unknown>(null)
  const [reloadToken, setReloadToken] = useState(0)
  const [pendingId, setPendingId] = useState<string | null>(null)
  const [confirmDeleteId, setConfirmDeleteId] = useState<string | null>(null)
  const [formAccountId, setFormAccountId] = useState("")
  const [formProxyId, setFormProxyId] = useState("")
  const [formName, setFormName] = useState("")
  const [formError, setFormError] = useState<string | null>(null)
  const [creating, setCreating] = useState(false)
  const confirmTimer = useRef<number | null>(null)

  const reload = useCallback(() => setReloadToken((value) => value + 1), [])

  useEffect(() => {
    if (!enabled) {
      setSnapshot(null); setSummary(null); setAccounts([]); setProxies([]); setError(null); setLoading(false)
      return
    }
    const controller = new AbortController()
    setLoading(true); setError(null)
    Promise.all([
      listStaticProxyExits({ signal: controller.signal }),
      getStaticProxyCapacitySummary({ signal: controller.signal }),
      listAccountOptions(controller.signal),
      listProxyOptions(controller.signal),
    ]).then(([exitsSnapshot, capacitySummary, accountOptions, proxyOptions]) => {
      setSnapshot(exitsSnapshot)
      setSummary(capacitySummary)
      setAccounts(accountOptions)
      setProxies(proxyOptions)
    }).catch((requestError: unknown) => {
      if (!(requestError instanceof DOMException && requestError.name === "AbortError")) setError(requestError)
    }).finally(() => setLoading(false))
    return () => controller.abort()
  }, [enabled, reloadToken])

  useEffect(() => () => {
    if (confirmTimer.current !== null) window.clearTimeout(confirmTimer.current)
  }, [])

  const busy = pendingId !== null || creating
  const exits = snapshot?.exits ?? []
  const stats = useMemo(() => {
    const perAssignment = summary?.capacityPerAssignment ?? 0
    const enabledCount = summary?.enabledCount ?? exits.filter((exit) => exit.enabled).length
    const enabledCapacity = summary?.enabledCapacity ?? enabledCount * perAssignment
    return {
      assignments: summary?.assignmentCount ?? exits.length,
      enabled: enabledCount,
      enabledCapacity,
      perAssignment,
      totalCapacity: summary?.totalCapacity ?? exits.length * perAssignment,
    }
  }, [summary, exits])

  function resetConfirmTimer() {
    if (confirmTimer.current !== null) window.clearTimeout(confirmTimer.current)
    confirmTimer.current = window.setTimeout(() => {
      confirmTimer.current = null
      setConfirmDeleteId(null)
    }, DELETE_CONFIRM_TIMEOUT_MS)
  }

  async function runExitMutation(exit: StaticProxyExit, action: () => Promise<unknown>) {
    setPendingId(exit.id)
    try {
      await action()
      reload()
    } finally {
      setPendingId(null)
    }
  }

  function handleToggle(exit: StaticProxyExit) {
    void runExitMutation(exit, () => toggleStaticProxyExit(exit.id, !exit.enabled))
  }

  function handleRequestDelete(exit: StaticProxyExit) {
    if (confirmDeleteId === exit.id) {
      if (confirmTimer.current !== null) window.clearTimeout(confirmTimer.current)
      confirmTimer.current = null
      setConfirmDeleteId(null)
      void runExitMutation(exit, () => deleteStaticProxyExit(exit.id))
      return
    }
    setConfirmDeleteId(exit.id)
    resetConfirmTimer()
  }

  async function handleCreate(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!formAccountId || !formProxyId) {
      setFormError("Select both an account and a proxy to create an exit.")
      return
    }
    setCreating(true); setFormError(null)
    try {
      await createStaticProxyExit({ accountId: formAccountId, proxyId: formProxyId, name: formName.trim() })
      setFormName("")
      reload()
    } catch (createError: unknown) {
      setFormError(createError instanceof ApiError ? createError.message : "Unable to create the static exit.")
    } finally {
      setCreating(false)
    }
  }

  if (!enabled) return <div className="state-panel">Load an admin session to manage static exits.</div>
  if (loading) return <div className="state-panel loading-state">Loading static exit inventory...</div>
  if (error) {
    const copy = errorCopy(error)
    return <div className="error-panel" role="alert"><strong>{copy.title}</strong><p>{copy.detail}</p><button className="button button-quiet" type="button" onClick={reload} style={{ marginTop: 14 }}>Retry</button></div>
  }

  return (
    <section className="static-exits-view" aria-label="Static proxy exits">
      <div className="stat-strip" aria-label="Static exit capacity summary">
        <div className="stat"><span className="stat-label">Assignments</span><span className="stat-value">{formatNumber(stats.assignments)}</span></div>
        <div className="stat"><span className="stat-label">Enabled exits</span><span className="stat-value">{formatNumber(stats.enabled)}</span></div>
        <div className="stat"><span className="stat-label">Live capacity (slots)</span><span className="stat-value">{formatNumber(stats.enabledCapacity)}</span></div>
        <div className="stat"><span className="stat-label">Total capacity (slots)</span><span className="stat-value">{formatNumber(stats.totalCapacity)}</span></div>
      </div>

      <form className="exit-create-panel" onSubmit={handleCreate}>
        <label><span>Account</span><select value={formAccountId} onChange={(event) => setFormAccountId(event.target.value)}>
          <option value="">Select an account…</option>
          {accounts.map((account) => <option key={account.id} value={account.id}>{account.label}</option>)}
        </select></label>
        <label><span>Proxy (static IP)</span><select value={formProxyId} onChange={(event) => setFormProxyId(event.target.value)}>
          <option value="">Select a proxy…</option>
          {proxies.map((proxy) => <option key={proxy.id} value={proxy.id}>{proxy.label}</option>)}
        </select></label>
        <label><span>Label (optional)</span><input type="text" value={formName} onChange={(event) => setFormName(event.target.value)} placeholder="e.g. us-east exit 1" /></label>
        <button className="button button-primary" type="submit" disabled={busy}>Add exit</button>
      </form>
      <p className="exit-capacity-note">Each enabled exit contributes {formatNumber(stats.perAssignment || 10)} concurrent slots; multiple exits on one account multiply the account capacity.</p>
      {formError ? <div className="error-panel" role="alert"><strong>Could not add the exit</strong><p>{formError}</p></div> : null}

      <div className="table-frame">
        <div className="table-toolbar"><h3>Static exit assignments</h3><button className="refresh-button" type="button" onClick={reload}>Refresh</button></div>
        {exits.length ? (
          <table>
            <thead><tr><th>Exit</th><th>Account</th><th>Proxy</th><th>Capacity</th><th>Health</th><th>Actions</th></tr></thead>
            <tbody>
              {exits.map((exit) => (
                <ExitRow
                  exit={exit}
                  key={exitRowKey(exit)}
                  busy={pendingId === exit.id || creating}
                  confirmDelete={confirmDeleteId === exit.id}
                  onToggle={handleToggle}
                  onRequestDelete={handleRequestDelete}
                />
              ))}
            </tbody>
          </table>
        ) : (
          <div className="empty-state"><strong>No static exits yet</strong><span>Bind a proxy to an account above to create the first static exit.</span></div>
        )}
      </div>
    </section>
  )
}

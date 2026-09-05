import { useCallback, useEffect, useMemo, useState } from "react"
import { ApiError } from "../../../lib/http/apiClient"
import { listAccounts } from "../api"
import { AccountsSnapshot, CapacityMetric, NormalizedAccount } from "../types"

function formatNumber(value: number): string {
  return new Intl.NumberFormat("en-US", { maximumFractionDigits: 2 }).format(value)
}

function formatCapacity(metric: CapacityMetric): string {
  return `${formatNumber(metric.used)} / ${metric.limit === null ? "unlimited" : formatNumber(metric.limit)}`
}

function capacityClass(metric: CapacityMetric): string {
  if (metric.utilization !== null && metric.utilization >= 1) return "is-full"
  if (metric.utilization !== null && metric.utilization >= 0.8) return "is-high"
  return ""
}

function CapacityMetricView({ metric }: { metric: CapacityMetric | null }) {
  if (!metric) return <span className="not-reported">Not reported</span>
  const width = metric.utilization === null ? 0 : Math.min(100, Math.max(0, metric.utilization * 100))
  return (
    <div className="capacity-cell">
      <div className="capacity-line">
        <span className="capacity-value">{formatCapacity(metric)}</span>
        <span className="capacity-meta">{metric.available === null ? "open" : `${formatNumber(metric.available)} available`}</span>
      </div>
      <div className="meter" aria-hidden="true"><span className={capacityClass(metric)} style={{ width: `${width}%` }} /></div>
    </div>
  )
}

function accountStatusClass(status: string): string {
  if (status === "active") return "tag-success"
  if (status === "error" || status === "disabled" || status === "inactive") return "tag-danger"
  return "tag-warning"
}

function AccountRow({ account }: { account: NormalizedAccount }) {
  return (
    <tr>
      <td><div className="account-identity"><span className="account-avatar" aria-hidden="true">{account.name.slice(0, 2).toUpperCase()}</span><div><div className="account-name">{account.name}</div><div className="account-id">ID {account.id}</div></div></div></td>
      <td><span className="tag">{account.platform}</span></td>
      <td><span className={`tag ${accountStatusClass(account.status)}`}>{account.status}</span></td>
      <td><CapacityMetricView metric={account.capacity.aggregate} /></td>
      <td>{account.capacity.perStaticIp.length ? <div className="static-list">{account.capacity.perStaticIp.map((item) => <div className="static-item" key={`${account.id}-${item.ip}`}><div className="static-ip">{item.ip}</div><CapacityMetricView metric={item.capacity} /></div>)}</div> : <span className="not-reported">Not reported</span>}</td>
    </tr>
  )
}

function errorCopy(error: unknown): { title: string; detail: string } {
  if (error instanceof ApiError && error.status === 401) return { title: "Authentication required", detail: "Load an admin API key or bearer JWT above, then retry." }
  if (error instanceof ApiError && error.status === 403) return { title: "Admin access denied", detail: "The current session does not have permission to read accounts." }
  if (error instanceof ApiError) return { title: "Accounts request failed", detail: error.message }
  return { title: "Unable to reach Sub3API", detail: "Check the endpoint and network connection, then retry." }
}

export function AccountsPage({ enabled }: { enabled: boolean }) {
  const [snapshot, setSnapshot] = useState<AccountsSnapshot | null>(null)
  const [loading, setLoading] = useState(enabled)
  const [error, setError] = useState<unknown>(null)
  const [reloadToken, setReloadToken] = useState(0)
  const reload = useCallback(() => setReloadToken((value) => value + 1), [])

  useEffect(() => {
    if (!enabled) { setSnapshot(null); setError(null); setLoading(false); return }
    const controller = new AbortController()
    setLoading(true); setError(null)
    listAccounts({ signal: controller.signal }).then(setSnapshot).catch((requestError: unknown) => { if (!(requestError instanceof DOMException && requestError.name === "AbortError")) setError(requestError) }).finally(() => setLoading(false))
    return () => controller.abort()
  }, [enabled, reloadToken])

  const stats = useMemo(() => {
    const accounts = snapshot?.accounts ?? []
    return { total: snapshot?.total ?? accounts.length, active: accounts.filter((account) => account.status === "active").length, reported: accounts.filter((account) => account.capacity.aggregate !== null).length }
  }, [snapshot])

  if (!enabled) return <div className="state-panel">Load an admin session to inspect accounts.</div>
  if (loading) return <div className="state-panel loading-state">Loading account inventory...</div>
  if (error) {
    const copy = errorCopy(error)
    return <div className="error-panel" role="alert"><strong>{copy.title}</strong><p>{copy.detail}</p><button className="button button-quiet" type="button" onClick={reload} style={{ marginTop: 14 }}>Retry</button></div>
  }

  const accounts = snapshot?.accounts ?? []
  return (
    <section className="accounts-view" aria-label="Accounts inventory">
      <div className="stat-strip" aria-label="Account summary"><div className="stat"><span className="stat-label">Accounts</span><span className="stat-value">{formatNumber(stats.total)}</span></div><div className="stat"><span className="stat-label">Active now</span><span className="stat-value">{formatNumber(stats.active)}</span></div><div className="stat"><span className="stat-label">Capacity reported</span><span className="stat-value">{formatNumber(stats.reported)}</span></div></div>
      <div className="table-frame"><div className="table-toolbar"><h3>Account inventory</h3><button className="refresh-button" type="button" onClick={reload}>Refresh</button></div>{accounts.length ? <table><thead><tr><th>Account</th><th>Platform</th><th>Status</th><th>Aggregate capacity</th><th>Per static IP</th></tr></thead><tbody>{accounts.map((account) => <AccountRow account={account} key={account.id} />)}</tbody></table> : <div className="empty-state"><strong>No accounts found</strong><span>The admin API returned an empty account inventory.</span></div>}</div>
    </section>
  )
}

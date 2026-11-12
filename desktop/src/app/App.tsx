import { FormEvent, useEffect, useState } from "react"
import { AccountsPage } from "../features/accounts/components/AccountsPage"
import { StaticExitsPage } from "../features/staticExits/components/StaticExitsPage"
import { AuthScheme, AuthSession, authTokenStore } from "../lib/auth/tokenStore"
import { apiBaseUrl } from "../lib/config"
import { UpdatePanel } from "../features/updater/components/UpdatePanel"

type AppView = "accounts" | "static-exits"

const VIEW_META: Record<AppView, { tab: string; title: string; summary: string; route: string }> = {
  accounts: {
    tab: "Accounts",
    title: "Accounts",
    summary: "Inspect account availability and capacity from the Sub3API admin API.",
    route: "GET /api/v1/admin/accounts",
  },
  "static-exits": {
    tab: "Static exits",
    title: "Static proxy exits",
    summary: "Bind static IP proxies to accounts and manage per-exit concurrency capacity.",
    route: "GET /api/v1/admin/static-proxy-exits",
  },
}

function SessionControls({ session, onSessionChange }: { session: AuthSession | null; onSessionChange: (next: AuthSession | null) => void }) {
  const [scheme, setScheme] = useState<AuthScheme>("api-key")
  const [tokenDraft, setTokenDraft] = useState("")
  const [saving, setSaving] = useState(false)
  const [message, setMessage] = useState<string | null>(null)

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const token = tokenDraft.trim()
    if (!token) return
    setSaving(true)
    setMessage(null)
    try {
      const nextSession = { token, scheme }
      await authTokenStore.set(nextSession)
      setTokenDraft("")
      onSessionChange(nextSession)
      setMessage("Session token loaded for this run.")
    } catch {
      setMessage("Unable to update the session token.")
    } finally {
      setSaving(false)
    }
  }

  async function clearSession() {
    await authTokenStore.clear()
    setTokenDraft("")
    setMessage("Session cleared.")
    onSessionChange(null)
  }

  return (
    <section className="session-panel" aria-label="Admin session">
      <div className="session-heading">
        <div><p className="eyebrow">Connection</p><h2>Admin session</h2></div>
        <span className={`session-indicator ${session ? "is-live" : ""}`}>{session ? "Loaded" : "Not connected"}</span>
      </div>
      <form className="session-form" onSubmit={submit}>
        <label><span>Auth method</span><select value={scheme} onChange={(event) => setScheme(event.target.value as AuthScheme)}><option value="api-key">Admin API key</option><option value="bearer">Bearer JWT</option></select></label>
        <label className="token-field"><span>{session ? "Replace session token" : "Admin token"}</span><input type="password" autoComplete="off" value={tokenDraft} onChange={(event) => setTokenDraft(event.target.value)} placeholder="Paste a token for this run" /></label>
        <button className="button button-primary" type="submit" disabled={saving || !tokenDraft.trim()}>{saving ? "Loading..." : "Load token"}</button>
        <button className="button button-quiet" type="button" onClick={clearSession} disabled={!session}>Clear</button>
      </form>
      <p className="session-note">{message ?? "Tokens stay in volatile memory and are never written to browser storage."}</p>
    </section>
  )
}

export function App() {
  const [session, setSession] = useState<AuthSession | null>(null)
  const [sessionReady, setSessionReady] = useState(false)
  const [view, setView] = useState<AppView>("accounts")

  useEffect(() => {
    let active = true
    authTokenStore.get().then((storedSession) => { if (active) setSession(storedSession) }).finally(() => { if (active) setSessionReady(true) })
    return () => { active = false }
  }, [])

  const meta = VIEW_META[view]

  return (
    <div className="app-shell">
      <header className="topbar">
        <div className="brand-lockup"><span className="brand-mark" aria-hidden="true">S3</span><div><p className="eyebrow">Control plane</p><h1>Sub3API Cockpit</h1></div></div>
        <div className="endpoint-chip" title={apiBaseUrl}><span className="status-dot" aria-hidden="true" /><span>{apiBaseUrl}</span></div>
      </header>
      <main className="main-content">
        <SessionControls session={session} onSessionChange={setSession} />
        <UpdatePanel />
        <nav className="view-tabs" aria-label="Control plane sections">
          {(Object.keys(VIEW_META) as AppView[]).map((viewKey) => (
            <button
              className={`view-tab ${view === viewKey ? "is-active" : ""}`}
              type="button"
              key={viewKey}
              onClick={() => setView(viewKey)}
            >
              {VIEW_META[viewKey].tab}
            </button>
          ))}
        </nav>
        <div className="page-heading"><div><p className="eyebrow">Operations</p><h2>{meta.title}</h2><p className="page-summary">{meta.summary}</p></div><span className="route-label">{meta.route}</span></div>
        {sessionReady ? (
          view === "accounts"
            ? <AccountsPage enabled={Boolean(session)} />
            : <StaticExitsPage enabled={Boolean(session)} />
        ) : <div className="state-panel">Preparing session...</div>}
      </main>
    </div>
  )
}

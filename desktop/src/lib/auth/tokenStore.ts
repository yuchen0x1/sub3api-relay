import { invoke } from "@tauri-apps/api/core"

export type AuthScheme = "api-key" | "bearer"

export interface AuthSession {
  token: string
  scheme: AuthScheme
}

let memorySession: AuthSession | null = null

function runningInTauri(): boolean {
  return typeof window !== "undefined" && "__TAURI_INTERNALS__" in window
}

export const authTokenStore = {
  async get(): Promise<AuthSession | null> {
    if (runningInTauri()) return invoke<AuthSession | null>("get_auth_session")
    return memorySession
  },

  async set(session: AuthSession): Promise<void> {
    if (runningInTauri()) {
      await invoke("set_auth_session", { session })
      return
    }
    memorySession = session
  },

  async clear(): Promise<void> {
    if (runningInTauri()) {
      await invoke("clear_auth_session")
      return
    }
    memorySession = null
  },
}

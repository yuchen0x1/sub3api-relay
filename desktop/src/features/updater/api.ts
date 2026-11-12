import { check, type DownloadEvent, type Update } from "@tauri-apps/plugin-updater"
import { relaunch } from "@tauri-apps/plugin-process"

export function isTauriRuntime(): boolean {
  return typeof window !== "undefined" && "__TAURI_INTERNALS__" in window
}

export async function checkForDesktopUpdate(): Promise<Update | null> {
  if (!isTauriRuntime()) return null
  return check({ timeout: 15_000 })
}

export async function installDesktopUpdate(
  update: Update,
  onEvent: (event: DownloadEvent) => void,
): Promise<void> {
  await update.downloadAndInstall(onEvent, { timeout: 15 * 60_000, restartAfterInstall: false })
  await relaunch()
}

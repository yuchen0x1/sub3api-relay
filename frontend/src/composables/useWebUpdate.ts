import { ref } from 'vue'
import type { RegisterSWOptions } from 'vite-plugin-pwa/types'

type Register = (options: RegisterSWOptions) => (reload?: boolean) => Promise<void>

const available = ref(false)
let activate: (() => Promise<void>) | undefined
let accepted = false
let dismissed = false
let activeInAnotherTab = false

export function registerWebUpdate(register: Register) {
  activate = register({
    immediate: true,
    onNeedRefresh() {
      if (!dismissed) available.value = true
    },
    onNeedReload() {
      activeInAnotherTab = true
      if (accepted) window.location.reload()
    },
    onRegisteredSW(_url, registration) {
      if (!registration) return
      window.setInterval(() => {
        if (navigator.onLine && !document.hidden) {
          void registration.update().catch(() => { /* Retry at the next check. */ })
        }
      }, 60 * 60 * 1000)
    },
  })
}

export function useWebUpdate() {
  async function refresh() {
    if (activeInAnotherTab) {
      window.location.reload()
      return
    }
    accepted = true
    try {
      await activate?.()
    } catch {
      accepted = false
    }
  }

  function later() {
    dismissed = true
    accepted = false
    available.value = false
  }

  return { available, refresh, later }
}

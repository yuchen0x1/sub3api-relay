import { computed, ref } from 'vue'

interface BeforeInstallPromptEvent extends Event {
  prompt: () => Promise<void>
  userChoice: Promise<{ outcome: 'accepted' | 'dismissed' }>
}

// Module-level state so multiple components share one pending prompt event:
// the browser dispatches beforeinstallprompt only once per page load.
const deferredPrompt = ref<BeforeInstallPromptEvent | null>(null)
const installed = ref(false)
let listening = false

function detectStandalone(): boolean {
  if (typeof window === 'undefined') return false
  return (
    window.matchMedia('(display-mode: standalone)').matches ||
    (navigator as Navigator & { standalone?: boolean }).standalone === true
  )
}

function handleBeforeInstallPrompt(event: Event): void {
  event.preventDefault()
  deferredPrompt.value = event as BeforeInstallPromptEvent
}

function handleAppInstalled(): void {
  installed.value = true
  deferredPrompt.value = null
}

function startListening(): void {
  if (listening || typeof window === 'undefined') return
  listening = true
  installed.value = detectStandalone()
  window.addEventListener('beforeinstallprompt', handleBeforeInstallPrompt)
  window.addEventListener('appinstalled', handleAppInstalled)
  window.matchMedia('(display-mode: standalone)').addEventListener('change', (event) => {
    installed.value = event.matches
  })
}

export function usePwaInstall() {
  startListening()

  const canInstall = computed(() => deferredPrompt.value !== null && !installed.value)

  const promptInstall = async (): Promise<boolean> => {
    const prompt = deferredPrompt.value
    if (!prompt) return false
    await prompt.prompt()
    const choice = await prompt.userChoice
    deferredPrompt.value = null
    if (choice.outcome === 'accepted') {
      installed.value = true
      return true
    }
    return false
  }

  return {
    canInstall,
    installed: computed(() => installed.value),
    promptInstall
  }
}

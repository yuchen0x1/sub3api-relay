export const PENDING_REGISTRATION_STORAGE_KEY = 'register_data'

export interface PendingRegistrationSeed {
  email: string
  password?: string
  turnstile_token?: string
  [key: string]: unknown
}

export interface PendingRegistration {
  email: string
  password: string
  turnstileToken: string
}

let inMemoryRegistration: PendingRegistration | null = null

function readStoredEmail(): string {
  try {
    const raw = sessionStorage.getItem(PENDING_REGISTRATION_STORAGE_KEY)
    if (!raw) return ''
    const parsed = JSON.parse(raw) as { email?: unknown }
    const email = typeof parsed.email === 'string' ? parsed.email : ''
    if (raw !== JSON.stringify({ email })) {
      storeEmail(email)
    }
    return email
  } catch {
    try {
      sessionStorage.removeItem(PENDING_REGISTRATION_STORAGE_KEY)
    } catch {
      // Ignore browser storage failures.
    }
    return ''
  }
}

function storeEmail(email: string): void {
  try {
    sessionStorage.setItem(PENDING_REGISTRATION_STORAGE_KEY, JSON.stringify({ email }))
  } catch {
    // The form can still continue in the current page without browser storage.
  }
}

/** Store the password and one-time challenge only in the current page's memory. */
export function storePendingRegistration(data: PendingRegistrationSeed): void {
  const email = typeof data.email === 'string' ? data.email : ''
  inMemoryRegistration = {
    email,
    password: typeof data.password === 'string' ? data.password : '',
    turnstileToken: typeof data.turnstile_token === 'string' ? data.turnstile_token : '',
  }
  storeEmail(email)
}

/**
 * Load registration state for the verification route. Secrets are consumed from
 * module memory once; after a reload only the email metadata remains available.
 */
export function loadPendingRegistration(): PendingRegistration {
  const storedEmail = readStoredEmail()
  const memory = inMemoryRegistration
  inMemoryRegistration = null
  const email = storedEmail || memory?.email || ''
  const hasMatchingMemory = !!memory && memory.email === email
  return {
    email,
    password: hasMatchingMemory ? memory.password : '',
    turnstileToken: hasMatchingMemory ? memory.turnstileToken : '',
  }
}

/** Read email metadata so returning to registration can prefill the form. */
export function readPendingRegistrationEmail(): string {
  return readStoredEmail() || inMemoryRegistration?.email || ''
}

/** Clear transient secrets and optionally retain only non-sensitive email metadata. */
export function clearPendingRegistration(preserveEmail?: string): void {
  inMemoryRegistration = null
  try {
    if (preserveEmail) {
      storeEmail(preserveEmail)
    } else {
      sessionStorage.removeItem(PENDING_REGISTRATION_STORAGE_KEY)
    }
  } catch {
    // Ignore browser storage failures.
  }
}

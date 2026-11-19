import { beforeEach, describe, expect, it } from 'vitest'
import {
  clearPendingRegistration,
  loadPendingRegistration,
  PENDING_REGISTRATION_STORAGE_KEY,
  readPendingRegistrationEmail,
  storePendingRegistration,
} from '@/utils/pendingRegistration'

describe('pendingRegistration', () => {
  beforeEach(() => {
    clearPendingRegistration()
    sessionStorage.clear()
  })

  it('persists only email metadata while preserving secrets for the current handoff', () => {
    storePendingRegistration({
      email: 'person@example.com',
      password: 'registration-password',
      turnstile_token: 'one-time-proof',
    })

    expect(JSON.parse(sessionStorage.getItem(PENDING_REGISTRATION_STORAGE_KEY) || '{}')).toEqual({
      email: 'person@example.com',
    })
    expect(sessionStorage.getItem(PENDING_REGISTRATION_STORAGE_KEY)).not.toContain('registration-password')
    expect(sessionStorage.getItem(PENDING_REGISTRATION_STORAGE_KEY)).not.toContain('one-time-proof')
    expect(loadPendingRegistration()).toEqual({
      email: 'person@example.com',
      password: 'registration-password',
      turnstileToken: 'one-time-proof',
    })
  })

  it('after a page reload keeps the email but requires password and challenge re-entry', () => {
    storePendingRegistration({
      email: 'person@example.com',
      password: 'registration-password',
      turnstile_token: 'one-time-proof',
    })

    // A reload discards module memory but leaves only the non-sensitive email metadata.
    clearPendingRegistration('person@example.com')

    expect(readPendingRegistrationEmail()).toBe('person@example.com')
    expect(loadPendingRegistration()).toEqual({
      email: 'person@example.com',
      password: '',
      turnstileToken: '',
    })
    expect(sessionStorage.getItem(PENDING_REGISTRATION_STORAGE_KEY)).toBe(
      JSON.stringify({ email: 'person@example.com' })
    )
  })

  it('scrubs secrets from registration metadata written by an older frontend', () => {
    sessionStorage.setItem(PENDING_REGISTRATION_STORAGE_KEY, JSON.stringify({
      email: 'person@example.com',
      password: 'legacy-password',
      turnstile_token: 'legacy-proof',
    }))

    expect(readPendingRegistrationEmail()).toBe('person@example.com')
    expect(sessionStorage.getItem(PENDING_REGISTRATION_STORAGE_KEY)).toBe(
      JSON.stringify({ email: 'person@example.com' })
    )
  })
})

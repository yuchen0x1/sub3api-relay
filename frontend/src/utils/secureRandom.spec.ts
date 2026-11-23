import { describe, expect, it } from 'vitest'
import { secureRandomString, secureRandomId } from './secureRandom'

describe('secureRandomString', () => {
  it('returns the requested length using only the supplied alphabet', () => {
    const value = secureRandomString(128, 'abc123')

    expect(value).toHaveLength(128)
    expect(value).toMatch(/^[abc123]+$/)
  })

  it('validates its inputs', () => {
    expect(() => secureRandomString(-1, 'abc')).toThrow(RangeError)
    expect(() => secureRandomString(1.5, 'abc')).toThrow(RangeError)
    expect(() => secureRandomString(1, '')).toThrow(RangeError)
  })

  it('creates an id from the platform secure random source', () => {
    expect(secureRandomId()).toMatch(/^[0-9a-f-]{32,36}$/)
  })
})

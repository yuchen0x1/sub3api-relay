const RANDOM_BUFFER_SIZE = 32
const HEX_ALPHABET = '0123456789abcdef'

export function secureRandomString(length: number, alphabet: string): string {
  if (!Number.isInteger(length) || length < 0) {
    throw new RangeError('length must be a non-negative integer')
  }
  if (!alphabet) {
    throw new RangeError('alphabet must not be empty')
  }

  const cryptoSource = globalThis.crypto
  if (!cryptoSource?.getRandomValues) {
    throw new Error('A secure random number generator is required')
  }

  const result: string[] = []
  const limit = 256 - (256 % alphabet.length)
  const buffer = new Uint8Array(RANDOM_BUFFER_SIZE)

  while (result.length < length) {
    cryptoSource.getRandomValues(buffer)
    for (const value of buffer) {
      if (value >= limit) continue
      result.push(alphabet[value % alphabet.length])
      if (result.length === length) break
    }
  }

  return result.join('')
}

export function secureRandomId(): string {
  const randomUUID = globalThis.crypto?.randomUUID
  if (randomUUID) return randomUUID.call(globalThis.crypto)
  return secureRandomString(32, HEX_ALPHABET)
}

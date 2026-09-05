import { describe, expect, it } from 'vitest'
import { shouldReportHighTTFT } from '../ttftDiagnostics'

describe('shouldReportHighTTFT', () => {
  it('uses the configured threshold', () => {
    expect(shouldReportHighTTFT(5_000, 10_000, 20)).toBe(false)
    expect(shouldReportHighTTFT(10_000, 10_000, 20)).toBe(true)
  })

  it('requires recorded TTFT samples', () => {
    expect(shouldReportHighTTFT(20_000, 500, 0)).toBe(false)
  })
})

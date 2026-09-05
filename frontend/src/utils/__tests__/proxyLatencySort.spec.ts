import { describe, expect, it } from 'vitest'

import type { Proxy } from '@/types'
import { sortProxiesByLatency } from '../proxyLatencySort'

const proxy = (id: number, latency_ms?: number, latency_status?: Proxy['latency_status']): Proxy => ({
  id,
  name: `Proxy ${id}`,
  protocol: 'socks5',
  host: '127.0.0.1',
  port: 1080 + id,
  username: null,
  tls_insecure: false,
  status: 'active',
  latency_ms,
  latency_status,
  expires_at: null,
  fallback_mode: 'none',
  expiry_warn_days: 7,
  created_at: '2026-08-19T00:00:00Z',
  updated_at: '2026-08-19T00:00:00Z'
})

describe('sortProxiesByLatency', () => {
  const rows = [
    proxy(1, 700, 'success'),
    proxy(2, undefined, 'failed'),
    proxy(3, 120, 'success'),
    proxy(4),
    proxy(5, 350, 'success'),
    proxy(6, Number.NaN, 'success'),
    proxy(7, 50, 'failed')
  ]

  it('sorts measured latency ascending and keeps invalid states at the end', () => {
    expect(sortProxiesByLatency(rows, 'asc').map(row => row.id)).toEqual([
      3, 5, 1, 2, 4, 6, 7
    ])
  })

  it('sorts measured latency descending and still keeps invalid states at the end', () => {
    expect(sortProxiesByLatency(rows, 'desc').map(row => row.id)).toEqual([
      1, 5, 3, 2, 4, 6, 7
    ])
  })

  it('preserves source order for equal latency and invalid states', () => {
    const tiedRows = [proxy(10, 200, 'success'), proxy(11), proxy(12, 200, 'success'), proxy(13)]

    expect(sortProxiesByLatency(tiedRows, 'asc').map(row => row.id)).toEqual([10, 12, 11, 13])
  })
})

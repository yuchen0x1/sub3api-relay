import type { Proxy } from '@/types'

export type ProxyLatencySortOrder = 'asc' | 'desc'

const sortableLatency = (proxy: Proxy): number | null => {
  if (proxy.latency_status === 'failed') return null
  return typeof proxy.latency_ms === 'number' && Number.isFinite(proxy.latency_ms)
    ? proxy.latency_ms
    : null
}

export const sortProxiesByLatency = (
  proxies: Proxy[],
  order: ProxyLatencySortOrder
): Proxy[] => {
  const direction = order === 'asc' ? 1 : -1

  return proxies
    .map((proxy, index) => ({ proxy, index, latency: sortableLatency(proxy) }))
    .sort((left, right) => {
      const leftHasLatency = left.latency !== null
      const rightHasLatency = right.latency !== null

      // Failed, untested, and empty latency values stay after measured values
      // in both directions. Their original order is preserved.
      if (leftHasLatency !== rightHasLatency) return leftHasLatency ? -1 : 1
      if (left.latency !== null && right.latency !== null && left.latency !== right.latency) {
        return (left.latency - right.latency) * direction
      }
      return left.index - right.index
    })
    .map(({ proxy }) => proxy)
}

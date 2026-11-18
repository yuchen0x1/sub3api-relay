import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { RegisterSWOptions } from 'vite-plugin-pwa/types'

async function start() {
  const { registerWebUpdate, useWebUpdate } = await import('../useWebUpdate')
  const activate = vi.fn().mockResolvedValue(undefined)
  let worker!: RegisterSWOptions
  registerWebUpdate((options) => { worker = options; return activate })
  return { worker, activate, ...useWebUpdate() }
}

describe('Web update consent', () => {
  beforeEach(() => vi.resetModules())

  it('waits for consent before activating a waiting worker', async () => {
    const app = await start()
    expect(app.available.value).toBe(false)
    app.worker.onNeedRefresh?.()
    expect(app.available.value).toBe(true)
    expect(app.activate).not.toHaveBeenCalled()
    await app.refresh()
    expect(app.activate).toHaveBeenCalledOnce()
  })

  it('keeps a dismissed notice hidden while the user continues editing', async () => {
    const app = await start()
    app.worker.onNeedRefresh?.()
    app.later()
    app.worker.onNeedRefresh?.()
    app.worker.onNeedReload?.()
    expect(app.available.value).toBe(false)
    expect(app.activate).not.toHaveBeenCalled()
  })
})

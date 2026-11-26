import { afterEach, describe, expect, it, vi } from "vitest"
import { getUpdateStorage, readUpdateInterval, saveUpdateInterval } from "../preferences"

const KEY = "sub3api.desktop.update-check-hours"

describe("update check interval preference", () => {
  afterEach(() => vi.unstubAllGlobals())

  it("handles browsers that deny access to the storage object itself", () => {
    vi.stubGlobal("window", Object.defineProperty({}, "localStorage", {
      get() { throw new Error("storage denied") },
    }))
    expect(getUpdateStorage()).toBeUndefined()
    expect(readUpdateInterval(getUpdateStorage())).toBe(6)
    expect(() => saveUpdateInterval(getUpdateStorage(), 24)).not.toThrow()
  })

  it("uses the available browser storage", () => {
    const storage = { getItem: vi.fn(), setItem: vi.fn() }
    vi.stubGlobal("window", { localStorage: storage })
    expect(getUpdateStorage()).toBe(storage)
  })

  it("defaults to six hours when no preference has been saved", () => {
    expect(readUpdateInterval({ getItem: () => null })).toBe(6)
    expect(readUpdateInterval(undefined)).toBe(6)
  })

  it("restores only supported intervals and falls back for malformed values", () => {
    const getItem = vi.fn((key: string) => key === KEY ? "24" : null)
    expect(readUpdateInterval({ getItem })).toBe(24)
    expect(readUpdateInterval({ getItem: () => "1" })).toBe(6)
    expect(readUpdateInterval({ getItem: () => "" })).toBe(6)
  })

  it("keeps the default when persistent storage throws", () => {
    expect(readUpdateInterval({ getItem: () => { throw new Error("storage disabled") } })).toBe(6)
  })

  it("saves supported values without failing when storage is unavailable", () => {
    const setItem = vi.fn()
    saveUpdateInterval({ setItem }, 0)
    expect(setItem).toHaveBeenCalledWith(KEY, "0")
    saveUpdateInterval({ setItem }, 5)
    expect(setItem).toHaveBeenCalledTimes(1)
    expect(() => saveUpdateInterval({ setItem: () => { throw new Error("storage disabled") } }, 24)).not.toThrow()
  })
})

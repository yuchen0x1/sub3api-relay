import { beforeEach, describe, expect, it, vi } from "vitest"
import { apiClient, ApiError, createApiClient } from "../../../lib/http/apiClient"
import {
  createStaticProxyExit,
  deleteStaticProxyExit,
  getStaticProxyCapacitySummary,
  listStaticProxyExits,
  toggleStaticProxyExit,
  updateStaticProxyExit,
} from "../api"

vi.mock("../../../lib/http/apiClient", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../../lib/http/apiClient")>()
  return { ...actual, apiClient: { get: vi.fn(), post: vi.fn(), put: vi.fn(), del: vi.fn() } }
})

const mockedGet = apiClient.get as ReturnType<typeof vi.fn>
const mockedPost = apiClient.post as ReturnType<typeof vi.fn>
const mockedPut = apiClient.put as ReturnType<typeof vi.fn>
const mockedDel = apiClient.del as ReturnType<typeof vi.fn>

function okEnvelope(data: unknown) {
  return { code: 0, data }
}

describe("static exits api calls", () => {
  beforeEach(() => {
    mockedGet.mockReset()
    mockedPost.mockReset()
    mockedPut.mockReset()
    mockedDel.mockReset()
  })

  it("lists exits with pagination and position ordering", async () => {
    mockedGet.mockResolvedValue(okEnvelope({ items: [{ id: 11, account_id: 42, proxy_id: 101, name: "a", position: 1, enabled: true, capacity: 10, health_status: "healthy" }], total: 1, page: 1, pages: 1 }))

    const snapshot = await listStaticProxyExits({ accountId: "42" })

    expect(mockedGet).toHaveBeenCalledWith("/api/v1/admin/static-proxy-exits", {
      query: { page: 1, page_size: 100, sort_by: "position", sort_order: "asc", account_id: "42" },
      signal: undefined,
    })
    expect(snapshot.exits[0]).toMatchObject({ id: "11", accountId: "42", enabled: true })
  })

  it("fetches the capacity summary, optionally scoped to an account", async () => {
    mockedGet.mockResolvedValue(okEnvelope({ assignment_count: 10, enabled_count: 8, capacity_per_assignment: 10, enabled_capacity: 80, total_capacity: 100 }))

    const summary = await getStaticProxyCapacitySummary({ accountId: "42" })

    expect(mockedGet).toHaveBeenCalledWith("/api/v1/admin/static-proxy-exits/capacity", {
      query: { account_id: "42" },
      signal: undefined,
    })
    expect(summary).toMatchObject({ enabledCount: 8, enabledCapacity: 80 })
  })

  it("creates an exit from camelCase input via the snake_case payload", async () => {
    mockedPost.mockResolvedValue(okEnvelope({ id: 13, account_id: 42, proxy_id: 101, name: "x", position: 0, enabled: true, capacity: 10, health_status: "unknown" }))

    const exit = await createStaticProxyExit({ accountId: "42", proxyId: "101", name: "x" })

    expect(mockedPost).toHaveBeenCalledWith("/api/v1/admin/static-proxy-exits", {
      body: { account_id: "42", proxy_id: "101", name: "x", position: 0, enabled: true },
      signal: undefined,
    })
    expect(exit).toMatchObject({ id: "13", accountId: "42", proxyId: "101" })
  })

  it("updates, toggles, and deletes exits by id", async () => {
    const exitPayload = okEnvelope({ id: 13, account_id: 42, proxy_id: 101, name: "renamed", position: 2, enabled: false, capacity: 10, health_status: "healthy" })
    mockedPut.mockResolvedValue(exitPayload)
    mockedPost.mockResolvedValue(exitPayload)
    mockedDel.mockResolvedValue(okEnvelope({ message: "deleted" }))

    await updateStaticProxyExit("13", { name: "renamed", position: 2 })
    expect(mockedPut).toHaveBeenCalledWith("/api/v1/admin/static-proxy-exits/13", {
      body: { account_id: undefined, proxy_id: undefined, name: "renamed", position: 2, enabled: undefined },
      signal: undefined,
    })

    await toggleStaticProxyExit("13", false)
    expect(mockedPost).toHaveBeenCalledWith("/api/v1/admin/static-proxy-exits/13/toggle", {
      body: { enabled: false },
      signal: undefined,
    })

    await deleteStaticProxyExit("13")
    expect(mockedDel).toHaveBeenCalledWith("/api/v1/admin/static-proxy-exits/13", { signal: undefined })
  })
})

describe("apiClient transport", () => {
  it("sends JSON bodies, auth headers, and unwraps the response envelope", async () => {
    const { authTokenStore } = await import("../../../lib/auth/tokenStore")
    await authTokenStore.set({ token: "secret-token", scheme: "bearer" })

    const fetchImpl = vi.fn().mockResolvedValue(new Response(JSON.stringify(okEnvelope({ id: 1 })), { status: 200 }))
    const client = createApiClient({ baseUrl: "http://test.local/", fetchImpl: fetchImpl as unknown as typeof fetch })

    const data = await client.post("/api/v1/admin/static-proxy-exits", { body: { enabled: true } })

    expect(data).toEqual({ id: 1 })
    const [url, init] = fetchImpl.mock.calls[0]
    expect(url.toString()).toBe("http://test.local/api/v1/admin/static-proxy-exits")
    expect(init.method).toBe("POST")
    expect(init.headers["Content-Type"]).toBe("application/json")
    expect(init.headers.Authorization).toBe("Bearer secret-token")
    expect(JSON.parse(init.body)).toEqual({ enabled: true })

    await authTokenStore.clear()
  })

  it("throws ApiError with the server message on failure", async () => {
    const fetchImpl = vi.fn().mockResolvedValue(new Response(JSON.stringify({ message: "proxy already bound" }), { status: 409 }))
    const client = createApiClient({ baseUrl: "http://test.local", fetchImpl: fetchImpl as unknown as typeof fetch })

    const failure = await client.post("/api/v1/admin/static-proxy-exits", { body: {} }).catch((error: unknown) => error)
    expect(failure).toBeInstanceOf(ApiError)
    expect((failure as ApiError).status).toBe(409)
    expect((failure as ApiError).message).toBe("proxy already bound")
  })
})

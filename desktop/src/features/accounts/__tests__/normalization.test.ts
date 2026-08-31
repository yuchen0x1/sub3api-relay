import { normalizeAccountsResponse } from "../normalization"

describe("normalizeAccountsResponse", () => {
  it("normalizes the Sub3API envelope and account capacity fields", () => {
    const snapshot = normalizeAccountsResponse({
      code: 0,
      data: {
        items: [{
          id: 41,
          name: "alpha@example.com",
          platform: "openai",
          type: "apikey",
          status: "active",
          concurrency: 10,
          current_concurrency: 3,
          per_static_ip_capacity: [
            { ip: "203.0.113.10", limit: 4, used: 1 },
            { ip: "203.0.113.11", max: 2, current: 2 },
          ],
        }],
        total: 1,
        page: 1,
        pages: 1,
      },
    })

    expect(snapshot.total).toBe(1)
    expect(snapshot.accounts[0]).toMatchObject({ id: "41", name: "alpha@example.com", platform: "openai" })
    expect(snapshot.accounts[0].capacity.aggregate).toMatchObject({ used: 3, limit: 10, available: 7 })
    expect(snapshot.accounts[0].capacity.perStaticIp).toEqual([
      { ip: "203.0.113.10", capacity: { used: 1, limit: 4, available: 3, utilization: 0.25 } },
      { ip: "203.0.113.11", capacity: { used: 2, limit: 2, available: 0, utilization: 1 } },
    ])
  })

  it("supports keyed static-IP capacity and omits unavailable capacity", () => {
    const snapshot = normalizeAccountsResponse({
      accounts: [{
        account_id: "edge-7",
        email: "edge@example.com",
        provider: "anthropic",
        status: "inactive",
        static_ip_capacity: {
          "198.51.100.4": { used: 2, capacity: 5 },
        },
      }],
    })

    expect(snapshot.accounts[0].capacity.aggregate).toBeNull()
    expect(snapshot.accounts[0].capacity.perStaticIp[0]).toMatchObject({ ip: "198.51.100.4" })
    expect(snapshot.accounts[0].capacity.perStaticIp[0].capacity.available).toBe(3)
  })

  it("handles an empty response as an explicit empty state", () => {
    expect(normalizeAccountsResponse({ code: 0, data: { items: [], total: 0 } })).toEqual({ accounts: [], total: 0, page: null, pages: null })
  })
})

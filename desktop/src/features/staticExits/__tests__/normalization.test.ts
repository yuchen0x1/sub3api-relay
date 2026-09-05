import {
  normalizeAccountOptions,
  normalizeProxyOptions,
  normalizeStaticProxyCapacitySummary,
  normalizeStaticProxyExitsResponse,
} from "../normalization"

describe("normalizeStaticProxyExitsResponse", () => {
  it("normalizes the envelope, pagination, and nested proxy metadata", () => {
    const snapshot = normalizeStaticProxyExitsResponse({
      code: 0,
      data: {
        items: [
          {
            id: 11,
            account_id: 42,
            proxy_id: 101,
            name: "us-east exit 1",
            position: 1,
            enabled: true,
            capacity: 10,
            health_status: "healthy",
            failure_count: 0,
            proxy: { id: 101, name: "us-east", protocol: "http", host: "203.0.113.10", port: 8080, status: "active" },
          },
          {
            id: 12,
            account_id: 42,
            proxy_id: 102,
            name: "",
            position: 2,
            enabled: false,
            capacity: 10,
            health_status: "unhealthy",
            failure_count: 3,
            last_failure_reason: "connect timeout",
          },
        ],
        total: 2,
        page: 1,
        pages: 1,
      },
    })

    expect(snapshot.total).toBe(2)
    expect(snapshot.exits).toEqual([
      {
        id: "11",
        accountId: "42",
        proxyId: "101",
        name: "us-east exit 1",
        position: 1,
        enabled: true,
        capacity: 10,
        healthStatus: "healthy",
        failureCount: 0,
        lastFailureReason: "",
        proxy: { id: "101", name: "us-east", protocol: "http", host: "203.0.113.10", port: 8080, status: "active" },
      },
      {
        id: "12",
        accountId: "42",
        proxyId: "102",
        name: "Exit 12",
        position: 2,
        enabled: false,
        capacity: 10,
        healthStatus: "unhealthy",
        failureCount: 3,
        lastFailureReason: "connect timeout",
        proxy: null,
      },
    ])
  })

  it("supports bare arrays and skips malformed rows", () => {
    const snapshot = normalizeStaticProxyExitsResponse([{ id: 1, enabled: "yes" }, null, { account_id: 7 }])
    expect(snapshot.exits).toHaveLength(1)
    expect(snapshot.exits[0]).toMatchObject({ id: "1", enabled: false, healthStatus: "unknown" })
  })
})

describe("normalizeStaticProxyCapacitySummary", () => {
  it("reads the summary fields with numeric coercion", () => {
    expect(normalizeStaticProxyCapacitySummary({
      code: 0,
      data: {
        assignment_count: "10",
        enabled_count: 8,
        disabled_count: 2,
        capacity_per_assignment: 10,
        total_capacity: 100,
        enabled_capacity: 80,
      },
    })).toEqual({
      accountId: null,
      assignmentCount: 10,
      enabledCount: 8,
      disabledCount: 2,
      capacityPerAssignment: 10,
      totalCapacity: 100,
      enabledCapacity: 80,
    })
  })

  it("keeps a scoped account id and defaults missing numbers to zero", () => {
    expect(normalizeStaticProxyCapacitySummary({ account_id: 42 })).toMatchObject({ accountId: "42", assignmentCount: 0, enabledCapacity: 0 })
    expect(normalizeStaticProxyCapacitySummary(null)).toMatchObject({ accountId: null, totalCapacity: 0 })
  })
})

describe("picker option normalizers", () => {
  it("builds proxy labels with protocol, host, and port", () => {
    expect(normalizeProxyOptions({
      code: 0,
      data: {
        items: [
          { id: 101, name: "us-east", protocol: "socks5", host: "203.0.113.10", port: 1080 },
          { id: 102, name: "", protocol: "http", host: "198.51.100.4", port: "8080" },
          { name: "no id" },
        ],
      },
    })).toEqual([
      { id: "101", label: "us-east — socks5://203.0.113.10:1080", host: "203.0.113.10", port: 1080, protocol: "socks5" },
      { id: "102", label: "http://198.51.100.4:8080", host: "198.51.100.4", port: 8080, protocol: "http" },
    ])
  })

  it("builds account labels from name or email with an id fallback", () => {
    expect(normalizeAccountOptions({
      code: 0,
      data: {
        accounts: [
          { id: 41, email: "alpha@example.com" },
          { id: 42, name: "beta" },
          { id: 43 },
        ],
      },
    })).toEqual([
      { id: "41", label: "alpha@example.com" },
      { id: "42", label: "beta" },
      { id: "43", label: "Account 43" },
    ])
  })
})

export interface CapacityMetric {
  used: number
  limit: number | null
  available: number | null
  utilization: number | null
}

export interface StaticIpCapacity {
  ip: string
  capacity: CapacityMetric
}

export interface AccountCapacity {
  aggregate: CapacityMetric | null
  perStaticIp: StaticIpCapacity[]
}

export interface NormalizedAccount {
  id: string
  name: string
  platform: string
  type: string
  status: string
  capacity: AccountCapacity
}

export interface AccountsSnapshot {
  accounts: NormalizedAccount[]
  total: number | null
  page: number | null
  pages: number | null
}

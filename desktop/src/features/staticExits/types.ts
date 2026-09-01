export interface StaticProxyExitProxy {
  id: string
  name: string
  protocol: string
  host: string
  port: number | null
  status: string
}

export interface StaticProxyExit {
  id: string
  accountId: string
  proxyId: string
  name: string
  position: number
  enabled: boolean
  capacity: number
  healthStatus: string
  failureCount: number
  lastFailureReason: string
  proxy: StaticProxyExitProxy | null
}

export interface StaticProxyExitCapacitySummary {
  accountId: string | null
  assignmentCount: number
  enabledCount: number
  disabledCount: number
  capacityPerAssignment: number
  totalCapacity: number
  enabledCapacity: number
}

export interface StaticExitsSnapshot {
  exits: StaticProxyExit[]
  total: number | null
  page: number | null
  pages: number | null
}

export interface PickerOption {
  id: string
  label: string
}

export interface ProxyPickerOption extends PickerOption {
  host: string
  port: number | null
  protocol: string
}

export interface CreateStaticProxyExitInput {
  accountId: string
  proxyId: string
  name?: string
  position?: number
  enabled?: boolean
}

export interface UpdateStaticProxyExitInput {
  accountId?: string
  proxyId?: string
  name?: string
  position?: number
  enabled?: boolean
}

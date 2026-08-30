/**
 * Setup API endpoints
 */
import axios from 'axios'
import { buildGatewayUrl } from './url'

// Create a separate client for setup endpoints (not under /api/v1)
const setupClient = axios.create({
  baseURL: buildGatewayUrl('/').replace(/\/+$/, ''),
  withCredentials: true,
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json'
  }
})

// Remote owner setup is protected by the deployment's one-time token. The
// deployment helper may print a URL with #setup_token=...; read it only into
// memory, then remove it from the address bar so it is not persisted in
// browser history or sent as a Referer. The token is never stored in
// localStorage or returned by the status API.
let setupToken = ''
function captureSetupToken(): string {
  if (typeof window === 'undefined') {
    return setupToken
  }
  const raw = window.location.hash.replace(/^#/, '')
  const fragment = new URLSearchParams(raw)
  const token = fragment.get('setup_token') || ''
  if (token) {
    setupToken = token
    window.history.replaceState({}, document.title, `${window.location.pathname}${window.location.search}`)
  }
  return setupToken
}

captureSetupToken()

setupClient.interceptors.request.use((config) => {
  const token = captureSetupToken()
  if (token && config.headers) {
    config.headers['X-Setup-Token'] = token
  }
  return config
})

export interface SetupStatus {
  needs_setup: boolean
  step: string
  owner_setup?: boolean
  token_required?: boolean
  mode?: string
}

export interface DatabaseConfig {
  host: string
  port: number
  user: string
  password: string
  dbname: string
  sslmode: string
}

export interface RedisConfig {
  host: string
  port: number
  username: string
  password: string
  db: number
  enable_tls: boolean
}

export interface AdminConfig {
  email: string
  password: string
}

export interface ServerConfig {
  host: string
  port: number
  mode: string
}

export interface InstallRequest {
  database: DatabaseConfig
  redis: RedisConfig
  admin: AdminConfig
  server: ServerConfig
}

export interface InstallResponse {
  message: string
  restart: boolean
}

/**
 * Get setup status
 */
export async function getSetupStatus(): Promise<SetupStatus> {
  const response = await setupClient.get('/setup/status')
  return response.data.data
}

/**
 * Test database connection
 */
export async function testDatabase(config: DatabaseConfig): Promise<void> {
  await setupClient.post('/setup/test-db', config)
}

/**
 * Test Redis connection
 */
export async function testRedis(config: RedisConfig): Promise<void> {
  await setupClient.post('/setup/test-redis', config)
}

/**
 * Perform installation
 */
export async function install(config: InstallRequest): Promise<InstallResponse> {
  const response = await setupClient.post('/setup/install', config)
  return response.data.data
}

/** Complete the automatic deployment's site-owner bootstrap. */
export async function completeAdminSetup(admin: AdminConfig): Promise<InstallResponse> {
  const response = await setupClient.post('/setup/admin', admin)
  return response.data.data
}

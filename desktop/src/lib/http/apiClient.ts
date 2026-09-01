import { apiBaseUrl } from "../config"
import { authTokenStore } from "../auth/tokenStore"

export class ApiError extends Error {
  constructor(
    message: string,
    public readonly status: number,
    public readonly body: unknown = null,
  ) {
    super(message)
    this.name = "ApiError"
  }
}

export interface ApiClientOptions {
  baseUrl?: string
  fetchImpl?: typeof fetch
}

export interface GetOptions {
  query?: Record<string, string | number | boolean | undefined>
  signal?: AbortSignal
}

export interface MutationOptions extends GetOptions {
  body?: unknown
}

function joinUrl(baseUrl: string, path: string): string {
  return `${baseUrl.replace(/\/+$/, "")}/${path.replace(/^\/+/, "")}`
}

function errorMessage(body: unknown, status: number): string {
  if (typeof body === "object" && body !== null) {
    const record = body as Record<string, unknown>
    for (const key of ["message", "detail", "error"]) {
      if (typeof record[key] === "string" && record[key]) return record[key] as string
    }
    if (typeof record.data === "object" && record.data !== null) {
      const nested = record.data as Record<string, unknown>
      if (typeof nested.message === "string") return nested.message
    }
  }
  return `Request failed with status ${status}.`
}

async function readBody(response: Response): Promise<unknown> {
  const text = await response.text()
  if (!text) return null
  try {
    return JSON.parse(text) as unknown
  } catch {
    return text
  }
}

function unwrapEnvelope<T>(body: unknown): T {
  if (typeof body === "object" && body !== null && "code" in body) {
    const envelope = body as { code?: unknown; data?: unknown }
    if (envelope.code !== 0 && envelope.code !== undefined) {
      throw new ApiError(errorMessage(body, 200), 200, body)
    }
    return envelope.data as T
  }
  return body as T
}

export function createApiClient(options: ApiClientOptions = {}) {
  const baseUrl = options.baseUrl ?? apiBaseUrl
  const fetchImpl = options.fetchImpl ?? fetch

  async function request<T>(method: string, path: string, options: MutationOptions = {}): Promise<T> {
    const url = new URL(joinUrl(baseUrl, path))
    Object.entries(options.query ?? {}).forEach(([key, value]) => {
      if (value !== undefined) url.searchParams.set(key, String(value))
    })

    const session = await authTokenStore.get()
    const headers: Record<string, string> = {
      Accept: "application/json",
      "X-Admin-UI-Request": "1",
    }
    if (options.body !== undefined) headers["Content-Type"] = "application/json"
    if (session?.token) {
      if (session.scheme === "bearer") headers.Authorization = `Bearer ${session.token}`
      else headers["x-api-key"] = session.token
    }

    const response = await fetchImpl(url, {
      method,
      headers,
      body: options.body === undefined ? undefined : JSON.stringify(options.body),
      signal: options.signal,
    })
    const body = await readBody(response)
    if (!response.ok) throw new ApiError(errorMessage(body, response.status), response.status, body)
    return unwrapEnvelope<T>(body)
  }

  return {
    get: <T>(path: string, getOptions: GetOptions = {}) => request<T>("GET", path, getOptions),
    post: <T>(path: string, mutationOptions: MutationOptions = {}) => request<T>("POST", path, mutationOptions),
    put: <T>(path: string, mutationOptions: MutationOptions = {}) => request<T>("PUT", path, mutationOptions),
    del: <T>(path: string, mutationOptions: MutationOptions = {}) => request<T>("DELETE", path, mutationOptions),
  }
}

export const apiClient = createApiClient()

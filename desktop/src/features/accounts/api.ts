import { apiClient } from "../../lib/http/apiClient"
import { normalizeAccountsResponse } from "./normalization"
import { AccountsSnapshot } from "./types"

export interface ListAccountsOptions {
  page?: number
  pageSize?: number
  signal?: AbortSignal
}

export async function listAccounts(options: ListAccountsOptions = {}): Promise<AccountsSnapshot> {
  const payload = await apiClient.get<unknown>("/api/v1/admin/accounts", {
    query: {
      page: options.page ?? 1,
      page_size: options.pageSize ?? 100,
      sort_by: "created_at",
      sort_order: "desc",
    },
    signal: options.signal,
  })
  return normalizeAccountsResponse(payload)
}

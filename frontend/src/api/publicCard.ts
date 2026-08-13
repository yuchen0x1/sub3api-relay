import { apiClient } from './client'

export interface PublicCardBalance {
  key_preview: string
  status: string
  quota: number
  quota_used: number
  quota_remaining: number
  expires_at?: string
  last_used_at?: string
  created_at: string
  updated_at: string
}

export interface PublicCardRedeemResult extends PublicCardBalance {
  api_key: string
}

export const publicCardAPI = {
  async redeem(code: string): Promise<PublicCardRedeemResult> {
    const response = await apiClient.post<PublicCardRedeemResult>('/public/cards/redeem', { code })
    return response.data
  },

  async query(apiKey: string): Promise<PublicCardBalance> {
    const response = await apiClient.post<PublicCardBalance>('/public/cards/query', { api_key: apiKey })
    return response.data
  },
}

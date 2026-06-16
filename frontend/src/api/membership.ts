import { apiClient } from './client'
import type { ApiKey, Group } from '@/types'

export interface MembershipLevel {
  id: number
  name: string
  code: string
  discount_rate: number
  enabled: boolean
  is_default: boolean
  sort_order: number
  description: string
  created_at: string
  updated_at: string
  groups?: Group[]
}

export interface MembershipManagedKey {
  id: number
  user_id: number
  group_id: number
  api_key_id: number
  membership_level_id: number
  status: 'active' | 'disabled'
  disabled_reason: string
  created_at: string
  updated_at: string
  group?: Group
  api_key?: ApiKey
}

export interface MembershipSummary {
  level: MembershipLevel | null
  starts_at: string
  expires_at: string | null
  groups: Group[]
  managed_keys: MembershipManagedKey[]
}

export async function getCurrent(): Promise<MembershipSummary> {
  const { data } = await apiClient.get<MembershipSummary>('/membership')
  return data
}

export const membershipAPI = {
  getCurrent
}

export default membershipAPI

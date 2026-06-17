import { apiClient } from './client'
import type { Group } from '@/types'

export interface MembershipGroup extends Group {
  effective_rate_multiplier?: number
}

export interface MembershipLevel {
  id: number
  name: string
  code: string
  color: string
  discount_rate: number
  enabled: boolean
  is_default: boolean
  sort_order: number
  description: string
  created_at: string
  updated_at: string
  groups?: MembershipGroup[]
}

export interface MembershipAPIKey {
  id: number
  user_id: number
  key_suffix?: string
  masked_key?: string
  name: string
  status: string
  group_id?: number | null
  created_at: string
  updated_at: string
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
  group?: MembershipGroup
  api_key?: MembershipAPIKey
}

export interface MembershipSummary {
  level: MembershipLevel | null
  starts_at: string
  expires_at: string | null
  levels: MembershipLevel[]
  groups: MembershipGroup[]
  managed_keys: MembershipManagedKey[]
}

export interface MembershipAssignmentUser {
  id: number
  email: string
  username: string
  status: string
}

export interface MembershipAssignment {
  id: number
  user_id: number
  membership_level_id: number
  starts_at: string
  expires_at: string | null
  status: string
  source: string
  created_at: string
  updated_at: string
  level?: MembershipLevel
  user?: MembershipAssignmentUser
}

export async function getCurrent(): Promise<MembershipSummary> {
  const { data } = await apiClient.get<MembershipSummary>('/membership')
  return data
}

export const membershipAPI = {
  getCurrent
}

export default membershipAPI

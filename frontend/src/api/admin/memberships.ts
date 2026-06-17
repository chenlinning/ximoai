import { apiClient } from '../client'
import type { MembershipAssignment, MembershipLevel, MembershipSummary } from '../membership'

export interface MembershipLevelRequest {
  discount_rate?: number
  group_ids?: number[]
}

export interface AssignMembershipRequest {
  membership_level_id: number
  expires_at?: string | null
  source?: 'admin' | 'system' | 'purchase' | string
}

export async function list(includeDisabled = true): Promise<MembershipLevel[]> {
  const { data } = await apiClient.get<MembershipLevel[]>('/admin/memberships', {
    params: { include_disabled: includeDisabled }
  })
  return data
}

export async function getById(id: number): Promise<MembershipLevel> {
  const { data } = await apiClient.get<MembershipLevel>(`/admin/memberships/${id}`)
  return data
}

export async function create(payload: MembershipLevelRequest): Promise<MembershipLevel> {
  const { data } = await apiClient.post<MembershipLevel>('/admin/memberships', payload)
  return data
}

export async function update(id: number, payload: MembershipLevelRequest): Promise<MembershipLevel> {
  const { data } = await apiClient.put<MembershipLevel>(`/admin/memberships/${id}`, payload)
  return data
}

export async function listAssignments(limit = 200): Promise<MembershipAssignment[]> {
  const { data } = await apiClient.get<MembershipAssignment[]>('/admin/memberships/assignments', {
    params: { limit }
  })
  return data
}

export async function disable(id: number): Promise<{ message: string }> {
  const { data } = await apiClient.delete<{ message: string }>(`/admin/memberships/${id}`)
  return data
}

export async function sync(id: number): Promise<{ message: string }> {
  const { data } = await apiClient.post<{ message: string }>(`/admin/memberships/${id}/sync`)
  return data
}

export async function assignUser(
  userId: number,
  payload: AssignMembershipRequest
): Promise<MembershipSummary> {
  const { data } = await apiClient.post<MembershipSummary>(`/admin/users/${userId}/membership`, payload)
  return data
}

export const membershipsAPI = {
  list,
  getById,
  create,
  update,
  listAssignments,
  disable,
  sync,
  assignUser
}

export default membershipsAPI

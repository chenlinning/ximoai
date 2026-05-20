import { apiClient } from '../client'
import type { Platform } from '@/types'

export interface PlatformRequest {
  slug: string
  display_name: string
  protocol: string
  base_url: string
  auth_modes?: string[]
  capabilities?: string[]
  color: string
  enabled?: boolean
}

export async function list(includeDisabled = false): Promise<Platform[]> {
  const { data } = await apiClient.get<Platform[]>('/admin/platforms', {
    params: includeDisabled ? { include_disabled: true } : undefined,
  })
  return data
}

export async function create(payload: PlatformRequest): Promise<Platform> {
  const { data } = await apiClient.post<Platform>('/admin/platforms', payload)
  return data
}

export async function update(slug: string, payload: PlatformRequest): Promise<Platform> {
  const { data } = await apiClient.put<Platform>(`/admin/platforms/${encodeURIComponent(slug)}`, payload)
  return data
}

export async function remove(slug: string): Promise<{ message: string }> {
  const { data } = await apiClient.delete<{ message: string }>(`/admin/platforms/${encodeURIComponent(slug)}`)
  return data
}

export const platformsAPI = { list, create, update, remove }

export default platformsAPI

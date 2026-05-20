import { apiClient } from './client'
import type { Platform } from '@/types'

export async function list(includeDisabled = false): Promise<Platform[]> {
  const { data } = await apiClient.get<Platform[]>('/platforms', {
    params: includeDisabled ? { include_disabled: true } : undefined,
  })
  return data
}

export const platformsAPI = { list }

export default platformsAPI

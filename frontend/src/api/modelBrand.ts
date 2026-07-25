import { apiClient } from './client'

export type ModelBrandSource = 'automatic' | 'administrator'

export interface ModelBrandEditor {
  automatic_brand: string
  source: ModelBrandSource
}

export interface ModelBrandState {
  brand: string
  editor?: ModelBrandEditor | null
}

export interface ModelBrandTarget {
  platform: string
  model: string
}

export async function save(target: ModelBrandTarget, brand: string): Promise<ModelBrandState> {
  const { data } = await apiClient.put<ModelBrandState>('/admin/model-plaza/brand', {
    ...target,
    brand
  })
  return data
}

export async function reset(target: ModelBrandTarget): Promise<ModelBrandState> {
  const { data } = await apiClient.delete<ModelBrandState>('/admin/model-plaza/brand', { data: target })
  return data
}

export const modelBrandAPI = { save, reset }

export default modelBrandAPI

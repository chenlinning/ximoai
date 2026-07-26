import { apiClient } from './client'

export type ModelType = 'conversation' | 'embedding' | 'image' | 'video' | 'tts' | 'asr'
export type ModelInvocationMode = 'sync' | 'stream' | 'async' | 'bidirectional' | 'batch'
export type ModelReasoningLevel = 'none' | 'minimal' | 'low' | 'medium' | 'high' | 'xhigh' | 'max'

export interface ModelMetadataValues {
  brand: string
  types: ModelType[]
  invocation_modes: ModelInvocationMode[]
  reasoning_levels?: ModelReasoningLevel[]
  thinking_supported?: boolean
}

export interface ModelMetadataOverride {
  brand?: string
  types?: ModelType[]
  invocation_modes?: ModelInvocationMode[]
  reasoning_levels?: ModelReasoningLevel[]
  thinking_supported?: boolean
}

export interface ModelMetadataOptions {
  types: ModelType[]
  invocation_modes: ModelInvocationMode[]
  reasoning_levels?: ModelReasoningLevel[]
}

export interface ModelMetadataEditor {
  automatic: ModelMetadataValues
  override?: ModelMetadataOverride | null
  options: ModelMetadataOptions
}

export interface ModelMetadataState extends ModelMetadataValues {
  editor?: ModelMetadataEditor | null
}

export interface ModelMetadataTarget {
  platform: string
  model: string
}

export async function save(target: ModelMetadataTarget, override: ModelMetadataOverride): Promise<ModelMetadataOverride> {
  const { data } = await apiClient.put<ModelMetadataOverride>('/admin/model-plaza/metadata', {
    ...target,
    ...override
  })
  return data
}

export async function reset(target: ModelMetadataTarget): Promise<ModelMetadataOverride> {
  const { data } = await apiClient.delete<ModelMetadataOverride>('/admin/model-plaza/metadata', { data: target })
  return data
}

export const modelMetadataAPI = { save, reset }

export default modelMetadataAPI

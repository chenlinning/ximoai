import { apiClient } from './client'

export type ModelAPIDocsCategory = 'conversation' | 'image' | 'video' | 'tts' | 'asr'
export type ModelAPIDocsMode = 'sync' | 'stream' | 'async' | 'bidirectional'
export type ModelAPIDocsTransport = 'http' | 'websocket'
export type ModelAPIDocsDelivery = 'json' | 'sse' | 'binary' | 'websocket_frames'

export interface ModelAPIDocsWorkflowStep {
  id: string
  title: string
  method: string
  path: string
  content_type?: string
  request_example?: string
  response_example?: string
}

export interface ModelAPIDocsEndpointVariant {
  id: string
  label: string
  mode: ModelAPIDocsMode
  transport: ModelAPIDocsTransport
  delivery: ModelAPIDocsDelivery
  steps: ModelAPIDocsWorkflowStep[]
}

export interface ModelAPIDocsEndpointProfile {
  id: string
  category: ModelAPIDocsCategory
  protocol: string
  title: string
  description: string
  variants: ModelAPIDocsEndpointVariant[]
}

export interface ModelAPIDocsEndpointBinding {
  profile: string
  variants: string[]
}

export interface ModelAPIDocsCategoryBinding {
  category: ModelAPIDocsCategory
  endpoints: ModelAPIDocsEndpointBinding[]
}

export interface ModelAPIDocsBinding {
  platform: string
  protocol: string
  model: string
  categories: ModelAPIDocsCategoryBinding[]
}

export interface ModelAPIDocsEditor {
  automatic_binding: ModelAPIDocsBinding
  available_profiles: ModelAPIDocsEndpointProfile[]
}

export interface ModelAPIDocsResponse {
  platform: string
  protocol: string
  model: string
  source: 'automatic' | 'administrator'
  binding: ModelAPIDocsBinding
  profiles: ModelAPIDocsEndpointProfile[]
  editor?: ModelAPIDocsEditor | null
}

export interface ModelAPIDocsTarget {
  platform: string
  protocol: string
  model: string
}

export async function save(target: ModelAPIDocsTarget, categories: ModelAPIDocsCategoryBinding[]): Promise<ModelAPIDocsBinding> {
  const { data } = await apiClient.put<ModelAPIDocsBinding>('/admin/model-plaza/docs', {
    ...target,
    categories
  })
  return data
}

export async function reset(target: ModelAPIDocsTarget): Promise<void> {
  await apiClient.delete('/admin/model-plaza/docs', { data: target })
}

export const modelApiDocsAPI = { save, reset }

export default modelApiDocsAPI

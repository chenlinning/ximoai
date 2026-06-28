import { apiClient } from '../client'

export interface XimoDeskUpdateRelease {
  id?: string
  app_key?: string
  enabled?: boolean
  channel: 'stable' | 'beta' | string
  os: 'windows' | 'macos' | 'linux' | 'android' | 'ios' | string
  arch: 'x86_64' | 'aarch64' | 'universal' | string
  locale?: string
  package_type?: string
  version: string
  version_code?: number
  min_supported_version?: string
  min_supported_version_code?: number
  download_url: string
  notes: string
  sha256: string
  force: boolean
  file_name?: string
  file_size?: number
  uploaded_at?: string
  published_at?: string
}

export interface XimoAppUpdateApp {
  key: string
  name: string
  description?: string
  client_type?: string
  response_mode?: string
  enabled?: boolean
  hidden?: boolean
}

export interface XimoDeskUpdateConfig {
  enabled: boolean
  apps?: XimoAppUpdateApp[]
  releases: XimoDeskUpdateRelease[]
}

export async function get(): Promise<XimoDeskUpdateConfig> {
  const { data } = await apiClient.get<XimoDeskUpdateConfig>('/admin/ximoapp/update')
  return data
}

export async function update(payload: XimoDeskUpdateConfig): Promise<XimoDeskUpdateConfig> {
  const { data } = await apiClient.put<XimoDeskUpdateConfig>('/admin/ximoapp/update', payload)
  return data
}

export interface XimoDeskPackageUploadResponse {
  release: XimoDeskUpdateRelease
  config: XimoDeskUpdateConfig
}

export async function uploadPackage(payload: FormData): Promise<XimoDeskPackageUploadResponse> {
  const { data } = await apiClient.post<XimoDeskPackageUploadResponse>('/admin/ximoapp/update/packages', payload, {
    headers: { 'Content-Type': 'multipart/form-data' },
    timeout: 120000
  })
  return data
}

export async function deleteRelease(id: string): Promise<XimoDeskUpdateConfig> {
  const { data } = await apiClient.delete<XimoDeskUpdateConfig>(`/admin/ximoapp/update/releases/${encodeURIComponent(id)}`)
  return data
}

export async function deleteApp(appKey: string): Promise<XimoDeskUpdateConfig> {
  const { data } = await apiClient.delete<XimoDeskUpdateConfig>(`/admin/ximoapp/update/apps/${encodeURIComponent(appKey)}`)
  return data
}

export const ximodeskUpdateAPI = {
  get,
  update,
  uploadPackage,
  deleteRelease,
  deleteApp
}

export default ximodeskUpdateAPI

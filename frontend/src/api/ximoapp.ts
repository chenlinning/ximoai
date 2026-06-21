import { apiClient } from './client'

export interface XimoAppDownloadRelease {
  id?: string
  app_key?: string
  channel: string
  os: string
  arch: string
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

export interface XimoAppDownloadApp {
  key: string
  name: string
  description?: string
  client_type?: string
  releases: XimoAppDownloadRelease[]
}

export interface XimoAppDownloadCenterResponse {
  apps: XimoAppDownloadApp[]
}

export async function getDownloadCenter(): Promise<XimoAppDownloadCenterResponse> {
  const { data } = await apiClient.get<XimoAppDownloadCenterResponse>('/ximoapp/download-center')
  return data
}

export const ximoAppAPI = {
  getDownloadCenter
}

export default ximoAppAPI

import { apiClient } from './client'

export interface XimoAIHomeTab {
  id: string
  label: string
  url: string
  cover_url?: string
  enabled: boolean
  workbench_sso?: boolean
  diamond_only?: boolean
  sort_order: number
}

export type XimoAIHomeTabInput = Omit<XimoAIHomeTab, 'sort_order'> & { sort_order?: number }

export const ximoaiHomeAPI = {
  async list(): Promise<XimoAIHomeTab[]> {
    const { data } = await apiClient.get<XimoAIHomeTab[]>('/settings/ximoai-home-tabs')
    return data
  },

  async listAdmin(): Promise<XimoAIHomeTab[]> {
    const { data } = await apiClient.get<XimoAIHomeTab[]>('/admin/settings/ximoai-home-tabs')
    return data
  },

  async update(tabs: XimoAIHomeTabInput[]): Promise<XimoAIHomeTab[]> {
    const { data } = await apiClient.put<XimoAIHomeTab[]>('/admin/settings/ximoai-home-tabs', { tabs })
    return data
  }
}

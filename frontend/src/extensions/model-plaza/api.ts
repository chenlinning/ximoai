/**
 * User Channels API endpoints (non-admin)
 * 用户侧「可用渠道」聚合查询：渠道 + 用户可访问的分组 + 支持模型（含定价）。
 */

import { apiClient } from '@/api/client'
import type { BillingMode } from '@/constants/channel'
import type { ModelInvocationMode, ModelMetadataEditor, ModelReasoningLevel, ModelType } from '@/api/modelMetadata'
import type {
  UserAvailableGroup,
  UserPricingInterval,
  UserSupportedModelPricing as NativeUserSupportedModelPricing,
} from '@/api/channels'

export type { UserAvailableGroup, UserPricingInterval }

export interface UserSupportedModelPricing extends NativeUserSupportedModelPricing {
  billing_mode: BillingMode
}

export interface UserSupportedModel {
  name: string
  platform: string
  brand?: string
  metadata_editor?: ModelMetadataEditor | null
  pricing: UserSupportedModelPricing | null
  types: ModelType[]
  invocation_modes: ModelInvocationMode[]
  reasoning_levels?: ModelReasoningLevel[]
  thinking_supported?: boolean
}

/**
 * 渠道下单个平台的子视图：用户可访问的分组 + 该平台支持的模型。
 * 后端把一个渠道按平台聚合成 sections，前端可以把渠道名作为 row-group
 * 一次渲染，后面按 sections 顺序用 rowspan 铺开。
 */
export interface UserChannelPlatformSection {
  platform: string
  display_name: string
  color: string
  protocol: string
  groups: UserAvailableGroup[]
  supported_models: UserSupportedModel[]
}

export interface UserAvailableChannel {
  name: string
  description: string
  platforms: UserChannelPlatformSection[]
}

export async function getModelPlaza(options?: { signal?: AbortSignal }): Promise<UserAvailableChannel[]> {
  const { data } = await apiClient.get<UserAvailableChannel[]>('/channels/model-plaza', {
    signal: options?.signal
  })
  return data
}

export const userChannelsAPI = { getModelPlaza }

export default userChannelsAPI

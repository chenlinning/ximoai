import { computed, ref, watch, type ComputedRef, type Ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import { useAppStore } from '@/stores/app'

type MaybeReadonlyRef<T> = Ref<T> | ComputedRef<T>

interface UsePreviewUpstreamModelsSyncOptions {
  platform: MaybeReadonlyRef<string>
  accountType: MaybeReadonlyRef<string>
  platformEnabled: MaybeReadonlyRef<boolean>
  apiKey: Ref<string>
  baseUrl: Ref<string>
  baseUrlFallback: MaybeReadonlyRef<string>
  anthropicAPIKeyAuthScheme?: MaybeReadonlyRef<string>
}

const previewSyncPlatforms = new Set([
  'anthropic', 'openai', 'gemini', 'kimi', 'zhipu', 'deepseek',
  'grok-video', 'openai-audio', 'kling_audio',
])

export function usePreviewUpstreamModelsSync(options: UsePreviewUpstreamModelsSyncOptions) {
  const { t } = useI18n()
  const appStore = useAppStore()
  const previewSyncedUpstreamModels = ref<string[]>([])

  const canPreviewSyncUpstreamModels = computed(() =>
    options.accountType.value === 'apikey' &&
    options.platformEnabled.value &&
    previewSyncPlatforms.has(options.platform.value)
  )

  const clearPreviewSyncedUpstreamModels = () => {
    previewSyncedUpstreamModels.value = []
  }

  const previewSyncUpstreamModels = async (): Promise<string[]> => {
    const apiKey = options.apiKey.value.trim()
    if (!apiKey) {
      appStore.showError(t('admin.accounts.pleaseEnterApiKey'))
      return []
    }

    const baseUrl = options.baseUrl.value.trim() || options.baseUrlFallback.value
    const extra = options.platform.value === 'anthropic' && options.anthropicAPIKeyAuthScheme?.value === 'authorization_bearer'
      ? { anthropic_apikey_auth_scheme: 'authorization_bearer' }
      : undefined
    const result = await adminAPI.accounts.syncUpstreamModelsPreview({
      platform: options.platform.value,
      type: 'apikey',
      credentials: {
        base_url: baseUrl,
        api_key: apiKey
      },
      ...(extra ? { extra } : {})
    })
    previewSyncedUpstreamModels.value = result.models
    return result.models
  }

  watch([options.baseUrl, options.apiKey], clearPreviewSyncedUpstreamModels)

  return {
    canPreviewSyncUpstreamModels,
    clearPreviewSyncedUpstreamModels,
    previewSyncedUpstreamModels,
    previewSyncUpstreamModels
  }
}

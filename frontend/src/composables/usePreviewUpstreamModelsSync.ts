import { computed, ref, watch, type ComputedRef, type Ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import { useAppStore } from '@/stores/app'

type MaybeReadonlyRef<T> = Ref<T> | ComputedRef<T>

interface UsePreviewUpstreamModelsSyncOptions {
  platform: MaybeReadonlyRef<string>
  accountType: MaybeReadonlyRef<string>
  selectedProtocol: MaybeReadonlyRef<string>
  platformEnabled: MaybeReadonlyRef<boolean>
  apiKey: Ref<string>
  baseUrl: Ref<string>
  baseUrlFallback: MaybeReadonlyRef<string>
}

const previewSyncProtocols = new Set(['openai', 'openai_compatible', 'anthropic', 'gemini'])

export function usePreviewUpstreamModelsSync(options: UsePreviewUpstreamModelsSyncOptions) {
  const { t } = useI18n()
  const appStore = useAppStore()
  const previewSyncedUpstreamModels = ref<string[]>([])

  const canPreviewSyncUpstreamModels = computed(() =>
    options.accountType.value === 'apikey' &&
    options.platform.value !== 'antigravity' &&
    options.platformEnabled.value &&
    previewSyncProtocols.has(
      options.selectedProtocol.value ||
      (options.platform.value === 'openai' ? 'openai' : '')
    )
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
    const result = await adminAPI.accounts.syncUpstreamModelsPreview({
      platform: options.platform.value,
      type: 'apikey',
      credentials: {
        base_url: baseUrl,
        api_key: apiKey
      }
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

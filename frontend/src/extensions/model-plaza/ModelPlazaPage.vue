<template>
  <AppLayout>
    <div class="mx-auto flex w-full max-w-none flex-col gap-6">
      <div class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
        <div>
          <h1 class="page-title">{{ t('modelPlaza.title') }}</h1>
          <p class="page-description">{{ t('modelPlaza.description') }}</p>
        </div>

        <button
          type="button"
          class="btn btn-secondary btn-md"
          :disabled="loading"
          :title="t('modelPlaza.refresh')"
          @click="loadData"
        >
          <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
        </button>
      </div>

      <div class="relative w-full lg:max-w-md">
        <Icon
          name="search"
          size="md"
          class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 dark:text-gray-500"
        />
        <input
          v-model="searchQuery"
          type="text"
          :placeholder="t('modelPlaza.searchPlaceholder')"
          class="input pl-10"
        />
      </div>

      <div class="space-y-3">
        <div data-filter-level="brand" class="flex w-full min-w-0 items-center gap-3">
          <span class="w-16 flex-shrink-0 text-sm font-medium text-gray-600 dark:text-gray-300">
            {{ t('modelPlaza.brand') }}
          </span>
          <div class="flex min-w-0 gap-2 overflow-x-auto pb-1">
            <button
              v-for="brand in brandList"
              :key="brand"
            type="button"
              :data-model-brand="brand"
              class="inline-flex h-9 flex-shrink-0 items-center rounded-lg border px-3 text-sm font-medium transition-colors"
              :class="filterButtonClass(brand, activeBrand)"
              @click="activeBrand = brand"
            >
              {{ brand === allBrandsKey ? t('modelPlaza.allBrands') : brand }}
            </button>
          </div>
        </div>

        <div data-filter-level="type" class="flex w-full min-w-0 items-center gap-3">
          <span class="w-16 flex-shrink-0 text-sm font-medium text-gray-600 dark:text-gray-300">
            {{ t('modelPlaza.modelType') }}
          </span>
          <div class="flex min-w-0 gap-2 overflow-x-auto pb-1">
            <button
              v-for="type in typeFilterOptions"
              :key="type"
              type="button"
              :data-model-category="type"
              class="inline-flex h-9 flex-shrink-0 items-center rounded-lg border px-3 text-sm font-medium transition-colors"
              :class="filterButtonClass(type, activeType)"
              @click="activeType = type"
            >
              {{ type === allTypesKey ? t('modelPlaza.allTypes') : typeLabel(type as ModelType) }}
            </button>
          </div>
        </div>

        <div data-filter-level="mode" class="flex w-full min-w-0 items-center gap-3">
          <span class="w-16 flex-shrink-0 text-sm font-medium text-gray-600 dark:text-gray-300">
            {{ t('modelPlaza.invocationMode') }}
          </span>
          <div class="flex min-w-0 gap-2 overflow-x-auto pb-1">
            <button
              v-for="mode in modeFilterOptions"
              :key="mode"
              type="button"
              :data-model-mode="mode"
              class="inline-flex h-9 flex-shrink-0 items-center rounded-lg border px-3 text-sm font-medium transition-colors"
              :class="filterButtonClass(mode, activeMode)"
              @click="activeMode = mode"
            >
              {{ mode === allModesKey ? t('modelPlaza.allModes') : modeLabel(mode as ModelInvocationMode) }}
            </button>
          </div>
        </div>
      </div>

      <div v-if="loading" class="model-card-grid grid gap-4">
        <div v-for="idx in 6" :key="idx" class="card p-5">
          <div class="skeleton mb-4 h-5 w-2/3"></div>
          <div class="skeleton mb-6 h-4 w-24"></div>
          <div class="space-y-3">
            <div class="skeleton h-20 w-full"></div>
            <div class="skeleton h-16 w-full"></div>
          </div>
        </div>
      </div>

      <div v-else-if="error" class="card flex flex-col items-center justify-center gap-4 p-10 text-center">
        <Icon name="exclamationCircle" size="xl" class="text-red-500" />
        <p class="text-sm text-gray-600 dark:text-gray-300">{{ error }}</p>
        <button type="button" class="btn btn-primary btn-md" @click="loadData">
          {{ t('modelPlaza.retry') }}
        </button>
      </div>

      <div v-else-if="filteredModels.length === 0" class="card flex flex-col items-center justify-center gap-3 p-12 text-center">
        <Icon name="inbox" size="xl" class="text-gray-300 dark:text-dark-600" />
        <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('modelPlaza.noModels') }}</p>
      </div>

      <div
        v-else
        data-model-grid
        data-card-min-width="20rem"
        data-card-max-width="24rem"
        class="model-card-grid grid gap-4"
      >
        <article
          v-for="model in filteredModels"
          :key="model.key"
          :data-model-key="model.key"
          class="card group relative w-full max-w-96 overflow-hidden p-5"
        >
          <div
            class="absolute inset-x-0 top-0 h-1"
            :class="platformAccentBarClass(model.platform)"
            :style="platformAccentStyle(model.platform)"
          ></div>

          <div class="mb-4 flex items-start justify-between gap-3">
            <div class="min-w-0">
              <h2 class="break-words text-base font-semibold leading-6 text-gray-900 dark:text-white">
                {{ model.name }}
              </h2>
              <div class="mt-2 flex flex-wrap items-center gap-2">
                <span
                  :data-model-brand-chip="model.brand"
                  class="inline-flex items-center rounded-md bg-primary-100 px-2 py-0.5 text-xs font-medium text-primary-700 dark:bg-primary-950/40 dark:text-primary-300"
                >
                  {{ model.brand }}
                </span>
                <span
                  v-for="type in model.types"
                  :key="`type-${type}`"
                  :data-model-type-chip="type"
                  class="inline-flex items-center rounded-md bg-sky-100 px-2 py-0.5 text-xs font-medium text-sky-700 dark:bg-sky-950/40 dark:text-sky-300"
                >
                  {{ typeLabel(type) }}
                </span>
                <span
                  v-for="mode in model.invocationModes"
                  :key="`mode-${mode}`"
                  :data-model-mode-chip="mode"
                  class="inline-flex items-center rounded-md bg-amber-100 px-2 py-0.5 text-xs font-medium text-amber-700 dark:bg-amber-950/40 dark:text-amber-300"
                >
                  {{ modeLabel(mode) }}
                </span>
                <span class="text-xs text-gray-500 dark:text-dark-400">
                  {{ t('modelPlaza.groupCount', { count: model.groups.length }) }}
                </span>
              </div>
            </div>

            <div class="flex flex-shrink-0 items-center gap-1">
              <button
                v-if="model.metadataEditor"
                type="button"
                :data-model-metadata-edit="model.key"
                class="btn btn-ghost btn-icon"
                :title="t('modelPlaza.metadataEditor.edit')"
                @click="selectedMetadataModel = model"
              >
                <Icon name="edit" size="sm" />
              </button>
              <button
                type="button"
                class="btn btn-ghost btn-icon"
                :title="t('modelPlaza.copyModelName')"
                @click="copyName(model.name)"
              >
                <Icon name="copy" size="sm" />
              </button>
            </div>
          </div>

          <div class="space-y-3">
            <section
              v-for="group in model.groups"
              :key="group.key"
              class="rounded-lg border border-gray-100 bg-gray-50/80 p-3 dark:border-dark-700 dark:bg-dark-900/50"
            >
              <div class="mb-2 flex items-start justify-between gap-3">
                <div class="min-w-0">
                  <p class="truncate text-sm font-medium text-gray-900 dark:text-white">
                    {{ group.name }}
                  </p>
                  <p class="mt-0.5 text-xs text-gray-500 dark:text-dark-400">
                    {{ group.channelName }}
                  </p>
                </div>
                <span
                  class="flex-shrink-0 rounded-md px-2 py-0.5 text-xs font-semibold"
                  :class="rateBadgeClass(group.rate)"
                >
                  {{ t('modelPlaza.rateValue', { rate: formatRate(group.rate) }) }}
                </span>
              </div>

              <div v-if="group.pricing" class="space-y-1.5 text-xs">
                <div class="mb-1 flex items-center justify-between gap-3 text-gray-500 dark:text-dark-400">
                  <span>{{ t('modelPlaza.billingMode') }}</span>
                  <span>{{ billingModeLabel(group.pricing.billing_mode) }}</span>
                </div>

                <template v-if="group.pricing.billing_mode === BILLING_MODE_TOKEN">
                  <PriceLine
                    :label="t('modelPlaza.inputPrice')"
                    :value="scaledPrice(group.pricing.input_price, group.rate)"
                    :unit="t('modelPlaza.perMillionUnit')"
                    :scale="perMillionScale"
                  />
                  <PriceLine
                    :label="t('modelPlaza.outputPrice')"
                    :value="scaledPrice(group.pricing.output_price, group.rate)"
                    :unit="t('modelPlaza.perMillionUnit')"
                    :scale="perMillionScale"
                  />
                  <PriceLine
                    v-if="group.pricing.cache_write_price != null"
                    :label="t('modelPlaza.cacheWritePrice')"
                    :value="scaledPrice(group.pricing.cache_write_price, group.rate)"
                    :unit="t('modelPlaza.perMillionUnit')"
                    :scale="perMillionScale"
                  />
                  <PriceLine
                    v-if="group.pricing.cache_read_price != null"
                    :label="t('modelPlaza.cacheReadPrice')"
                    :value="scaledPrice(group.pricing.cache_read_price, group.rate)"
                    :unit="t('modelPlaza.perMillionUnit')"
                    :scale="perMillionScale"
                  />
                  <PriceLine
                    v-if="group.pricing.image_output_price != null"
                    :label="t('modelPlaza.imageOutputPrice')"
                    :value="scaledPrice(group.pricing.image_output_price, group.rate)"
                    :unit="t('modelPlaza.perMillionUnit')"
                    :scale="perMillionScale"
                  />
                </template>

                <PriceLine
                  v-else-if="group.pricing.billing_mode === BILLING_MODE_PER_REQUEST || group.pricing.billing_mode === BILLING_MODE_VIDEO"
                  :label="group.pricing.billing_mode === BILLING_MODE_VIDEO ? t('modelPlaza.videoPrice') : t('modelPlaza.perRequestPrice')"
                  :value="scaledPrice(group.pricing.per_request_price, group.rate)"
                  :unit="group.pricing.billing_mode === BILLING_MODE_VIDEO ? t('modelPlaza.perVideoUnit') : t('modelPlaza.perRequestUnit')"
                  :scale="1"
                />

                <template v-else-if="group.pricing.billing_mode === BILLING_MODE_IMAGE">
                  <PriceLine
                    :label="t('modelPlaza.imageOutputPrice')"
                    :value="scaledPrice(displayUnitPrice(group.pricing), group.rate)"
                    :unit="t('modelPlaza.perImageUnit')"
                    :scale="1"
                  />
                </template>

                <div
                  v-if="displayIntervals(group.pricing, group.rate).length > 0"
                  class="mt-2 border-t border-gray-200 pt-2 dark:border-dark-700"
                >
                  <p class="mb-1 text-xs font-medium text-gray-500 dark:text-dark-400">
                    {{ t('modelPlaza.intervals') }}
                  </p>
                  <div class="space-y-1">
                    <div
                      v-for="interval in displayIntervals(group.pricing, group.rate)"
                      :key="interval.key"
                      class="flex items-start justify-between gap-3 text-xs"
                    >
                      <span class="text-gray-500 dark:text-dark-400">
                        {{ interval.label }}
                      </span>
                      <span class="text-right font-mono text-gray-800 dark:text-gray-100">
                        {{ interval.price }}
                      </span>
                    </div>
                  </div>
                </div>
              </div>

              <p v-else class="text-xs text-gray-500 dark:text-dark-400">
                {{ t('modelPlaza.noPricing') }}
              </p>
            </section>
          </div>
        </article>
      </div>

      <ModelMetadataDialog
        v-if="selectedMetadataModel && selectedMetadataModel.metadataEditor"
        :show="Boolean(selectedMetadataModel)"
        :platform="selectedMetadataModel.platform"
        :model-name="selectedMetadataModel.name"
        :brand="selectedMetadataModel.brand"
        :types="selectedMetadataModel.types"
        :invocation-modes="selectedMetadataModel.invocationModes"
        :editor="selectedMetadataModel.metadataEditor"
        @updated="updateModelMetadata"
        @close="selectedMetadataModel = null"
      />
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, h, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import ModelMetadataDialog from './ModelMetadataDialog.vue'
import userChannelsAPI, {
  type UserAvailableChannel,
  type UserPricingInterval,
  type UserSupportedModelPricing
} from '@/api/channels'
import type {
  ModelInvocationMode,
  ModelMetadataEditor,
  ModelMetadataState,
  ModelReasoningLevel,
  ModelType
} from '@/api/modelMetadata'
import {
  BILLING_MODE_IMAGE,
  BILLING_MODE_PER_REQUEST,
  BILLING_MODE_TOKEN,
  BILLING_MODE_VIDEO,
  type BillingMode
} from '@/constants/channel'
import { formatScaled } from '@/utils/pricing'
import {
  platformAccentBarClass,
  platformDisplayColor
} from '@/utils/platformColors'
import { extractApiErrorMessage } from '@/utils/apiError'
import { useAppStore } from '@/stores/app'
import type { Platform } from '@/types'
import { displayUnitPrice } from '@/utils/modelPricingDisplay'

const PriceLine = (
  props: {
    label: string
    value: number | null
    unit: string
    scale: number
  }
) => {
  if (!hasDisplayPrice(props.value)) return null

  return h('div', { class: 'flex items-start justify-between gap-3' }, [
    h('span', { class: 'text-gray-500 dark:text-dark-400' }, props.label),
    h(
      'span',
      { class: 'text-right font-mono font-medium text-orange-600 dark:text-orange-400' },
      `${formatScaled(props.value, props.scale)} ${props.unit}`
    )
  ])
}

interface ModelGroup {
  key: string
  id: number
  name: string
  channelName: string
  rate: number
  pricing: UserSupportedModelPricing | null
}

interface ModelEntry {
  key: string
  name: string
  platform: string
  brand: string
  metadataEditor: ModelMetadataEditor | null
  types: ModelType[]
  invocationModes: ModelInvocationMode[]
  reasoningLevels: ModelReasoningLevel[]
  thinkingSupported: boolean
  groups: ModelGroup[]
}

interface DisplayInterval {
  key: string
  label: string
  price: string
}

const { t } = useI18n()
const appStore = useAppStore()
const perMillionScale = 1_000_000
const allBrandsKey = 'all'
const allTypesKey = 'all'
const allModesKey = 'all'
const typeOptions: ModelType[] = ['conversation', 'embedding', 'image', 'video', 'tts', 'asr']
const invocationModeOptions: ModelInvocationMode[] = ['sync', 'stream', 'async', 'bidirectional', 'batch']
const typeFilterOptions = [allTypesKey, ...typeOptions]
const modeFilterOptions = [allModesKey, ...invocationModeOptions]

const loading = ref(false)
const error = ref('')
const models = ref<ModelEntry[]>([])
const platforms = ref<Platform[]>([])
const searchQuery = ref('')
const activeBrand = ref(allBrandsKey)
const activeType = ref(allTypesKey)
const activeMode = ref(allModesKey)
const selectedMetadataModel = ref<ModelEntry | null>(null)

const brandList = computed(() => {
  const brands = Array.from(new Set(models.value.map((model) => model.brand))).sort()
  return [allBrandsKey, ...brands]
})

const filteredModels = computed(() => {
  const q = searchQuery.value.trim().toLowerCase()
  return models.value.filter((model) => {
    if (activeBrand.value !== allBrandsKey && model.brand !== activeBrand.value) {
      return false
    }
    if (activeType.value !== allTypesKey && !model.types.includes(activeType.value as ModelType)) {
      return false
    }
    if (activeMode.value !== allModesKey && !model.invocationModes.includes(activeMode.value as ModelInvocationMode)) {
      return false
    }
    if (!q) return true
    return (
      model.name.toLowerCase().includes(q) ||
      model.brand.toLowerCase().includes(q) ||
      model.platform.toLowerCase().includes(q) ||
      model.groups.some(
        (group) =>
          group.name.toLowerCase().includes(q) ||
          group.channelName.toLowerCase().includes(q)
      )
    )
  })
})

function filterButtonClass(value: string, active: string): string {
  if (value === active) {
    return 'border-primary-500 bg-primary-500 text-white shadow-sm hover:bg-primary-600'
  }
  return 'border-gray-200 bg-white text-gray-700 hover:border-primary-300 hover:text-primary-600 dark:border-dark-700 dark:bg-dark-800 dark:text-gray-300 dark:hover:border-primary-600 dark:hover:text-primary-400'
}

function platformColor(platform: string): string {
  return platformDisplayColor(platforms.value, platform)
}

function typeLabel(type: ModelType) {
  return t(`modelPlaza.types.${type}`)
}

function modeLabel(mode: ModelInvocationMode) {
  return t(`modelPlaza.modes.${mode}`)
}

function updateModelMetadata(state: ModelMetadataState) {
  if (!selectedMetadataModel.value) return
  selectedMetadataModel.value.brand = state.brand
  selectedMetadataModel.value.types = [...state.types]
  selectedMetadataModel.value.invocationModes = [...state.invocation_modes]
  selectedMetadataModel.value.reasoningLevels = [...(state.reasoning_levels ?? [])]
  selectedMetadataModel.value.thinkingSupported = Boolean(state.thinking_supported)
  selectedMetadataModel.value.metadataEditor = state.editor || null
  activeBrand.value = state.brand
  selectedMetadataModel.value = null
}

function platformAccentStyle(platform: string) {
  const color = platformColor(platform)
  return color ? { background: color } : undefined
}

function rateBadgeClass(rate: number): string {
  if (rate <= 1) return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/40 dark:text-emerald-300'
  if (rate <= 1.5) return 'bg-amber-100 text-amber-700 dark:bg-amber-900/40 dark:text-amber-300'
  return 'bg-orange-100 text-orange-700 dark:bg-orange-900/40 dark:text-orange-300'
}

function formatRate(rate: number): string {
  return Number.isInteger(rate) ? String(rate) : rate.toFixed(4).replace(/0+$/, '').replace(/\.$/, '')
}

function billingModeLabel(mode: BillingMode): string {
  switch (mode) {
    case BILLING_MODE_PER_REQUEST:
      return t('modelPlaza.billingModePerRequest')
    case BILLING_MODE_IMAGE:
      return t('modelPlaza.billingModeImage')
    case BILLING_MODE_VIDEO:
      return t('modelPlaza.billingModeVideo')
    case BILLING_MODE_TOKEN:
    default:
      return t('modelPlaza.billingModeToken')
  }
}

function scaledPrice(value: number | null, rate: number): number | null {
  return value == null ? null : value * rate
}

function hasDisplayPrice(value: number | null): value is number {
  return value != null && value > 0
}

function formatDisplayPrice(value: number | null, scale: number): string | null {
  return hasDisplayPrice(value) ? formatScaled(value, scale) : null
}

function intervalLabel(interval: UserPricingInterval): string {
  if (interval.tier_label) return interval.tier_label
  const max = interval.max_tokens == null ? t('modelPlaza.intervalUnlimited') : interval.max_tokens
  return `${interval.min_tokens} - ${max}`
}

function intervalPrice(interval: UserPricingInterval, mode: BillingMode, rate: number): string | null {
  if (mode === BILLING_MODE_PER_REQUEST || mode === BILLING_MODE_IMAGE || mode === BILLING_MODE_VIDEO) {
    const unitKey =
      mode === BILLING_MODE_IMAGE
        ? 'modelPlaza.perImageUnit'
        : mode === BILLING_MODE_VIDEO
          ? 'modelPlaza.perVideoUnit'
          : 'modelPlaza.perRequestUnit'
    const price = formatDisplayPrice(scaledPrice(interval.per_request_price, rate), 1)
    return price ? `${price} ${t(unitKey)}` : null
  }
  const input = formatDisplayPrice(scaledPrice(interval.input_price, rate), perMillionScale)
  const output = formatDisplayPrice(scaledPrice(interval.output_price, rate), perMillionScale)
  const parts = [
    input ? `${t('modelPlaza.inputPrice')} ${input}` : null,
    output ? `${t('modelPlaza.outputPrice')} ${output}` : null
  ].filter((part): part is string => Boolean(part))

  return parts.length > 0 ? `${parts.join(' / ')} ${t('modelPlaza.perMillionUnit')}` : null
}

function displayIntervals(pricing: UserSupportedModelPricing, rate: number): DisplayInterval[] {
  return pricing.intervals
    .map((interval, idx) => {
      const price = intervalPrice(interval, pricing.billing_mode, rate)
      if (!price) return null
      return {
        key: `${idx}:${intervalLabel(interval)}`,
        label: intervalLabel(interval),
        price
      }
    })
    .filter((interval): interval is DisplayInterval => interval != null)
}

function buildModels(channels: UserAvailableChannel[]): ModelEntry[] {
  const modelMap = new Map<string, ModelEntry>()
  const groupKeySet = new Set<string>()

  for (const channel of channels) {
    for (const section of channel.platforms || []) {
      const platform = section.platform || 'unknown'
      const groups = section.groups || []
      if (groups.length === 0) continue

      for (const supportedModel of section.supported_models || []) {
        if (!supportedModel.name) continue
        const key = `${platform}:${supportedModel.name}`
        if (!modelMap.has(key)) {
          modelMap.set(key, {
            key,
            name: supportedModel.name,
            platform,
            brand: supportedModel.brand || 'Other',
            metadataEditor: supportedModel.metadata_editor || null,
            types: supportedModel.types || [],
            invocationModes: supportedModel.invocation_modes || [],
            reasoningLevels: supportedModel.reasoning_levels || [],
            thinkingSupported: Boolean(supportedModel.thinking_supported),
            groups: []
          })
        }

        const entry = modelMap.get(key)!
        for (const group of groups) {
          const groupKey = `${key}:${channel.name}:${group.id}`
          if (groupKeySet.has(groupKey)) continue
          groupKeySet.add(groupKey)
          entry.groups.push({
            key: groupKey,
            id: group.id,
            name: group.name,
            channelName: channel.name,
            rate: group.rate_multiplier ?? 1,
            pricing: supportedModel.pricing
          })
        }
      }
    }
  }

  return Array.from(modelMap.values())
    .map((model) => ({
      ...model,
      groups: model.groups.sort((a, b) => a.name.localeCompare(b.name) || a.channelName.localeCompare(b.channelName))
    }))
    .sort((a, b) => a.name.localeCompare(b.name) || a.platform.localeCompare(b.platform))
}

function buildPlatforms(channels: UserAvailableChannel[]): Platform[] {
  const items = new Map<string, Platform>()
  for (const channel of channels) {
    for (const section of channel.platforms || []) {
      if (items.has(section.platform)) continue
      items.set(section.platform, {
        slug: section.platform,
        display_name: section.display_name || section.platform,
        protocol: section.protocol,
        base_url: '',
        auth_modes: [],
        capabilities: [],
        color: section.color || '',
        enabled: true,
        builtin: false,
        created_at: '',
        updated_at: ''
      })
    }
  }
  return Array.from(items.values())
}

async function copyName(name: string) {
  await navigator.clipboard.writeText(name)
  appStore.showSuccess(t('modelPlaza.copied'))
}

async function loadData() {
  loading.value = true
  error.value = ''
  try {
    const channels = await userChannelsAPI.getModelPlaza()
    platforms.value = buildPlatforms(channels)
    models.value = buildModels(channels)
    if (activeBrand.value !== allBrandsKey && !brandList.value.includes(activeBrand.value)) {
      activeBrand.value = allBrandsKey
    }
  } catch (err: unknown) {
    error.value = extractApiErrorMessage(err, t('modelPlaza.loadError'))
  } finally {
    loading.value = false
  }
}

onMounted(loadData)
</script>

<style scoped>
.model-card-grid {
  grid-template-columns: repeat(auto-fill, minmax(min(100%, 20rem), 1fr));
}

.model-card-grid > * {
  width: 100%;
  max-width: 24rem;
}
</style>

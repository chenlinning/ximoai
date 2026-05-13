<template>
  <AppLayout>
    <div class="mx-auto max-w-7xl px-4 py-6 sm:px-6 lg:px-8">
      <!-- Header -->
      <div class="mb-6 flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 class="text-2xl font-bold text-accent-900 dark:text-white">
            {{ t('modelPlaza.title') }}
          </h1>
          <p class="mt-1 text-sm text-accent-500 dark:text-accent-400">
            {{ t('modelPlaza.subtitle') }}
          </p>
        </div>
        <div class="flex items-center gap-3">
          <!-- Search -->
          <div class="relative w-full sm:w-72">
            <Icon
              name="search"
              size="md"
              class="absolute left-3 top-1/2 -translate-y-1/2 text-accent-400 dark:text-accent-500"
            />
            <input
              v-model="searchQuery"
              type="text"
              :placeholder="t('modelPlaza.searchPlaceholder')"
              class="input pl-10"
            />
          </div>
          <!-- Platform filter -->
          <select
            v-model="selectedPlatform"
            class="input w-auto min-w-[140px]"
          >
            <option value="">{{ t('modelPlaza.allPlatforms') }}</option>
            <option v-for="p in platformOptions" :key="p" :value="p">{{ p }}</option>
          </select>
          <!-- Refresh -->
          <button
            @click="loadData"
            :disabled="loading"
            class="btn btn-secondary"
            :title="t('common.refresh', 'Refresh')"
          >
            <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
          </button>
        </div>
      </div>

      <!-- Loading -->
      <div v-if="loading" class="flex items-center justify-center py-20">
        <Icon name="refresh" size="xl" class="h-10 w-10 animate-spin text-accent-400" />
      </div>

      <!-- Empty -->
      <div v-else-if="modelList.length === 0" class="flex flex-col items-center py-20">
        <Icon name="inbox" size="xl" class="mb-3 h-12 w-12 text-accent-400" />
        <p class="text-sm text-accent-500 dark:text-accent-400">{{ t('modelPlaza.empty') }}</p>
      </div>

      <!-- No results after filter -->
      <div v-else-if="filteredModels.length === 0" class="flex flex-col items-center py-20">
        <Icon name="search" size="xl" class="mb-3 h-12 w-12 text-accent-400" />
        <p class="text-sm text-accent-500 dark:text-accent-400">{{ t('modelPlaza.noResults') }}</p>
      </div>

      <!-- Model Cards Grid -->
      <div v-else class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
        <div
          v-for="model in filteredModels"
          :key="model.uniqueKey"
          class="card group relative overflow-hidden transition-shadow hover:shadow-lg"
          :class="[platformBorderClass(model.platform)]"
        >
          <!-- Accent bar at top -->
          <div class="h-1" :class="[platformAccentBarClass(model.platform)]"></div>

          <div class="p-4">
            <!-- Model name + copy button -->
            <div class="mb-3 flex items-start justify-between gap-2">
              <div class="min-w-0 flex-1">
                <h3
                  class="cursor-pointer truncate text-sm font-semibold text-accent-900 dark:text-white"
                  :title="model.name"
                >
                  {{ model.name }}
                </h3>
              </div>
              <button
                @click="copyModelName(model.name)"
                class="flex-shrink-0 rounded-md p-1 text-accent-400 transition-colors hover:bg-accent-100 hover:text-accent-600 dark:hover:bg-dark-700 dark:hover:text-accent-300"
                :title="t('modelPlaza.copyModelName')"
              >
                <Icon name="copy" size="sm" />
              </button>
            </div>

            <!-- Platform badge -->
            <div class="mb-3">
              <span
                :class="[
                  'inline-flex items-center gap-1 rounded-md border px-2 py-0.5 text-[11px] font-medium uppercase',
                  platformBadgeClass(model.platform),
                ]"
              >
                <PlatformIcon :platform="model.platform as GroupPlatform" size="xs" />
                {{ model.platform }}
              </span>
            </div>

            <!-- Pricing info -->
            <div v-if="model.pricing" class="space-y-1.5 text-xs">
              <template v-if="model.pricing.billing_mode === BILLING_MODE_TOKEN">
                <div class="flex justify-between">
                  <span class="text-accent-500 dark:text-accent-400">{{ t('modelPlaza.inputPrice') }}</span>
                  <span :class="[platformTextClass(model.platform)]" class="font-medium">
                    {{ formatScaled(model.pricing.input_price, perMillionScale) }}
                    <span class="text-accent-400 dark:text-accent-500">/1M</span>
                  </span>
                </div>
                <div class="flex justify-between">
                  <span class="text-accent-500 dark:text-accent-400">{{ t('modelPlaza.outputPrice') }}</span>
                  <span :class="[platformTextClass(model.platform)]" class="font-medium">
                    {{ formatScaled(model.pricing.output_price, perMillionScale) }}
                    <span class="text-accent-400 dark:text-accent-500">/1M</span>
                  </span>
                </div>
                <div v-if="model.pricing.cache_read_price != null" class="flex justify-between">
                  <span class="text-accent-500 dark:text-accent-400">{{ t('modelPlaza.cacheReadPrice') }}</span>
                  <span class="font-medium text-accent-600 dark:text-accent-300">
                    {{ formatScaled(model.pricing.cache_read_price, perMillionScale) }}
                    <span class="text-accent-400 dark:text-accent-500">/1M</span>
                  </span>
                </div>
              </template>
              <template v-else-if="model.pricing.billing_mode === BILLING_MODE_PER_REQUEST">
                <div class="flex justify-between">
                  <span class="text-accent-500 dark:text-accent-400">{{ t('modelPlaza.perRequestPrice') }}</span>
                  <span :class="[platformTextClass(model.platform)]" class="font-medium">
                    {{ formatScaled(model.pricing.per_request_price, 1) }}
                    <span class="text-accent-400 dark:text-accent-500">/req</span>
                  </span>
                </div>
              </template>
              <template v-else-if="model.pricing.billing_mode === BILLING_MODE_IMAGE">
                <div class="flex justify-between">
                  <span class="text-accent-500 dark:text-accent-400">{{ t('modelPlaza.imagePrice') }}</span>
                  <span :class="[platformTextClass(model.platform)]" class="font-medium">
                    {{ formatScaled(model.pricing.image_output_price, 1) }}
                    <span class="text-accent-400 dark:text-accent-500">/img</span>
                  </span>
                </div>
              </template>
            </div>
            <div v-else class="text-xs text-accent-400 dark:text-accent-500">
              {{ t('modelPlaza.noPricing') }}
            </div>

            <!-- Group rate multipliers -->
            <div v-if="model.groups.length > 0" class="mt-3 border-t border-accent-100 pt-3 dark:border-dark-700">
              <div class="mb-1.5 text-[10px] font-medium uppercase tracking-wider text-accent-500 dark:text-accent-400">
                {{ t('modelPlaza.groupRates') }}
              </div>
              <div class="flex flex-wrap gap-1">
                <span
                  v-for="g in model.groups"
                  :key="g.id"
                  class="inline-flex items-center rounded px-1.5 py-0.5 text-[11px]"
                  :class="g.isExclusive
                    ? 'bg-purple-500/10 text-purple-600 dark:text-purple-400'
                    : 'bg-accent-100 text-accent-600 dark:bg-dark-700 dark:text-accent-300'"
                >
                  {{ g.name }}
                  <span class="ml-1 font-semibold" :class="getMultiplierColor(g.effectiveRate)">
                    ×{{ g.effectiveRate.toFixed(2) }}
                  </span>
                </span>
              </div>
            </div>
          </div>

          <!-- Copied toast -->
          <Transition name="fade">
            <div
              v-if="copiedModel === model.name"
              class="absolute inset-0 flex items-center justify-center bg-white/80 dark:bg-dark-900/80"
            >
              <span class="flex items-center gap-1 text-sm font-medium text-emerald-600 dark:text-emerald-400">
                <Icon name="check" size="sm" />
                {{ t('modelPlaza.copied') }}
              </span>
            </div>
          </Transition>
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import userChannelsAPI, {
  type UserAvailableChannel,
  type UserAvailableGroup,
  type UserSupportedModelPricing,
} from '@/api/channels'
import userGroupsAPI from '@/api/groups'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatScaled } from '@/utils/pricing'
import {
  platformBadgeClass,
  platformBorderClass,
  platformAccentBarClass,
  platformTextClass,
} from '@/utils/platformColors'
import {
  BILLING_MODE_TOKEN,
  BILLING_MODE_PER_REQUEST,
  BILLING_MODE_IMAGE,
} from '@/constants/channel'
import type { GroupPlatform } from '@/types'

const { t } = useI18n()
const appStore = useAppStore()

const perMillionScale = 1_000_000

// ── State ──────────────────────────────────────────────────────────
const channels = ref<UserAvailableChannel[]>([])
const userGroupRates = ref<Record<number, number>>({})
const loading = ref(false)
const searchQuery = ref('')
const selectedPlatform = ref('')
const copiedModel = ref<string | null>(null)
let copyTimer: ReturnType<typeof setTimeout> | null = null

// ── Flattened model list with group info ───────────────────────────
interface ModelGroup {
  id: number
  name: string
  rateMultiplier: number
  userRateMultiplier: number | null
  isExclusive: boolean
  effectiveRate: number
}

interface FlatModel {
  uniqueKey: string
  name: string
  platform: string
  pricing: UserSupportedModelPricing | null
  groups: ModelGroup[]
}

const modelList = computed<FlatModel[]>(() => {
  const result: FlatModel[] = []
  const seen = new Map<string, FlatModel>()

  for (const channel of channels.value) {
    for (const section of channel.platforms) {
      for (const m of section.supported_models) {
        const key = `${m.platform}::${m.name}`
        const existing = seen.get(key)

        const groups: ModelGroup[] = section.groups.map((g: UserAvailableGroup) => {
          const userRate = userGroupRates.value[g.id] ?? null
          const effectiveRate = userRate !== null ? userRate : g.rate_multiplier
          return {
            id: g.id,
            name: g.name,
            rateMultiplier: g.rate_multiplier,
            userRateMultiplier: userRate,
            isExclusive: g.is_exclusive,
            effectiveRate,
          }
        })

        if (existing) {
          // Merge groups from different channels for the same model
          for (const g of groups) {
            if (!existing.groups.some((eg) => eg.id === g.id)) {
              existing.groups.push(g)
            }
          }
        } else {
          const flat: FlatModel = {
            uniqueKey: key,
            name: m.name,
            platform: m.platform,
            pricing: m.pricing,
            groups,
          }
          seen.set(key, flat)
          result.push(flat)
        }
      }
    }
  }

  return result
})

// ── Platform options for filter ────────────────────────────────────
const platformOptions = computed(() => {
  const platforms = new Set(modelList.value.map((m) => m.platform))
  return Array.from(platforms).sort()
})

// ── Filtered models ────────────────────────────────────────────────
const filteredModels = computed(() => {
  const q = searchQuery.value.trim().toLowerCase()
  let result = modelList.value

  if (selectedPlatform.value) {
    result = result.filter((m) => m.platform === selectedPlatform.value)
  }

  if (q) {
    result = result.filter(
      (m) =>
        m.name.toLowerCase().includes(q) ||
        m.platform.toLowerCase().includes(q) ||
        m.groups.some((g) => g.name.toLowerCase().includes(q)),
    )
  }

  return result
})

// ── Color for rate multiplier ──────────────────────────────────────
function getMultiplierColor(rate: number): string {
  if (rate <= 1) return 'text-emerald-600 dark:text-emerald-400'
  if (rate <= 1.5) return 'text-accent-600 dark:text-accent-300'
  if (rate <= 3) return 'text-orange-600 dark:text-orange-400'
  return 'text-red-600 dark:text-red-400'
}

// ── Copy model name ────────────────────────────────────────────────
async function copyModelName(name: string) {
  try {
    await navigator.clipboard.writeText(name)
    copiedModel.value = name
    if (copyTimer) clearTimeout(copyTimer)
    copyTimer = setTimeout(() => {
      copiedModel.value = null
    }, 1200)
  } catch {
    // Fallback for older browsers
    const textarea = document.createElement('textarea')
    textarea.value = name
    textarea.style.position = 'fixed'
    textarea.style.opacity = '0'
    document.body.appendChild(textarea)
    textarea.select()
    document.execCommand('copy')
    document.body.removeChild(textarea)
    copiedModel.value = name
    if (copyTimer) clearTimeout(copyTimer)
    copyTimer = setTimeout(() => {
      copiedModel.value = null
    }, 1200)
  }
}

// ── Data loading ───────────────────────────────────────────────────
async function loadData() {
  loading.value = true
  try {
    const [list, rates] = await Promise.all([
      userChannelsAPI.getAvailable(),
      userGroupsAPI.getUserGroupRates().catch((err: unknown) => {
        console.error('Failed to load user group rates:', err)
        return {} as Record<number, number>
      }),
    ])
    channels.value = list
    userGroupRates.value = rates
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('common.error')))
  } finally {
    loading.value = false
  }
}

onMounted(loadData)
</script>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>

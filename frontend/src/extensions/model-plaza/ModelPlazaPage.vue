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
        <p class="text-sm text-accent-500 dark:text-accent-400">{{ t('modelPlaza.noModels') }}</p>
      </div>

      <!-- No results after filter -->
      <div v-else-if="filteredModels.length === 0" class="flex flex-col items-center py-20">
        <Icon name="search" size="xl" class="mb-3 h-12 w-12 text-accent-400" />
        <p class="text-sm text-accent-500 dark:text-accent-400">{{ t('modelPlaza.noResults') }}</p>
      </div>

      <!-- Model Cards Grid -->
      <div v-else class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        <div
          v-for="model in filteredModels"
          :key="model.name"
          class="card group relative overflow-hidden rounded-lg border border-transparent bg-white transition-all duration-300 ease-out hover:-translate-y-1 hover:shadow-xl hover:border-current/10 dark:bg-dark-800 dark:hover:border-current/10"
        >
          <!-- Accent bar at top - expands on hover -->
          <div class="h-1 w-0 transition-all duration-500 ease-out group-hover:w-full bg-gradient-to-r from-accent-500 to-accent-400"></div>

          <div class="p-4">
            <!-- Model name + copy button -->
            <div class="mb-3 flex items-start justify-between gap-2">
              <div class="min-w-0 flex-1">
                <h3
                  class="cursor-pointer truncate text-sm font-semibold text-accent-900 transition-all duration-300 group-hover:translate-x-0.5 group-hover:text-accent-700 dark:text-white dark:group-hover:text-accent-200"
                  :title="model.name"
                >
                  {{ model.name }}
                </h3>
              </div>
              <button
                @click="copyModelName(model.name)"
                class="flex-shrink-0 rounded-md p-1 text-accent-400 transition-all duration-300 hover:bg-accent-100 hover:text-accent-600 opacity-0 group-hover:opacity-100 hover:scale-110 dark:hover:bg-dark-700 dark:hover:text-accent-300"
                :title="t('modelPlaza.copyModelName')"
              >
                <Icon name="copy" size="sm" />
              </button>
            </div>

            <!-- Platform entries -->
            <div class="space-y-3">
              <div
                v-for="pe in model.platformEntries"
                :key="pe.platform"
                class="rounded-md border border-accent-100 p-3 dark:border-dark-700"
              >
                <!-- Platform badge -->
                <div class="mb-2">
                  <span
                    :class="[
                      'inline-flex items-center gap-1 rounded-md border px-2 py-0.5 text-[11px] font-medium uppercase',
                      platformBadgeClass(pe.platform),
                    ]"
                  >
                    <PlatformIcon :platform="pe.platform as GroupPlatform" size="xs" />
                    {{ pe.platform }}
                  </span>
                </div>

                <!-- Pricing info -->
                <div v-if="pe.pricing" class="space-y-1.5 text-xs">
                  <template v-if="pe.pricing.billing_mode === BILLING_MODE_TOKEN">
                    <div class="flex justify-between">
                      <span class="text-accent-500 dark:text-accent-400">{{ t('modelPlaza.inputPrice') }}</span>
                      <span :class="[platformTextClass(pe.platform)]" class="font-medium">
                        {{ formatScaled(pe.pricing.input_price, perMillionScale) }}
                        <span class="text-accent-400 dark:text-accent-500">{{ t('modelPlaza.perMillionUnit') }}</span>
                      </span>
                    </div>
                    <div class="flex justify-between">
                      <span class="text-accent-500 dark:text-accent-400">{{ t('modelPlaza.outputPrice') }}</span>
                      <span :class="[platformTextClass(pe.platform)]" class="font-medium">
                        {{ formatScaled(pe.pricing.output_price, perMillionScale) }}
                        <span class="text-accent-400 dark:text-accent-500">{{ t('modelPlaza.perMillionUnit') }}</span>
                      </span>
                    </div>
                    <div v-if="pe.pricing.cache_read_price != null" class="flex justify-between">
                      <span class="text-accent-500 dark:text-accent-400">{{ t('modelPlaza.cacheReadPrice') }}</span>
                      <span class="font-medium text-accent-600 dark:text-accent-300">
                        {{ formatScaled(pe.pricing.cache_read_price, perMillionScale) }}
                        <span class="text-accent-400 dark:text-accent-500">{{ t('modelPlaza.perMillionUnit') }}</span>
                      </span>
                    </div>
                  </template>
                  <template v-else-if="pe.pricing.billing_mode === BILLING_MODE_PER_REQUEST">
                    <div class="flex justify-between">
                      <span class="text-accent-500 dark:text-accent-400">{{ t('modelPlaza.perRequestPrice') }}</span>
                      <span :class="[platformTextClass(pe.platform)]" class="font-medium">
                        {{ formatScaled(pe.pricing.per_request_price, 1) }}
                        <span class="text-accent-400 dark:text-accent-500">{{ t('modelPlaza.perRequestUnit') }}</span>
                      </span>
                    </div>
                  </template>
                  <template v-else-if="pe.pricing.billing_mode === BILLING_MODE_IMAGE">
                    <div class="flex justify-between">
                      <span class="text-accent-500 dark:text-accent-400">{{ t('modelPlaza.imagePrice') }}</span>
                      <span :class="[platformTextClass(pe.platform)]" class="font-medium">
                        {{ formatScaled(pe.pricing.image_output_price, 1) }}
                        <span class="text-accent-400 dark:text-accent-500">{{ t('modelPlaza.perImageUnit') }}</span>
                      </span>
                    </div>
                  </template>
                </div>
                <div v-else class="text-xs text-accent-400 dark:text-accent-500">
                  {{ t('modelPlaza.noPricing') }}
                </div>
              </div>
            </div>

            <!-- Group pricing: show each group's effective price -->
            <div v-if="model.visibleGroups.length > 0" class="mt-3 border-t border-accent-100 pt-3 dark:border-dark-700">
              <div class="mb-2 text-[10px] font-medium uppercase tracking-wider text-accent-500 dark:text-accent-400">
                {{ t('modelPlaza.groupRates') }}
              </div>
              <div class="space-y-2">
                <div
                  v-for="g in model.visibleGroups"
                  :key="g.id"
                  class="rounded-md border border-accent-100 p-2 dark:border-dark-700"
                >
                  <!-- Group name + multiplier -->
                  <div class="mb-1.5 flex items-center justify-between">
                    <span class="text-xs font-medium text-accent-700 dark:text-accent-200">
                      {{ g.name }}
                    </span>
                    <span class="text-[11px] font-semibold" :class="getMultiplierColor(g.effectiveRate)">
                      x{{ g.effectiveRate.toFixed(2) }}
                    </span>
                  </div>
                  <!-- Effective prices for each group -->
                  <div class="space-y-1 text-[11px]">
                    <template v-for="pe in model.platformEntries" :key="pe.platform">
                      <template v-if="pe.pricing">
                        <template v-if="pe.pricing.billing_mode === BILLING_MODE_TOKEN">
                          <div class="flex justify-between text-accent-500 dark:text-accent-400">
                            <span>{{ t('modelPlaza.inputPrice') }}</span>
                            <span class="font-medium text-accent-700 dark:text-accent-200">
                              {{ formatScaled((pe.pricing.input_price ?? 0) * g.effectiveRate, perMillionScale) }}
                              <span class="text-accent-400 dark:text-accent-500">{{ t('modelPlaza.perMillionUnit') }}</span>
                            </span>
                          </div>
                          <div class="flex justify-between text-accent-500 dark:text-accent-400">
                            <span>{{ t('modelPlaza.outputPrice') }}</span>
                            <span class="font-medium text-accent-700 dark:text-accent-200">
                              {{ formatScaled((pe.pricing.output_price ?? 0) * g.effectiveRate, perMillionScale) }}
                              <span class="text-accent-400 dark:text-accent-500">{{ t('modelPlaza.perMillionUnit') }}</span>
                            </span>
                          </div>
                          <div v-if="pe.pricing.cache_read_price != null" class="flex justify-between text-accent-500 dark:text-accent-400">
                            <span>{{ t('modelPlaza.cacheReadPrice') }}</span>
                            <span class="font-medium text-accent-700 dark:text-accent-200">
                              {{ formatScaled((pe.pricing.cache_read_price ?? 0) * g.effectiveRate, perMillionScale) }}
                              <span class="text-accent-400 dark:text-accent-500">{{ t('modelPlaza.perMillionUnit') }}</span>
                            </span>
                          </div>
                        </template>
                        <template v-else-if="pe.pricing.billing_mode === BILLING_MODE_PER_REQUEST">
                          <div class="flex justify-between text-accent-500 dark:text-accent-400">
                            <span>{{ t('modelPlaza.perRequestPrice') }}</span>
                            <span class="font-medium text-accent-700 dark:text-accent-200">
                              {{ formatScaled((pe.pricing.per_request_price ?? 0) * g.effectiveRate, 1) }}
                              <span class="text-accent-400 dark:text-accent-500">{{ t('modelPlaza.perRequestUnit') }}</span>
                            </span>
                          </div>
                        </template>
                        <template v-else-if="pe.pricing.billing_mode === BILLING_MODE_IMAGE">
                          <div class="flex justify-between text-accent-500 dark:text-accent-400">
                            <span>{{ t('modelPlaza.imagePrice') }}</span>
                            <span class="font-medium text-accent-700 dark:text-accent-200">
                              {{ formatScaled((pe.pricing.image_output_price ?? 0) * g.effectiveRate, 1) }}
                              <span class="text-accent-400 dark:text-accent-500">{{ t('modelPlaza.perImageUnit') }}</span>
                            </span>
                          </div>
                        </template>
                      </template>
                    </template>
                  </div>
                </div>
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
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import { type UserSupportedModelPricing } from '@/api/channels'
import userGroupsAPI from '@/api/groups'
import adminChannelsAPI from '@/api/admin/channels'
import adminGroupsAPI from '@/api/admin/groups'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatScaled } from '@/utils/pricing'
import {
  platformBadgeClass,
  platformTextClass,
} from '@/utils/platformColors'
import {
  BILLING_MODE_TOKEN,
  BILLING_MODE_PER_REQUEST,
  BILLING_MODE_IMAGE,
} from '@/constants/channel'
import type { GroupPlatform, AdminGroup } from '@/types'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const authStore = useAuthStore()
const appStore = useAppStore()

const perMillionScale = 1_000_000

// ── State ──────────────────────────────────────────────────────────
const loading = ref(false)
const searchQuery = ref('')
const copiedModel = ref<string | null>(null)
let copyTimer: ReturnType<typeof setTimeout> | null = null

// ── Grouped model: one card per model name ─────────────────────────
interface ModelGroup {
  id: number
  name: string
  rateMultiplier: number
  userRateMultiplier: number | null
  effectiveRate: number
}

interface PlatformEntry {
  platform: string
  pricing: UserSupportedModelPricing | null
}

interface GroupedModel {
  name: string
  platformEntries: PlatformEntry[]
  /** All non-exclusive groups this model is available in */
  visibleGroups: ModelGroup[]
}

const modelList = ref<GroupedModel[]>([])

// ── Filtered models ────────────────────────────────────────────────
const filteredModels = computed(() => {
  const q = searchQuery.value.trim().toLowerCase()
  if (!q) return modelList.value

  return modelList.value.filter(
    (m) =>
      m.name.toLowerCase().includes(q) ||
      m.platformEntries.some((pe) => pe.platform.toLowerCase().includes(q)) ||
      m.visibleGroups.some((g) => g.name.toLowerCase().includes(q)),
  )
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

// ── Helper: merge group into map, skip exclusive ──────────────────
function mergeGroupInto(
  map: Map<number, ModelGroup>,
  g: ModelGroup,
) {
  if (g.rateMultiplier === 0 && g.userRateMultiplier === null) return
  // Skip exclusive groups
  // (isExclusive is already filtered before calling this, but double-check)
  const existing = map.get(g.id)
  if (!existing) {
    map.set(g.id, { ...g })
  }
}

// ── Data loading ──────────────────────────────────────────────────────
// All authenticated users can see model pricing (public info).
// Use admin channels API for everyone, then overlay user-specific
// group rates for non-admin users.
async function loadData() {
  loading.value = true
  try {
    // 1) Load channels + groups via admin API (works for all logged-in users)
    const [channelsResp, allGroups] = await Promise.all([
      adminChannelsAPI.list(1, 200),
      adminGroupsAPI.getAll().catch(() => [] as AdminGroup[]),
    ])

    // 2) For non-admin users, also load their personal group rates
    let userGroupRates: Record<number, number> = {}
    if (!authStore.isAdmin) {
      userGroupRates = await userGroupsAPI.getUserGroupRates()
        .then((r) => (r && typeof r === 'object' && !Array.isArray(r) ? r as Record<number, number> : {}))
        .catch(() => ({} as Record<number, number>))
    }

    const channels = channelsResp.items
    const groupMap = new Map<number, AdminGroup>()
    for (const g of allGroups) groupMap.set(g.id, g)

    // Build result grouped by model name
    const result: GroupedModel[] = []
    const seen = new Map<string, GroupedModel>()

    for (const channel of channels) {
      if (channel.status !== 'active') continue

      const channelGroupIds = channel.group_ids || []
      // Build non-exclusive groups for this channel
      const channelGroups: ModelGroup[] = channelGroupIds
        .map((gid: number) => {
          const g = groupMap.get(gid)
          if (!g) return null as ModelGroup | null
          if (g.is_exclusive === true) return null as ModelGroup | null
          // Non-admin: overlay user-specific rate
          const userRate = userGroupRates[g.id] ?? null
          const effectiveRate = userRate !== null ? userRate : g.rate_multiplier
          return {
            id: g.id,
            name: g.name,
            rateMultiplier: g.rate_multiplier,
            userRateMultiplier: userRate,
            effectiveRate,
          } as ModelGroup
        })
        .filter((g: ModelGroup | null): g is ModelGroup => g !== null)

      for (const mp of channel.model_pricing || []) {
        const pricing: UserSupportedModelPricing = {
          billing_mode: mp.billing_mode,
          input_price: mp.input_price,
          output_price: mp.output_price,
          cache_write_price: mp.cache_write_price,
        cache_read_price: mp.cache_read_price,
        image_output_price: mp.image_output_price ?? null,
        per_request_price: mp.per_request_price ?? null,
        intervals: [],
      }

      for (const modelName of mp.models || []) {
        let model = seen.get(modelName)

        if (!model) {
          model = { name: modelName, platformEntries: [], visibleGroups: [] }
          seen.set(modelName, model)
          result.push(model)
        }

        // Add platform entry if not already present for this platform
        const existingPlatform = model.platformEntries.find(
          (pe) => pe.platform === mp.platform,
        )
        if (!existingPlatform) {
          model.platformEntries.push({ platform: mp.platform, pricing })
        }

        // Merge non-exclusive groups
        const groupMapForModel = new Map<number, ModelGroup>()
        for (const g of model.visibleGroups) groupMapForModel.set(g.id, g)
        for (const g of channelGroups) {
          mergeGroupInto(groupMapForModel, g)
        }
        model.visibleGroups = Array.from(groupMapForModel.values())
      }
    }
  }

  modelList.value = result
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

/* Subtle glow pulse on card hover */
.card {
  transition: transform 0.3s ease-out, box-shadow 0.3s ease-out, border-color 0.3s ease-out;
}
.card:hover {
  animation: glow-pulse 2s ease-in-out infinite;
}
@keyframes glow-pulse {
  0%, 100% { box-shadow: 0 10px 25px -5px rgb(0 0 0 / 0.08), 0 4px 10px -6px rgb(0 0 0 / 0.04); }
  50% { box-shadow: 0 12px 30px -5px rgb(0 0 0 / 0.12), 0 6px 15px -6px rgb(0 0 0 / 0.06); }
}
:is(.dark *) .card:hover {
  animation: glow-pulse-dark 2s ease-in-out infinite;
}
@keyframes glow-pulse-dark {
  0%, 100% { box-shadow: 0 10px 25px -5px rgb(0 0 0 / 0.3), 0 4px 10px -6px rgb(0 0 0 / 0.2); }
  50% { box-shadow: 0 12px 30px -5px rgb(0 0 0 / 0.45), 0 6px 15px -6px rgb(0 0 0 / 0.3); }
}
</style>

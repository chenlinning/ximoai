<template>
  <AppLayout>
    <div class="mx-auto max-w-7xl px-4 py-6 sm:px-6 lg:px-8">
      <!-- Header -->
      <div class="mb-6">
        <h1 class="text-2xl font-bold text-accent-900 dark:text-accent-50">
          {{ t('modelPlaza.title') }}
        </h1>
        <p class="mt-1 text-sm text-accent-500 dark:text-accent-400">
          {{ t('modelPlaza.subtitle') }}
        </p>
      </div>

      <!-- Search + Platform Filter -->
      <div class="mb-6 flex flex-wrap items-center gap-3">
        <div class="relative flex-1 min-w-[200px]">
          <Icon name="search" size="sm" class="absolute left-3 top-1/2 -translate-y-1/2 text-accent-400" />
          <input
            v-model="searchQuery"
            type="text"
            :placeholder="t('modelPlaza.searchPlaceholder')"
            class="w-full rounded-lg border border-accent-200 bg-white py-2 pl-9 pr-3 text-sm placeholder:text-accent-400 focus:border-accent-500 focus:outline-none focus:ring-1 focus:ring-accent-500 dark:border-accent-700 dark:bg-dark-800 dark:placeholder:text-accent-500"
          />
        </div>
        <div class="flex flex-wrap items-center gap-2">
          <button
            v-for="p in platformOptions"
            :key="p.value"
            :class="[
              'rounded-lg px-3 py-1.5 text-xs font-medium transition-colors',
              selectedPlatform === p.value
                ? 'bg-accent-600 text-white shadow-sm'
                : 'bg-accent-100 text-accent-600 hover:bg-accent-200 dark:bg-accent-800 dark:text-accent-300 dark:hover:bg-accent-700',
            ]"
            @click="selectedPlatform = p.value"
          >
            {{ p.label }}
          </button>
        </div>
      </div>

      <!-- Loading -->
      <div v-if="loading" class="flex items-center justify-center py-20">
        <div class="h-8 w-8 animate-spin rounded-full border-4 border-accent-200 border-t-accent-600"></div>
      </div>

      <!-- Empty -->
      <div v-else-if="modelList.length === 0" class="py-20 text-center text-accent-400 dark:text-accent-500">
        <Icon name="inbox" size="lg" class="mx-auto mb-3" />
        <p>{{ t('modelPlaza.noModels') }}</p>
      </div>

      <!-- No results -->
      <div v-else-if="filteredModels.length === 0" class="py-20 text-center text-accent-400 dark:text-accent-500">
        <Icon name="search" size="lg" class="mx-auto mb-3" />
        <p>{{ t('modelPlaza.noResults') }}</p>
      </div>

      <!-- Model Grid: 4 columns -->
      <div v-else class="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4">
        <div
          v-for="model in filteredModels"
          :key="model.name"
          class="card group relative rounded-xl border border-accent-200 bg-white shadow-sm hover:-translate-y-1 hover:shadow-xl dark:border-accent-700 dark:bg-dark-800 overflow-hidden"
        >
          <!-- Top accent bar -->
          <div
            v-if="model.platformEntries.length > 0"
            :class="[platformBadgeClass(model.platformEntries[0].platform), 'h-1 w-0 group-hover:w-full transition-all duration-500']"
          ></div>

          <div class="p-4">
            <!-- Model name + copy -->
            <div class="mb-2 flex items-start justify-between gap-2">
              <h3
                class="truncate text-sm font-semibold text-accent-900 transition-transform duration-200 group-hover:translate-x-0.5 dark:text-accent-50"
                :title="model.name"
              >
                {{ model.name }}
              </h3>
              <button
                class="shrink-0 rounded-md p-1 text-accent-400 opacity-0 transition-all duration-200 hover:scale-110 hover:bg-accent-100 hover:text-accent-600 group-hover:opacity-100 dark:hover:bg-accent-700 dark:hover:text-accent-300"
                @click="copyModelName(model.name)"
              >
                <Icon name="copy" size="sm" />
              </button>
            </div>

            <!-- Platform badges -->
            <div class="mb-3 flex flex-wrap gap-1.5">
              <span
                v-for="pe in model.platformEntries"
                :key="pe.platform"
                :class="[platformBadgeClass(pe.platform), 'inline-flex items-center gap-1 rounded-md px-2 py-0.5 text-[10px] font-medium']"
              >
                <PlatformIcon :platform="pe.platform as GroupPlatform" size="xs" />
                {{ pe.platform }}
              </span>
            </div>

            <!-- Group rate table -->
            <div
              v-if="model.visibleGroups.length > 0"
              class="space-y-1.5 text-[11px]"
            >
              <div
                v-for="g in model.visibleGroups"
                :key="g.id"
                class="group/rate rounded-md border border-accent-100 px-2 py-1 dark:border-accent-700/50 transition-transform duration-150 hover:scale-[1.02]"
              >
                <div class="flex items-center justify-between">
                  <span class="truncate text-accent-600 dark:text-accent-400">{{ g.name }}</span>
                  <span :class="getMultiplierColor(g.effectiveRate)" class="font-medium">
                    ×{{ g.effectiveRate }}
                  </span>
                </div>
                <template v-if="peHasPricing(model, g)">
                  <template v-for="pe in model.platformEntries" :key="pe.platform + '-pricing'">
                    <template v-if="pe.pricing">
                      <div class="mt-1 space-y-0.5" :class="platformTextClass(pe.platform)">
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
                          <div v-if="pe.pricing.cache_read_price" class="flex justify-between text-accent-500 dark:text-accent-400">
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
                      </div>
                    </template>
                  </template>
                </template>
              </div>
            </div>
            <div v-else class="text-[11px] text-accent-400 dark:text-accent-500">
              —
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
import { type UserSupportedModelPricing, type UserAvailableGroup, default as userChannelsAPI } from '@/api/channels'
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
const selectedPlatform = ref('')
const copiedModel = ref<string | null>(null)
let copyTimer: ReturnType<typeof setTimeout> | null = null

// ── Platform filter options ────────────────────────────────────────
const platformOptions = computed(() => {
  const platforms = new Set<string>()
  for (const m of modelList.value) {
    for (const pe of m.platformEntries) {
      platforms.add(pe.platform)
    }
  }
  return [
    { value: '', label: t('modelPlaza.allPlatforms') },
    ...Array.from(platforms).sort().map((p) => ({ value: p, label: p })),
  ]
})

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
  let result = modelList.value
  const q = searchQuery.value.trim().toLowerCase()
  const platform = selectedPlatform.value

  if (q) {
    result = result.filter(
      (m) =>
        m.name.toLowerCase().includes(q) ||
        m.platformEntries.some((pe) => pe.platform.toLowerCase().includes(q)) ||
        m.visibleGroups.some((g) => g.name.toLowerCase().includes(q)),
    )
  }

  if (platform) {
    result = result.filter((m) =>
      m.platformEntries.some((pe) => pe.platform === platform),
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

// ── Helper: check if any platform entry has pricing ────────────────
function peHasPricing(model: GroupedModel, _g: ModelGroup): boolean {
  return model.platformEntries.some((pe) => pe.pricing !== null)
}

// ── Helper: merge group into map, skip exclusive ──────────────────
function mergeGroupInto(
  map: Map<number, ModelGroup>,
  g: ModelGroup,
) {
  if (g.rateMultiplier === 0 && g.userRateMultiplier === null) return
  const existing = map.get(g.id)
  if (!existing) {
    map.set(g.id, { ...g })
  }
}

// ── Data loading: Admin path ───────────────────────────────────────
async function loadAdminData() {
  const [channelsResp, allGroups] = await Promise.all([
    adminChannelsAPI.list(1, 200),
    adminGroupsAPI.getAll().catch(() => [] as AdminGroup[]),
  ])

  const channels = channelsResp.items
  const groupMap = new Map<number, AdminGroup>()
  for (const g of allGroups) groupMap.set(g.id, g)

  const result: GroupedModel[] = []
  const seen = new Map<string, GroupedModel>()

  for (const channel of channels) {
    if (channel.status !== 'active') continue

    const channelGroupIds = channel.group_ids || []
    const channelGroups: ModelGroup[] = channelGroupIds
      .map((gid: number) => {
        const g = groupMap.get(gid)
        if (!g) return null as ModelGroup | null
        if (g.is_exclusive === true) return null as ModelGroup | null
        return {
          id: g.id,
          name: g.name,
          rateMultiplier: g.rate_multiplier,
          userRateMultiplier: null,
          effectiveRate: g.rate_multiplier,
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

        const existingPlatform = model.platformEntries.find(
          (pe) => pe.platform === mp.platform,
        )
        if (!existingPlatform) {
          model.platformEntries.push({ platform: mp.platform, pricing })
        }

        const groupMapForModel = new Map<number, ModelGroup>()
        for (const g of model.visibleGroups) groupMapForModel.set(g.id, g)
        for (const g of channelGroups) {
          mergeGroupInto(groupMapForModel, g)
        }
        model.visibleGroups = Array.from(groupMapForModel.values())
      }
    }
  }

  return result
}

// ── Data loading: User path ────────────────────────────────────────
async function loadUserData() {
  const [channels, userGroupRates] = await Promise.all([
    userChannelsAPI.getAvailable().catch(() => []),
    userGroupsAPI.getUserGroupRates()
      .then((r) => (r && typeof r === 'object' && !Array.isArray(r) ? r as Record<number, number> : {}))
      .catch(() => ({} as Record<number, number>)),
  ])

  const result: GroupedModel[] = []
  const seen = new Map<string, GroupedModel>()

  for (const channel of channels) {
    for (const section of channel.platforms || []) {
      const channelGroups: ModelGroup[] = (section.groups || [])
        .filter((rawG: UserAvailableGroup) => rawG.is_exclusive !== true)
        .map((rawG: UserAvailableGroup) => {
          const userRate = userGroupRates[rawG.id] ?? null
          const effectiveRate = userRate !== null ? userRate : rawG.rate_multiplier
          return {
            id: rawG.id,
            name: rawG.name,
            rateMultiplier: rawG.rate_multiplier,
            userRateMultiplier: userRate,
            effectiveRate,
          } as ModelGroup
        })

      for (const m of section.supported_models || []) {
        let model = seen.get(m.name)
        if (!model) {
          model = { name: m.name, platformEntries: [], visibleGroups: [] }
          seen.set(m.name, model)
          result.push(model)
        }

        const existingPlatform = model.platformEntries.find(
          (pe) => pe.platform === section.platform,
        )
        if (!existingPlatform) {
          model.platformEntries.push({
            platform: section.platform,
            pricing: m.pricing || null,
          })
        }

        const groupMapForModel = new Map<number, ModelGroup>()
        for (const g of model.visibleGroups) groupMapForModel.set(g.id, g)
        for (const g of channelGroups) {
          mergeGroupInto(groupMapForModel, g)
        }
        model.visibleGroups = Array.from(groupMapForModel.values())
      }
    }
  }

  return result
}

// ── Main load function ─────────────────────────────────────────────
async function loadData() {
  loading.value = true
  try {
    modelList.value = authStore.isAdmin
      ? await loadAdminData()
      : await loadUserData()
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

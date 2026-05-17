<template>
  <AppLayout>
  <div class="p-6" style="max-width: 80%; margin: 0 auto">
    <!-- Header -->
    <div class="flex items-center justify-between mb-6">
      <h1 class="text-2xl font-bold text-foreground">{{ t('modelPlaza.title') }}</h1>
    </div>

    <!-- Search & Filters -->
    <div class="flex items-center gap-3 mb-6">
      <div class="relative w-64">
        <Icon name="search" class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-accent-400" />
        <input
          v-model="searchQuery"
          type="text"
          :placeholder="t('modelPlaza.searchPlaceholder')"
          class="w-full pl-9 pr-3 py-2 rounded-lg border border-border bg-background text-foreground text-sm focus:outline-none focus:ring-2 focus:ring-primary/50"
        />
      </div>
      <div class="flex items-center gap-1.5">
        <button
          v-for="p in platformList"
          :key="p"
          @click="togglePlatform(p)"
          :class="[
            'px-3 py-1.5 rounded-lg text-sm font-medium transition-colors',
            activePlatform === p
              ? 'bg-primary text-white'
              : 'bg-accent-100 dark:bg-accent-800 text-accent-600 dark:text-accent-300 hover:bg-accent-200 dark:hover:bg-accent-700'
          ]"
        >
          {{ p === 'all' ? t('modelPlaza.allPlatforms') : p }}
        </button>
      </div>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="flex items-center justify-center py-20">
      <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
    </div>

    <!-- Error -->
    <div v-else-if="error" class="text-center py-20">
      <p class="text-red-500 text-lg">{{ error }}</p>
      <button @click="loadData" class="mt-4 px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90">
        {{ t('modelPlaza.retry') }}
      </button>
    </div>

    <!-- Model Grid -->
    <div v-else>
      <div class="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-5">
        <div
          v-for="model in filteredModels"
          :key="model.name"
          class="group bg-card border border-border rounded-xl p-5 transition-all duration-300 hover:-translate-y-1 hover:shadow-lg hover:shadow-primary/5 hover:border-primary/30 relative overflow-hidden"
        >
          <!-- Accent bar -->
          <div class="absolute top-0 left-0 w-1 h-0 bg-primary transition-all duration-300 group-hover:h-full rounded-l-xl"></div>

          <!-- Model Name + Copy -->
          <div class="flex items-start justify-between mb-3">
            <h3 class="text-lg font-bold text-foreground leading-tight flex-1 mr-2">{{ model.name }}</h3>
            <button
              @click="copyName(model.name)"
              class="opacity-0 group-hover:opacity-100 transition-opacity p-1.5 rounded-lg hover:bg-accent-100 dark:hover:bg-accent-800"
              :title="t('modelPlaza.copyModelName')"
            >
              <Icon name="copy" class="w-4 h-4 text-accent-500" />
            </button>
          </div>

          <!-- Platform Tag -->
          <div class="mb-4">
            <span :class="['inline-flex items-center px-2.5 py-1 rounded-full text-xs font-medium', platformTagClass(model.platform)]">
              {{ model.platform }}
            </span>
          </div>

          <!-- Group Rates -->
          <div class="space-y-3">
            <div
              v-for="group in model.groups"
              :key="group.name"
              class="bg-accent-50 dark:bg-accent-900/30 rounded-lg px-3 py-2.5"
            >
              <div class="flex items-center justify-between mb-2">
                <div class="flex items-center gap-1.5">
                  <span class="text-[10px] uppercase tracking-wider text-accent-400 font-medium">{{ t('modelPlaza.groupLabel') }}</span>
                  <span class="text-sm font-semibold text-emerald-700 dark:text-emerald-400">{{ group.name }}</span>
                </div>
                <div class="flex items-center gap-1.5">
                  <span class="text-[10px] uppercase tracking-wider text-accent-400 font-medium">{{ t('modelPlaza.rateLabel') }}</span>
                  <span :class="['text-sm font-bold', rateColorClass(group.rate)]">×{{ group.rate }}</span>
                </div>
              </div>
              <!-- Token billing -->
              <template v-if="group.billing_mode === BILLING_MODE_TOKEN">
                <div v-if="group.input_price != null" class="flex justify-between text-xs">
                  <span class="text-accent-500">{{ t('modelPlaza.inputPrice') }}</span>
                  <span class="text-orange-600 dark:text-orange-400 font-bold">{{ formatTokenPrice(group.input_price) }} {{ t('modelPlaza.perMillionUnit') }}</span>
                </div>
                <div v-if="group.output_price != null" class="flex justify-between text-xs">
                  <span class="text-accent-500">{{ t('modelPlaza.outputPrice') }}</span>
                  <span class="text-orange-600 dark:text-orange-400 font-bold">{{ formatTokenPrice(group.output_price) }} {{ t('modelPlaza.perMillionUnit') }}</span>
                </div>
                <div v-if="group.cache_read_price != null" class="flex justify-between text-xs">
                  <span class="text-accent-500">{{ t('modelPlaza.cacheReadPrice') }}</span>
                  <span class="text-orange-600 dark:text-orange-400 font-bold">{{ formatTokenPrice(group.cache_read_price) }} {{ t('modelPlaza.perMillionUnit') }}</span>
                </div>
              </template>
              <!-- Per-request billing -->
              <div v-if="group.billing_mode === BILLING_MODE_PER_REQUEST && group.per_request_price != null" class="flex justify-between text-xs">
                <span class="text-accent-500">{{ t('modelPlaza.perRequestPrice') }}</span>
                <span class="text-orange-600 dark:text-orange-400 font-bold">{{ formatPerRequestPrice(group.per_request_price) }} {{ t('modelPlaza.perRequestUnit') }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Empty State -->
      <div v-if="filteredModels.length === 0 && !loading" class="text-center py-20">
        <p class="text-accent-500 text-lg">{{ t('modelPlaza.noModels') }}</p>
      </div>
    </div>
  </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import adminChannelsAPI from '@/api/admin/channels'
import adminGroupsAPI from '@/api/admin/groups'
import userChannelsAPI from '@/api/channels'
import userGroupsAPI from '@/api/groups'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import { formatScaled } from '@/utils/pricing'

const { t } = useI18n()
const authStore = useAuthStore()
const appStore = useAppStore()

/** Format per-token price to per-million-token display (e.g. 0.000003 → "$3") */
const PER_MILLION = 1_000_000
function formatTokenPrice(value: number | null): string {
  return formatScaled(value, PER_MILLION)
}
/** Format per-request price (no scaling) */
function formatPerRequestPrice(value: number | null): string {
  return formatScaled(value, 1)
}

const BILLING_MODE_TOKEN = 'token'
const BILLING_MODE_PER_REQUEST = 'per_request'

interface ModelGroup {
  name: string
  rate: number
  billing_mode: string
  input_price: number | null
  output_price: number | null
  cache_read_price: number | null
  per_request_price: number | null
}

interface ModelEntry {
  name: string
  platform: string
  groups: ModelGroup[]
}

const loading = ref(true)
const error = ref('')
const models = ref<ModelEntry[]>([])
const searchQuery = ref('')
const activePlatform = ref('all')

const platformList = computed(() => {
  const platforms = new Set(models.value.map(m => m.platform))
  return ['all', ...Array.from(platforms).sort()]
})

const filteredModels = computed(() => {
  let result = models.value
  if (activePlatform.value !== 'all') {
    result = result.filter(m => m.platform === activePlatform.value)
  }
  if (searchQuery.value.trim()) {
    const q = searchQuery.value.toLowerCase().trim()
    result = result.filter(m => m.name.toLowerCase().includes(q))
  }
  return result
})

function togglePlatform(p: string) {
  activePlatform.value = p
}

function platformTagClass(platform: string): string {
  const map: Record<string, string> = {
    openai: 'bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-400',
    anthropic: 'bg-orange-100 dark:bg-orange-900/30 text-orange-700 dark:text-orange-400',
    google: 'bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-400',
    gemini: 'bg-cyan-100 dark:bg-cyan-900/30 text-cyan-700 dark:text-cyan-400',
    azure: 'bg-indigo-100 dark:bg-indigo-900/30 text-indigo-700 dark:text-indigo-400',
  }
  return map[platform] || 'bg-accent-100 dark:bg-accent-800 text-accent-600 dark:text-accent-300'
}

function rateColorClass(rate: number): string {
  if (rate <= 1) return 'text-emerald-600 dark:text-emerald-400'
  if (rate <= 1.5) return 'text-amber-600 dark:text-amber-400'
  if (rate <= 3) return 'text-orange-600 dark:text-orange-400'
  return 'text-red-600 dark:text-red-400'
}

function copyName(name: string) {
  navigator.clipboard.writeText(name).then(() => {
    appStore.showSuccess(t('modelPlaza.copied'))
  })
}

async function loadData() {
  loading.value = true
  error.value = ''
  try {
    const isAdmin = authStore.user?.role === 'admin'
    if (isAdmin) {
      await loadAdminData()
    } else {
      await loadUserData()
    }
  } catch (e: any) {
    error.value = e.message || t('modelPlaza.loadError')
  } finally {
    loading.value = false
  }
}

/**
 * Admin data loading:
 * - Channel has group_ids: number[] and model_pricing: ChannelModelPricing[]
 * - Each ChannelModelPricing has platform: string, models: string[],
 *   input_price/output_price/cache_read_price
 * - AdminGroup has id, name, rate_multiplier, is_exclusive
 */
async function loadAdminData() {
  const [channelsRes, groups] = await Promise.all([
    adminChannelsAPI.list(1, 9999),
    adminGroupsAPI.getAll()
  ])

  const channels: any[] = channelsRes?.items || []

  // Build group lookup by ID
  const groupById = new Map<number, { name: string; rate: number; is_exclusive: boolean }>()
  for (const g of groups) {
    groupById.set(g.id, { name: g.name, rate: g.rate_multiplier ?? 1, is_exclusive: g.is_exclusive })
  }

  const modelMap = new Map<string, ModelEntry>()

  for (const ch of channels) {
    if (!ch.model_pricing || ch.model_pricing.length === 0) continue

    // Resolve this channel's non-exclusive groups
    const channelGroups: { name: string; rate: number }[] = []
    for (const gid of (ch.group_ids || [])) {
      const gi = groupById.get(gid)
      if (gi && !gi.is_exclusive) {
        channelGroups.push({ name: gi.name, rate: gi.rate })
      }
    }
    if (channelGroups.length === 0) continue

    // Each model_pricing entry has platform + models[] + prices
    for (const mp of ch.model_pricing) {
      const platform = mp.platform || 'unknown'
      const modelNames: string[] = mp.models || []
      const billingMode = mp.billing_mode || BILLING_MODE_TOKEN
      const inputPrice = mp.input_price ?? null
      const outputPrice = mp.output_price ?? null
      const cacheReadPrice = mp.cache_read_price ?? null
      const perRequestPrice = mp.per_request_price ?? null

      for (const modelName of modelNames) {
        if (!modelName) continue
        if (!modelMap.has(modelName)) {
          modelMap.set(modelName, { name: modelName, platform, groups: [] })
        }
        const entry = modelMap.get(modelName)!

        // Add this channel's groups to the model (if not already added)
        for (const cg of channelGroups) {
          const existing = entry.groups.find(g => g.name === cg.name)
          if (!existing) {
            entry.groups.push({
              name: cg.name,
              rate: cg.rate,
              billing_mode: billingMode,
              input_price: inputPrice,
              output_price: outputPrice,
              cache_read_price: cacheReadPrice,
              per_request_price: perRequestPrice,
            })
          }
        }
      }
    }
  }

  models.value = Array.from(modelMap.values()).sort((a, b) => a.name.localeCompare(b.name))
}

/**
 * User data loading:
 * - UserAvailableChannel has platforms: UserChannelPlatformSection[]
 * - Each section has platform, groups: UserAvailableGroup[], supported_models: UserSupportedModel[]
 * - getUserGroupRates() returns Record<number, number> (group_id → custom rate)
 */
async function loadUserData() {
  const [channels, userRates] = await Promise.all([
    userChannelsAPI.getAvailable(),
    userGroupsAPI.getUserGroupRates()
  ])

  const modelMap = new Map<string, ModelEntry>()

  for (const ch of channels) {
    if (!ch.platforms || ch.platforms.length === 0) continue

    for (const section of ch.platforms) {
      const platform = section.platform || 'unknown'

      // Filter non-exclusive groups
      const visibleGroups = (section.groups || []).filter((g: any) => !g.is_exclusive)
      if (visibleGroups.length === 0) continue

      // Process supported models
      for (const sm of (section.supported_models || [])) {
        const modelName = sm.name
        if (!modelName) continue

        if (!modelMap.has(modelName)) {
          modelMap.set(modelName, { name: modelName, platform, groups: [] })
        }
        const entry = modelMap.get(modelName)!

        // Add groups with their rates
        for (const g of visibleGroups) {
          const existing = entry.groups.find(eg => eg.name === g.name)
          if (!existing) {
            // Use user custom rate if available, otherwise group default
            const rate = userRates[g.id] ?? g.rate_multiplier ?? 1
            const pricing = sm.pricing
            entry.groups.push({
              name: g.name,
              rate,
              billing_mode: pricing?.billing_mode || BILLING_MODE_TOKEN,
              input_price: pricing?.input_price ?? null,
              output_price: pricing?.output_price ?? null,
              cache_read_price: pricing?.cache_read_price ?? null,
              per_request_price: pricing?.per_request_price ?? null,
            })
          }
        }
      }
    }
  }

  models.value = Array.from(modelMap.values()).sort((a, b) => a.name.localeCompare(b.name))
}

onMounted(() => {
  loadData()
})
</script>

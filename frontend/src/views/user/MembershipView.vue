<template>
  <AppLayout>
    <div class="mx-auto w-full max-w-6xl space-y-6">
      <div class="flex flex-col gap-2">
        <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">{{ t('membership.title') }}</h1>
        <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('membership.description') }}</p>
      </div>

      <div v-if="loading" class="rounded-lg border border-gray-200 bg-white p-6 text-sm text-gray-500 dark:border-dark-700 dark:bg-dark-800 dark:text-dark-400">
        {{ t('membership.loading') }}
      </div>

      <template v-else-if="summary">
        <section class="rounded-lg border border-gray-200 bg-white p-5 dark:border-dark-700 dark:bg-dark-800">
          <div class="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
            <div>
              <div class="text-sm text-gray-500 dark:text-dark-400">{{ t('membership.currentLevel') }}</div>
              <div class="mt-1 flex flex-wrap items-center gap-2">
                <MembershipLevelMark
                  v-if="summary.level"
                  :code="summary.level.code"
                  :color="summary.level.color"
                  size="xl"
                />
                <span class="text-2xl font-semibold text-gray-900 dark:text-white">
                  {{ summary.level ? membershipLevelDisplayName(summary.level.code, summary.level.name) : t('membership.unassigned') }}
                </span>
                <span
                  v-if="summary.level"
                  class="inline-flex items-center rounded-md border px-2 py-0.5 text-xs font-medium"
                  :style="membershipBadgeStyle(membershipLevelColor(summary.level.code, summary.level.color))"
                >
                  {{ summary.level.code }}
                </span>
                <span v-if="summary.level?.is_default" class="badge badge-gray">{{ t('membership.defaultLevel') }}</span>
              </div>
            </div>
            <div class="grid grid-cols-2 gap-4 text-sm md:min-w-[360px]">
              <div>
                <div class="text-gray-500 dark:text-dark-400">{{ t('membership.discountRate') }}</div>
                <div class="mt-1 font-semibold text-gray-900 dark:text-white">
                  {{ formatRate(summary.level?.discount_rate) }}
                </div>
              </div>
              <div>
                <div class="text-gray-500 dark:text-dark-400">{{ t('membership.expiresAt') }}</div>
                <div class="mt-1 font-semibold text-gray-900 dark:text-white">
                  {{ summary.expires_at ? formatDateTime(summary.expires_at) : t('membership.longTerm') }}
                </div>
              </div>
            </div>
          </div>
        </section>

        <section class="rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
          <div class="border-b border-gray-100 px-5 py-4 dark:border-dark-700">
            <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('membership.benefits') }}</h2>
            <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ t('membership.benefitsDescription') }}</p>
          </div>
          <div class="p-5">
            <div class="grid grid-cols-1 gap-3 lg:grid-cols-5">
              <div
                v-for="level in membershipLevels"
                :key="level.id"
                class="flex min-h-[220px] flex-col rounded-lg border border-gray-200 border-t-4 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-900/60"
                :style="{ borderTopColor: membershipLevelColor(level.code, level.color) }"
              >
                <div class="flex flex-col items-center text-center">
                  <MembershipLevelMark :code="level.code" :color="level.color" size="lg" />
                  <div class="mt-2 min-w-0">
                    <div class="truncate font-semibold text-gray-900 dark:text-white">
                      {{ membershipLevelDisplayName(level.code, level.name) }}
                    </div>
                    <div class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ level.code }}</div>
                  </div>
                  <span
                    v-if="isCurrentLevel(level)"
                    class="mt-2 rounded-md px-2 py-0.5 text-xs font-medium"
                    :style="membershipBadgeStyle(membershipLevelColor(level.code, level.color))"
                  >
                    {{ t('membership.current') }}
                  </span>
                </div>

                <div class="mt-4 space-y-2 text-sm">
                  <div class="rounded-md border border-gray-200 bg-white px-3 py-2 dark:border-dark-700 dark:bg-dark-800">
                    <div class="text-xs text-gray-500 dark:text-dark-400">{{ t('membership.discountRate') }}</div>
                    <div class="mt-1 font-semibold text-gray-900 dark:text-white">{{ formatRate(level.discount_rate) }}</div>
                  </div>
                  <div class="rounded-md border border-gray-200 bg-white px-3 py-2 dark:border-dark-700 dark:bg-dark-800">
                    <div class="text-xs text-gray-500 dark:text-dark-400">{{ t('membership.exclusiveGroups') }}</div>
                    <div class="mt-1 font-semibold text-gray-900 dark:text-white">{{ t('membership.exclusiveGroupCount', { count: exclusiveGroups(level).length }) }}</div>
                  </div>
                </div>

                <div class="mt-3 flex flex-1 flex-col gap-1">
                  <span
                    v-for="group in exclusiveGroups(level)"
                    :key="group.id"
                    class="rounded-md border border-gray-200 bg-white px-2 py-1 text-xs text-gray-700 dark:border-dark-700 dark:bg-dark-800 dark:text-dark-200"
                  >
                    {{ group.name }}
                    <span class="text-gray-400 dark:text-dark-500">· {{ formatRate(group.rate_multiplier) }}</span>
                  </span>
                  <span v-if="exclusiveGroups(level).length === 0" class="text-xs text-gray-500 dark:text-dark-400">
                    {{ t('membership.noExclusiveGroups') }}
                  </span>
                </div>
              </div>

              <div v-if="membershipLevels.length === 0" class="col-span-full px-5 py-8 text-center text-sm text-gray-500 dark:text-dark-400">
                {{ t('membership.noBenefits') }}
              </div>
            </div>
          </div>
        </section>

        <section class="rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
          <div class="border-b border-gray-100 px-5 py-4 dark:border-dark-700">
            <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('membership.availableGroups') }}</h2>
          </div>
          <div class="divide-y divide-gray-100 dark:divide-dark-700">
            <div
              v-for="group in groups"
              :key="group.id"
              class="flex flex-col gap-2 px-5 py-4 md:flex-row md:items-center md:justify-between"
            >
              <div>
                <div class="font-medium text-gray-900 dark:text-white">{{ group.name }}</div>
                <div class="text-xs text-gray-500 dark:text-dark-400">
                  {{ group.platform }} · {{ group.is_exclusive ? t('membership.exclusiveGroups') : t('membership.publicGroup') }}
                </div>
              </div>
              <div class="text-sm text-gray-600 dark:text-dark-300 md:text-right">
                <div>{{ t('membership.groupRate') }} {{ formatRate(group.rate_multiplier) }}</div>
                <div>{{ t('membership.effectiveRate') }} {{ formatRate(effectiveRate(group)) }}</div>
              </div>
            </div>
            <div v-if="groups.length === 0" class="px-5 py-8 text-center text-sm text-gray-500 dark:text-dark-400">
              {{ t('membership.noGroups') }}
            </div>
          </div>
        </section>

        <section class="rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
          <div class="border-b border-gray-100 px-5 py-4 dark:border-dark-700">
            <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('membership.managedKeys') }}</h2>
          </div>
          <div class="divide-y divide-gray-100 dark:divide-dark-700">
            <div
              v-for="managed in managedKeys"
              :key="managed.id"
              class="flex flex-col gap-2 px-5 py-4 md:flex-row md:items-center md:justify-between"
            >
              <div>
                <div class="flex flex-wrap items-center gap-2">
                  <span class="font-medium text-gray-900 dark:text-white">{{ managed.api_key?.name || `API Key #${managed.api_key_id}` }}</span>
                  <span :class="['badge', managed.status === 'active' ? 'badge-success' : 'badge-gray']">
                    {{ managed.status === 'active' ? t('membership.enabled') : t('membership.disabled') }}
                  </span>
                </div>
                <div class="text-xs text-gray-500 dark:text-dark-400">
                  {{ managed.group?.name || t('membership.groupFallback', { id: managed.group_id }) }}
                  <span v-if="managed.disabled_reason"> · {{ disabledReasonText(managed.disabled_reason) }}</span>
                </div>
              </div>
              <div class="flex flex-col gap-1 md:items-end">
                <div v-if="managed.group" class="text-xs text-gray-500 dark:text-dark-400">
                  {{ t('membership.effectiveRate') }} {{ formatRate(effectiveRate(managed.group)) }}
                </div>
                <code v-if="managed.api_key?.masked_key" class="code text-xs">{{ managed.api_key.masked_key }}</code>
              </div>
            </div>
            <div v-if="managedKeys.length === 0" class="px-5 py-8 text-center text-sm text-gray-500 dark:text-dark-400">
              {{ t('membership.noManagedKeys') }}
            </div>
          </div>
        </section>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { membershipAPI, type MembershipGroup, type MembershipSummary } from '@/api/membership'
import AppLayout from '@/components/layout/AppLayout.vue'
import MembershipLevelMark from '@/components/membership/MembershipLevelMark.vue'
import { useAppStore } from '@/stores/app'
import { formatDateTime } from '@/utils/format'
import {
  membershipBadgeStyle,
  membershipLevelColor,
  membershipLevelDisplayName
} from '@/utils/membershipStyle'

const appStore = useAppStore()
const { t } = useI18n()
const loading = ref(false)
const summary = ref<MembershipSummary | null>(null)
const membershipLevels = computed(() => summary.value?.levels ?? [])
const groups = computed(() => summary.value?.groups ?? [])
const managedKeys = computed(() => summary.value?.managed_keys ?? [])

const formatRate = (value?: number | null) => `${Number(value ?? 1).toFixed(4).replace(/\.?0+$/, '')}x`
const effectiveRate = (group: MembershipGroup) =>
  group.effective_rate_multiplier ?? group.rate_multiplier * (summary.value?.level?.discount_rate ?? 1)
const exclusiveGroups = (level: NonNullable<MembershipSummary['levels']>[number]) =>
  (level.groups ?? []).filter((group) => group.is_exclusive)
const isCurrentLevel = (level: NonNullable<MembershipSummary['levels']>[number]) =>
  summary.value?.level?.id === level.id

const disabledReasonText = (reason: string) => {
  const key = `membership.disabledReasons.${reason}`
  const translated = t(key)
  return translated === key ? reason : translated
}

const loadMembership = async () => {
  loading.value = true
  try {
    summary.value = await membershipAPI.getCurrent()
  } catch (error) {
    appStore.showError(t('membership.loadFailed'))
  } finally {
    loading.value = false
  }
}

onMounted(loadMembership)
</script>

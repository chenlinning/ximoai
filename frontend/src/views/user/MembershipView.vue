<template>
  <AppLayout>
    <div class="mx-auto w-full max-w-6xl space-y-6">
      <div class="flex flex-col gap-2">
        <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">会员中心</h1>
        <p class="text-sm text-gray-500 dark:text-dark-400">查看当前会员等级、可用分组和系统托管 API Key。</p>
      </div>

      <div v-if="loading" class="rounded-lg border border-gray-200 bg-white p-6 text-sm text-gray-500 dark:border-dark-700 dark:bg-dark-800 dark:text-dark-400">
        正在加载会员信息...
      </div>

      <template v-else-if="summary">
        <section class="rounded-lg border border-gray-200 bg-white p-5 dark:border-dark-700 dark:bg-dark-800">
          <div class="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
            <div>
              <div class="text-sm text-gray-500 dark:text-dark-400">当前会员等级</div>
              <div class="mt-1 flex flex-wrap items-center gap-2">
                <span class="text-2xl font-semibold text-gray-900 dark:text-white">
                  {{ summary.level?.name || '未绑定会员等级' }}
                </span>
                <span v-if="summary.level" class="badge badge-primary">{{ summary.level.code }}</span>
                <span v-if="summary.level?.is_default" class="badge badge-gray">默认等级</span>
              </div>
            </div>
            <div class="grid grid-cols-2 gap-4 text-sm md:min-w-[360px]">
              <div>
                <div class="text-gray-500 dark:text-dark-400">折扣倍率</div>
                <div class="mt-1 font-semibold text-gray-900 dark:text-white">
                  {{ formatRate(summary.level?.discount_rate) }}
                </div>
              </div>
              <div>
                <div class="text-gray-500 dark:text-dark-400">到期时间</div>
                <div class="mt-1 font-semibold text-gray-900 dark:text-white">
                  {{ summary.expires_at ? formatDateTime(summary.expires_at) : '长期有效' }}
                </div>
              </div>
            </div>
          </div>
        </section>

        <section class="rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
          <div class="border-b border-gray-100 px-5 py-4 dark:border-dark-700">
            <h2 class="text-base font-semibold text-gray-900 dark:text-white">可用分组</h2>
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
                  {{ group.platform }} · {{ group.is_exclusive ? '专属分组' : '公开分组' }}
                </div>
              </div>
              <div class="text-sm text-gray-600 dark:text-dark-300">
                分组倍率 {{ group.rate_multiplier }}x · 实际倍率 {{ formatRate(group.rate_multiplier * (summary.level?.discount_rate ?? 1)) }}
              </div>
            </div>
            <div v-if="groups.length === 0" class="px-5 py-8 text-center text-sm text-gray-500 dark:text-dark-400">
              当前会员等级没有配置可用分组。
            </div>
          </div>
        </section>

        <section class="rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
          <div class="border-b border-gray-100 px-5 py-4 dark:border-dark-700">
            <h2 class="text-base font-semibold text-gray-900 dark:text-white">系统托管 Key</h2>
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
                    {{ managed.status === 'active' ? '启用' : '停用' }}
                  </span>
                </div>
                <div class="text-xs text-gray-500 dark:text-dark-400">
                  {{ managed.group?.name || `分组 #${managed.group_id}` }}
                  <span v-if="managed.disabled_reason"> · {{ disabledReasonText(managed.disabled_reason) }}</span>
                </div>
              </div>
              <code v-if="managed.api_key?.key" class="code text-xs">{{ maskManagedKey(managed.api_key.key) }}</code>
            </div>
            <div v-if="managedKeys.length === 0" class="px-5 py-8 text-center text-sm text-gray-500 dark:text-dark-400">
              暂无系统托管 Key。
            </div>
          </div>
        </section>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { membershipAPI, type MembershipSummary } from '@/api/membership'
import AppLayout from '@/components/layout/AppLayout.vue'
import { useAppStore } from '@/stores/app'
import { formatDateTime } from '@/utils/format'
import { maskApiKey } from '@/utils/maskApiKey'

const appStore = useAppStore()
const loading = ref(false)
const summary = ref<MembershipSummary | null>(null)
const groups = computed(() => summary.value?.groups ?? [])
const managedKeys = computed(() => summary.value?.managed_keys ?? [])

const formatRate = (value?: number | null) => `${Number(value ?? 1).toFixed(4).replace(/\.?0+$/, '')}x`
const maskManagedKey = (key: string) => maskApiKey(key)

const disabledReasonText = (reason: string) => {
  const labels: Record<string, string> = {
    membership_expired: '会员到期',
    membership_group_removed: '会员等级移除分组',
    membership_level_disabled: '会员等级停用',
    repair_disabled: '自动修复停用'
  }
  return labels[reason] || reason
}

const loadMembership = async () => {
  loading.value = true
  try {
    summary.value = await membershipAPI.getCurrent()
  } catch (error) {
    appStore.showError('会员信息加载失败')
  } finally {
    loading.value = false
  }
}

onMounted(loadMembership)
</script>

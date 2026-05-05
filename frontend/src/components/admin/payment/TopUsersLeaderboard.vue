<template>
  <div class="card p-4">
    <h3 class="mb-4 text-sm font-semibold text-accent-900 dark:text-white">
      {{ t('payment.admin.topUsers') }}
    </h3>
    <div
      v-if="!users?.length"
      class="flex h-32 items-center justify-center text-sm text-accent-500 dark:text-accent-400"
    >
      {{ t('payment.admin.noData') }}
    </div>
    <div v-else class="space-y-2">
      <div
        v-for="(user, idx) in users"
        :key="user.user_id"
        class="flex items-center justify-between rounded-lg px-3 py-2 hover:bg-accent-50 dark:hover:bg-dark-700"
      >
        <div class="flex items-center gap-3">
          <span
            :class="[
              'flex h-6 w-6 items-center justify-center rounded-full text-xs font-bold',
              rankClass(idx),
            ]"
          >
            {{ idx + 1 }}
          </span>
          <span class="text-sm text-accent-700 dark:text-accent-300">{{ user.email }}</span>
        </div>
        <span class="text-sm font-medium text-accent-900 dark:text-white">
          ${{ user.amount.toFixed(2) }}
        </span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

defineProps<{
  users: { user_id: number; email: string; amount: number }[]
}>()

function rankClass(idx: number): string {
  if (idx === 0) return 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400'
  if (idx === 1) return 'bg-accent-200 text-accent-600 dark:bg-accent-700 dark:text-accent-300'
  if (idx === 2) return 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400'
  return 'bg-accent-100 text-accent-500 dark:bg-dark-700 dark:text-accent-400'
}
</script>

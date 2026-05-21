<script setup lang="ts">
import AccountTestModal from '@/components/account/AccountTestModal.vue'
import type { Account } from '@/types'

defineProps<{
  show: boolean
  account: Account | null
}>()

const emit = defineEmits<{
  (event: 'close'): void
}>()
</script>

<template>
  <AccountTestModal :show="show" :account="account" @close="emit('close')" />
</template>

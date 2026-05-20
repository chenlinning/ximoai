<template>
  <div>
    <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
      {{ t('payment.paymentMethod') }}
    </label>
    <div class="grid grid-cols-2 gap-3 sm:flex">
      <button
        v-for="method in sortedMethods"
        :key="method.type"
        type="button"
        :disabled="!method.available"
        :class="[
          'relative flex h-[60px] flex-col items-center justify-center rounded-lg border px-3 transition-all sm:flex-1',
          !method.available
            ? 'cursor-not-allowed border-gray-200 bg-gray-50 opacity-50 dark:border-dark-700 dark:bg-dark-800/50'
            : selected === method.type
              ? methodSelectedClass(method.type)
              : 'border-gray-300 bg-white text-gray-700 hover:border-gray-400 dark:border-dark-600 dark:bg-dark-800 dark:text-gray-200 dark:hover:border-dark-500',
        ]"
        @click="method.available && emit('select', method.type)"
      >
        <span class="flex items-center gap-2">
          <PaymentMethodIcon :type="method.type" :class="['h-7 w-7', methodIconClass(method.type)]" />
          <span class="flex flex-col items-start leading-none">
            <span class="text-base font-semibold">{{ t(`payment.methods.${method.type}`) }}</span>
            <span
              v-if="method.fee_rate > 0"
              class="text-[10px] tracking-wide text-gray-500 dark:text-dark-400"
            >
              {{ t('payment.fee') }} {{ method.fee_rate }}%
            </span>
          </span>
        </span>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { METHOD_ORDER } from './providerConfig'
import PaymentMethodIcon from './PaymentMethodIcon.vue'

export interface PaymentMethodOption {
  type: string
  fee_rate: number
  available: boolean
}

const props = defineProps<{
  methods: PaymentMethodOption[]
  selected: string
}>()

const emit = defineEmits<{
  select: [type: string]
}>()

const { t } = useI18n()

const sortedMethods = computed(() => {
  const order: readonly string[] = METHOD_ORDER
  return [...props.methods].sort((a, b) => {
    const ai = order.indexOf(a.type)
    const bi = order.indexOf(b.type)
    return (ai === -1 ? 999 : ai) - (bi === -1 ? 999 : bi)
  })
})

function methodIconClass(type: string): string {
  if (type.includes('alipay')) return 'text-alipay-500'
  if (type.includes('wxpay')) return 'text-wxpay-500'
  if (type === 'airwallex') return 'text-airwallex-500'
  if (type === 'stripe') return 'text-stripe-500'
  return 'text-primary-500'
}

function methodSelectedClass(type: string): string {
  if (type.includes('alipay')) return 'border-alipay-500 bg-alipay-50 text-gray-900 shadow-sm dark:bg-alipay-950 dark:text-gray-100'
  if (type.includes('wxpay')) return 'border-wxpay-500 bg-wxpay-50 text-gray-900 shadow-sm dark:bg-wxpay-950 dark:text-gray-100'
  if (type === 'stripe') return 'border-stripe-500 bg-stripe-50 text-gray-900 shadow-sm dark:bg-stripe-950 dark:text-gray-100'
  if (type === 'airwallex') return 'border-airwallex-500 bg-airwallex-50 text-gray-900 shadow-sm dark:bg-airwallex-950 dark:text-gray-100'
  return 'border-primary-500 bg-primary-50 text-gray-900 shadow-sm dark:bg-primary-950 dark:text-gray-100'
}
</script>

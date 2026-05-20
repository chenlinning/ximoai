<template>
  <span
    class="payment-method-icon"
    :style="iconStyle"
    aria-hidden="true"
  />
</template>

<script setup lang="ts">
import { computed } from 'vue'
import alipayIcon from '@/assets/icons/alipay.svg'
import wxpayIcon from '@/assets/icons/wxpay.svg'
import stripeIcon from '@/assets/icons/stripe.svg'
import airwallexIcon from '@/assets/icons/airwallex.svg'

const props = defineProps<{
  type: string
}>()

const iconUrl = computed(() => {
  if (props.type.includes('alipay')) return alipayIcon
  if (props.type.includes('wxpay')) return wxpayIcon
  if (props.type === 'airwallex') return airwallexIcon
  if (props.type === 'stripe') return stripeIcon
  return stripeIcon
})

const iconStyle = computed(() => ({
  '--payment-method-icon-url': `url("${iconUrl.value}")`,
}))
</script>

<style scoped>
.payment-method-icon {
  display: inline-block;
  flex-shrink: 0;
  background-color: currentColor;
  mask: var(--payment-method-icon-url) center / contain no-repeat;
  -webkit-mask: var(--payment-method-icon-url) center / contain no-repeat;
}
</style>

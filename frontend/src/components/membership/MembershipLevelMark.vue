<template>
  <img
    :class="['membership-level-mark', sizeClass]"
    :src="badgeSrc"
    :alt="label"
    loading="lazy"
    decoding="async"
  >
</template>

<script setup lang="ts">
import { computed } from 'vue'
import bronzeBadgeUrl from '@/assets/membership/bronze.png'
import silverBadgeUrl from '@/assets/membership/silver.png'
import goldBadgeUrl from '@/assets/membership/gold.png'
import platinumBadgeUrl from '@/assets/membership/platinum.png'
import diamondBadgeUrl from '@/assets/membership/diamond.png'
import { membershipLevelDisplayName } from '@/utils/membershipStyle'

const props = withDefaults(defineProps<{
  code?: string | null
  color?: string | null
  size?: 'sm' | 'md' | 'lg' | 'xl'
}>(), {
  code: '',
  color: '',
  size: 'md'
})

const badgeImages: Record<string, string> = {
  bronze: bronzeBadgeUrl,
  silver: silverBadgeUrl,
  gold: goldBadgeUrl,
  platinum: platinumBadgeUrl,
  diamond: diamondBadgeUrl
}

const normalizedCode = computed(() => props.code?.trim().toLowerCase() || 'bronze')

const badgeSrc = computed(() => badgeImages[normalizedCode.value] || bronzeBadgeUrl)

const label = computed(() => membershipLevelDisplayName(props.code))

const sizeClass = computed(() => ({
  sm: 'h-5 w-5',
  md: 'h-7 w-7',
  lg: 'h-9 w-9',
  xl: 'h-12 w-12'
}[props.size]))
</script>

<style scoped>
.membership-level-mark {
  flex: 0 0 auto;
  border-radius: 9999px;
  object-fit: cover;
  box-shadow: 0 1px 3px rgba(17, 24, 39, 0.24);
}
</style>

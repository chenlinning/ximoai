<template>
  <div class="home-entry-grid" :class="`home-entry-grid--columns-${columns}`">
    <div
      v-for="row in rows"
      :key="row.index"
      class="home-entry-row"
      :class="row.compact ? 'home-entry-row--compact' : 'home-entry-row--primary'"
    >
      <div
        v-for="entry in row.items"
        :key="entry.item.id"
        class="home-entry-card-slot"
      >
        <XimoAIHomeCard
          :tab="entry.item"
          :theme="theme"
          :animation-delay="`${entry.index * 80}ms`"
          @activate="emit('activate', $event)"
          @hover="emit('hover', $event)"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { XimoAIHomeTab } from '@/api'
import XimoAIHomeCard from './XimoAIHomeCard.vue'
import type { HomeColumnCount, HomeRow } from './spotlightLayout'

defineProps<{
  rows: HomeRow<XimoAIHomeTab>[]
  columns: HomeColumnCount
  theme: 'light' | 'dark'
}>()

const emit = defineEmits<{
  activate: [tabID: string]
  hover: [tabID: string]
}>()
</script>

<style scoped>
.home-entry-grid {
  display: flex;
  width: 100%;
  flex-direction: column;
  gap: 1.5rem;
}

.home-entry-row {
  display: flex;
  width: 100%;
  align-items: flex-start;
  justify-content: center;
  gap: 1.5rem;
}

.home-entry-card-slot {
  flex: 0 0 auto;
  min-width: 0;
}

.home-entry-grid--columns-3 .home-entry-row--primary .home-entry-card-slot {
  width: calc((100% - 3rem) / 3);
}

.home-entry-grid--columns-3 .home-entry-row--compact .home-entry-card-slot {
  width: calc((80% - 2.4rem) / 3);
}

.home-entry-grid--columns-2 .home-entry-row--primary .home-entry-card-slot {
  width: min(calc((100% - 1.5rem) / 2), 24.5rem);
}

.home-entry-grid--columns-2 .home-entry-row--compact .home-entry-card-slot {
  width: min(calc((80% - 1.2rem) / 2), 19.6rem);
}

.home-entry-grid--columns-1 .home-entry-row--primary .home-entry-card-slot {
  width: min(100%, 24.5rem);
}

.home-entry-grid--columns-1 .home-entry-row--compact .home-entry-card-slot {
  width: min(80%, 19.6rem);
}
</style>

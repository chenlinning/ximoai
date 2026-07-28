<template>
  <div
    class="home-entry-spotlight"
    :class="[
      `home-entry-spotlight--${layout.mode}`,
      `home-entry-spotlight--columns-${columns}`
    ]"
  >
    <div class="home-entry-active-slot">
      <XimoAIHomeCard
        :tab="layout.active.item"
        :theme="theme"
        active
        spotlight
        @activate="emit('activate', $event)"
        @hover="emit('hover', $event)"
      />
    </div>

    <template v-if="columns === 3 && layout.mode === 'center'">
      <div class="home-entry-spotlight-rail home-entry-spotlight-rail--left">
        <div
          v-for="entry in layout.left"
          :key="entry.item.id"
          class="home-entry-companion-slot"
        >
          <XimoAIHomeCard
            :tab="entry.item"
            :theme="theme"
            spotlight
            @activate="emit('activate', $event)"
            @hover="emit('hover', $event)"
          />
        </div>
      </div>
      <div class="home-entry-spotlight-rail home-entry-spotlight-rail--right">
        <div
          v-for="entry in layout.right"
          :key="entry.item.id"
          class="home-entry-companion-slot"
        >
          <XimoAIHomeCard
            :tab="entry.item"
            :theme="theme"
            spotlight
            @activate="emit('activate', $event)"
            @hover="emit('hover', $event)"
          />
        </div>
      </div>
    </template>

    <div
      v-else-if="columns >= 2 && layout.mode !== 'center'"
      class="home-entry-spotlight-companion-grid"
    >
      <div
        v-for="entry in layout.companions"
        :key="entry.item.id"
        class="home-entry-companion-slot"
      >
        <XimoAIHomeCard
          :tab="entry.item"
          :theme="theme"
          spotlight
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
import type { HomeColumnCount, HomeSpotlightLayout } from './spotlightLayout'

defineProps<{
  layout: HomeSpotlightLayout<XimoAIHomeTab>
  columns: HomeColumnCount
  theme: 'light' | 'dark'
}>()

const emit = defineEmits<{
  activate: [tabID: string]
  hover: [tabID: string]
}>()
</script>

<style scoped>
.home-entry-spotlight {
  display: grid;
  width: 100%;
  align-items: center;
  justify-content: center;
  gap: var(--home-horizontal-gap);
}

.home-entry-active-slot {
  grid-area: active;
  width: var(--home-active-width);
}

.home-entry-spotlight--center {
  display: flex;
  justify-content: center;
}

.home-entry-spotlight--columns-3.home-entry-spotlight--center {
  display: grid;
  grid-template-areas: 'left active right';
  grid-template-columns:
    var(--home-companion-width)
    var(--home-active-width)
    var(--home-companion-width);
}

.home-entry-spotlight--left {
  grid-template-areas: 'active companions';
  grid-template-columns: var(--home-active-width) var(--home-companion-grid-width);
}

.home-entry-spotlight--right {
  grid-template-areas: 'companions active';
  grid-template-columns: var(--home-companion-grid-width) var(--home-active-width);
}

.home-entry-spotlight--columns-2.home-entry-spotlight--left {
  grid-template-columns: var(--home-active-width) var(--home-companion-width);
}

.home-entry-spotlight--columns-2.home-entry-spotlight--right {
  grid-template-columns: var(--home-companion-width) var(--home-active-width);
}

.home-entry-spotlight-rail {
  display: flex;
  width: var(--home-companion-width);
  align-self: stretch;
  flex-direction: column;
  justify-content: center;
  gap: var(--home-vertical-gap);
}

.home-entry-spotlight-rail--left {
  grid-area: left;
}

.home-entry-spotlight-rail--right {
  grid-area: right;
}

.home-entry-spotlight-companion-grid {
  display: flex;
  grid-area: companions;
  width: var(--home-companion-grid-width);
  align-self: stretch;
  flex-wrap: wrap;
  align-content: center;
  justify-content: center;
  gap: var(--home-vertical-gap) var(--home-horizontal-gap);
}

.home-entry-companion-slot {
  width: var(--home-companion-width);
}

@media (prefers-reduced-motion: reduce) {
  .home-entry-active-slot,
  .home-entry-companion-slot {
    transition: none;
  }
}
</style>

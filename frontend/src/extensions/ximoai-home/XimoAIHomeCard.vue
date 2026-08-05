<template>
  <article
    class="home-entry-card group min-w-0 text-left"
    :style="{ animationDelay }"
  >
    <div
      ref="mediaRef"
      class="home-entry-media relative aspect-[5/3] overflow-hidden rounded-lg border border-white/60 shadow-lg transition group-hover:border-primary-300 group-hover:shadow-xl dark:border-dark-700/80 dark:group-hover:border-primary-600"
    >
      <img
        v-if="tab.cover_url && coverType(tab.cover_url) === 'image'"
        :src="tab.cover_url"
        :alt="tab.label"
        class="h-full w-full object-contain"
      />
      <video
        v-else-if="tab.cover_url && coverType(tab.cover_url) === 'video'"
        :src="tab.cover_url"
        :aria-label="tab.label"
        class="h-full w-full object-contain"
        autoplay
        muted
        loop
        playsinline
      />
      <iframe
        v-else-if="tab.cover_url && coverType(tab.cover_url) === 'html'"
        :srcdoc="htmlCover(tab.cover_url)"
        :title="tab.label"
        class="home-entry-cover-frame absolute left-0 top-0 border-0"
        :style="htmlCoverStyle"
        sandbox=""
      />
      <div
        v-else
        class="flex h-full items-center justify-center text-5xl font-semibold text-primary-600 dark:text-primary-300"
      >
        {{ tab.label.slice(0, 1).toUpperCase() }}
      </div>
      <button
        type="button"
        class="absolute inset-0 z-10 rounded-lg focus:outline-none focus:ring-2 focus:ring-inset focus:ring-primary-500"
        @click="emit('activate', tab.id)"
      >
        <span class="sr-only">{{ tab.label }}</span>
      </button>
    </div>
    <div class="home-entry-label mt-3 break-words text-center text-base font-semibold text-gray-900 dark:text-white">
      {{ tab.label }}
    </div>
  </article>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import type { XimoAIHomeTab } from '@/api'
import { decodeXimoAIHomeHTMLCover, resolveXimoAIHomeCoverType } from '@/utils/ximoaiHomeCover'

const props = withDefaults(defineProps<{
  tab: XimoAIHomeTab
  theme: 'light' | 'dark'
  animationDelay?: string
}>(), {
  animationDelay: '0ms'
})

const emit = defineEmits<{
  activate: [tabID: string]
}>()

const coverType = resolveXimoAIHomeCoverType
const htmlCover = decodeXimoAIHomeHTMLCover
const HTML_COVER_WIDTH = 1200
const HTML_COVER_HEIGHT = 720
const mediaRef = ref<HTMLElement | null>(null)
const htmlCoverScale = ref(1)
let mediaResizeObserver: ResizeObserver | null = null

const htmlCoverStyle = computed(() => ({
  width: `${HTML_COVER_WIDTH}px`,
  height: `${HTML_COVER_HEIGHT}px`,
  transform: `scale(${htmlCoverScale.value})`,
  transformOrigin: 'top left',
  colorScheme: props.theme
}))

function updateHTMLCoverScale(width: number) {
  if (width > 0) htmlCoverScale.value = width / HTML_COVER_WIDTH
}

onMounted(() => {
  const media = mediaRef.value
  if (!media || !props.tab.cover_url || coverType(props.tab.cover_url) !== 'html') return

  updateHTMLCoverScale(media.getBoundingClientRect().width)
  if (typeof ResizeObserver !== 'undefined') {
    mediaResizeObserver = new ResizeObserver((entries) => {
      updateHTMLCoverScale(entries[0]?.contentRect.width || 0)
    })
    mediaResizeObserver.observe(media)
  }
})

onBeforeUnmount(() => {
  mediaResizeObserver?.disconnect()
})
</script>

<style scoped>
.home-entry-card {
  position: relative;
  width: 100%;
  animation: home-entry-card-in 620ms cubic-bezier(0.22, 1, 0.36, 1) both;
}

@keyframes home-entry-card-in {
  from { opacity: 0; }
  to { opacity: 1; }
}

@media (prefers-reduced-motion: reduce) {
  .home-entry-card {
    animation: none;
  }

}
</style>

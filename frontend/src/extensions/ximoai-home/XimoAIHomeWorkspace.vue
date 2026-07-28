<template>
  <main class="relative flex min-h-screen flex-col overflow-x-hidden bg-gray-50 dark:bg-dark-950">
    <template v-if="!activeTabID">
      <LoginGalaxyBackground class="fixed inset-0 z-0" />
      <AppHeader
        variant="floating"
        show-theme-toggle
        show-console-button
      />

      <section class="relative z-10 flex min-h-screen items-center justify-center px-4 py-24 sm:px-6">
        <div class="w-full">
          <div class="mb-8 text-center">
            <img
              v-if="siteLogo"
              :src="siteLogo"
              :alt="siteName"
              class="mx-auto mb-4 h-16 w-16 object-contain"
            />
            <h1 class="text-3xl font-semibold text-gray-950 dark:text-white sm:text-4xl">
              {{ siteName }}
            </h1>
          </div>

          <div class="home-entry-grid">
            <div
              v-for="(tab, index) in tabs"
              :key="tab.id"
              class="home-entry-card group min-w-0 text-left"
              :class="{
                'home-entry-card--active': hoveredTabID === tab.id,
                'home-entry-card--dimmed': hoveredTabID !== '' && hoveredTabID !== tab.id
              }"
              :style="{ animationDelay: `${index * 80}ms` }"
              @mouseenter="hoveredTabID = tab.id"
              @mouseleave="clearHoveredTab"
            >
              <div class="home-entry-media relative aspect-[5/3] overflow-hidden rounded-lg border border-white/60 shadow-lg transition group-hover:border-primary-300 group-hover:shadow-xl dark:border-dark-700/80 dark:group-hover:border-primary-600">
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
                  class="home-entry-cover-frame h-full w-full border-0"
                  :style="{ colorScheme: theme }"
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
                  @click="activateTab(tab.id)"
                >
                  <span class="sr-only">{{ tab.label }}</span>
                </button>
              </div>
              <div class="home-entry-label mt-3 break-words text-center text-base font-semibold text-gray-900 dark:text-white">
                {{ tab.label }}
              </div>
            </div>
          </div>
        </div>
      </section>
    </template>

    <template v-else>
      <AppHeader
        variant="workspace"
        show-theme-toggle
        show-console-button
      >
        <template #left>
          <div class="min-w-0 max-w-full">
            <nav class="flex flex-wrap items-center justify-center gap-1" :aria-label="t('ximoaiHome.tabs')">
              <button
                v-for="tab in tabs"
                :key="tab.id"
                type="button"
                class="shrink-0 rounded-md border px-3 py-2 text-sm font-medium transition-colors"
                :class="activeTabID === tab.id
                  ? 'border-primary-500 bg-primary-500 text-white shadow-sm hover:bg-primary-600 hover:shadow-md'
                  : 'border-gray-300 text-gray-600 hover:bg-gray-100 hover:text-gray-900 dark:border-dark-600 dark:text-dark-300 dark:hover:bg-dark-800 dark:hover:text-white'"
                @click="activateTab(tab.id)"
              >
                {{ tab.label }}
              </button>
            </nav>
          </div>
        </template>
      </AppHeader>

      <section class="relative min-h-0 flex-1 bg-white dark:bg-dark-950">
        <div
          v-for="tab in tabs"
          v-show="activeTabID === tab.id"
          :key="tab.id"
          class="absolute inset-0"
        >
          <div v-if="loadingTabIDs.has(tab.id)" class="flex h-full items-center justify-center text-sm text-gray-500 dark:text-dark-300">
            {{ t('common.loading') }}
          </div>
          <div v-else-if="tabErrors[tab.id]" class="flex h-full flex-col items-center justify-center gap-4 px-6 text-center">
            <p class="text-sm text-red-600 dark:text-red-400">{{ tabErrors[tab.id] }}</p>
            <button type="button" class="btn btn-primary" @click="retryTab(tab.id)">
              <Icon name="refresh" size="sm" />
              {{ t('ximoaiHome.retry') }}
            </button>
          </div>
          <iframe
            v-else-if="loadedTabIDs.has(tab.id) && frameURLs[tab.id]"
            :ref="(element) => setFrameRef(tab.id, element)"
            :src="frameURLs[tab.id]"
            :title="tab.label"
            class="h-full w-full border-0"
            allow="clipboard-read; clipboard-write; fullscreen"
            allowfullscreen
            @load="handleFrameLoad(tab.id)"
          />
        </div>
      </section>
    </template>
  </main>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { workbenchAPI, type XimoAIHomeTab } from '@/api'
import LoginGalaxyBackground from '@/components/auth/LoginGalaxyBackground.vue'
import Icon from '@/components/icons/Icon.vue'
import AppHeader from '@/components/layout/AppHeader.vue'
import { useAppStore } from '@/stores'
import { sanitizeUrl } from '@/utils/url'
import { decodeXimoAIHomeHTMLCover, resolveXimoAIHomeCoverType } from '@/utils/ximoaiHomeCover'
import {
  addLoadedTab,
  buildPreferencesMessage,
  frameOrigin,
  isPreferencesReadyMessage
} from './homeState'

const props = defineProps<{ tabs: XimoAIHomeTab[] }>()
const appStore = useAppStore()
const { t, locale } = useI18n()

const activeTabID = ref('')
const loadedTabIDs = ref<Set<string>>(new Set())
const loadingTabIDs = ref<Set<string>>(new Set())
const frameURLs = reactive<Record<string, string>>({})
const tabErrors = reactive<Record<string, string>>({})
const frameRefs = new Map<string, HTMLIFrameElement>()
const theme = ref<'light' | 'dark'>(document.documentElement.classList.contains('dark') ? 'dark' : 'light')
const hoveredTabID = ref('')
let themeObserver: MutationObserver | null = null

const siteName = computed(() => appStore.siteName || 'XimoAI')
const siteLogo = computed(() => sanitizeUrl(appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }))
const coverType = resolveXimoAIHomeCoverType
const htmlCover = decodeXimoAIHomeHTMLCover

function setLoading(tabID: string, loading: boolean) {
  const next = new Set(loadingTabIDs.value)
  if (loading) next.add(tabID)
  else next.delete(tabID)
  loadingTabIDs.value = next
}

async function ensureTabLoaded(tab: XimoAIHomeTab) {
  if (loadedTabIDs.value.has(tab.id) || loadingTabIDs.value.has(tab.id)) return

  setLoading(tab.id, true)
  delete tabErrors[tab.id]
  try {
    frameURLs[tab.id] = tab.workbench_sso
      ? (await workbenchAPI.createSSOTicket(tab.url)).entry_url
      : tab.url
    loadedTabIDs.value = addLoadedTab(loadedTabIDs.value, tab.id)
  } catch {
    tabErrors[tab.id] = t('ximoaiHome.loadFailed')
  } finally {
    setLoading(tab.id, false)
  }
}

function activateTab(tabID: string) {
  const tab = props.tabs.find((item) => item.id === tabID)
  if (!tab) return
  activeTabID.value = tab.id
  void ensureTabLoaded(tab)
}

function clearHoveredTab() {
  hoveredTabID.value = ''
}

function retryTab(tabID: string) {
  const tab = props.tabs.find((item) => item.id === tabID)
  if (!tab) return
  delete frameURLs[tabID]
  loadedTabIDs.value = new Set([...loadedTabIDs.value].filter((id) => id !== tabID))
  void ensureTabLoaded(tab)
}

function setFrameRef(tabID: string, element: unknown) {
  if (element instanceof HTMLIFrameElement) frameRefs.set(tabID, element)
  else frameRefs.delete(tabID)
}

function postPreferences(tab: XimoAIHomeTab) {
  const frame = frameRefs.get(tab.id)
  const origin = frameOrigin(tab.url)
  if (!frame?.contentWindow || !origin) return
  frame.contentWindow.postMessage(buildPreferencesMessage(theme.value, locale.value), origin)
}

function broadcastPreferences() {
  props.tabs.forEach((tab) => {
    if (loadedTabIDs.value.has(tab.id)) postPreferences(tab)
  })
}

function handleFrameLoad(tabID: string) {
  const tab = props.tabs.find((item) => item.id === tabID)
  if (tab) postPreferences(tab)
}

function handleMessage(event: MessageEvent) {
  for (const tab of props.tabs) {
    const frameWindow = frameRefs.get(tab.id)?.contentWindow
    if (frameWindow && isPreferencesReadyMessage(event, tab.url, frameWindow)) {
      postPreferences(tab)
      return
    }
  }
}

watch([theme, locale], broadcastPreferences)

watch(() => props.tabs, (nextTabs) => {
  if (activeTabID.value && !nextTabs.some((tab) => tab.id === activeTabID.value)) {
    activeTabID.value = ''
  }
}, { deep: true })

onMounted(() => {
  themeObserver = new MutationObserver(() => {
    theme.value = document.documentElement.classList.contains('dark') ? 'dark' : 'light'
  })
  themeObserver.observe(document.documentElement, { attributes: true, attributeFilter: ['class'] })
  window.addEventListener('message', handleMessage)
})

onBeforeUnmount(() => {
  themeObserver?.disconnect()
  window.removeEventListener('message', handleMessage)
})
</script>

<style scoped>
.home-entry-grid {
  width: min(100%, 78rem);
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(min(100%, 20rem), 24rem));
  justify-content: center;
  gap: 1.5rem;
}

.home-entry-card {
  position: relative;
  transform: translate3d(0, 0, 0) scale(1);
  transform-origin: center;
  opacity: 1;
  filter: saturate(1);
  transition:
    transform 420ms cubic-bezier(0.22, 1, 0.36, 1),
    opacity 300ms ease,
    filter 300ms ease;
  animation: home-entry-card-in 700ms cubic-bezier(0.22, 1, 0.36, 1) both;
}

.home-entry-media::after {
  position: absolute;
  inset: 0;
  z-index: 1;
  pointer-events: none;
  content: '';
  background: linear-gradient(
    115deg,
    transparent 20%,
    rgb(var(--color-primary-500) / 0.2) 48%,
    transparent 76%
  );
  opacity: 0;
  transform: translateX(-120%);
}

@media (hover: hover) and (pointer: fine) {
  .home-entry-card--active {
    z-index: 2;
    transform: translate3d(0, -0.5rem, 0) scale(1.055);
  }

  .home-entry-card--active .home-entry-media {
    box-shadow:
      0 1.5rem 2.5rem -1.5rem rgb(var(--color-primary-500) / 0.65),
      0 0 0 1px rgb(var(--color-primary-500) / 0.35);
  }

  .home-entry-card--active .home-entry-media::after {
    opacity: 1;
    animation: home-entry-card-sheen 900ms ease-out;
  }

  .home-entry-card--dimmed {
    transform: scale(0.94);
    opacity: 0.64;
    filter: saturate(0.72);
  }
}

@keyframes home-entry-card-in {
  from {
    opacity: 0;
    transform: translate3d(0, 1.25rem, 0) scale(0.96);
  }
  to {
    opacity: 1;
    transform: translate3d(0, 0, 0) scale(1);
  }
}

@keyframes home-entry-card-sheen {
  from {
    transform: translateX(-120%);
  }
  to {
    transform: translateX(120%);
  }
}

@media (prefers-reduced-motion: reduce) {
  .home-entry-card {
    animation: none;
    transition: opacity 150ms ease;
  }

  .home-entry-card--active {
    transform: none;
  }

  .home-entry-card--active .home-entry-media::after {
    animation: none;
  }
}

@media (max-width: 639px) {
  .home-entry-grid {
    grid-template-columns: minmax(0, 1fr);
    gap: 1rem;
  }
}
</style>

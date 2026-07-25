<template>
  <AppLayout>
    <div class="mx-auto flex w-full max-w-6xl flex-col gap-5 p-4 sm:p-6">
      <header>
        <h1 class="page-title">{{ t('videoCollector.title') }}</h1>
        <p class="page-description">{{ t('videoCollector.description') }}</p>
      </header>

      <div class="grid min-w-0 gap-5 xl:grid-cols-[minmax(0,1fr)_280px]">
        <section class="min-w-0 overflow-hidden rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
          <div class="border-b border-gray-100 p-5 dark:border-dark-700">
            <label class="input-label" for="video-source-url">{{ t('videoCollector.sourceURL') }}</label>
            <div class="flex min-w-0 flex-col gap-3 sm:flex-row">
              <div class="relative min-w-0 flex-1">
                <Icon name="link" size="md" class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
                <input
                  id="video-source-url"
                  v-model.trim="sourceURL"
                  type="url"
                  class="input pl-10"
                  :placeholder="t('videoCollector.urlPlaceholder')"
                  :disabled="parsing || taskActive"
                  @keydown.enter="parseSource"
                />
              </div>
              <button type="button" class="btn btn-primary min-w-[132px]" :disabled="!canParse" @click="parseSource">
                <Icon name="search" size="sm" :class="parsing ? 'animate-pulse' : ''" />
                {{ parsing ? t('videoCollector.parsing') : t('videoCollector.parse') }}
              </button>
            </div>
          </div>

          <div v-if="errorMessage" class="border-b border-red-200 bg-red-50 px-5 py-3 text-sm text-red-700 dark:border-red-900/60 dark:bg-red-950/30 dark:text-red-200">
            {{ errorMessage }}
          </div>

          <div v-if="parsing" class="flex min-h-[280px] flex-col items-center justify-center gap-4 p-8 text-center">
            <Icon name="refresh" size="xl" class="animate-spin text-primary-500" />
            <p class="text-sm text-gray-500 dark:text-dark-300">{{ t('videoCollector.readingMedia') }}</p>
          </div>

          <div v-else-if="media" class="min-w-0">
            <div class="grid gap-4 border-b border-gray-100 p-5 dark:border-dark-700 sm:grid-cols-[144px_minmax(0,1fr)]">
              <div class="flex aspect-[4/3] items-center justify-center overflow-hidden rounded-lg bg-gray-100 dark:bg-dark-900">
                <img v-if="media.thumbnail" :src="media.thumbnail" :alt="media.title" class="h-full w-full object-cover" />
                <Icon v-else name="play" size="xl" class="text-gray-400" />
              </div>
              <div class="min-w-0 self-center">
                <span class="inline-flex rounded-md bg-primary-50 px-2 py-1 text-xs font-medium text-primary-700 dark:bg-primary-900/30 dark:text-primary-200">
                  {{ media.extractor }}
                </span>
                <h2 class="mt-3 break-words text-lg font-semibold text-gray-900 dark:text-white">{{ media.title }}</h2>
                <p class="mt-2 text-sm text-gray-500 dark:text-dark-300">{{ media.uploader }}</p>
                <p class="mt-1 text-xs text-gray-400 dark:text-dark-400">{{ formatDuration(media.duration) }}</p>
              </div>
            </div>

            <fieldset class="p-5" :disabled="taskActive || task?.state === 'completed'">
              <legend class="mb-3 text-sm font-semibold text-gray-900 dark:text-white">{{ t('videoCollector.chooseFormat') }}</legend>
              <div class="grid gap-2 sm:grid-cols-2">
                <label
                  v-for="format in media.formats"
                  :key="format.id"
                  class="flex min-w-0 cursor-pointer items-center gap-3 rounded-lg border px-4 py-3 transition-colors"
                  :class="selectedFormatID === format.id
                    ? 'border-primary-500 bg-primary-50 dark:border-primary-400 dark:bg-primary-900/20'
                    : 'border-gray-200 bg-white hover:border-primary-300 dark:border-dark-600 dark:bg-dark-800 dark:hover:border-primary-700'"
                >
                  <input v-model="selectedFormatID" type="radio" name="video-format" :value="format.id" class="h-4 w-4 accent-primary-500" />
                  <span class="min-w-0 flex-1">
                    <strong class="block truncate text-sm text-gray-900 dark:text-white">{{ formatLabel(format) }}</strong>
                    <small class="mt-1 block truncate text-xs text-gray-500 dark:text-dark-300">{{ formatDetails(format) }}</small>
                  </span>
                  <span class="shrink-0 text-xs tabular-nums text-gray-500 dark:text-dark-300">{{ formatBytes(format.approximateBytes) }}</span>
                </label>
              </div>
            </fieldset>

            <div v-if="task" class="border-t border-gray-100 p-5 dark:border-dark-700">
              <div class="flex items-center justify-between gap-4 text-sm">
                <span class="font-medium text-gray-900 dark:text-white">{{ taskStateLabel }}</span>
                <span class="tabular-nums text-primary-600 dark:text-primary-300">{{ Math.round(task.percent) }}%</span>
              </div>
              <div class="mt-3 h-2 overflow-hidden rounded-full bg-gray-100 dark:bg-dark-900">
                <div class="h-full rounded-full bg-primary-500 transition-[width] duration-300" :style="{ width: `${Math.max(0, Math.min(100, task.percent))}%` }"></div>
              </div>
              <div class="mt-2 flex flex-wrap justify-between gap-2 text-xs text-gray-500 dark:text-dark-300">
                <span>{{ task.speed || formatTransfer(task.downloadedBytes, task.totalBytes) }}</span>
                <span v-if="task.eta">{{ t('videoCollector.eta', { value: task.eta }) }}</span>
              </div>
              <p v-if="task.error" class="mt-3 text-sm text-red-600 dark:text-red-300">{{ task.error }}</p>
            </div>

            <div class="flex flex-col gap-3 border-t border-gray-100 p-5 dark:border-dark-700 sm:flex-row sm:justify-end">
              <button v-if="taskActive" type="button" class="btn btn-secondary" @click="cancelTask">
                <Icon name="x" size="sm" />
                {{ t('videoCollector.cancel') }}
              </button>
              <button v-else-if="task?.state === 'completed'" type="button" class="btn btn-primary" :disabled="saving" @click="saveToComputer">
                <Icon name="download" size="sm" />
                {{ saving ? t('videoCollector.saving', { percent: saveProgress }) : t('videoCollector.save') }}
              </button>
              <button v-else type="button" class="btn btn-primary" :disabled="!selectedFormat" @click="prepareDownload">
                <Icon name="download" size="sm" />
                {{ t('videoCollector.prepare') }}
              </button>
            </div>
          </div>

          <div v-else class="flex min-h-[320px] flex-col items-center justify-center gap-3 p-8 text-center">
            <div class="flex h-14 w-14 items-center justify-center rounded-full bg-primary-50 text-primary-600 dark:bg-primary-900/25 dark:text-primary-300">
              <Icon name="play" size="lg" />
            </div>
            <p class="text-sm font-medium text-gray-700 dark:text-gray-200">{{ t('videoCollector.empty') }}</p>
          </div>
        </section>

        <aside class="flex flex-col gap-4">
          <section class="rounded-lg border border-gray-200 bg-white p-5 dark:border-dark-700 dark:bg-dark-800">
            <div class="flex items-center gap-2 text-sm font-semibold text-gray-900 dark:text-white">
              <Icon name="shield" size="sm" class="text-primary-500" />
              {{ t('videoCollector.temporaryTitle') }}
            </div>
            <dl class="mt-4 space-y-3 text-sm">
              <div class="flex items-center justify-between gap-3">
                <dt class="text-gray-500 dark:text-dark-300">{{ t('videoCollector.afterDownload') }}</dt>
                <dd class="font-medium text-gray-900 dark:text-white">10 {{ t('videoCollector.minutes') }}</dd>
              </div>
              <div class="flex items-center justify-between gap-3">
                <dt class="text-gray-500 dark:text-dark-300">{{ t('videoCollector.unclaimed') }}</dt>
                <dd class="font-medium text-gray-900 dark:text-white">30 {{ t('videoCollector.minutes') }}</dd>
              </div>
            </dl>
            <p v-if="deleteAt" class="mt-4 border-t border-gray-100 pt-4 text-xs leading-5 text-gray-500 dark:border-dark-700 dark:text-dark-300">
              {{ t('videoCollector.deleteAt', { value: formatDateTime(deleteAt) }) }}
            </p>
          </section>

          <section class="rounded-lg border border-gray-200 bg-white p-5 text-xs leading-5 text-gray-500 dark:border-dark-700 dark:bg-dark-800 dark:text-dark-300">
            {{ t('videoCollector.legal') }}
          </section>
        </aside>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores'
import {
  cancelVideoTask,
  getVideoTask,
  parseVideoURL,
  startVideoTask,
  type MediaFormat,
  type MediaInfo,
  type VideoTask
} from './api'
import { saveVideoTask } from './download'

const { t } = useI18n()
const appStore = useAppStore()
const sourceURL = ref('')
const parsing = ref(false)
const saving = ref(false)
const saveProgress = ref(0)
const media = ref<MediaInfo | null>(null)
const selectedFormatID = ref('')
const task = ref<VideoTask | null>(null)
const errorMessage = ref('')
const deleteAt = ref<string | null>(null)
let pollTimer: ReturnType<typeof setTimeout> | null = null

const activeStates = new Set(['queued', 'downloading', 'processing'])
const taskActive = computed(() => !!task.value && activeStates.has(task.value.state))
const canParse = computed(() => sourceURL.value.length > 0 && !parsing.value && !taskActive.value)
const selectedFormat = computed(() => media.value?.formats.find(format => format.id === selectedFormatID.value) || null)
const taskStateLabel = computed(() => t(`videoCollector.states.${task.value?.state || 'queued'}`))

async function parseSource() {
  if (!canParse.value) return
  stopPolling()
  parsing.value = true
  errorMessage.value = ''
  media.value = null
  task.value = null
  deleteAt.value = null
  try {
    media.value = await parseVideoURL(sourceURL.value)
    selectedFormatID.value = media.value.formats[0]?.id || ''
  } catch (error) {
    errorMessage.value = apiErrorMessage(error, t('videoCollector.parseFailed'))
  } finally {
    parsing.value = false
  }
}

async function prepareDownload() {
  if (!media.value || !selectedFormat.value) return
  errorMessage.value = ''
  try {
    task.value = await startVideoTask({
      sourceUrl: media.value.sourceUrl,
      mediaId: media.value.id,
      title: media.value.title,
      formatId: selectedFormat.value.id,
      hasAudio: selectedFormat.value.hasAudio
    })
    if (taskActive.value) {
      schedulePoll(300)
    }
  } catch (error) {
    errorMessage.value = apiErrorMessage(error, t('videoCollector.prepareFailed'))
  }
}

async function pollTask() {
  if (!task.value) return
  try {
    task.value = await getVideoTask(task.value.id)
    if (taskActive.value) {
      schedulePoll(1000)
    }
  } catch (error) {
    errorMessage.value = apiErrorMessage(error, t('videoCollector.statusFailed'))
    stopPolling()
  }
}

async function cancelTask() {
  if (!task.value || !taskActive.value) return
  try {
    await cancelVideoTask(task.value.id)
    task.value = { ...task.value, state: 'cancelled' }
    stopPolling()
  } catch (error) {
    errorMessage.value = apiErrorMessage(error, t('videoCollector.cancelFailed'))
  }
}

async function saveToComputer() {
  if (!task.value || task.value.state !== 'completed') return
  saving.value = true
  saveProgress.value = 0
  errorMessage.value = ''
  try {
    deleteAt.value = await saveVideoTask(task.value.id, task.value.fileName || `${media.value?.title || 'video'}.mp4`, value => {
      saveProgress.value = value
    })
    appStore.showSuccess(t('videoCollector.saved'))
  } catch (error) {
    if ((error as { name?: string })?.name !== 'AbortError') {
      errorMessage.value = apiErrorMessage(error, t('videoCollector.saveFailed'))
    }
  } finally {
    saving.value = false
  }
}

function schedulePoll(delay: number) {
  stopPolling()
  pollTimer = setTimeout(() => void pollTask(), delay)
}

function stopPolling() {
  if (pollTimer) {
    clearTimeout(pollTimer)
    pollTimer = null
  }
}

function formatLabel(format: MediaFormat): string {
  if (!format.hasVideo) return t('videoCollector.audioOnly')
  return format.height ? `${format.height}p` : t('videoCollector.video')
}

function formatDetails(format: MediaFormat): string {
  const values = [format.extension?.toUpperCase(), format.videoCodec, format.audioCodec]
  if (format.hasVideo && !format.hasAudio) values.push(t('videoCollector.mergeAudio'))
  return values.filter(Boolean).join(' · ')
}

function formatBytes(value?: number): string {
  if (!value || value <= 0) return '—'
  const units = ['B', 'KB', 'MB', 'GB']
  let size = value
  let unit = 0
  while (size >= 1024 && unit < units.length - 1) {
    size /= 1024
    unit++
  }
  return `${size.toFixed(unit === 0 ? 0 : 1)} ${units[unit]}`
}

function formatTransfer(downloaded?: number, total?: number): string {
  if (!downloaded && !total) return ''
  return `${formatBytes(downloaded)} / ${formatBytes(total)}`
}

function formatDuration(value?: number): string {
  if (!value || value <= 0) return ''
  const minutes = Math.floor(value / 60)
  const seconds = Math.floor(value % 60).toString().padStart(2, '0')
  return `${minutes}:${seconds}`
}

function formatDateTime(value: string): string {
  return new Date(value).toLocaleString()
}

function apiErrorMessage(error: unknown, fallback: string): string {
  const message = (error as { message?: unknown })?.message
  return typeof message === 'string' && message.trim() ? message : fallback
}

onBeforeUnmount(stopPolling)
</script>

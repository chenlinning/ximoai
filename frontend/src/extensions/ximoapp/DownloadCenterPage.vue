<template>
  <AppLayout>
    <div class="mx-auto flex h-full w-full max-w-none flex-col gap-5 p-4 sm:p-6">
      <div class="flex flex-col justify-between gap-3 sm:flex-row sm:items-center">
        <div>
          <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">
            {{ t('downloadCenter.title', 'Download Center') }}
          </h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
            {{ t('downloadCenter.description', 'Download published packages and view app introductions and release notes.') }}
          </p>
        </div>
        <button type="button" class="btn btn-secondary inline-flex items-center gap-2" :disabled="loading" @click="loadDownloadCenter">
          <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
          {{ t('downloadCenter.refresh', 'Refresh') }}
        </button>
      </div>

      <div v-if="loading" class="flex min-h-[260px] items-center justify-center rounded-lg border border-gray-200 bg-white text-sm text-gray-500 dark:border-dark-700 dark:bg-dark-800 dark:text-gray-400">
        <Icon name="refresh" size="md" class="mr-2 animate-spin" />
        {{ t('downloadCenter.loading', 'Loading download center...') }}
      </div>

      <div v-else-if="displayApps.length === 0" class="rounded-lg border border-gray-200 bg-white px-6 py-16 text-center dark:border-dark-700 dark:bg-dark-800">
        <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
          {{ t('downloadCenter.empty', 'No downloadable software') }}
        </h2>
        <p class="mt-2 text-sm text-gray-500 dark:text-gray-400">
          {{ t('downloadCenter.emptyDescription', 'Uploaded and enabled packages will appear here.') }}
        </p>
      </div>

      <div v-else class="download-app-grid grid gap-4">
        <article
          v-for="app in displayApps"
          :key="app.key"
          class="flex min-w-0 flex-col overflow-hidden rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800"
        >
          <header class="border-b border-gray-100 px-5 py-4 dark:border-dark-700">
            <div class="flex items-start justify-between gap-3">
              <div class="min-w-0">
                <div class="flex flex-wrap items-center gap-2">
                  <h2 class="break-words text-lg font-semibold text-gray-900 dark:text-white">
                    {{ app.name || app.key }}
                  </h2>
                  <span class="rounded-full bg-primary-50 px-2 py-0.5 text-xs font-medium text-primary-700 dark:bg-primary-900/25 dark:text-primary-200">
                    {{ app.client_type || 'custom' }}
                  </span>
                </div>
                <p class="mt-2 text-sm leading-6 text-gray-600 dark:text-gray-300">
                  {{ app.description || t('downloadCenter.appDescriptionFallback', 'No app description.') }}
                </p>
              </div>
              <div class="shrink-0 text-right text-xs text-gray-500 dark:text-gray-400">
                {{ t('downloadCenter.releaseCount', { count: app.releases.length }) }}
              </div>
            </div>
          </header>

          <section v-if="app.latestRelease" class="px-5 py-4">
            <div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
              <div class="min-w-0">
                <div class="flex flex-wrap items-center gap-2">
                  <h3 class="font-semibold text-gray-900 dark:text-white">
                    {{ app.latestRelease.version }}
                  </h3>
                  <span class="rounded-full bg-emerald-50 px-2 py-0.5 text-xs font-medium text-emerald-700 dark:bg-emerald-900/25 dark:text-emerald-200">
                    {{ t('downloadCenter.latestVersion', 'Latest Version') }}
                  </span>
                  <span
                    v-if="app.latestRelease.force"
                    class="rounded-full bg-red-50 px-2 py-0.5 text-xs font-medium text-red-700 dark:bg-red-900/25 dark:text-red-200"
                  >
                    {{ t('downloadCenter.forceUpdate', 'Force Update') }}
                  </span>
                </div>

                <dl class="mt-3 grid grid-cols-1 gap-x-4 gap-y-2 text-sm sm:grid-cols-2">
                  <div>
                    <dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('downloadCenter.target', 'OS / Architecture') }}</dt>
                    <dd class="font-mono text-gray-900 dark:text-gray-100">{{ app.latestRelease.os }} / {{ app.latestRelease.arch }}</dd>
                  </div>
                  <div>
                    <dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('downloadCenter.channel', 'Channel') }}</dt>
                    <dd class="font-mono text-gray-900 dark:text-gray-100">{{ app.latestRelease.channel }}</dd>
                  </div>
                  <div>
                    <dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('downloadCenter.locale', 'Language') }}</dt>
                    <dd class="font-mono text-gray-900 dark:text-gray-100">{{ app.latestRelease.locale || 'all' }}</dd>
                  </div>
                  <div>
                    <dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('downloadCenter.packageType', 'Package') }}</dt>
                    <dd class="font-mono text-gray-900 dark:text-gray-100">{{ app.latestRelease.package_type || '-' }}</dd>
                  </div>
                  <div v-if="app.latestRelease.version_code">
                    <dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('downloadCenter.versionCode', 'Version Code') }}</dt>
                    <dd class="font-mono text-gray-900 dark:text-gray-100">{{ app.latestRelease.version_code }}</dd>
                  </div>
                  <div>
                    <dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('downloadCenter.fileSize', 'File Size') }}</dt>
                    <dd class="font-mono text-gray-900 dark:text-gray-100">{{ formatBytes(app.latestRelease.file_size) }}</dd>
                  </div>
                </dl>
              </div>
              <a
                class="btn btn-primary shrink-0"
                :href="app.latestRelease.download_url"
                target="_blank"
                rel="noopener"
              >
                <Icon name="download" size="sm" class="mr-2" />
                {{ t('downloadCenter.download', 'Download') }}
              </a>
            </div>

            <div class="mt-4 space-y-2 text-sm">
              <div>
                <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('downloadCenter.releaseNotes', 'Release Notes') }}</div>
                <p class="mt-1 whitespace-pre-wrap text-gray-700 dark:text-gray-300">
                  {{ app.latestRelease.notes || t('downloadCenter.noReleaseNotes', 'No release notes') }}
                </p>
              </div>
              <div class="break-all font-mono text-xs text-gray-500 dark:text-gray-400">
                {{ t('downloadCenter.sha256', 'SHA256') }}: {{ app.latestRelease.sha256 }}
              </div>
              <div v-if="releaseTime(app.latestRelease)" class="font-mono text-xs text-gray-500 dark:text-gray-400">
                {{ t('downloadCenter.publishedAt', 'Published At') }}: {{ releaseTime(app.latestRelease) }}
              </div>
            </div>
          </section>

          <section v-if="app.historyReleases.length > 0" class="mt-auto border-t border-gray-100 dark:border-dark-700">
            <button
              type="button"
              class="flex w-full items-center justify-between gap-3 px-5 py-3 text-left text-sm font-medium text-gray-700 transition-colors hover:bg-gray-50 dark:text-gray-200 dark:hover:bg-dark-700/50"
              @click="toggleHistory(app.key)"
            >
              <span>
                {{ t('downloadCenter.historyVersions', 'History Versions') }}
                <span class="ml-1 text-xs font-normal text-gray-500 dark:text-gray-400">
                  {{ t('downloadCenter.releaseCount', { count: app.historyReleases.length }) }}
                </span>
              </span>
              <Icon :name="expandedHistoryKeys.has(app.key) ? 'chevronUp' : 'chevronDown'" size="sm" />
            </button>

            <div v-if="expandedHistoryKeys.has(app.key)" class="divide-y divide-gray-100 dark:divide-dark-700">
              <div
                v-for="release in app.historyReleases"
                :key="release.id || release.download_url"
                class="grid gap-2 px-5 py-3 text-sm sm:grid-cols-[minmax(8rem,1fr)_minmax(12rem,2fr)_auto]"
              >
                <div class="min-w-0">
                  <div class="truncate font-semibold text-gray-900 dark:text-white">{{ release.version }}</div>
                  <div v-if="release.version_code" class="mt-0.5 font-mono text-xs text-gray-500 dark:text-gray-400">
                    {{ t('downloadCenter.versionCode', 'Version Code') }} {{ release.version_code }}
                  </div>
                </div>
                <div class="min-w-0 font-mono text-xs text-gray-600 dark:text-gray-300">
                  <div class="truncate">{{ release.channel }} / {{ release.os }} / {{ release.arch }}</div>
                  <div class="mt-0.5 truncate text-gray-500 dark:text-gray-400">
                    {{ release.locale || 'all' }} / {{ release.package_type || '-' }}
                  </div>
                </div>
                <div class="text-left text-xs text-gray-500 dark:text-gray-400 sm:text-right">
                  <div>{{ formatBytes(release.file_size) }}</div>
                  <div v-if="releaseTime(release)" class="mt-0.5 font-mono">{{ releaseTime(release) }}</div>
                </div>
              </div>
            </div>
          </section>
        </article>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import { ximoAppAPI } from '@/api'
import type { XimoAppDownloadApp, XimoAppDownloadRelease } from '@/api/ximoapp'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'

interface DisplayDownloadApp extends XimoAppDownloadApp {
  latestRelease: XimoAppDownloadRelease | null
  historyReleases: XimoAppDownloadRelease[]
}

const { t } = useI18n()
const appStore = useAppStore()
const loading = ref(false)
const apps = ref<XimoAppDownloadApp[]>([])
const expandedHistoryKeys = ref(new Set<string>())

const displayApps = computed<DisplayDownloadApp[]>(() =>
  [...apps.value]
    .sort((left, right) => (left.name || left.key).localeCompare(right.name || right.key))
    .map((app) => {
      const releases = [...(app.releases || [])].sort(compareReleases)
      return {
        ...app,
        releases,
        latestRelease: releases[0] || null,
        historyReleases: releases.slice(1)
      }
    })
)

async function loadDownloadCenter() {
  loading.value = true
  try {
    const data = await ximoAppAPI.getDownloadCenter()
    apps.value = data.apps || []
    expandedHistoryKeys.value = new Set(
      [...expandedHistoryKeys.value].filter((key) => apps.value.some((app) => app.key === key))
    )
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('downloadCenter.loadFailed', 'Failed to load download center')))
  } finally {
    loading.value = false
  }
}

function toggleHistory(appKey: string) {
  const next = new Set(expandedHistoryKeys.value)
  if (next.has(appKey)) {
    next.delete(appKey)
  } else {
    next.add(appKey)
  }
  expandedHistoryKeys.value = next
}

function compareReleases(left: XimoAppDownloadRelease, right: XimoAppDownloadRelease): number {
  const version = compareVersions(right.version, left.version)
  if (version !== 0) return version
  if ((right.version_code || 0) !== (left.version_code || 0)) {
    return (right.version_code || 0) - (left.version_code || 0)
  }
  return releaseTimestamp(right) - releaseTimestamp(left)
}

function compareVersions(left: string, right: string): number {
  const leftParts = left.trim().replace(/^v/i, '').split('.')
  const rightParts = right.trim().replace(/^v/i, '').split('.')
  const length = Math.max(leftParts.length, rightParts.length)
  for (let idx = 0; idx < length; idx += 1) {
    const leftValue = leadingInt(leftParts[idx] || '')
    const rightValue = leadingInt(rightParts[idx] || '')
    if (leftValue !== rightValue) return leftValue - rightValue
  }
  return left.localeCompare(right)
}

function leadingInt(value: string): number {
  const match = value.match(/^\d+/)
  return match ? Number(match[0]) : 0
}

function releaseTimestamp(release: XimoAppDownloadRelease): number {
  const date = new Date(release.published_at || release.uploaded_at || '')
  return Number.isNaN(date.getTime()) ? 0 : date.getTime()
}

function releaseTime(release: XimoAppDownloadRelease): string {
  return formatDate(release.published_at || release.uploaded_at || '')
}

function formatBytes(value?: number) {
  if (!value || value <= 0) {
    return '-'
  }
  const units = ['B', 'KB', 'MB', 'GB']
  let size = value
  let unit = 0
  while (size >= 1024 && unit < units.length - 1) {
    size /= 1024
    unit += 1
  }
  return `${size.toFixed(unit === 0 ? 0 : 1)} ${units[unit]}`
}

function formatDate(value: string) {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return value
  }
  return date.toLocaleString()
}

loadDownloadCenter()
</script>

<style scoped>
.download-app-grid {
  grid-template-columns: repeat(auto-fit, minmax(min(100%, 22rem), 1fr));
}
</style>

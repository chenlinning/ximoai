<template>
  <AppLayout>
    <div class="mx-auto flex h-full w-full max-w-6xl flex-col gap-5 p-4 sm:p-6">
      <div class="flex flex-col justify-between gap-3 sm:flex-row sm:items-center">
        <div>
          <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">
            {{ t('downloadCenter.title', '下载中心') }}
          </h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
            {{ t('downloadCenter.description', '下载已发布的软件安装包，查看软件介绍和版本说明。') }}
          </p>
        </div>
        <button type="button" class="btn btn-secondary inline-flex items-center gap-2" :disabled="loading" @click="loadDownloadCenter">
          <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
          {{ t('downloadCenter.refresh', '刷新') }}
        </button>
      </div>

      <div v-if="loading" class="flex min-h-[260px] items-center justify-center rounded-lg border border-gray-200 bg-white text-sm text-gray-500 dark:border-dark-700 dark:bg-dark-800 dark:text-gray-400">
        <Icon name="refresh" size="md" class="mr-2 animate-spin" />
        {{ t('downloadCenter.loading', '正在加载下载中心...') }}
      </div>

      <div v-else-if="sortedApps.length === 0" class="rounded-lg border border-gray-200 bg-white px-6 py-16 text-center dark:border-dark-700 dark:bg-dark-800">
        <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
          {{ t('downloadCenter.empty', '暂无可下载软件') }}
        </h2>
        <p class="mt-2 text-sm text-gray-500 dark:text-gray-400">
          {{ t('downloadCenter.emptyDescription', '管理员上传并启用安装包后，会在这里显示下载入口。') }}
        </p>
      </div>

      <div v-else class="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <article
          v-for="app in sortedApps"
          :key="app.key"
          class="rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800"
        >
          <header class="border-b border-gray-100 px-5 py-4 dark:border-dark-700">
            <div class="flex items-start justify-between gap-3">
              <div class="min-w-0">
                <div class="flex flex-wrap items-center gap-2">
                  <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
                    {{ app.name || app.key }}
                  </h2>
                  <span class="rounded-full bg-primary-50 px-2 py-0.5 text-xs font-medium text-primary-700 dark:bg-primary-900/25 dark:text-primary-200">
                    {{ app.client_type || 'custom' }}
                  </span>
                </div>
                <p class="mt-2 text-sm leading-6 text-gray-600 dark:text-gray-300">
                  {{ app.description || t('downloadCenter.appDescriptionFallback', '暂无软件介绍。') }}
                </p>
              </div>
              <div class="shrink-0 text-right text-xs text-gray-500 dark:text-gray-400">
                {{ t('downloadCenter.releaseCount', { count: app.releases.length }) }}
              </div>
            </div>
          </header>

          <div class="divide-y divide-gray-100 dark:divide-dark-700">
            <section
              v-for="(release, index) in app.releases"
              :key="release.id || release.download_url"
              class="px-5 py-4"
            >
              <div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
                <div class="min-w-0">
                  <div class="flex flex-wrap items-center gap-2">
                    <h3 class="font-semibold text-gray-900 dark:text-white">
                      {{ release.version }}
                    </h3>
                    <span
                      v-if="index === 0"
                      class="rounded-full bg-emerald-50 px-2 py-0.5 text-xs font-medium text-emerald-700 dark:bg-emerald-900/25 dark:text-emerald-200"
                    >
                      {{ t('downloadCenter.latestVersion', '最新版本') }}
                    </span>
                    <span
                      v-if="release.force"
                      class="rounded-full bg-red-50 px-2 py-0.5 text-xs font-medium text-red-700 dark:bg-red-900/25 dark:text-red-200"
                    >
                      {{ t('downloadCenter.forceUpdate', '强制更新') }}
                    </span>
                  </div>
                  <dl class="mt-3 grid grid-cols-1 gap-x-4 gap-y-2 text-sm sm:grid-cols-2">
                    <div>
                      <dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('downloadCenter.target', '系统 / 架构') }}</dt>
                      <dd class="font-mono text-gray-900 dark:text-gray-100">{{ release.os }} / {{ release.arch }}</dd>
                    </div>
                    <div>
                      <dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('downloadCenter.channel', '渠道') }}</dt>
                      <dd class="font-mono text-gray-900 dark:text-gray-100">{{ release.channel }}</dd>
                    </div>
                    <div>
                      <dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('downloadCenter.locale', '语言') }}</dt>
                      <dd class="font-mono text-gray-900 dark:text-gray-100">{{ release.locale || 'all' }}</dd>
                    </div>
                    <div>
                      <dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('downloadCenter.packageType', '安装包') }}</dt>
                      <dd class="font-mono text-gray-900 dark:text-gray-100">{{ release.package_type || '-' }}</dd>
                    </div>
                    <div v-if="release.version_code">
                      <dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('downloadCenter.versionCode', '构建号') }}</dt>
                      <dd class="font-mono text-gray-900 dark:text-gray-100">{{ release.version_code }}</dd>
                    </div>
                    <div>
                      <dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('downloadCenter.fileSize', '文件大小') }}</dt>
                      <dd class="font-mono text-gray-900 dark:text-gray-100">{{ formatBytes(release.file_size) }}</dd>
                    </div>
                  </dl>
                </div>
                <a
                  class="btn btn-primary shrink-0"
                  :href="release.download_url"
                  target="_blank"
                  rel="noopener"
                >
                  {{ t('downloadCenter.download', '下载') }}
                </a>
              </div>

              <div class="mt-4 space-y-2 text-sm">
                <div>
                  <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('downloadCenter.releaseNotes', '版本说明') }}</div>
                  <p class="mt-1 whitespace-pre-wrap text-gray-700 dark:text-gray-300">
                    {{ release.notes || t('downloadCenter.noReleaseNotes', '暂无版本说明') }}
                  </p>
                </div>
                <div class="break-all font-mono text-xs text-gray-500 dark:text-gray-400">
                  {{ t('downloadCenter.sha256', 'SHA256') }}: {{ release.sha256 }}
                </div>
                <div v-if="release.published_at" class="font-mono text-xs text-gray-500 dark:text-gray-400">
                  {{ t('downloadCenter.publishedAt', '发布时间') }}: {{ formatDate(release.published_at) }}
                </div>
              </div>
            </section>
          </div>
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
import type { XimoAppDownloadApp } from '@/api/ximoapp'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'

const { t } = useI18n()
const appStore = useAppStore()
const loading = ref(false)
const apps = ref<XimoAppDownloadApp[]>([])

const sortedApps = computed(() =>
  [...apps.value].sort((left, right) => (left.name || left.key).localeCompare(right.name || right.key))
)

async function loadDownloadCenter() {
  loading.value = true
  try {
    const data = await ximoAppAPI.getDownloadCenter()
    apps.value = data.apps || []
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('downloadCenter.loadFailed', '下载中心加载失败')))
  } finally {
    loading.value = false
  }
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
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return value
  }
  return date.toLocaleString()
}

loadDownloadCenter()
</script>

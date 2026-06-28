<template>
  <AppLayout>
    <div class="mx-auto flex h-full w-full max-w-none flex-col gap-4 p-4 sm:p-6">
      <div class="flex flex-col justify-between gap-3 sm:flex-row sm:items-center">
        <div>
          <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">
            {{ t('ximoappUpdate.title', 'Application Center') }}
          </h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
            {{ t('ximoappUpdate.description', 'Maintain update versions, packages, and checksums for each app.') }}
          </p>
        </div>
        <button type="button" class="btn btn-secondary inline-flex items-center gap-2" :disabled="loading" @click="loadConfig">
          <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
          {{ t('common.refresh', 'Refresh') }}
        </button>
      </div>

      <div class="rounded-lg border border-blue-200 bg-blue-50 px-4 py-3 text-sm text-blue-800 dark:border-blue-800/40 dark:bg-blue-900/20 dark:text-blue-200">
        {{ t('ximoappUpdate.publicEndpoint', 'Update endpoint: POST /api/ximoapp/:appKey/version/latest; download path: /downloads/ximoapp/:file.') }}
      </div>

      <section class="rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
        <div class="flex flex-col gap-3 px-5 py-4 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
              {{ t('ximoappUpdate.globalSettings', 'Global Settings') }}
            </h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
              {{ t('ximoappUpdate.globalSettingsHint', 'When disabled, all update-check endpoints return 204.') }}
            </p>
          </div>
          <div class="flex flex-wrap items-center gap-3">
            <label class="inline-flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
              <input v-model="form.enabled" type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
              {{ t('ximoappUpdate.enabled', 'Enable update source') }}
            </label>
            <button type="button" class="btn btn-primary" :disabled="saving" @click="saveConfig">
              <Icon v-if="saving" name="refresh" size="sm" class="mr-2 animate-spin" />
              {{ saving ? t('common.saving', 'Saving') : t('common.save', 'Save') }}
            </button>
          </div>
        </div>
      </section>

      <section class="rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
        <div class="flex flex-col gap-3 border-b border-gray-100 px-5 py-4 dark:border-dark-700 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
              {{ t('ximoappUpdate.apps', 'Apps') }}
            </h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
              {{ t('ximoappUpdate.appsHint', 'Use appKey to separate clients. XimoDesk uses ximodesk.') }}
            </p>
          </div>
          <button type="button" class="btn btn-secondary" @click="addApp">
            {{ t('ximoappUpdate.addApp', 'Add App') }}
          </button>
        </div>

        <div v-if="loading" class="flex items-center justify-center py-16 text-gray-500 dark:text-gray-400">
          <Icon name="refresh" size="md" class="mr-2 animate-spin" />
          {{ t('common.loading', 'Loading') }}
        </div>

        <div v-else class="ximoapp-app-grid grid gap-4 p-5">
          <article
            v-for="(app, index) in form.apps"
            :key="`${app.key || 'app'}:${index}`"
            class="flex min-w-0 flex-col overflow-hidden rounded-lg border border-gray-200 bg-gray-50/70 dark:border-dark-700 dark:bg-dark-900/40"
          >
            <header class="border-b border-gray-100 px-4 py-3 dark:border-dark-700">
              <div class="flex items-start justify-between gap-3">
                <div class="min-w-0">
                  <h3 class="break-words text-base font-semibold text-gray-900 dark:text-white">
                    {{ app.name || app.key || t('ximoappUpdate.addApp', 'Add App') }}
                  </h3>
                  <p class="mt-1 truncate font-mono text-xs text-gray-500 dark:text-gray-400">
                    {{ app.key || 'app-key' }}
                  </p>
                </div>
                <div class="flex shrink-0 flex-wrap items-center justify-end gap-2">
                  <label class="inline-flex items-center gap-2 text-xs text-gray-700 dark:text-gray-300">
                    <input v-model="app.enabled" type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
                    {{ t('ximoappUpdate.releaseEnabled', 'Enabled') }}
                  </label>
                  <button type="button" class="btn btn-danger btn-sm" :disabled="deletingAppKey === normalizeAppKey(app.key)" @click="deleteApp(app)">
                    {{ deletingAppKey === normalizeAppKey(app.key) ? t('common.deleting', 'Deleting') : t('common.delete', 'Delete') }}
                  </button>
                </div>
              </div>
            </header>

            <section class="border-b border-gray-100 p-4 dark:border-dark-700">
              <div class="mb-4">
                <h4 class="text-sm font-semibold text-gray-900 dark:text-white">
                  {{ t('ximoappUpdate.appRegistration', 'App Registration') }}
                </h4>
                <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                  {{ t('ximoappUpdate.appRegistrationHint', 'Each app keeps its own package uploads and version list.') }}
                </p>
              </div>

              <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
                <div>
                  <label class="input-label">{{ t('ximoappUpdate.appKey', 'App Key') }}</label>
                  <input v-model.trim="app.key" class="input h-9 font-mono text-xs" placeholder="ximodesk" />
                </div>
                <div>
                  <label class="input-label">{{ t('ximoappUpdate.appName', 'Display Name') }}</label>
                  <input v-model.trim="app.name" class="input h-9" placeholder="XimoDesk" />
                </div>
                <div>
                  <label class="input-label">{{ t('ximoappUpdate.clientType', 'Client Type') }}</label>
                  <select v-model="app.client_type" class="input h-9">
                    <option value="desktop">desktop</option>
                    <option value="mobile">mobile</option>
                    <option value="custom">custom</option>
                  </select>
                </div>
                <div class="flex items-end">
                  <label class="flex min-h-9 items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
                    <input v-model="app.hidden" type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
                    <span>{{ t('ximoappUpdate.hiddenInDownloadCenter', 'Hide from download center') }}</span>
                  </label>
                </div>
                <div class="sm:col-span-2">
                  <label class="input-label">{{ t('ximoappUpdate.appDescription', 'App Description') }}</label>
                  <textarea
                    v-model.trim="app.description"
                    class="input min-h-[82px] text-sm"
                    :placeholder="t('ximoappUpdate.appDescriptionPlaceholder', 'Shown in the user download center')"
                  />
                </div>
              </div>

              <div class="mt-4 flex justify-end">
                <button type="button" class="btn btn-primary" :disabled="saving" @click="saveConfig">
                  <Icon v-if="saving" name="refresh" size="sm" class="mr-2 animate-spin" />
                  {{ saving ? t('common.saving', 'Saving') : t('common.save', 'Save') }}
                </button>
              </div>
            </section>

            <form class="border-b border-gray-100 p-4 dark:border-dark-700" @submit.prevent="uploadPackageForApp(app, index, $event)">
              <div class="mb-4">
                <h4 class="text-sm font-semibold text-gray-900 dark:text-white">
                  {{ t('ximoappUpdate.uploadPackage', 'Upload Package') }}
                </h4>
                <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                  {{ t('ximoappUpdate.uploadPackageHint', 'Uploads automatically calculate sha256 and generate a download URL.') }}
                </p>
              </div>

              <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
                <div>
                  <label class="input-label">{{ t('ximoappUpdate.version', 'Version') }}</label>
                  <input v-model.trim="uploadState(index, app).version" class="input h-9 font-mono text-xs" placeholder="1.0.1" required />
                </div>
                <div>
                  <label class="input-label">{{ t('ximoappUpdate.channel', 'Channel') }}</label>
                  <select v-model="uploadState(index, app).channel" class="input h-9">
                    <option value="stable">stable</option>
                    <option value="beta">beta</option>
                  </select>
                </div>
                <div>
                  <label class="input-label">{{ t('ximoappUpdate.os', 'OS') }}</label>
                  <select v-model="uploadState(index, app).os" class="input h-9">
                    <option value="windows">windows</option>
                    <option value="macos">macos</option>
                    <option value="linux">linux</option>
                    <option value="android">android</option>
                    <option value="ios">ios</option>
                  </select>
                </div>
                <div>
                  <label class="input-label">{{ t('ximoappUpdate.arch', 'Architecture') }}</label>
                  <select v-model="uploadState(index, app).arch" class="input h-9">
                    <option value="x86_64">x86_64</option>
                    <option value="aarch64">aarch64</option>
                    <option value="universal">universal</option>
                  </select>
                </div>
                <div>
                  <label class="input-label">{{ t('ximoappUpdate.locale', 'Language') }}</label>
                  <input v-model.trim="uploadState(index, app).locale" class="input h-9 font-mono text-xs" placeholder="zh-CN / en-US / all" />
                </div>
                <div>
                  <label class="input-label">{{ t('ximoappUpdate.packageType', 'Package Type') }}</label>
                  <select v-model="uploadState(index, app).package_type" class="input h-9">
                    <option value="">{{ t('ximoappUpdate.autoDetect', 'Auto detect') }}</option>
                    <option value="msi">msi</option>
                    <option value="zip">zip</option>
                    <option value="exe">exe</option>
                    <option value="dmg">dmg</option>
                    <option value="pkg">pkg</option>
                    <option value="apk">apk</option>
                    <option value="aab">aab</option>
                    <option value="ipa">ipa</option>
                  </select>
                </div>
                <div>
                  <label class="input-label">{{ t('ximoappUpdate.minSupportedVersion', 'Minimum Supported Version') }}</label>
                  <input v-model.trim="uploadState(index, app).min_supported_version" class="input h-9 font-mono text-xs" placeholder="1.0.0" />
                </div>
                <div>
                  <label class="input-label">{{ t('ximoappUpdate.minSupportedVersionCode', 'Minimum Supported Version Code') }}</label>
                  <input v-model.trim="uploadState(index, app).min_supported_version_code" class="input h-9 font-mono text-xs" placeholder="90" />
                </div>
              </div>

              <div class="mt-3 flex flex-wrap gap-4">
                <label class="inline-flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
                  <input v-model="uploadState(index, app).force" type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
                  {{ t('ximoappUpdate.force', 'Force update') }}
                </label>
                <label class="inline-flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
                  <input v-model="uploadState(index, app).enabled" type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
                  {{ t('ximoappUpdate.releaseEnabled', 'Enabled') }}
                </label>
              </div>

              <div class="mt-3">
                <label class="input-label">{{ t('ximoappUpdate.notes', 'Release Notes') }}</label>
                <textarea v-model.trim="uploadState(index, app).notes" class="input min-h-[82px] text-sm" placeholder="Fix connection stability and polish the interface" />
              </div>

              <div class="mt-3 grid grid-cols-1 gap-3 sm:grid-cols-[1fr_auto] sm:items-end">
                <div>
                  <label class="input-label">{{ t('ximoappUpdate.packageFile', 'Package File') }}</label>
                  <input
                    type="file"
                    class="input"
                    accept=".msi,.zip,.exe,.dmg,.pkg,.apk,.aab,.ipa"
                    required
                    @change="onFileChange(index, app, $event)"
                  />
                  <p class="input-hint">
                    {{ uploadState(index, app).fileName || t('ximoappUpdate.packageFileHint', 'Supports .msi / .zip / .exe / .dmg / .pkg / .apk / .aab / .ipa.') }}
                  </p>
                </div>
                <button type="submit" class="btn btn-primary min-w-[132px]" :disabled="isUploading(index)">
                  <Icon v-if="isUploading(index)" name="refresh" size="sm" class="mr-2 animate-spin" />
                  {{ isUploading(index) ? t('ximoappUpdate.uploading', 'Uploading') : t('ximoappUpdate.upload', 'Upload package') }}
                </button>
              </div>
            </form>

            <section class="flex flex-1 flex-col p-4">
              <div class="mb-3 flex items-start justify-between gap-3">
                <div>
                  <h4 class="text-sm font-semibold text-gray-900 dark:text-white">
                    {{ t('ximoappUpdate.releaseList', 'Published Packages') }}
                  </h4>
                  <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                    {{ t('ximoappUpdate.releaseListHint', 'Packages are grouped by app to avoid mixing different clients.') }}
                  </p>
                </div>
                <button type="button" class="btn btn-secondary btn-sm shrink-0" :disabled="saving" @click="saveConfig">
                  {{ t('ximoappUpdate.saveList', 'Save list settings') }}
                </button>
              </div>

              <div v-if="appReleases(app.key).length === 0" class="flex flex-1 items-center justify-center rounded-md border border-dashed border-gray-200 px-4 py-8 text-center text-sm text-gray-500 dark:border-dark-700 dark:text-gray-400">
                {{ t('ximoappUpdate.emptyReleases', 'No packages uploaded yet.') }}
              </div>

              <div v-else class="divide-y divide-gray-100 overflow-hidden rounded-md border border-gray-200 bg-white dark:divide-dark-700 dark:border-dark-700 dark:bg-dark-800">
                <section v-for="release in appReleases(app.key)" :key="release.id || release.download_url">
                  <div class="flex w-full items-center gap-3 px-4 py-3 hover:bg-gray-50 dark:hover:bg-dark-700/50">
                    <button
                      type="button"
                      class="flex min-w-0 flex-1 items-center gap-3 text-left"
                      @click="toggleReleaseDetails(release)"
                    >
                      <Icon :name="isReleaseExpanded(release) ? 'chevronDown' : 'chevronRight'" size="xs" class="shrink-0 text-gray-500 dark:text-gray-400" />
                      <span class="min-w-0 break-all font-semibold text-gray-900 dark:text-white">{{ release.version }}</span>
                    </button>
                    <div class="flex shrink-0 items-center gap-2">
                      <label class="inline-flex items-center gap-1 text-xs text-gray-700 dark:text-gray-300">
                        <input v-model="release.enabled" type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
                        {{ t('ximoappUpdate.releaseEnabled', 'Enabled') }}
                      </label>
                      <button type="button" class="btn btn-danger btn-sm" :disabled="deletingId === release.id" @click="deleteRelease(release)">
                        {{ deletingId === release.id ? t('common.deleting', 'Deleting') : t('common.delete', 'Delete') }}
                      </button>
                    </div>
                  </div>

                  <div v-if="isReleaseExpanded(release)" class="space-y-3 border-t border-gray-100 px-4 py-4 dark:border-dark-700">
                    <div class="grid grid-cols-1 gap-3 text-xs sm:grid-cols-2">
                      <div class="min-w-0">
                        <div class="text-gray-500 dark:text-gray-400">{{ t('ximoappUpdate.channel', 'Channel') }}</div>
                        <div class="mt-1 font-mono text-gray-800 dark:text-gray-200">{{ release.channel }}</div>
                      </div>
                      <div class="min-w-0">
                        <div class="text-gray-500 dark:text-gray-400">{{ t('ximoappUpdate.target', 'Target') }}</div>
                        <div class="mt-1 font-mono text-gray-800 dark:text-gray-200">{{ release.os }} / {{ release.arch }} / {{ release.locale || 'all' }}</div>
                        <div v-if="releaseTime(release)" class="mt-1 font-mono text-gray-500 dark:text-gray-400">{{ releaseTime(release) }}</div>
                      </div>
                      <div class="min-w-0">
                        <div class="text-gray-500 dark:text-gray-400">{{ t('ximoappUpdate.packageFile', 'Package File') }}</div>
                        <div class="mt-1 break-all font-mono text-gray-800 dark:text-gray-200">{{ release.file_name || '-' }}</div>
                        <div class="mt-1 text-gray-500 dark:text-gray-400">{{ formatBytes(release.file_size) }}</div>
                      </div>
                      <div class="min-w-0 sm:col-span-2">
                        <div class="text-gray-500 dark:text-gray-400">{{ t('ximoappUpdate.downloadUrl', 'Download URL') }}</div>
                        <div class="mt-1 break-all font-mono text-gray-800 dark:text-gray-200">{{ release.download_url }}</div>
                      </div>
                      <div class="min-w-0 sm:col-span-2">
                        <div class="text-gray-500 dark:text-gray-400">sha256</div>
                        <div class="mt-1 break-all font-mono text-gray-500 dark:text-gray-400">{{ release.sha256 }}</div>
                      </div>
                    </div>

                    <input v-model.trim="release.notes" class="input h-9 text-xs" :placeholder="t('ximoappUpdate.notes', 'Release Notes')" />

                    <div class="flex flex-wrap gap-2">
                      <button type="button" class="btn btn-secondary btn-sm" @click="copyUrl(release.download_url)">
                        {{ t('common.copy', 'Copy') }}
                      </button>
                      <label class="flex items-center gap-2 text-xs text-gray-700 dark:text-gray-300">
                        <input v-model="release.force" type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
                        {{ t('ximoappUpdate.force', 'Force update') }}
                      </label>
                    </div>
                  </div>
                </section>
              </div>
            </section>
          </article>
        </div>
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import { adminAPI } from '@/api/admin'
import type { XimoAppUpdateApp, XimoDeskUpdateConfig, XimoDeskUpdateRelease } from '@/api/admin/ximodeskUpdate'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'

interface UploadState {
  channel: string
  os: string
  arch: string
  locale: string
  version: string
  min_supported_version: string
  min_supported_version_code: string
  package_type: string
  notes: string
  force: boolean
  enabled: boolean
  file: File | null
  fileName: string
}

const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(false)
const saving = ref(false)
const uploadingKey = ref('')
const deletingId = ref('')
const deletingAppKey = ref('')
const expandedReleaseKeys = ref(new Set<string>())

const form = reactive<XimoDeskUpdateConfig>({
  enabled: false,
  apps: [],
  releases: []
})

const uploadForms = reactive<Record<string, UploadState>>({})

const releasesByApp = computed(() => {
  const map = new Map<string, XimoDeskUpdateRelease[]>()
  for (const release of form.releases || []) {
    const key = normalizeAppKey(release.app_key || 'ximodesk')
    if (!map.has(key)) {
      map.set(key, [])
    }
    map.get(key)!.push(release)
  }
  for (const releases of map.values()) {
    releases.sort(compareReleases)
  }
  return map
})

function normalizeApp(app: XimoAppUpdateApp): XimoAppUpdateApp {
  return {
    key: normalizeAppKey(app.key || ''),
    name: app.name || app.key || '',
    description: app.description || '',
    client_type: app.client_type || 'custom',
    response_mode: app.response_mode || 'standard',
    enabled: app.enabled !== false,
    hidden: app.hidden === true
  }
}

function normalizeRelease(release: XimoDeskUpdateRelease): XimoDeskUpdateRelease {
  return {
    ...release,
    app_key: normalizeAppKey(release.app_key || 'ximodesk'),
    enabled: release.enabled !== false,
    locale: release.locale || 'all',
    package_type: release.package_type || ''
  }
}

function normalizeAppKey(value: string): string {
  return (value || '').trim().toLowerCase().replace(/_/g, '-')
}

function appUploadKey(index: number): string {
  return `app-${index}`
}

function createUploadState(app?: XimoAppUpdateApp): UploadState {
  const isMobile = app?.client_type === 'mobile'
  return {
    channel: 'stable',
    os: isMobile ? 'android' : 'windows',
    arch: isMobile ? 'universal' : 'x86_64',
    locale: 'zh-CN',
    version: '',
    min_supported_version: '',
    min_supported_version_code: '',
    package_type: '',
    notes: '',
    force: false,
    enabled: true,
    file: null,
    fileName: ''
  }
}

function uploadState(index: number, app?: XimoAppUpdateApp): UploadState {
  const key = appUploadKey(index)
  if (!uploadForms[key]) {
    uploadForms[key] = createUploadState(app)
  }
  return uploadForms[key]
}

function isUploading(index: number): boolean {
  return uploadingKey.value === appUploadKey(index)
}

function appReleases(appKey: string): XimoDeskUpdateRelease[] {
  return releasesByApp.value.get(normalizeAppKey(appKey)) || []
}

function releaseKey(release: XimoDeskUpdateRelease): string {
  return release.id || [
    normalizeAppKey(release.app_key || 'ximodesk'),
    release.channel,
    release.os,
    release.arch,
    release.locale || 'all',
    release.package_type || '',
    release.version,
    release.download_url
  ].join(':')
}

function isReleaseExpanded(release: XimoDeskUpdateRelease): boolean {
  return expandedReleaseKeys.value.has(releaseKey(release))
}

function toggleReleaseDetails(release: XimoDeskUpdateRelease) {
  const key = releaseKey(release)
  const next = new Set(expandedReleaseKeys.value)
  if (next.has(key)) {
    next.delete(key)
  } else {
    next.add(key)
  }
  expandedReleaseKeys.value = next
}

function applyConfig(config: XimoDeskUpdateConfig) {
  form.enabled = !!config.enabled
  form.apps = (config.apps || []).map(normalizeApp)
  form.releases.splice(0, form.releases.length, ...((config.releases || []).map(normalizeRelease)))
  const keys = new Set(form.releases.map(releaseKey))
  expandedReleaseKeys.value = new Set([...expandedReleaseKeys.value].filter((key) => keys.has(key)))
  form.apps.forEach((app, index) => {
    uploadState(index, app)
  })
}

function buildPayload(): XimoDeskUpdateConfig {
  return {
    enabled: form.enabled,
    apps: (form.apps || []).map((app) => ({
      ...app,
      key: normalizeAppKey(app.key),
      name: (app.name || '').trim(),
      description: (app.description || '').trim(),
      client_type: app.client_type || 'custom',
      response_mode: app.response_mode || 'standard',
      enabled: app.enabled !== false,
      hidden: app.hidden === true
    })),
    releases: form.releases.map((release) => ({
      ...release,
      app_key: normalizeAppKey(release.app_key || 'ximodesk'),
      enabled: release.enabled !== false,
      locale: release.locale || 'all',
      package_type: release.package_type || undefined,
      published_at: release.published_at || undefined,
      uploaded_at: release.uploaded_at || undefined,
      file_name: release.file_name || undefined
    }))
  }
}

function addApp() {
  const index = (form.apps || []).length + 1
  if (!form.apps) {
    form.apps = []
  }
  form.apps.push({
    key: `custom-app-${index}`,
    name: `Custom App ${index}`,
    description: '',
    client_type: 'custom',
    response_mode: 'standard',
    enabled: true
  })
}

async function deleteApp(app: XimoAppUpdateApp) {
  const appKey = normalizeAppKey(app.key)
  if (!appKey) {
    appStore.showError(t('ximoappUpdate.appKeyRequired', 'Please enter the app key before deleting'))
    return
  }
  const ok = window.confirm(t('ximoappUpdate.confirmDeleteApp', 'Delete this app and all of its package releases?'))
  if (!ok) {
    return
  }
  deletingAppKey.value = appKey
  try {
    const config = await adminAPI.ximodeskUpdate.deleteApp(appKey)
    applyConfig(config)
    appStore.showSuccess(t('ximoappUpdate.appDeleted', 'App deleted'))
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('ximoappUpdate.deleteAppFailed', 'Failed to delete app')))
  } finally {
    deletingAppKey.value = ''
  }
}

function onFileChange(index: number, app: XimoAppUpdateApp, event: Event) {
  const input = event.target as HTMLInputElement
  const state = uploadState(index, app)
  state.file = input.files?.[0] || null
  state.fileName = state.file?.name || ''
}

async function loadConfig() {
  loading.value = true
  try {
    const config = await adminAPI.ximodeskUpdate.get()
    applyConfig(config)
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('ximoappUpdate.loadFailed', 'Failed to load XimoAPP update config')))
  } finally {
    loading.value = false
  }
}

async function saveConfig() {
  saving.value = true
  try {
    const saved = await adminAPI.ximodeskUpdate.update(buildPayload())
    applyConfig(saved)
    appStore.showSuccess(t('ximoappUpdate.saved', 'XimoAPP update config saved'))
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('ximoappUpdate.saveFailed', 'Failed to save XimoAPP update config')))
  } finally {
    saving.value = false
  }
}

async function uploadPackageForApp(app: XimoAppUpdateApp, index: number, event: Event) {
  const appKey = normalizeAppKey(app.key)
  if (!appKey) {
    appStore.showError(t('ximoappUpdate.appKeyRequired', 'Please enter the app key before uploading'))
    return
  }
  const state = uploadState(index, app)
  if (!state.file) {
    appStore.showError(t('ximoappUpdate.packageFileRequired', 'Please choose a package file'))
    return
  }
  uploadingKey.value = appUploadKey(index)
  try {
    const payload = new FormData()
    payload.append('file', state.file)
    payload.append('app_key', appKey)
    payload.append('channel', state.channel)
    payload.append('os', state.os)
    payload.append('arch', state.arch)
    payload.append('locale', state.locale || 'all')
    payload.append('version', state.version)
    payload.append('min_supported_version', state.min_supported_version)
    payload.append('min_supported_version_code', state.min_supported_version_code)
    payload.append('package_type', state.package_type)
    payload.append('notes', state.notes)
    payload.append('force', String(state.force))
    payload.append('enabled', String(state.enabled))

    const result = await adminAPI.ximodeskUpdate.uploadPackage(payload)
    applyConfig(result.config)
    state.version = ''
    state.notes = ''
    state.force = false
    state.file = null
    state.fileName = ''
    const input = (event.currentTarget as HTMLFormElement | null)?.querySelector<HTMLInputElement>('input[type="file"]')
    if (input) {
      input.value = ''
    }
    appStore.showSuccess(t('ximoappUpdate.uploaded', 'Package uploaded'))
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('ximoappUpdate.uploadFailed', 'Failed to upload package')))
  } finally {
    uploadingKey.value = ''
  }
}

async function deleteRelease(release: XimoDeskUpdateRelease) {
  if (!release.id) {
    appStore.showError(t('ximoappUpdate.releaseIdMissing', 'Release ID is missing, cannot delete'))
    return
  }
  const ok = window.confirm(t('ximoappUpdate.confirmDelete', 'Delete this package release?'))
  if (!ok) {
    return
  }
  deletingId.value = release.id
  try {
    const config = await adminAPI.ximodeskUpdate.deleteRelease(release.id)
    applyConfig(config)
    appStore.showSuccess(t('ximoappUpdate.deleted', 'Package release deleted'))
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('ximoappUpdate.deleteFailed', 'Failed to delete package release')))
  } finally {
    deletingId.value = ''
  }
}

async function copyUrl(url: string) {
  try {
    await navigator.clipboard.writeText(url)
    appStore.showSuccess(t('common.copied', 'Copied'))
  } catch {
    appStore.showError(t('common.copyFailed', 'Copy failed'))
  }
}

function compareReleases(left: XimoDeskUpdateRelease, right: XimoDeskUpdateRelease): number {
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

function releaseTimestamp(release: XimoDeskUpdateRelease): number {
  const date = new Date(release.published_at || release.uploaded_at || '')
  return Number.isNaN(date.getTime()) ? 0 : date.getTime()
}

function releaseTime(release: XimoDeskUpdateRelease): string {
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

loadConfig()
</script>

<style scoped>
.ximoapp-app-grid {
  align-items: start;
  grid-template-columns: repeat(auto-fill, minmax(min(100%, 32rem), 1fr));
}
</style>

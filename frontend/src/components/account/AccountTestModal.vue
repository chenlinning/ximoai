<template>
  <BaseDialog
    :show="show"
    :title="t('admin.accounts.testAccountConnection')"
    width="normal"
    @close="handleClose"
  >
    <div class="space-y-4">
      <!-- Account Info Card -->
      <div
        v-if="account"
        class="flex items-center justify-between rounded-xl border border-gray-200 bg-gradient-to-r from-gray-50 to-gray-100 p-3 dark:border-dark-500 dark:from-dark-700 dark:to-dark-600"
      >
        <div class="flex items-center gap-3">
          <div
            class="flex h-10 w-10 items-center justify-center rounded-lg bg-gradient-to-br from-primary-500 to-primary-600"
          >
            <Icon name="play" size="md" class="text-white" :stroke-width="2" />
          </div>
          <div>
            <div class="font-semibold text-gray-900 dark:text-gray-100">{{ account.name }}</div>
            <div class="flex items-center gap-1.5 text-xs text-gray-500 dark:text-gray-400">
              <span
                class="rounded bg-gray-200 px-1.5 py-0.5 text-[10px] font-medium uppercase dark:bg-dark-500"
              >
                {{ account.type }}
              </span>
              <span>{{ t('admin.accounts.account') }}</span>
            </div>
          </div>
        </div>
        <span
          :class="[
            'rounded-full px-2.5 py-1 text-xs font-semibold',
            account.status === 'active'
              ? 'bg-green-100 text-green-700 dark:bg-green-500/20 dark:text-green-400'
              : 'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-400'
          ]"
        >
          {{ account.status }}
        </span>
      </div>

      <div class="space-y-1.5">
        <label class="text-sm font-medium text-gray-700 dark:text-gray-300">
          {{ t('admin.accounts.selectTestModel') }}
        </label>
        <Select
          v-model="selectedModelId"
          :options="availableModels"
          :disabled="loadingModels || status === 'connecting'"
          value-key="id"
          label-key="display_name"
          :placeholder="loadingModels ? t('common.loading') + '...' : t('admin.accounts.selectTestModel')"
        />
      </div>

      <div v-if="isOpenAIAccount" class="space-y-1.5">
        <label class="text-sm font-medium text-gray-700 dark:text-gray-300">
          {{ t('admin.accounts.testFormat') }}
        </label>
        <Select
          v-model="testType"
          :options="testFormatOptions"
          :disabled="status === 'connecting'"
        />
      </div>

      <div v-if="isOpenAIAccount && effectiveTestType === 'text'" class="space-y-1.5">
        <label class="text-sm font-medium text-gray-700 dark:text-gray-300">
          {{ t('admin.accounts.openai.testMode') }}
        </label>
        <Select
          v-model="testMode"
          :options="openAITestModeOptions"
          :disabled="status === 'connecting'"
        />
      </div>

      <div v-if="isImageTest" class="space-y-1.5">
        <TextArea
          v-model="testPrompt"
          :label="t('admin.accounts.imagePromptLabel')"
          :placeholder="t('admin.accounts.imagePromptPlaceholder')"
          :hint="t('admin.accounts.imageTestHint')"
          :disabled="status === 'connecting'"
          rows="3"
        />
      </div>

      <div v-if="isAudioTest" class="space-y-1.5">
        <TextArea
          v-model="audioInput"
          :label="t('admin.accounts.audioInputLabel')"
          :placeholder="t('admin.accounts.audioInputPlaceholder')"
          :disabled="status === 'connecting'"
          rows="2"
        />
      </div>

      <div v-if="isVideoTest" class="space-y-3">
        <TextArea
          v-model="videoPrompt"
          :label="t('admin.accounts.videoPromptLabel')"
          :placeholder="t('admin.accounts.videoPromptPlaceholder')"
          :disabled="status === 'connecting'"
          rows="3"
        />
        <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <label class="space-y-1.5">
            <span class="block text-sm font-medium text-gray-700 dark:text-gray-300">
              {{ t('admin.accounts.videoSecondsLabel') }}
            </span>
            <input
              v-model.number="videoSeconds"
              type="number"
              min="1"
              max="20"
              :disabled="status === 'connecting'"
              class="h-10 w-full rounded-lg border border-gray-300 bg-white px-3 text-sm text-gray-900 outline-none transition-colors focus:border-primary-500 focus:ring-2 focus:ring-primary-500/20 disabled:cursor-not-allowed disabled:bg-gray-100 dark:border-dark-500 dark:bg-dark-700 dark:text-gray-100 dark:disabled:bg-dark-600"
            />
          </label>
          <div class="space-y-1.5">
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
              {{ t('admin.accounts.videoSizeLabel') }}
            </label>
            <Select
              v-model="videoSize"
              :options="videoSizeOptions"
              :disabled="status === 'connecting'"
            />
          </div>
        </div>
      </div>

      <!-- Terminal Output -->
      <div class="group relative">
        <div
          ref="terminalRef"
          class="max-h-[240px] min-h-[120px] overflow-y-auto rounded-xl border border-gray-700 bg-gray-900 p-4 font-mono text-sm dark:border-gray-800 dark:bg-black"
        >
          <!-- Status Line -->
          <div v-if="status === 'idle'" class="flex items-center gap-2 text-gray-500">
            <Icon name="play" size="sm" :stroke-width="2" />
            <span>{{ t('admin.accounts.readyToTest') }}</span>
          </div>
          <div v-else-if="status === 'connecting'" class="flex items-center gap-2 text-yellow-400">
            <Icon name="refresh" size="sm" class="animate-spin" :stroke-width="2" />
            <span>{{ t('admin.accounts.connectingToApi') }}</span>
          </div>

          <!-- Output Lines -->
          <div v-for="(line, index) in outputLines" :key="index" :class="line.class">
            {{ line.text }}
          </div>

          <!-- Streaming Content -->
          <div v-if="streamingContent" class="text-green-400">
            {{ streamingContent }}<span class="animate-pulse">_</span>
          </div>

          <!-- Result Status -->
          <div
            v-if="status === 'success'"
            class="mt-3 flex items-center gap-2 border-t border-gray-700 pt-3 text-green-400"
          >
            <Icon name="check" size="sm" :stroke-width="2" />
            <span>{{ t('admin.accounts.testCompleted') }}</span>
          </div>
          <div
            v-else-if="status === 'error'"
            class="mt-3 flex items-center gap-2 border-t border-gray-700 pt-3 text-red-400"
          >
            <Icon name="x" size="sm" :stroke-width="2" />
            <span>{{ errorMessage }}</span>
          </div>
        </div>

        <!-- Copy Button -->
        <button
          v-if="outputLines.length > 0"
          @click="copyOutput"
          class="absolute right-2 top-2 rounded-lg bg-gray-800/80 p-1.5 text-gray-400 opacity-0 transition-all hover:bg-gray-700 hover:text-white group-hover:opacity-100"
          :title="t('admin.accounts.copyOutput')"
        >
          <Icon name="link" size="sm" :stroke-width="2" />
        </button>
      </div>

      <div v-if="generatedAudios.length > 0" class="space-y-2">
        <div class="text-xs font-medium text-gray-600 dark:text-gray-300">
          {{ t('admin.accounts.audioPreview') }}
        </div>
        <div class="space-y-2">
          <div
            v-for="(audio, index) in generatedAudios"
            :key="`${audio.url}-${index}`"
            class="rounded-xl border border-gray-200 bg-white p-3 shadow-sm dark:border-dark-500 dark:bg-dark-700"
          >
            <audio :src="audio.url" controls class="w-full" />
            <div class="mt-2 text-xs text-gray-500 dark:text-gray-300">
              {{ audio.mimeType || 'audio/*' }}
            </div>
          </div>
        </div>
      </div>

      <div v-if="generatedVideos.length > 0" class="space-y-2">
        <div class="text-xs font-medium text-gray-600 dark:text-gray-300">
          {{ t('admin.accounts.videoPreview') }}
        </div>
        <div class="space-y-3">
          <div
            v-for="(video, index) in generatedVideos"
            :key="`${video.url}-${index}`"
            class="rounded-xl border border-gray-200 bg-white p-3 shadow-sm dark:border-dark-500 dark:bg-dark-700"
          >
            <video
              :src="video.url"
              controls
              class="max-h-[420px] w-full rounded-lg bg-black object-contain"
            />
            <div class="mt-2 flex items-center justify-between gap-3 text-xs text-gray-500 dark:text-gray-300">
              <span>{{ video.mimeType || 'video/*' }}</span>
              <a
                v-if="isExternalURL(video.url)"
                :href="video.url"
                target="_blank"
                rel="noopener noreferrer"
                class="text-primary-600 hover:text-primary-700 dark:text-primary-400"
              >
                {{ t('admin.accounts.openVideoResult') }}
              </a>
            </div>
          </div>
        </div>
      </div>

      <div v-if="generatedImages.length > 0" class="space-y-2">
        <div class="text-xs font-medium text-gray-600 dark:text-gray-300">
          {{ t('admin.accounts.imagePreview') }}
        </div>
        <div class="flex flex-wrap justify-center gap-3">
          <div
            v-for="(image, index) in generatedImages"
            :key="`${image.url}-${index}`"
            class="group/img relative cursor-pointer overflow-hidden rounded-xl border border-gray-200 bg-white shadow-sm transition hover:border-primary-300 hover:shadow-md dark:border-dark-500 dark:bg-dark-700"
            @click="previewImageUrl = image.url"
          >
            <img :src="image.url" :alt="`test-image-${index + 1}`" class="max-h-[360px] w-full object-contain" />
            <div class="absolute inset-0 flex items-center justify-center bg-black/0 transition-colors group-hover/img:bg-black/20">
              <Icon name="eye" size="lg" class="text-white opacity-0 drop-shadow-lg transition-opacity group-hover/img:opacity-100" :stroke-width="2" />
            </div>
            <div class="border-t border-gray-100 px-3 py-1.5 text-xs text-gray-500 dark:border-dark-500 dark:text-gray-300">
              {{ image.mimeType || 'image/*' }}
            </div>
          </div>
        </div>
      </div>

      <!-- Image Lightbox -->
      <Teleport to="body">
        <Transition name="fade">
          <div
            v-if="previewImageUrl"
            class="fixed inset-0 z-[100] flex items-center justify-center bg-black/80 p-4"
            @click.self="previewImageUrl = ''"
          >
            <button
              class="absolute right-4 top-4 rounded-full bg-black/50 p-2 text-white transition-colors hover:bg-black/70"
              @click="previewImageUrl = ''"
            >
              <Icon name="x" size="lg" :stroke-width="2" />
            </button>
            <img
              :src="previewImageUrl"
              alt="preview"
              class="max-h-[90vh] max-w-[90vw] rounded-lg object-contain shadow-2xl"
            />
          </div>
        </Transition>
      </Teleport>

      <!-- Test Info -->
      <div class="flex items-center justify-between px-1 text-xs text-gray-500 dark:text-gray-400">
        <div class="flex items-center gap-3">
          <span class="flex items-center gap-1">
            <Icon name="grid" size="sm" :stroke-width="2" />
            {{ t('admin.accounts.testModel') }}
          </span>
        </div>
        <span class="flex items-center gap-1">
          <Icon name="chat" size="sm" :stroke-width="2" />
          {{ testInfoText }}
        </span>
      </div>
    </div>

    <template #footer>
      <div class="flex justify-end gap-3">
        <button
          @click="handleClose"
          class="rounded-lg bg-gray-100 px-4 py-2 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-200 dark:bg-dark-600 dark:text-gray-300 dark:hover:bg-dark-500"
        >
          {{ t('common.close') }}
        </button>
        <button
          @click="startTest"
          :disabled="status === 'connecting' || !selectedModelId"
          :class="[
            'flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-medium transition-all',
            status === 'connecting' || !selectedModelId
              ? 'cursor-not-allowed bg-primary-400 text-white'
              : status === 'success'
                ? 'bg-green-500 text-white hover:bg-green-600'
                : status === 'error'
                  ? 'bg-orange-500 text-white hover:bg-orange-600'
                  : 'bg-primary-500 text-white hover:bg-primary-600'
          ]"
        >
          <Icon
            v-if="status === 'connecting'"
            name="refresh"
            size="sm"
            class="animate-spin"
            :stroke-width="2"
          />
          <Icon v-else-if="status === 'idle'" name="play" size="sm" :stroke-width="2" />
          <Icon v-else name="refresh" size="sm" :stroke-width="2" />
          <span>
            {{
              status === 'connecting'
                ? t('admin.accounts.testing')
                : status === 'idle'
                  ? t('admin.accounts.startTest')
                  : t('admin.accounts.retry')
            }}
          </span>
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Select from '@/components/common/Select.vue'
import TextArea from '@/components/common/TextArea.vue'
import { Icon } from '@/components/icons'
import { useClipboard } from '@/composables/useClipboard'
import { adminAPI } from '@/api/admin'
import type { Account, ClaudeModel, Platform } from '@/types'

const { t } = useI18n()
const { copyToClipboard } = useClipboard()

interface OutputLine {
  text: string
  class: string
}

interface PreviewImage {
  url: string
  mimeType?: string
}

interface PreviewAudio {
  url: string
  mimeType?: string
}

interface PreviewVideo {
  url: string
  mimeType?: string
}

type AccountTestType = 'auto' | 'text' | 'image' | 'audio' | 'video'

const props = defineProps<{
  show: boolean
  account: Account | null
}>()

const emit = defineEmits<{
  (e: 'close'): void
}>()

const terminalRef = ref<HTMLElement | null>(null)
const status = ref<'idle' | 'connecting' | 'success' | 'error'>('idle')
const outputLines = ref<OutputLine[]>([])
const streamingContent = ref('')
const errorMessage = ref('')
const availableModels = ref<ClaudeModel[]>([])
const selectedModelId = ref('')
const testPrompt = ref('')
const audioInput = ref('hi')
const videoPrompt = ref('A tiny test video of a sunrise over mountains.')
const videoSeconds = ref(4)
const videoSize = ref('720x1280')
const loadingModels = ref(false)
let abortController: AbortController | null = null
const generatedImages = ref<PreviewImage[]>([])
const generatedAudios = ref<PreviewAudio[]>([])
const generatedVideos = ref<PreviewVideo[]>([])
const testMode = ref<'default' | 'compact'>('default')
const testType = ref<AccountTestType>('auto')
const activeTestType = ref<AccountTestType>('text')
const platformProtocols = ref<Record<string, string>>({})
const isOpenAIAccount = computed(() => {
  if (!props.account) return false
  if (props.account.platform === 'openai') return true
  const protocol = platformProtocols.value[props.account.platform]
  if (protocol) {
    return protocol === 'openai' || protocol === 'openai_compatible'
  }
  return props.account.type === 'apikey' && !['anthropic', 'gemini', 'antigravity'].includes(props.account.platform)
})
const openAITestModeOptions = computed(() => [
  { value: 'default', label: t('admin.accounts.openai.testModeDefault') },
  { value: 'compact', label: t('admin.accounts.openai.testModeCompact') }
])
const testFormatOptions = computed(() => [
  { value: 'auto', label: t('admin.accounts.testFormatAuto') },
  { value: 'text', label: t('admin.accounts.testFormatText') },
  { value: 'image', label: t('admin.accounts.testFormatImage') },
  { value: 'audio', label: t('admin.accounts.testFormatAudio') },
  { value: 'video', label: t('admin.accounts.testFormatVideo') }
])
const videoSizeOptions = [
  { value: '720x1280', label: '720x1280' },
  { value: '1280x720', label: '1280x720' },
  { value: '1024x1024', label: '1024x1024' }
]
const previewImageUrl = ref('')
const prioritizedGeminiModels = ['gemini-3.1-flash-image', 'gemini-2.5-flash-image', 'gemini-3.5-flash', 'gemini-2.5-flash', 'gemini-2.5-pro', 'gemini-3-flash-preview', 'gemini-3-pro-preview', 'gemini-2.0-flash']
const isOpenAIImageModel = (modelID: string) => modelID.startsWith('gpt-image-')
const isOpenAIAudioModel = (modelID: string) => modelID.includes('audio') || modelID.includes('realtime') || modelID.includes('tts') || modelID.startsWith('whisper')
const isOpenAIVideoModel = (modelID: string) => modelID.includes('video') || modelID.startsWith('sora-') || modelID.startsWith('veo') || modelID.includes('t2v') || modelID.includes('i2v') || modelID.includes('r2v')
const supportsGeminiImageTest = computed(() => {
  const modelID = selectedModelId.value.toLowerCase()
  if (!modelID.startsWith('gemini-') || !modelID.includes('-image')) return false

  return props.account?.platform === 'gemini' || (props.account?.platform === 'antigravity' && props.account?.type === 'apikey')
})

const autoDetectedTestType = computed<AccountTestType>(() => {
  const modelID = selectedModelId.value.toLowerCase()
  if (isOpenAIImageModel(modelID)) return 'image'
  if (isOpenAIAudioModel(modelID)) return 'audio'
  if (isOpenAIVideoModel(modelID)) return 'video'
  return 'text'
})

const effectiveTestType = computed<AccountTestType>(() => {
  if (!isOpenAIAccount.value) return supportsGeminiImageTest.value ? 'image' : 'text'
  return testType.value === 'auto' ? autoDetectedTestType.value : testType.value
})
const isImageTest = computed(() => supportsGeminiImageTest.value || (isOpenAIAccount.value && effectiveTestType.value === 'image'))
const isAudioTest = computed(() => isOpenAIAccount.value && effectiveTestType.value === 'audio')
const isVideoTest = computed(() => isOpenAIAccount.value && effectiveTestType.value === 'video')
const testInfoText = computed(() => {
  switch (effectiveTestType.value) {
    case 'image':
      return t('admin.accounts.imageTestMode')
    case 'audio':
      return t('admin.accounts.audioTestMode')
    case 'video':
      return t('admin.accounts.videoTestMode')
    default:
      return t('admin.accounts.testPrompt')
  }
})

const sortTestModels = (models: ClaudeModel[]) => {
  const priorityMap = new Map(prioritizedGeminiModels.map((id, index) => [id, index]))

  return [...models].sort((a, b) => {
    const aPriority = priorityMap.get(a.id) ?? Number.MAX_SAFE_INTEGER
    const bPriority = priorityMap.get(b.id) ?? Number.MAX_SAFE_INTEGER
    if (aPriority !== bPriority) return aPriority - bPriority
    return 0
  })
}

// Load available models when modal opens
watch(
  () => props.show,
  async (newVal) => {
    if (newVal && props.account) {
      testPrompt.value = ''
      audioInput.value = 'hi'
      videoPrompt.value = 'A tiny test video of a sunrise over mountains.'
      videoSeconds.value = 4
      videoSize.value = '720x1280'
      testMode.value = 'default'
      testType.value = 'auto'
      resetState()
      await loadPlatformProtocols()
      await loadAvailableModels()
    } else {
      abortStream()
    }
  }
)

watch([selectedModelId, testType], () => {
  if (isImageTest.value && !testPrompt.value.trim()) {
    testPrompt.value = t('admin.accounts.imagePromptDefault')
  }
  if (isAudioTest.value && !audioInput.value.trim()) {
    audioInput.value = 'hi'
  }
  if (isVideoTest.value && !videoPrompt.value.trim()) {
    videoPrompt.value = 'A tiny test video of a sunrise over mountains.'
  }
})

const loadPlatformProtocols = async () => {
  if (Object.keys(platformProtocols.value).length > 0) return
  try {
    const platforms = await adminAPI.platforms.list(true)
    platformProtocols.value = platforms.reduce<Record<string, string>>((acc, platform: Platform) => {
      acc[platform.slug] = platform.protocol
      return acc
    }, {})
  } catch (error) {
    console.error('Failed to load platform protocols:', error)
  }
}

const loadAvailableModels = async () => {
  if (!props.account) return

  loadingModels.value = true
  selectedModelId.value = '' // Reset selection before loading
  try {
    const models = await adminAPI.accounts.getAvailableModels(props.account.id)
    availableModels.value = props.account.platform === 'gemini' || props.account.platform === 'antigravity'
      ? sortTestModels(models)
      : models
    // Default selection by platform
    if (availableModels.value.length > 0) {
      if (props.account.platform === 'gemini') {
        selectedModelId.value = availableModels.value[0].id
      } else {
        // Try to select Sonnet as default, otherwise use first model
        const sonnetModel = availableModels.value.find((m) => m.id.includes('sonnet'))
        selectedModelId.value = sonnetModel?.id || availableModels.value[0].id
      }
    }
  } catch (error) {
    console.error('Failed to load available models:', error)
    // Fallback to empty list
    availableModels.value = []
    selectedModelId.value = ''
  } finally {
    loadingModels.value = false
  }
}

const resetState = () => {
  status.value = 'idle'
  outputLines.value = []
  streamingContent.value = ''
  errorMessage.value = ''
  generatedImages.value = []
  generatedAudios.value = []
  generatedVideos.value = []
  previewImageUrl.value = ''
}

const isExternalURL = (value: string) => /^https?:\/\//i.test(value)

const handleClose = () => {
  abortStream()
  emit('close')
}

const abortStream = () => {
  if (abortController) {
    abortController.abort()
    abortController = null
  }
}

const addLine = (text: string, className: string = 'text-gray-300') => {
  outputLines.value.push({ text, class: className })
  scrollToBottom()
}

const scrollToBottom = async () => {
  await nextTick()
  if (terminalRef.value) {
    terminalRef.value.scrollTop = terminalRef.value.scrollHeight
  }
}

const startTest = async () => {
  if (!props.account || !selectedModelId.value) return

  activeTestType.value = effectiveTestType.value
  resetState()
  status.value = 'connecting'
  addLine(t('admin.accounts.startingTestForAccount', { name: props.account.name }), 'text-blue-400')
  addLine(t('admin.accounts.testAccountTypeLabel', { type: props.account.type }), 'text-gray-400')
  addLine('', 'text-gray-300')

  abortStream()

  abortController = new AbortController()

  try {
    // Create EventSource for SSE
    const url = `/api/v1/admin/accounts/${props.account.id}/test`

    // Use fetch with streaming for SSE since EventSource doesn't support POST
    const response = await fetch(url, {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${localStorage.getItem('auth_token')}`,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        model_id: selectedModelId.value,
        test_type: activeTestType.value,
        prompt: buildTestPrompt(activeTestType.value),
        seconds: activeTestType.value === 'video' ? videoSeconds.value : undefined,
        size: activeTestType.value === 'video' ? videoSize.value : undefined,
        mode: isOpenAIAccount.value && activeTestType.value === 'text' ? testMode.value : 'default'
      }),
      signal: abortController.signal
    })

    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`)
    }

    const reader = response.body?.getReader()
    if (!reader) {
      throw new Error('No response body')
    }

    const decoder = new TextDecoder()
    let buffer = ''

    while (true) {
      const { done, value } = await reader.read()
      if (done) break

      buffer += decoder.decode(value, { stream: true })
      const lines = buffer.split('\n')
      buffer = lines.pop() || ''

      for (const line of lines) {
        if (line.startsWith('data: ')) {
          const jsonStr = line.slice(6).trim()
          if (jsonStr) {
            try {
              const event = JSON.parse(jsonStr)
              handleEvent(event)
            } catch (e) {
              console.error('Failed to parse SSE event:', e)
            }
          }
        }
      }
    }
  } catch (error: unknown) {
    if (error instanceof DOMException && error.name === 'AbortError') {
      status.value = 'idle'
      return
    }
    status.value = 'error'
    const msg = error instanceof Error ? error.message : 'Unknown error'
    errorMessage.value = msg
    addLine(`Error: ${msg}`, 'text-red-400')
  }
}

const buildTestPrompt = (type: AccountTestType) => {
  switch (type) {
    case 'image':
      return testPrompt.value.trim()
    case 'audio':
      return audioInput.value.trim()
    case 'video':
      return videoPrompt.value.trim()
    default:
      return ''
  }
}

const handleEvent = (event: {
  type: string
  text?: string
  model?: string
  success?: boolean
  error?: string
  image_url?: string
  audio_url?: string
  video_url?: string
  mime_type?: string
}) => {
  switch (event.type) {
    case 'test_start':
      addLine(t('admin.accounts.connectedToApi'), 'text-green-400')
      if (event.model) {
        addLine(t('admin.accounts.usingModel', { model: event.model }), 'text-cyan-400')
      }
      addLine(
        activeTestType.value === 'image'
          ? t('admin.accounts.sendingImageRequest')
          : activeTestType.value === 'audio'
            ? t('admin.accounts.sendingAudioRequest')
            : activeTestType.value === 'video'
              ? t('admin.accounts.sendingVideoRequest')
              : t('admin.accounts.sendingTestMessage'),
        'text-gray-400'
      )
      addLine('', 'text-gray-300')
      addLine(t('admin.accounts.response'), 'text-yellow-400')
      break

    case 'content':
      if (event.text) {
        if (activeTestType.value === 'text') {
          streamingContent.value += event.text
          scrollToBottom()
        } else {
          addLine(event.text, 'text-green-300')
        }
      }
      break

    case 'status':
      if (event.text) {
        addLine(event.text, 'text-cyan-300')
      }
      break

    case 'image':
      if (event.image_url) {
        generatedImages.value.push({
          url: event.image_url,
          mimeType: event.mime_type
        })
        addLine(t('admin.accounts.imageReceived', { count: generatedImages.value.length }), 'text-purple-300')
      }
      break

    case 'audio':
      if (event.audio_url) {
        generatedAudios.value.push({
          url: event.audio_url,
          mimeType: event.mime_type
        })
        addLine(t('admin.accounts.audioReceived', { count: generatedAudios.value.length }), 'text-purple-300')
      }
      break

    case 'video':
      if (event.video_url) {
        generatedVideos.value.push({
          url: event.video_url,
          mimeType: event.mime_type
        })
        addLine(t('admin.accounts.videoReceived', { count: generatedVideos.value.length }), 'text-purple-300')
      }
      break

    case 'test_complete':
      // Move streaming content to output lines
      if (streamingContent.value) {
        addLine(streamingContent.value, 'text-green-300')
        streamingContent.value = ''
      }
      if (event.success) {
        status.value = 'success'
      } else {
        status.value = 'error'
        errorMessage.value = event.error || 'Test failed'
      }
      break

    case 'error':
      status.value = 'error'
      errorMessage.value = event.error || 'Unknown error'
      if (streamingContent.value) {
        addLine(streamingContent.value, 'text-green-300')
        streamingContent.value = ''
      }
      break
  }
}

const copyOutput = () => {
  const text = outputLines.value.map((l) => l.text).join('\n')
  copyToClipboard(text, t('admin.accounts.outputCopied'))
}
</script>

<style>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>

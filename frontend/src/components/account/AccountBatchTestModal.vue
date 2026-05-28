<template>
  <BaseDialog
    :show="show"
    :title="t('admin.accounts.batchTest.title')"
    width="wide"
    @close="handleClose"
  >
    <div class="space-y-4">
      <div class="rounded-lg border border-gray-200 bg-gray-50 p-3 text-sm text-gray-700 dark:border-dark-500 dark:bg-dark-700 dark:text-gray-200">
        {{ t('admin.accounts.batchTest.selectedCount', { count: accountIds.length }) }}
      </div>

      <div
        v-if="resultSummary"
        class="flex flex-wrap items-center gap-3 rounded-lg border border-gray-200 px-3 py-2 text-sm dark:border-dark-500"
      >
        <span class="font-medium text-gray-700 dark:text-gray-200">
          {{ t('admin.accounts.batchTest.summary') }}
        </span>
        <span class="text-green-600 dark:text-green-400">
          {{ t('admin.accounts.batchTest.successCount', { count: resultSummary.success }) }}
        </span>
        <span class="text-red-600 dark:text-red-400">
          {{ t('admin.accounts.batchTest.failedCount', { count: resultSummary.failed }) }}
        </span>
      </div>

      <div class="grid grid-cols-1 gap-3 md:grid-cols-2">
        <div class="space-y-1.5">
          <label class="text-sm font-medium text-gray-700 dark:text-gray-300">
            {{ t('admin.accounts.selectTestModel') }}
          </label>
          <Select
            v-model="selectedModelId"
            :options="modelOptions"
            :disabled="modelSelectDisabled"
            :placeholder="modelSelectPlaceholder"
            searchable
          />
        </div>

        <div class="space-y-1.5">
          <label class="text-sm font-medium text-gray-700 dark:text-gray-300">
            {{ t('admin.accounts.testFormat') }}
          </label>
          <Select v-model="testType" :options="testFormatOptions" :disabled="testing" />
        </div>
      </div>

      <div v-if="effectiveTestType === 'text'" class="space-y-1.5">
        <label class="text-sm font-medium text-gray-700 dark:text-gray-300">
          {{ t('admin.accounts.openai.testMode') }}
        </label>
        <Select v-model="testMode" :options="openAITestModeOptions" :disabled="testing" />
      </div>

      <TextArea
        v-if="effectiveTestType === 'image'"
        v-model="imagePrompt"
        :label="t('admin.accounts.imagePromptLabel')"
        :placeholder="t('admin.accounts.imagePromptPlaceholder')"
        :disabled="testing"
        rows="3"
      />

      <TextArea
        v-if="effectiveTestType === 'audio'"
        v-model="audioInput"
        :label="t('admin.accounts.audioInputLabel')"
        :placeholder="t('admin.accounts.audioInputPlaceholder')"
        :disabled="testing"
        rows="2"
      />

      <div v-if="effectiveTestType === 'video'" class="space-y-3">
        <TextArea
          v-model="videoPrompt"
          :label="t('admin.accounts.videoPromptLabel')"
          :placeholder="t('admin.accounts.videoPromptPlaceholder')"
          :disabled="testing"
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
              :disabled="testing"
              class="h-10 w-full rounded-lg border border-gray-300 bg-white px-3 text-sm text-gray-900 outline-none transition-colors focus:border-primary-500 focus:ring-2 focus:ring-primary-500/20 disabled:cursor-not-allowed disabled:bg-gray-100 dark:border-dark-500 dark:bg-dark-700 dark:text-gray-100 dark:disabled:bg-dark-600"
            />
          </label>
          <div class="space-y-1.5">
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
              {{ t('admin.accounts.videoSizeLabel') }}
            </label>
            <Select v-model="videoSize" :options="videoSizeOptions" :disabled="testing" />
          </div>
        </div>
      </div>

      <div class="max-h-72 overflow-y-auto rounded-lg border border-gray-200 dark:border-dark-500">
        <div
          v-for="item in selectedAccountItems"
          :key="item.id"
          class="flex items-center justify-between gap-3 border-b border-gray-100 px-3 py-2 text-sm last:border-b-0 dark:border-dark-600"
        >
          <div class="min-w-0">
            <div class="truncate font-medium text-gray-900 dark:text-gray-100">{{ item.name }}</div>
            <div class="truncate text-xs text-gray-500 dark:text-gray-400">
              {{ item.subtitle }}
            </div>
          </div>
          <div class="flex shrink-0 items-center gap-2">
            <Icon
              v-if="resultById[item.id]?.success"
              name="check"
              size="sm"
              class="text-green-500"
              :stroke-width="2"
            />
            <Icon
              v-else-if="resultById[item.id]?.error"
              name="x"
              size="sm"
              class="text-red-500"
              :stroke-width="2"
            />
            <Icon
              v-else-if="testing"
              name="refresh"
              size="sm"
              class="animate-spin text-yellow-500"
              :stroke-width="2"
            />
            <span
              :class="[
                'max-w-[260px] truncate text-xs',
                resultById[item.id]?.success
                  ? 'text-green-600 dark:text-green-400'
                  : resultById[item.id]?.error
                    ? 'text-red-600 dark:text-red-400'
                    : 'text-gray-500 dark:text-gray-400'
              ]"
            >
              {{ resultText(item.id) }}
            </span>
          </div>
        </div>
      </div>
    </div>

    <template #footer>
      <div class="flex justify-end gap-3">
        <button
          class="rounded-lg bg-gray-100 px-4 py-2 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-200 disabled:cursor-not-allowed disabled:opacity-60 dark:bg-dark-600 dark:text-gray-300 dark:hover:bg-dark-500"
          :disabled="testing"
          @click="handleClose"
        >
          {{ t('common.close') }}
        </button>
        <button
          class="flex items-center gap-2 rounded-lg bg-primary-500 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-primary-600 disabled:cursor-not-allowed disabled:bg-primary-400"
          :disabled="testing || !selectedModelId || accountIds.length === 0"
          @click="startBatchTest"
        >
          <Icon v-if="testing" name="refresh" size="sm" class="animate-spin" :stroke-width="2" />
          <Icon v-else name="play" size="sm" :stroke-width="2" />
          <span>{{ testing ? t('admin.accounts.testing') : t('admin.accounts.batchTest.start') }}</span>
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Select from '@/components/common/Select.vue'
import TextArea from '@/components/common/TextArea.vue'
import { Icon } from '@/components/icons'
import { adminAPI } from '@/api/admin'
import type { BatchTestAccountResult, AccountBatchTestType } from '@/api/admin/accounts'
import type { Account, ClaudeModel } from '@/types'

const { t } = useI18n()

const props = defineProps<{
  show: boolean
  accountIds: number[]
  accounts: Account[]
}>()

const emit = defineEmits<{
  (event: 'close'): void
  (event: 'done'): void
}>()

type TestMode = 'default' | 'compact'

const selectedModelId = ref('')
const testType = ref<AccountBatchTestType>('auto')
const testMode = ref<TestMode>('default')
const imagePrompt = ref('')
const audioInput = ref('hi')
const videoPrompt = ref('A tiny test video of a sunrise over mountains.')
const videoSeconds = ref(4)
const videoSize = ref('720x1280')
const loadingModels = ref(false)
const modelLoadSeq = ref(0)
const testing = ref(false)
const modelOptions = ref<Array<{ value: string; label: string }>>([])
const results = ref<BatchTestAccountResult[]>([])
const accountCache = ref<Record<number, Account>>({})

const selectedModelLabel = computed(() => {
  return modelOptions.value.find((option) => option.value === selectedModelId.value)?.label || ''
})

const isInitialModelLoading = computed(() => loadingModels.value && modelOptions.value.length === 0)

const modelSelectDisabled = computed(() => testing.value || isInitialModelLoading.value)

const modelSelectPlaceholder = computed(() => {
  if (selectedModelLabel.value) return selectedModelLabel.value
  if (isInitialModelLoading.value) return `${t('common.loading')}...`
  return t('admin.accounts.selectTestModel')
})

const selectedAccountItems = computed(() => {
  return props.accountIds.map((id) => {
    const account = accountCache.value[id]
    return {
      id,
      name: account?.name || t('admin.accounts.batchTest.accountId', { id }),
      subtitle: account ? `${account.platform} / ${account.type}` : t('admin.accounts.batchTest.loadingAccount')
    }
  })
})

const resultById = computed<Record<number, BatchTestAccountResult>>(() => {
  return results.value.reduce<Record<number, BatchTestAccountResult>>((acc, item) => {
    acc[item.account_id] = item
    return acc
  }, {})
})

const resultSummary = computed(() => {
  if (results.value.length === 0) return null
  const success = results.value.filter((item) => item.success).length
  return {
    success,
    failed: results.value.length - success
  }
})

const testFormatOptions = computed(() => [
  { value: 'auto', label: t('admin.accounts.testFormatAuto') },
  { value: 'text', label: t('admin.accounts.testFormatText') },
  { value: 'image', label: t('admin.accounts.testFormatImage') },
  { value: 'audio', label: t('admin.accounts.testFormatAudio') },
  { value: 'video', label: t('admin.accounts.testFormatVideo') }
])

const openAITestModeOptions = computed(() => [
  { value: 'default', label: t('admin.accounts.openai.testModeDefault') },
  { value: 'compact', label: t('admin.accounts.openai.testModeCompact') }
])

const videoSizeOptions = [
  { value: '720x1280', label: '720x1280' },
  { value: '1280x720', label: '1280x720' },
  { value: '1024x1024', label: '1024x1024' }
]

const isOpenAIImageModel = (modelID: string) => modelID.startsWith('gpt-image-')
const isOpenAIAudioModel = (modelID: string) => modelID.includes('audio') || modelID.includes('realtime') || modelID.includes('tts') || modelID.startsWith('whisper')
const isOpenAIVideoModel = (modelID: string) => modelID.includes('video') || modelID.startsWith('sora-') || modelID.startsWith('veo') || modelID.includes('t2v') || modelID.includes('i2v') || modelID.includes('r2v')

const effectiveTestType = computed<AccountBatchTestType>(() => {
  if (testType.value !== 'auto') return testType.value
  const modelID = selectedModelId.value.toLowerCase()
  if (isOpenAIImageModel(modelID)) return 'image'
  if (isOpenAIAudioModel(modelID)) return 'audio'
  if (isOpenAIVideoModel(modelID)) return 'video'
  return 'text'
})

watch(
  () => props.show,
  async (show) => {
    if (!show) return
    selectedModelId.value = ''
    modelOptions.value = []
    testType.value = 'auto'
    testMode.value = 'default'
    imagePrompt.value = t('admin.accounts.imagePromptDefault')
    audioInput.value = 'hi'
    videoPrompt.value = 'A tiny test video of a sunrise over mountains.'
    videoSeconds.value = 4
    videoSize.value = '720x1280'
    results.value = []
    seedAccountCache()
    await loadMissingAccounts()
    await loadModels()
  }
)

watch(
  () => props.accounts,
  () => {
    seedAccountCache()
  },
  { deep: true }
)

const seedAccountCache = () => {
  const next = { ...accountCache.value }
  for (const account of props.accounts) {
    if (props.accountIds.includes(account.id)) {
      next[account.id] = account
    }
  }
  accountCache.value = next
}

const loadMissingAccounts = async () => {
  const missingIds = props.accountIds.filter((id) => !accountCache.value[id])
  if (missingIds.length === 0) return

  const loaded = await Promise.all(
    missingIds.map(async (id) => {
      try {
        return await adminAPI.accounts.getById(id)
      } catch (error) {
        console.error('Failed to load selected account:', id, error)
        return null
      }
    })
  )

  const next = { ...accountCache.value }
  for (const account of loaded) {
    if (account) next[account.id] = account
  }
  accountCache.value = next
}

const loadModels = async () => {
  const loadSeq = ++modelLoadSeq.value
  loadingModels.value = true
  try {
    const applyModels = (models: ClaudeModel[]) => {
      if (loadSeq !== modelLoadSeq.value) return
      const modelMap = new Map<string, string>()
      mergeModels(modelMap, models)
      modelOptions.value = [...modelMap.entries()].map(([value, label]) => ({ value, label }))
      selectedModelId.value = modelOptions.value[0]?.value || ''
    }

    for (const accountID of props.accountIds) {
      if (loadSeq !== modelLoadSeq.value) return
      try {
        const models = await adminAPI.accounts.getAvailableModels(accountID)
        if (models.length > 0) {
          applyModels(models)
          return
        }
      } catch (error) {
        console.error('Failed to load account models:', accountID, error)
      }
    }
  } finally {
    if (loadSeq === modelLoadSeq.value) {
      loadingModels.value = false
    }
  }
}

const mergeModels = (target: Map<string, string>, models: ClaudeModel[]) => {
  for (const model of models) {
    if (!model.id || target.has(model.id)) continue
    target.set(model.id, model.display_name || model.id)
  }
}

const resultText = (accountID: number) => {
  const result = resultById.value[accountID]
  if (!result) return testing.value ? t('admin.accounts.batchTest.waiting') : t('admin.accounts.batchTest.notStarted')
  if (result.success) return t('admin.accounts.batchTest.success')
  return result.error || t('admin.accounts.batchTest.failed')
}

const buildPrompt = () => {
  switch (effectiveTestType.value) {
    case 'image':
      return imagePrompt.value.trim()
    case 'audio':
      return audioInput.value.trim()
    case 'video':
      return videoPrompt.value.trim()
    default:
      return ''
  }
}

const startBatchTest = async () => {
  if (!selectedModelId.value || props.accountIds.length === 0) return

  testing.value = true
  results.value = []
  try {
    const response = await adminAPI.accounts.batchTest({
      account_ids: props.accountIds,
      model_id: selectedModelId.value,
      test_type: effectiveTestType.value,
      prompt: buildPrompt(),
      mode: effectiveTestType.value === 'text' ? testMode.value : 'default',
      seconds: effectiveTestType.value === 'video' ? videoSeconds.value : undefined,
      size: effectiveTestType.value === 'video' ? videoSize.value : undefined
    })
    results.value = response.results || []
    emit('done')
  } finally {
    testing.value = false
  }
}

const handleClose = () => {
  if (testing.value) return
  emit('close')
}
</script>

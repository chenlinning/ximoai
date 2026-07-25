<template>
  <BaseDialog
    :show="show"
    :title="t('modelPlaza.brandEditor.title', { model: modelName })"
    width="normal"
    @close="emit('close')"
  >
    <div class="space-y-5">
      <div>
        <label for="model-brand" class="label">{{ t('modelPlaza.brandEditor.brand') }}</label>
        <input
          id="model-brand"
          v-model="draftBrand"
          data-testid="model-brand-input"
          class="input"
          type="text"
          maxlength="64"
          autocomplete="off"
        />
      </div>

      <div class="border-l-2 border-primary-500 pl-3">
        <p class="text-xs text-gray-500 dark:text-dark-400">
          {{ t('modelPlaza.brandEditor.automaticBrand') }}
        </p>
        <p class="mt-1 text-sm font-medium text-gray-900 dark:text-white">
          {{ editor.automatic_brand }}
        </p>
      </div>
    </div>

    <template #footer>
      <div class="flex w-full flex-wrap items-center justify-between gap-3">
        <button
          v-if="editor.source === 'administrator'"
          type="button"
          class="btn btn-ghost btn-md text-red-600 dark:text-red-400"
          :disabled="saving"
          @click="resetBrand"
        >
          <Icon name="refresh" size="sm" />
          {{ t('modelPlaza.brandEditor.reset') }}
        </button>
        <span v-else></span>
        <div class="flex gap-2">
          <button type="button" class="btn btn-secondary btn-md" :disabled="saving" @click="emit('close')">
            {{ t('modelPlaza.brandEditor.cancel') }}
          </button>
          <button type="button" class="btn btn-primary btn-md" :disabled="saving || !canSave" @click="saveBrand">
            <Icon v-if="saving" name="refresh" size="sm" class="animate-spin" />
            {{ t('modelPlaza.brandEditor.save') }}
          </button>
        </div>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import { modelBrandAPI, type ModelBrandEditor, type ModelBrandState } from '@/api/modelBrand'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'

const props = defineProps<{
  show: boolean
  platform: string
  modelName: string
  brand: string
  editor: ModelBrandEditor
}>()

const emit = defineEmits<{
  (event: 'close'): void
  (event: 'updated', state: ModelBrandState): void
}>()

const { t } = useI18n()
const appStore = useAppStore()
const saving = ref(false)
const draftBrand = ref('')
const canSave = computed(() => {
  const normalized = draftBrand.value.trim()
  return normalized.length > 0 && normalized !== props.brand
})

function target() {
  return { platform: props.platform, model: props.modelName }
}

async function saveBrand() {
  const brand = draftBrand.value.trim()
  if (!brand) return
  saving.value = true
  try {
    const state = await modelBrandAPI.save(target(), brand)
    appStore.showSuccess(t('modelPlaza.brandEditor.saved'))
    emit('updated', state)
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('modelPlaza.brandEditor.saveError')))
  } finally {
    saving.value = false
  }
}

async function resetBrand() {
  saving.value = true
  try {
    const state = await modelBrandAPI.reset(target())
    appStore.showSuccess(t('modelPlaza.brandEditor.resetDone'))
    emit('updated', state)
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('modelPlaza.brandEditor.resetError')))
  } finally {
    saving.value = false
  }
}

watch(
  () => [props.show, props.brand],
  ([show]) => {
    if (show) draftBrand.value = props.brand
  },
  { immediate: true }
)
</script>

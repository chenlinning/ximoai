<template>
  <main class="flex min-h-screen items-center justify-center bg-gray-50 px-4 py-10 dark:bg-dark-950">
    <section class="w-full max-w-md rounded-lg border border-gray-200 bg-white px-6 py-8 text-center shadow-sm dark:border-dark-700 dark:bg-dark-900 sm:px-8">
      <div
        v-if="status !== 'error'"
        class="mx-auto flex h-12 w-12 items-center justify-center rounded-full bg-primary-50 dark:bg-primary-900/30"
        aria-hidden="true"
      >
        <span class="h-5 w-5 animate-spin rounded-full border-2 border-primary-200 border-t-primary-600 dark:border-primary-800 dark:border-t-primary-300" />
      </div>
      <div
        v-else
        class="mx-auto flex h-12 w-12 items-center justify-center rounded-full bg-red-50 text-xl font-semibold text-red-600 dark:bg-red-950/40 dark:text-red-300"
        aria-hidden="true"
      >
        !
      </div>

      <h1 class="mt-5 text-xl font-semibold text-gray-900 dark:text-white">
        {{ t('desktopAuthorization.title') }}
      </h1>
      <p class="mt-2 text-sm leading-6 text-gray-600 dark:text-gray-300" role="status">
        {{ statusMessage }}
      </p>

      <button
        v-if="status === 'error'"
        type="button"
        class="btn btn-primary mt-6"
        @click="authorize"
      >
        {{ t('desktopAuthorization.retry') }}
      </button>
    </section>
  </main>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'

import { issueDesktopAuthorization } from './api'
import { completeDesktopAuthorization } from './authorization'

type AuthorizationStatus = 'authorizing' | 'returning' | 'error'

const route = useRoute()
const { t } = useI18n()
const status = ref<AuthorizationStatus>('authorizing')

const statusMessage = computed(() => {
  if (status.value === 'returning') return t('desktopAuthorization.returning')
  if (status.value === 'error') return t('desktopAuthorization.failed')
  return t('desktopAuthorization.authorizing')
})

async function authorize(): Promise<void> {
  status.value = 'authorizing'
  try {
    await completeDesktopAuthorization(
      route.query,
      issueDesktopAuthorization,
      (callbackURL) => {
        status.value = 'returning'
        window.location.assign(callbackURL)
      },
    )
  } catch {
    status.value = 'error'
  }
}

onMounted(authorize)
</script>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'

const isOnPlaza = ref(false)

function checkRoute() {
  const router = (window as any).__APP_ROUTER__
  if (router) {
    isOnPlaza.value = router.currentRoute.value.path === '/model-plaza'
  }
}

let unregisterGuard: (() => void) | null = null

onMounted(() => {
  checkRoute()
  const router = (window as any).__APP_ROUTER__
  if (router) {
    const remove = router.afterEach(() => {
      checkRoute()
    })
    unregisterGuard = remove
  }
})

onUnmounted(() => {
  if (unregisterGuard) unregisterGuard()
})

function toggle() {
  const router = (window as any).__APP_ROUTER__
  if (!router) return
  if (isOnPlaza.value) {
    router.back()
  } else {
    router.push('/model-plaza')
  }
}
</script>

<template>
  <div class="model-plaza-fab">
    <button
      class="fab-button"
      :class="{ active: isOnPlaza }"
      @click="toggle"
      :title="isOnPlaza ? '返回' : '模型广场'"
    >
      <svg v-if="!isOnPlaza" xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <rect width="7" height="7" x="3" y="3" rx="1" />
        <rect width="7" height="7" x="14" y="3" rx="1" />
        <rect width="7" height="7" x="14" y="14" rx="1" />
        <rect width="7" height="7" x="3" y="14" rx="1" />
      </svg>
      <svg v-else xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <path d="M18 6 6 18" /><path d="m6 6 12 12" />
      </svg>
    </button>
  </div>
</template>

<style scoped>
.model-plaza-fab {
  position: fixed;
  bottom: 24px;
  right: 24px;
  z-index: 9999;
  pointer-events: auto;
}

.fab-button {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  border: none;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  background-color: var(--accent-500, #f59e0b);
  color: white;
  box-shadow: 0 4px 14px rgba(0, 0, 0, 0.3);
  transition: all 0.2s ease;
}

.fab-button:hover {
  transform: scale(1.1);
  box-shadow: 0 6px 20px rgba(0, 0, 0, 0.4);
}

.fab-button.active {
  background-color: var(--destructive, #ef4444);
}
</style>

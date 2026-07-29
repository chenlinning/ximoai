<template>
  <ModelPlazaPage v-if="authStore.isAuthenticated" />
  <ModelPlazaView v-else />
</template>

<script setup lang="ts">
import { useAuthStore } from '@/stores/auth'
import ModelPlazaView from '@/views/ModelPlazaView.vue'
import ModelPlazaPage from './ModelPlazaPage.vue'

const authStore = useAuthStore()
</script>

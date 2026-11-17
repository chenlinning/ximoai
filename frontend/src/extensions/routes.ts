import type { RouteRecordRaw } from 'vue-router'

export const ximoAIRoutes: RouteRecordRaw[] = [
  {
    path: '/ximoai-model-plaza',
    name: 'XimoAIModelPlaza',
    component: () => import('@/extensions/model-plaza/ModelPlazaPage.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: false,
      title: 'XimoAI Model Plaza',
      titleKey: 'nav.ximoaiModelPlaza',
      descriptionKey: 'ximoaiModelPlaza.description'
    }
  }
]

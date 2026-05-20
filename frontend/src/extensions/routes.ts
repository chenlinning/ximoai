import type { RouteRecordRaw } from 'vue-router'

export const ximoAIRoutes: RouteRecordRaw[] = [
  {
    path: '/model-plaza',
    name: 'ModelPlaza',
    component: () => import('@/extensions/model-plaza/ModelPlazaPage.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: false,
      title: 'Model Plaza',
      titleKey: 'modelPlaza.title',
      descriptionKey: 'modelPlaza.description'
    }
  },
  {
    path: '/admin/platforms',
    name: 'AdminPlatforms',
    component: () => import('@/views/admin/PlatformsView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: true,
      title: 'Platform Management',
      titleKey: 'admin.platforms.title',
      descriptionKey: 'admin.platforms.description'
    }
  },
]

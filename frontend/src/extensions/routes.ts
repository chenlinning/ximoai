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
  {
    path: '/download-center',
    name: 'DownloadCenter',
    component: () => import('@/extensions/ximoapp/DownloadCenterPage.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: false,
      title: 'Download Center',
      titleKey: 'downloadCenter.title',
      descriptionKey: 'downloadCenter.description'
    }
  },
  {
    path: '/admin/ximoapp-update',
    name: 'AdminXimoAppUpdate',
    component: () => import('@/extensions/ximodesk/XimoDeskUpdatePage.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: true,
      title: 'XimoAPP Update Center',
      titleKey: 'ximoappUpdate.title',
      descriptionKey: 'ximoappUpdate.description'
    }
  },
]

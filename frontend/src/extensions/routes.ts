import type { RouteRecordRaw } from 'vue-router'

export const ximoAIRoutes: RouteRecordRaw[] = [
  {
    path: '/desktop/authorize',
    name: 'DesktopAuthorize',
    component: () => import('@/extensions/desktop/DesktopAuthorizePage.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: false,
      title: 'Desktop Authorization',
      titleKey: 'desktopAuthorization.title'
    }
  },
  {
    path: '/ximoai-model-plaza',
    name: 'XimoAIModelPlaza',
    component: () => import('@/extensions/model-plaza/ModelPlazaPage.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: false,
      title: 'XimoAI Model Plaza',
      titleKey: 'nav.ximoaiModelPlaza',
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

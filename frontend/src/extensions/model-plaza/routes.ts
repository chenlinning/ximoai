/**
 * 模型广场 - 路由注册
 *
 * 通过 window.__APP_ROUTER__ 获取源项目 router 实例
 * 动态添加模型广场路由，不修改源项目路由文件
 */

export function registerRoutes(): void {
  const router = (window as any).__APP_ROUTER__
  if (!router) {
    console.warn('[ModelPlaza] Router instance not found')
    return
  }

  // 动态导入页面组件
  router.addRoute({
    path: '/model-plaza',
    name: 'ModelPlaza',
    component: () => import('./ModelPlazaPage.vue'),
    meta: { title: 'Model Plaza' },
  })

  console.log('[ModelPlaza] Route registered: /model-plaza')
}

/**
 * 模型广场 - 挂载入口
 *
 * 1. 注入 i18n 翻译
 * 2. 注册路由
 * 3. 挂载浮动导航按钮
 */

import { injectModelPlazaI18n } from './i18n'
import { registerRoutes } from './routes'

let mounted = false

export function mountModelPlaza(): void {
  if (mounted) return

  // 注入 i18n
  injectModelPlazaI18n()

  // 注册路由
  registerRoutes()

  // 挂载浮动导航按钮
  mountFloatingNav()

  mounted = true
  console.log('[ModelPlaza] Extension mounted')
}

function mountFloatingNav(): void {
  // 确保 DOM 就绪后挂载
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', createFloatingNav)
  } else {
    // 延迟确保源项目 layout 已渲染
    setTimeout(createFloatingNav, 1000)
  }
}

function createFloatingNav(): void {
  // 避免重复挂载
  if (document.getElementById('model-plaza-nav')) return

  const nav = document.createElement('div')
  nav.id = 'model-plaza-nav'
  document.body.appendChild(nav)

  // 动态导入 Vue 和组件来创建独立应用
  import('vue').then(({ createApp, h }) => {
    import('./FloatingNav.vue').then(({ default: FloatingNav }) => {
      const app = createApp({
        render: () => h(FloatingNav),
      })
      app.mount(nav)
      console.log('[ModelPlaza] Floating nav mounted')
    })
  })
}

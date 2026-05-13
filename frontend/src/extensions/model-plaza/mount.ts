/**
 * 模型广场 - 挂载入口
 *
 * 1. 注入 i18n 翻译
 *
 * 路由已在 router/index.ts 中静态注册（规则3豁免），无需动态 addRoute。
 * 侧边栏菜单项已通过 AppSidebar.vue 原生添加（规则3豁免），
 * 无需 DOM 注入。
 */

import { injectModelPlazaI18n } from './i18n'

let mounted = false

export function mountModelPlaza(): void {
  if (mounted) return

  // 注入 i18n
  injectModelPlazaI18n()

  mounted = true
  console.log('[ModelPlaza] Extension mounted (i18n only, route is static)')
}

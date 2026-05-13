/**
 * 模型广场 - 挂载入口
 *
 * 1. 注入 i18n 翻译
 * 2. 注册路由
 * 3. 通过 DOM 注入将"模型广场"菜单项插入侧边栏
 *
 * 注入策略：
 *   - 使用轮询 + MutationObserver 确保在侧边栏渲染后注入
 *   - 登录页无侧边栏，需等待用户登录后 dashboard 渲染
 *   - Vue 重新渲染侧边栏时 MutationObserver 自动重新注入
 */

import { injectModelPlazaI18n } from './i18n'
import { registerRoutes } from './routes'
import { watch } from 'vue'

let mounted = false
let observer: MutationObserver | null = null

export function mountModelPlaza(): void {
  if (mounted) return

  // 注入 i18n
  injectModelPlazaI18n()

  // 注册路由
  registerRoutes()

  // 注入侧边栏菜单项
  injectSidebarEntry()

  mounted = true
  console.log('[ModelPlaza] Extension mounted')
}

/* ───────── sidebar DOM injection ───────── */

const SIDEBAR_MARKER = 'data-model-plaza'
const POLL_INTERVAL = 500
const MAX_POLL_ATTEMPTS = 120 // 最多轮询 60 秒

function injectSidebarEntry(): void {
  // 注入全局样式
  injectGlobalStyles()

  // 开始轮询等待侧边栏出现
  startPolling()

  // 监听路由变化：更新激活态 + 确保注入元素存在
  const router = (window as any).__APP_ROUTER__
  if (router) {
    router.afterEach(() => {
      // 延迟执行，等 Vue 渲染新侧边栏
      setTimeout(() => {
        const sidebarNav = document.querySelector('.sidebar-nav')
        if (sidebarNav) {
          // 如果注入元素丢失（路由切换导致侧边栏重新渲染），重新注入
          if (!sidebarNav.querySelector(`[${SIDEBAR_MARKER}]`)) {
            doInject(sidebarNav as HTMLElement)
            // 重启 MutationObserver 监听新的侧边栏 DOM
            if (observer) observer.disconnect()
            startObserver(sidebarNav as HTMLElement)
          }
        }
        updateActiveState()
      }, 150)
    })
  }

  // 监听 locale 变化以更新侧边栏标签语言
  watchLocaleChange()
}

/** 获取当前语言的侧边栏标签 */
function getLabel(): string {
  const i18n = (window as any).__APP_I18N__
  try {
    return i18n?.global?.t?.('modelPlaza.title') || '模型广场'
  } catch {
    const locale = i18n?.global?.locale?.value || 'zh'
    return locale.startsWith('zh') ? '模型广场' : 'Model Plaza'
  }
}

/** 更新侧边栏标签文本（语言切换时调用） */
function updateSidebarLabel(): void {
  const wrapper = document.querySelector(`[${SIDEBAR_MARKER}]`)
  if (!wrapper) return
  const labelEl = wrapper.querySelector('.sidebar-label')
  const linkEl = wrapper.querySelector('a')
  const label = getLabel()
  if (labelEl) labelEl.textContent = label
  if (linkEl) linkEl.title = label
}

/** 监听 locale 变化 */
function watchLocaleChange(): void {
  const i18n = (window as any).__APP_I18N__
  if (!i18n?.global?.locale) return

  try {
    watch(
      () => i18n.global.locale.value,
      () => {
        updateSidebarLabel()
      }
    )
  } catch {
    // fallback: 侧边栏 MutationObserver 重新注入时也会更新
  }
}

function startPolling(): void {
  let attempts = 0

  const poll = () => {
    attempts++
    const sidebarNav = document.querySelector('.sidebar-nav')
    const sections = sidebarNav?.querySelectorAll('.sidebar-section')

    if (sidebarNav && sections && sections.length > 0) {
      // 侧边栏已渲染，执行注入
      doInject(sidebarNav as HTMLElement)
      startObserver(sidebarNav as HTMLElement)
      return // 停止轮询
    }

    if (attempts < MAX_POLL_ATTEMPTS) {
      setTimeout(poll, POLL_INTERVAL)
    } else {
      console.warn('[ModelPlaza] Sidebar not found after max polling attempts')
    }
  }

  // 初始延迟 800ms，给 Vue 首次渲染留出时间
  setTimeout(poll, 800)
}

function doInject(sidebarNav: HTMLElement): void {
  // 避免重复注入
  if (sidebarNav.querySelector(`[${SIDEBAR_MARKER}]`)) {
    // 即使已注入，也更新语言标签
    updateSidebarLabel()
    return
  }

  const label = getLabel()

  // 创建容器 div（与现有 sidebar-link 的父级结构一致）
  const wrapper = document.createElement('div')
  wrapper.setAttribute(SIDEBAR_MARKER, 'true')

  // 创建链接元素 — 使用 <a> 模拟 router-link 外观
  const link = document.createElement('a')
  link.href = '/model-plaza'
  link.className = 'sidebar-link mb-1'
  link.title = label
  link.innerHTML = `
    <svg class="h-5 w-5 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
      <path stroke-linecap="round" stroke-linejoin="round" d="M3.75 6A2.25 2.25 0 016 3.75h2.25A2.25 2.25 0 0110.5 6v2.25a2.25 2.25 0 01-2.25 2.25H6a2.25 2.25 0 01-2.25-2.25V6zM3.75 15.75A2.25 2.25 0 016 13.5h2.25a2.25 2.25 0 012.25 2.25V18a2.25 2.25 0 01-2.25 2.25H6A2.25 2.25 0 013.75 18v-2.25zM13.5 6a2.25 2.25 0 012.25-2.25H18A2.25 2.25 0 0120.25 6v2.25A2.25 2.25 0 0118 10.5h-2.25a2.25 2.25 0 01-2.25-2.25V6zM13.5 15.75a2.25 2.25 0 012.25-2.25H18a2.25 2.25 0 012.25 2.25V18A2.25 2.25 0 0118 20.25h-2.25a2.25 2.25 0 01-2.25-2.25z" />
    </svg>
    <span class="sidebar-label" aria-hidden="false">${label}</span>
  `

  // 点击处理 — 使用 Vue router 进行 SPA 导航（不刷新页面）
  link.addEventListener('click', (e: Event) => {
    e.preventDefault()
    const router = (window as any).__APP_ROUTER__
    if (router) {
      router.push('/model-plaza')
    } else {
      // fallback
      window.location.href = '/model-plaza'
    }
  })

  wrapper.appendChild(link)

  // 确定插入位置：注入到第一个 sidebar-section（管理菜单区）
  const sections = sidebarNav.querySelectorAll('.sidebar-section')
  if (sections.length === 0) return

  const targetSection = sections[0] as HTMLElement

  // 尝试在"系统设置"之前插入，否则追加到末尾
  const settingsLink = targetSection.querySelector('a[href="/admin/settings"]')
  if (settingsLink && settingsLink.parentElement) {
    // 如果 settingsLink 的父级是 router-link 渲染的 <a>，其 parentElement 可能就是 section
    // 也可能 settingsLink 本身就是 <a>
    const insertBefore = settingsLink.parentElement.hasAttribute(SIDEBAR_MARKER)
      ? settingsLink.parentElement
      : settingsLink
    targetSection.insertBefore(wrapper, insertBefore)
  } else {
    targetSection.appendChild(wrapper)
  }

  console.log('[ModelPlaza] Sidebar entry injected')
  updateActiveState()
}

function updateActiveState(): void {
  const wrapper = document.querySelector(`[${SIDEBAR_MARKER}]`) as HTMLElement | null
  if (!wrapper) return

  const link = wrapper.querySelector('a') as HTMLElement | null
  if (!link) return

  const router = (window as any).__APP_ROUTER__
  const isOnPlaza = router?.currentRoute?.value?.path === '/model-plaza'

  if (isOnPlaza) {
    link.classList.add('sidebar-link-active')
  } else {
    link.classList.remove('sidebar-link-active')
  }
}

function startObserver(sidebarNav: HTMLElement): void {
  observer = new MutationObserver(() => {
    // 如果注入的元素被移除（侧边栏重新渲染），重新注入
    if (!sidebarNav.querySelector(`[${SIDEBAR_MARKER}]`)) {
      // 延迟注入，等 Vue 渲染完成
      setTimeout(() => doInject(sidebarNav), 100)
    }
    // 更新激活态
    updateActiveState()
  })
  observer.observe(sidebarNav, { childList: true, subtree: true })
}

function injectGlobalStyles(): void {
  if (document.getElementById('model-plaza-styles')) return

  const style = document.createElement('style')
  style.id = 'model-plaza-styles'
  style.textContent = `
    /* 模型广场注入菜单项样式补充 */
    [data-model-plaza] .sidebar-label {
      display: block;
      min-width: 0;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
      transition: max-width 0.2s ease, opacity 0.12s ease, transform 0.12s ease;
      max-width: 12rem;
    }
    [data-model-plaza] a.sidebar-link {
      display: flex;
      align-items: center;
      gap: 0.75rem;
      width: 100%;
      cursor: pointer;
    }
    [data-model-plaza] a.sidebar-link-active {
      background-color: var(--primary-50, #eff6ff);
      color: var(--primary-700, #1d4ed8);
    }
    .dark [data-model-plaza] a.sidebar-link-active {
      background-color: rgba(var(--primary-900-rgb, 30 58 138), 0.2);
      color: var(--primary-400, #60a5fa);
    }
  `
  document.head.appendChild(style)
}

import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const testDirectory = dirname(fileURLToPath(import.meta.url))
const layoutSource = readFileSync(resolve(testDirectory, '../AppLayout.vue'), 'utf8')
const headerSource = readFileSync(resolve(testDirectory, '../AppHeader.vue'), 'utf8')
const sidebarSource = readFileSync(resolve(testDirectory, '../AppSidebar.vue'), 'utf8')
const homeWorkspaceSource = readFileSync(resolve(testDirectory, '../../../extensions/ximoai-home/XimoAIHomeWorkspace.vue'), 'utf8')
const styleSource = readFileSync(resolve(testDirectory, '../../../style.css'), 'utf8')

describe('AppLayout dark background', () => {
  it('keeps the page lighter than the unchanged sidebar', () => {
    expect(layoutSource).toContain('dark:bg-dark-800')

    const sidebarBlock = styleSource.match(/\.sidebar\s*\{[\s\S]*?\n {2}\}/)?.[0]
    expect(sidebarBlock).toContain('dark:bg-dark-900')
  })
})

describe('AppHeader background', () => {
  it('matches the sidebar in both themes', () => {
    expect(headerSource).toContain("'bg-white sticky top-0 z-30 border-b border-gray-200/50 dark:bg-dark-900 dark:border-dark-700/50'")
  })

  it('keeps the membership avatar trigger without adjacent user text', () => {
    const userTrigger = headerSource.match(/<button\s+@click="toggleDropdown"[\s\S]*?<\/button>/)?.[0]

    expect(userTrigger).toContain('membershipAvatarStyle(currentMembershipColor)')
    expect(userTrigger).toContain('membershipBadgeStyle(currentMembershipColor)')
    expect(userTrigger).not.toContain('hidden text-left md:block')
    expect(userTrigger).not.toContain('chevronDown')
  })

  it('shows the balance as plain model-plaza-colored text', () => {
    const balanceDisplay = headerSource.match(/<!-- Balance Display -->[\s\S]*?<!-- User Dropdown -->/)?.[0]

    expect(balanceDisplay).toContain('class="group relative hidden items-center gap-2 sm:flex"')
    expect(balanceDisplay).toContain('text-orange-600 dark:text-orange-400')
    expect(balanceDisplay).not.toContain('<svg')
    expect(balanceDisplay).not.toContain('bg-primary-50')
    expect(balanceDisplay).not.toContain('bg-amber-100')
  })
})

describe('Primary navigation emphasis', () => {
  it('uses the standard primary button colors for home navigation', () => {
    const homeButtonBlock = sidebarSource.match(/\.sidebar-home-button\s*\{[\s\S]*?\n\}/)?.[0]

    expect(homeButtonBlock).toContain('@apply bg-primary-500 text-white shadow-sm transition-colors;')
    expect(homeButtonBlock).toContain('@apply hover:bg-primary-600 hover:text-white hover:shadow-md;')
    expect(homeButtonBlock).not.toMatch(/\bborder(?:-|\b)/)
    expect(homeWorkspaceSource).toContain("? 'bg-primary-500 text-white shadow-sm hover:bg-primary-600 hover:shadow-md'")
  })
})

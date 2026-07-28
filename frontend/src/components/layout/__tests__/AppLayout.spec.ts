import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const testDirectory = dirname(fileURLToPath(import.meta.url))
const layoutSource = readFileSync(resolve(testDirectory, '../AppLayout.vue'), 'utf8')
const headerSource = readFileSync(resolve(testDirectory, '../AppHeader.vue'), 'utf8')
const sidebarSource = readFileSync(resolve(testDirectory, '../AppSidebar.vue'), 'utf8')
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

  it('restores the native user trigger while retaining the membership frame', () => {
    const userTrigger = headerSource.match(/<button\s+@click="toggleDropdown"[\s\S]*?<\/button>/)?.[0]

    expect(userTrigger).toContain('membershipAvatarStyle(currentMembershipColor)')
    expect(userTrigger).toContain('membershipBadgeStyle(currentMembershipColor)')
    expect(userTrigger).toContain('gap-2 rounded-xl p-1.5')
    expect(userTrigger).toContain('hidden text-left md:block')
    expect(userTrigger).toContain('chevronDown')
    expect(userTrigger).toContain('max-w-14')
    expect(userTrigger).toContain('px-0.5 py-0 text-[8px]')
  })

  it('shows the balance as plain model-plaza-colored text', () => {
    const balanceDisplay = headerSource.match(/<!-- Balance Display -->[\s\S]*?<!-- User Dropdown -->/)?.[0]

    expect(balanceDisplay).toContain('class="group relative hidden items-center gap-2 sm:flex"')
    expect(balanceDisplay).toContain('text-orange-600 dark:text-orange-400')
    expect(balanceDisplay).not.toContain('<svg')
    expect(balanceDisplay).not.toContain('bg-primary-50')
    expect(balanceDisplay).not.toContain('bg-amber-100')
  })

  it('moves the home entry to the far left of the header and matches the console size', () => {
    const homeButton = headerSource.match(/<!-- Home Entry -->[\s\S]*?<\/router-link>/)?.[0]
    const consoleButton = headerSource.match(/<router-link\s+v-if="props\.showConsoleButton"[\s\S]*?<\/router-link>/)?.[0]

    expect(homeButton).toContain('to="/home"')
    expect(homeButton).toContain('class="header-navigation-button"')
    expect(homeButton).toContain('<Icon name="home" size="sm" />')
    expect(consoleButton).toContain('class="header-navigation-button"')
    expect(headerSource).toContain('@apply inline-flex h-9 shrink-0 items-center gap-1.5 rounded-xl px-3 text-sm font-medium;')
    expect(sidebarSource).not.toContain('class="sidebar-home-button"')
    expect(sidebarSource).not.toContain('.sidebar-home-button')
  })

})

describe('Primary navigation emphasis', () => {
  it('uses the standard primary button colors for home navigation', () => {
    const homeButtonBlock = headerSource.match(/\.header-navigation-button\s*\{[\s\S]*?\n\}/)?.[0]

    expect(homeButtonBlock).toContain('@apply bg-primary-500 text-white shadow-sm transition-colors;')
    expect(homeButtonBlock).toContain('@apply hover:bg-primary-600 hover:text-white hover:shadow-md;')
    expect(homeButtonBlock).not.toMatch(/\bborder(?:-|\b)/)
  })
})

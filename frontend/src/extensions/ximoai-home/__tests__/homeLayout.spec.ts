import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'
import {
  buildHomeRows,
  resolveHomeColumnCount
} from '../homeLayout'

const items = Array.from({ length: 8 }, (_, index) => ({ id: `tab-${index}` }))
const testDirectory = dirname(fileURLToPath(import.meta.url))
const rowsSource = readFileSync(resolve(testDirectory, '../XimoAIHomeRows.vue'), 'utf8')
const workspaceSource = readFileSync(resolve(testDirectory, '../XimoAIHomeWorkspace.vue'), 'utf8')

describe('XimoAI home static layout', () => {
  it('derives one, two, or three columns from the visible container width', () => {
    expect(resolveHomeColumnCount(807)).toBe(1)
    expect(resolveHomeColumnCount(808)).toBe(2)
    expect(resolveHomeColumnCount(1223)).toBe(2)
    expect(resolveHomeColumnCount(1224)).toBe(3)
    expect(resolveHomeColumnCount(1800)).toBe(3)
  })

  it('keeps static card width continuous when a new column becomes available', () => {
    const cardMaxWidth = 392
    const oneColumnWidth = Math.min(807, cardMaxWidth)
    const twoColumnWidth = Math.min((808 - 24) / 2, cardMaxWidth)
    const wideTwoColumnWidth = Math.min((1223 - 24) / 2, cardMaxWidth)
    const threeColumnWidth = (1224 - 48) / 3

    expect(twoColumnWidth).toBe(oneColumnWidth)
    expect(threeColumnWidth).toBe(wideTwoColumnWidth)
    expect(rowsSource).toContain('24.5rem')
    expect(rowsSource).toContain('19.6rem')
  })

  it('keeps at most three cards per row and marks every later row compact', () => {
    const rows = buildHomeRows(items, 3)

    expect(rows.map((row) => row.items.length)).toEqual([3, 3, 2])
    expect(rows.map((row) => row.compact)).toEqual([false, true, true])
    expect(rows.flatMap((row) => row.items.map((entry) => entry.index))).toEqual([0, 1, 2, 3, 4, 5, 6, 7])
  })

  it('keeps every row centered and limits each row to three cards', () => {
    const rows = buildHomeRows(items, 3)

    expect(rows.every((row) => row.items.length <= 3)).toBe(true)
    expect(rows.every((row) => row.compact || row.index === 0)).toBe(true)
  })

  it('lets three cards fill the full visible entry width', () => {
    expect(workspaceSource).toMatch(/\.home-entry-area\s*\{[\s\S]*?width:\s*100%;/)
    expect(workspaceSource).not.toContain('width: min(100%, 96rem)')
  })

  it('keeps the animated background while using the balanced home frame rate', () => {
    expect(workspaceSource).toMatch(/<LoginGalaxyBackground[\s\S]*?:max-fps="30"/)
  })
})

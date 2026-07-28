import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'
import {
  buildHomeRows,
  buildSpotlightLayout,
  calculateSpotlightDimensions,
  resolveHomeColumnCount
} from '../spotlightLayout'

const items = Array.from({ length: 8 }, (_, index) => ({ id: `tab-${index}` }))
const testDirectory = dirname(fileURLToPath(import.meta.url))
const rowsSource = readFileSync(resolve(testDirectory, '../XimoAIHomeRows.vue'), 'utf8')
const workspaceSource = readFileSync(resolve(testDirectory, '../XimoAIHomeWorkspace.vue'), 'utf8')

describe('XimoAI home spotlight layout', () => {
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

  it('centers a middle card with at most two companions on each side', () => {
    const layout = buildSpotlightLayout(items, 1, 3)

    expect(layout.mode).toBe('center')
    expect(layout.left.map((entry) => entry.index)).toEqual([0])
    expect(layout.right.map((entry) => entry.index)).toEqual([2, 3])
    expect(layout.companions.map((entry) => entry.index)).toEqual([0, 2, 3])
    expect(layout.remainder.map((entry) => entry.index)).toEqual([4, 5, 6, 7])
    expect(new Set([
      layout.active.item.id,
      ...layout.companions.map((entry) => entry.item.id),
      ...layout.remainder.map((entry) => entry.item.id)
    ]).size).toBe(items.length)
  })

  it('places an edge card opposite a four-card companion grid', () => {
    const leftLayout = buildSpotlightLayout(items, 3, 3)
    const rightLayout = buildSpotlightLayout(items, 5, 3)

    expect(leftLayout.mode).toBe('left')
    expect(leftLayout.companions.map((entry) => entry.index)).toEqual([4, 5, 6, 7])
    expect(rightLayout.mode).toBe('right')
    expect(rightLayout.companions.map((entry) => entry.index)).toEqual([1, 2, 3, 4])
  })

  it('derives spotlight placement from the visible position in an incomplete row', () => {
    expect(buildSpotlightLayout(items.slice(0, 4), 3, 3).mode).toBe('center')
    expect(buildSpotlightLayout(items.slice(0, 5), 3, 3).mode).toBe('left')
    expect(buildSpotlightLayout(items.slice(0, 5), 4, 3).mode).toBe('right')
    expect(buildSpotlightLayout(items.slice(0, 7), 6, 2).companions).toHaveLength(0)
  })

  it('keeps replacement animation handles registered until that animation ends', () => {
    expect(workspaceSource).toContain('if (layoutAnimations.get(tabID) === animation)')
  })

  it('fills the wide spotlight width and matches two companion rows to the active height', () => {
    const dimensions = calculateSpotlightDimensions(1536, 3)
    const activeHeight = dimensions.activeWidth * 3 / 5 + dimensions.labelBlockHeight
    const companionRowsHeight = 2 * (
      dimensions.companionWidth * 3 / 5 + dimensions.labelBlockHeight
    ) + dimensions.verticalGap

    expect(dimensions.activeWidth + 2 * dimensions.companionWidth + 2 * dimensions.horizontalGap)
      .toBeCloseTo(1536, 5)
    expect(companionRowsHeight).toBeCloseTo(activeHeight, 5)
  })

  it('uses the full width for the reduced two-column and one-column spotlight layouts', () => {
    const medium = calculateSpotlightDimensions(900, 2)
    const narrow = calculateSpotlightDimensions(420, 1)

    expect(medium.activeWidth + medium.companionWidth + medium.horizontalGap).toBeCloseTo(900, 5)
    expect(narrow.activeWidth).toBe(420)
    expect(narrow.companionWidth).toBe(0)
  })
})

export type HomeColumnCount = 1 | 2 | 3
export type HomeSpotlightMode = 'left' | 'center' | 'right'

export interface IndexedHomeItem<T> {
  item: T
  index: number
}

export interface HomeRow<T> {
  index: number
  compact: boolean
  items: IndexedHomeItem<T>[]
}

export interface HomeSpotlightLayout<T> {
  active: IndexedHomeItem<T>
  activeRowIndex: number
  mode: HomeSpotlightMode
  left: IndexedHomeItem<T>[]
  right: IndexedHomeItem<T>[]
  companions: IndexedHomeItem<T>[]
  remainder: IndexedHomeItem<T>[]
}

export interface HomeSpotlightDimensions {
  activeWidth: number
  companionWidth: number
  companionGridWidth: number
  horizontalGap: number
  verticalGap: number
  labelBlockHeight: number
}

export const HOME_HORIZONTAL_GAP = 24
export const HOME_VERTICAL_GAP = 16
export const HOME_LABEL_BLOCK_HEIGHT = 36

const TWO_COLUMN_MIN_WIDTH = 808
const THREE_COLUMN_MIN_WIDTH = 1224
const MEDIA_HEIGHT_RATIO = 3 / 5

export function resolveHomeColumnCount(width: number): HomeColumnCount {
  if (width >= THREE_COLUMN_MIN_WIDTH) return 3
  if (width >= TWO_COLUMN_MIN_WIDTH) return 2
  return 1
}

export function buildHomeRows<T>(items: readonly T[], columns: HomeColumnCount): HomeRow<T>[] {
  const indexedItems = items.map((item, index) => ({ item, index }))
  const rows: HomeRow<T>[] = []

  for (let offset = 0; offset < indexedItems.length; offset += columns) {
    const index = rows.length
    rows.push({
      index,
      compact: index > 0,
      items: indexedItems.slice(offset, offset + columns)
    })
  }

  return rows
}

function spotlightMode(index: number, itemCount: number, columns: HomeColumnCount): HomeSpotlightMode {
  if (columns === 1) return 'center'

  const rowStart = Math.floor(index / columns) * columns
  const rowLength = Math.min(columns, itemCount - rowStart)
  const position = index - rowStart

  if (rowLength === 1) return 'center'
  if (position === 0) return 'left'
  if (position === rowLength - 1) return 'right'
  return 'center'
}

function companionCapacity(columns: HomeColumnCount): number {
  if (columns === 3) return 4
  if (columns === 2) return 1
  return 0
}

export function buildSpotlightLayout<T>(
  items: readonly T[],
  activeIndex: number,
  columns: HomeColumnCount
): HomeSpotlightLayout<T> {
  if (activeIndex < 0 || activeIndex >= items.length) {
    throw new RangeError('activeIndex is outside the home item list')
  }

  const indexedItems = items.map((item, index) => ({ item, index }))
  const active = indexedItems[activeIndex]
  const mode = spotlightMode(activeIndex, items.length, columns)
  const capacity = companionCapacity(columns)
  let left: IndexedHomeItem<T>[] = []
  let right: IndexedHomeItem<T>[] = []

  if (mode === 'center') {
    const sideCapacity = columns === 3 ? 2 : 0
    left = indexedItems.slice(Math.max(0, activeIndex - sideCapacity), activeIndex)
    right = indexedItems.slice(activeIndex + 1, activeIndex + 1 + sideCapacity)
  } else if (mode === 'left') {
    right = indexedItems.slice(activeIndex + 1, activeIndex + 1 + capacity)
  } else {
    left = indexedItems.slice(Math.max(0, activeIndex - capacity), activeIndex)
  }

  const companions = [...left, ...right]
  const selectedIndexes = new Set([activeIndex, ...companions.map((entry) => entry.index)])
  const remainder = indexedItems.filter((entry) => !selectedIndexes.has(entry.index))

  return {
    active,
    activeRowIndex: Math.floor(activeIndex / columns),
    mode,
    left,
    right,
    companions,
    remainder
  }
}

export function calculateSpotlightDimensions(
  containerWidth: number,
  columns: HomeColumnCount
): HomeSpotlightDimensions {
  const width = Math.max(0, containerWidth)

  if (columns === 1) {
    return {
      activeWidth: width,
      companionWidth: 0,
      companionGridWidth: 0,
      horizontalGap: HOME_HORIZONTAL_GAP,
      verticalGap: HOME_VERTICAL_GAP,
      labelBlockHeight: HOME_LABEL_BLOCK_HEIGHT
    }
  }

  if (columns === 2) {
    const availableWidth = Math.max(0, width - HOME_HORIZONTAL_GAP)
    const activeWidth = availableWidth * 0.6
    const companionWidth = availableWidth - activeWidth
    return {
      activeWidth,
      companionWidth,
      companionGridWidth: companionWidth,
      horizontalGap: HOME_HORIZONTAL_GAP,
      verticalGap: HOME_VERTICAL_GAP,
      labelBlockHeight: HOME_LABEL_BLOCK_HEIGHT
    }
  }

  const heightCompensation = (HOME_LABEL_BLOCK_HEIGHT + HOME_VERTICAL_GAP) / MEDIA_HEIGHT_RATIO
  const companionWidth = Math.max(
    0,
    (width - 2 * HOME_HORIZONTAL_GAP - heightCompensation) / 4
  )
  const activeWidth = Math.max(
    0,
    width - 2 * HOME_HORIZONTAL_GAP - 2 * companionWidth
  )

  return {
    activeWidth,
    companionWidth,
    companionGridWidth: 2 * companionWidth + HOME_HORIZONTAL_GAP,
    horizontalGap: HOME_HORIZONTAL_GAP,
    verticalGap: HOME_VERTICAL_GAP,
    labelBlockHeight: HOME_LABEL_BLOCK_HEIGHT
  }
}

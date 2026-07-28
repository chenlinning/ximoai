export type HomeColumnCount = 1 | 2 | 3

export interface IndexedHomeItem<T> {
  item: T
  index: number
}

export interface HomeRow<T> {
  index: number
  compact: boolean
  items: IndexedHomeItem<T>[]
}

const TWO_COLUMN_MIN_WIDTH = 808
const THREE_COLUMN_MIN_WIDTH = 1224

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

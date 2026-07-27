export type XimoAIHomeCoverType = 'image' | 'video' | 'html'

const videoExtensions = new Set(['.mp4', '.webm', '.ogg', '.mov', '.m4v'])
const htmlExtensions = new Set(['.html', '.htm', '.xhtml'])

export function resolveXimoAIHomeCoverType(value: string): XimoAIHomeCoverType {
  const trimmed = value.trim()
  if (!trimmed) return 'image'

  const dataMatch = trimmed.match(/^data:([^;,]+)[;,]/i)
  if (dataMatch) return coverTypeFromMime(dataMatch[1])

  try {
    const baseURL = typeof window === 'undefined' ? 'https://ximoai.invalid' : window.location.origin
    const pathname = new URL(trimmed, baseURL).pathname.toLowerCase()
    const extension = pathname.slice(pathname.lastIndexOf('.'))
    if (videoExtensions.has(extension)) return 'video'
    if (htmlExtensions.has(extension)) return 'html'
  } catch {
    // Keep legacy image behavior for invalid or opaque cover values.
  }

  return 'image'
}

export function decodeXimoAIHomeHTMLCover(value: string): string {
  const trimmed = value.trim()
  if (resolveXimoAIHomeCoverType(trimmed) !== 'html') return ''

  const separator = trimmed.indexOf(',')
  if (separator < 0 || !trimmed.slice(0, separator).toLowerCase().includes(';base64')) return ''

  try {
    const binary = atob(trimmed.slice(separator + 1))
    const bytes = Uint8Array.from(binary, (character) => character.charCodeAt(0))
    return new TextDecoder().decode(bytes)
  } catch {
    return ''
  }
}

function coverTypeFromMime(mime: string): XimoAIHomeCoverType {
  const normalized = mime.toLowerCase()
  if (normalized.startsWith('video/')) return 'video'
  if (normalized === 'text/html' || normalized === 'application/xhtml+xml') return 'html'
  return 'image'
}

import { buildApiUrl } from '@/api/url'

interface StreamWriter {
  write(chunk: Uint8Array): Promise<unknown>
  close(): Promise<unknown>
  abort?(reason?: unknown): Promise<unknown>
}

interface SaveFilePickerWindow extends Window {
  showSaveFilePicker?: (options: { suggestedName: string }) => Promise<{
    createWritable(): Promise<StreamWriter>
  }>
}

export async function streamResponseToWriter(
  response: Response,
  writer: StreamWriter,
  onProgress?: (percent: number) => void
): Promise<void> {
  if (!response.body) {
    await writer.abort?.(new Error('Download stream is unavailable'))
    throw new Error('Download stream is unavailable')
  }
  const total = Number(response.headers.get('Content-Length')) || 0
  const reader = response.body.getReader()
  let received = 0
  try {
    for (;;) {
      const { done, value } = await reader.read()
      if (done) break
      await writer.write(value)
      received += value.byteLength
      if (total > 0) {
        onProgress?.(Math.min(100, Math.round((received / total) * 100)))
      }
    }
    await writer.close()
    onProgress?.(100)
  } catch (error) {
    await writer.abort?.(error)
    throw error
  }
}

export async function saveVideoTask(
  taskID: string,
  fileName: string,
  onProgress?: (percent: number) => void
): Promise<string | null> {
  const picker = (window as SaveFilePickerWindow).showSaveFilePicker
  let writer: StreamWriter | null = null
  if (picker) {
    const handle = await picker({ suggestedName: fileName || 'video.mp4' })
    writer = await handle.createWritable()
  }

  const token = localStorage.getItem('auth_token')
  const response = await fetch(buildApiUrl(`/video-collector/tasks/${taskID}/download`), {
    headers: token ? { Authorization: `Bearer ${token}` } : {},
    credentials: 'include',
    cache: 'no-store'
  })
  if (!response.ok) {
    const payload = await response.json().catch(() => null) as { message?: string } | null
    await writer?.abort?.(new Error(payload?.message || 'Download failed'))
    throw new Error(payload?.message || 'Download failed')
  }

  if (writer) {
    await streamResponseToWriter(response, writer, onProgress)
  } else {
    const blob = await response.blob()
    const href = URL.createObjectURL(blob)
    const anchor = document.createElement('a')
    anchor.href = href
    anchor.download = fileName || 'video.mp4'
    anchor.click()
    window.setTimeout(() => URL.revokeObjectURL(href), 1000)
    onProgress?.(100)
  }
  return response.headers.get('X-Delete-At')
}

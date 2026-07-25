import { apiClient } from '@/api/client'

export interface MediaFormat {
  id: string
  extension: string
  width?: number
  height?: number
  videoCodec?: string
  audioCodec?: string
  approximateBytes?: number
  bitrateKbps?: number
  hasVideo: boolean
  hasAudio: boolean
}

export interface MediaInfo {
  id: string
  sourceUrl: string
  title: string
  uploader: string
  thumbnail?: string
  duration?: number
  extractor: string
  formats: MediaFormat[]
}

export type VideoTaskState = 'queued' | 'downloading' | 'processing' | 'completed' | 'cancelled' | 'failed' | 'expired'

export interface VideoTask {
  id: string
  state: VideoTaskState
  percent: number
  speed?: string
  eta?: string
  downloadedBytes?: number
  totalBytes?: number
  fileName?: string
  fileSize?: number
  error?: string
  createdAt: string
  completedAt?: string
  deleteAt?: string
}

export interface StartVideoTaskInput {
  sourceUrl: string
  mediaId: string
  title: string
  formatId: string
  hasAudio: boolean
}

export async function parseVideoURL(url: string): Promise<MediaInfo> {
  const { data } = await apiClient.post<MediaInfo>('/video-collector/parse', { url })
  return data
}

export async function startVideoTask(input: StartVideoTaskInput): Promise<VideoTask> {
  const { data } = await apiClient.post<VideoTask>('/video-collector/tasks', input)
  return data
}

export async function getVideoTask(taskID: string): Promise<VideoTask> {
  const { data } = await apiClient.get<VideoTask>(`/video-collector/tasks/${taskID}`)
  return data
}

export async function cancelVideoTask(taskID: string): Promise<void> {
  await apiClient.delete(`/video-collector/tasks/${taskID}`)
}

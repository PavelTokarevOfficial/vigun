import type { TimelineSegment } from '@/entities/template/model/types'

export type PipelineClip = {
  id: string
  title: string
  streamerName: string
  twitchClipId: string
  thumbnailUrl: string
  duration: number
  hasSource: boolean
  status: string
  error: string
  currentStep: string
  progress: number
  lastJobType: string
  lastJobStatus: string
  isReadyFragment: boolean
  editTimeline?: { segments: TimelineSegment[] }
}

export type RenderedVideo = {
  id: string
  clipId: string
  processingJobId: string
  title: string
  streamer: string
  twitchUrl: string
  thumbnailUrl: string
  templateName: string
  url: string
  createdAt: string
}

export type ProcessingJob = {
  id: string
  clipId: string
  clipTitle: string
  type: string
  status: 'pending' | 'running' | 'completed' | 'failed'
  currentStep: string
  error: string
  progress: number
  templateName: string
  isTrain: boolean
  fragmentCount: number
  createdAt: string
}

export type PipelineVideoPreview = {
  key: string
  title: string
  streamer: string
  templateName?: string
  url: string
  mode: 'video' | 'twitch'
}

export type InstagramContainer = {
  id: string
  status: string
  error?: string
}

export type InstagramPublishForm = {
  tunnelUrl: string
  caption: string
  shareToFeed: boolean
  collaborators: string
  coverUrl: string
  audioName: string
  locationId: string
  thumbOffset: string
}

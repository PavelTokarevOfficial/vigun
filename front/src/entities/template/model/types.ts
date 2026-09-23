export type LayerType =
  | 'video'
  | 'blur'
  | 'input_video'
  | 'asset_video'
  | 'image'
  | 'gif'
  | 'subtitles'
  | 'text'
  | 'audio'
  | 'color'

export type Layer = {
  id: string
  name: string
  type: LayerType
  x: number
  y: number
  width: number
  height: number
  visible: boolean
  opacity: number
  trackId?: string
  timelineSegmentId?: string
  startTime?: number
  endTime?: number
  source?: 'clip' | 'asset'
  fit?: 'cover' | 'contain' | 'stretch'
  assetId?: string
  text?: string
  textSource?: 'custom' | 'streamer_name'
  color?: string
  filters?: { blur?: number; brightness?: number }
  style?: {
    fontSize?: number
    alignment?: number
    marginV?: number
    outline?: number
    primaryColor?: string
    outlineColor?: string
  }
}

export type TemplateConfig = {
  version: 1
  canvas: { width: number; height: number; fps: number; background: string }
  layers: Layer[]
  timeline?: { segments: TimelineSegment[] }
  train?: { enabled: boolean; transitionAssetIds?: string[] }
}

export type TimelineSegment = {
  id: string
  source?: 'clip' | 'asset'
  assetId?: string
  clipId?: string
  start: number
  end: number
  sourceDuration?: number
}

export type VideoTemplate = {
  id: string
  name: string
  description: string
  previewAssetId: string | null
  previewUrl?: string
  isDefault: boolean
  configVersion: number
  config: TemplateConfig
  createdAt: string
  updatedAt: string
}

export function createDefaultConfig(): TemplateConfig {
  return {
    version: 1,
    canvas: { width: 1080, height: 1920, fps: 30, background: '#000000' },
    layers: [
      {
        id: 'background',
        name: 'Видео на фоне',
        type: 'video',
        source: 'clip',
        x: 0,
        y: 0,
        width: 1080,
        height: 1920,
        visible: true,
        opacity: 1,
        fit: 'cover',
      },
      {
        id: 'background-blur',
        name: 'Блюр фона',
        type: 'blur',
        x: 0,
        y: 0,
        width: 1080,
        height: 1920,
        visible: true,
        opacity: 1,
        filters: { blur: 25, brightness: -0.2 },
      },
      {
        id: 'clip',
        name: 'Видео',
        type: 'video',
        source: 'clip',
        x: 0,
        y: 0,
        width: 1080,
        height: 1920,
        visible: true,
        opacity: 1,
        fit: 'contain',
      },
      {
        id: 'subtitles',
        name: 'Субтитры',
        type: 'subtitles',
        x: 90,
        y: 1540,
        width: 900,
        height: 240,
        visible: true,
        opacity: 1,
        style: {
          fontSize: 8,
          alignment: 2,
          marginV: 100,
          outline: 2,
          primaryColor: '&H00FFFFFF',
          outlineColor: '&H00000000',
        },
      },
    ],
  }
}

/** Converts templates created by the first editor into the compact v2 layer UI. */
export function normalizeConfig(config: TemplateConfig): TemplateConfig {
  return {
    ...config,
    canvas: { ...config.canvas },
    timeline: config.timeline
      ? {
          segments: config.timeline.segments.map((segment) => ({
            ...segment,
            source: segment.source ?? 'clip',
          })),
        }
      : undefined,
    layers: config.layers.flatMap((layer) => {
      if (layer.type === 'input_video') {
        const video: Layer = {
          ...layer,
          type: 'video',
          source: 'clip',
          filters: undefined,
        }
        if (layer.filters?.blur || layer.filters?.brightness) {
          return [
            video,
            {
              ...layer,
              id: `${layer.id}-blur`,
              name: `Блюр · ${layer.name}`,
              type: 'blur',
              opacity: 1,
              fit: undefined,
            } satisfies Layer,
          ]
        }
        return [video]
      }
      if (layer.type === 'asset_video') {
        return [{ ...layer, type: 'video', source: 'asset' }]
      }
      if (layer.type === 'gif') return [{ ...layer, type: 'image' }]
      return [layer]
    }),
  }
}

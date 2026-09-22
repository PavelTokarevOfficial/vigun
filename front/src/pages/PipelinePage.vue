<script setup lang="ts">
import {
  ArrowLeft,
  Check,
  Download,
  ExternalLink,
  Play,
  Plus,
  Redo2,
  RotateCcw,
  Scissors,
  Send,
  Trash,
  Undo2,
  X,
} from '@lucide/vue'
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import type {
  Asset,
  AssetFolder,
  AssetLibrary,
} from '../entities/asset/model/types'
import {
  createDefaultConfig,
  type Layer,
  normalizeConfig,
  type TemplateConfig,
  type TimelineSegment,
  type VideoTemplate,
} from '../entities/template/model/types'
import { useTemplateEditor } from '../features/template-editor/model/useTemplateEditor'
import LayerPanel from '../features/template-editor/ui/LayerPanel.vue'
import PropertiesPanel from '../features/template-editor/ui/PropertiesPanel.vue'
import TemplateCanvas from '../features/template-editor/ui/TemplateCanvas.vue'
import VideoTimeline from '../features/template-editor/ui/VideoTimeline.vue'
import { readData, readError } from '../shared/api/http'
import AppButton from '../shared/ui/AppButton.vue'
import ErrorState from '../shared/ui/ErrorState.vue'

type Clip = {
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

type Video = {
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

type Job = {
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

type VideoPreview = {
  key: string
  title: string
  streamer: string
  templateName?: string
  url: string
  mode: 'video' | 'twitch'
}

type InstagramContainer = {
  id: string
  status: string
  error?: string
}

type InstagramForm = {
  tunnelUrl: string
  caption: string
  shareToFeed: boolean
  collaborators: string
  coverUrl: string
  audioName: string
  locationId: string
  thumbOffset: string
}

type Column = 'fragments' | 'renders'

const defaultInstagramCaption = `Лучшие моменты со стримов в коротком формате 🎬

Подписывайся, чтобы не пропустить новые клипы!

#стрим #стример #твич #twitch #twitchclips #клипы #нарезка #моменты #приколы #reels`

const clips = ref<Clip[]>([])
const videos = ref<Video[]>([])
const jobs = ref<Job[]>([])
const error = ref('')
const busy = ref('')
const dragged = ref<Clip | null>(null)
const queuedProcessIDs = ref(new Set<string>())
const processClipID = ref<string | null>(null)
const templates = ref<VideoTemplate[]>([])
const templatesLoading = ref(false)
const processStage = ref<'choose' | 'edit'>('choose')
const selectedTemplate = ref<VideoTemplate | null>(null)
const assets = ref<Asset[]>([])
const folders = ref<AssetFolder[]>([])
const sourceURL = ref('')
const sourceURLs = ref<Record<string, string>>({})
const processClipIDs = ref<string[]>([])
const timelineTime = ref(0)
const selectedTimelineSegmentID = ref<string | null>(null)
const previewVideo = ref<VideoPreview | null>(null)
const instagramVideo = ref<Video | null>(null)
const instagramBusy = ref(false)
const instagramMessage = ref('')
const trainSelectionMode = ref(false)
const selectedTrainClipIDs = ref(new Set<string>())
const fragmentClip = ref<Clip | null>(null)
const fragmentSourceURL = ref('')
const fragmentSegments = ref<TimelineSegment[]>([])
const fragmentSelectedSegmentID = ref<string | null>(null)
const fragmentVideo = ref<HTMLVideoElement | null>(null)
const fragmentPreviewTime = ref(0)
const fragmentPreviewSegmentIndex = ref(0)
const instagramForm = ref<InstagramForm>({
  tunnelUrl: '',
  caption: '',
  shareToFeed: true,
  collaborators: '',
  coverUrl: '',
  audioName: '',
  locationId: '',
  thumbOffset: '',
})
const renderEditor = useTemplateEditor(createDefaultConfig())

const downloaded = computed(() =>
  clips.value.filter(
    (clip) =>
      !clip.isReadyFragment &&
      (clip.hasSource ||
        ['saved', 'downloading'].includes(clip.status) ||
        clip.status === 'failed'),
  ),
)
const readyFragments = computed(() =>
  clips.value.filter((clip) => clip.isReadyFragment && clip.hasSource),
)
const trainJobs = computed(() =>
  jobs.value
    .filter(
      (job) =>
        job.type === 'process' &&
        job.isTrain &&
        (job.status !== 'completed' ||
          videos.value.some((video) => video.processingJobId === job.id)),
    )
    .slice(0, 6),
)
const availableTemplates = computed(() =>
  processClipIDs.value.length > 1
    ? templates.value.filter((template) => template.config.train?.enabled)
    : templates.value,
)
const processClip = computed(
  () => clips.value.find((clip) => clip.id === processClipID.value) ?? null,
)
async function load() {
  try {
    const [clipResponse, videoResponse, jobResponse] = await Promise.all([
      fetch('/api/clips'),
      fetch('/api/videos'),
      fetch('/api/jobs'),
    ])
    if (!clipResponse.ok || !videoResponse.ok || !jobResponse.ok) {
      throw new Error('Не удалось загрузить доску')
    }
    clips.value = (await clipResponse.json()).data || []
    videos.value = (await videoResponse.json()).data || []
    jobs.value = (await jobResponse.json()).data || []
    const downloadedIDs = new Set(
      clips.value
        .filter(
          (clip) =>
            clip.hasSource && ['downloaded', 'completed'].includes(clip.status),
        )
        .map((clip) => clip.id),
    )
    queuedProcessIDs.value = new Set(
      [...queuedProcessIDs.value].filter((id) => downloadedIDs.has(id)),
    )
  } catch (cause) {
    error.value =
      cause instanceof Error ? cause.message : 'Не удалось загрузить доску'
  }
}

async function action(
  id: string,
  path: 'download' | 'process' | 'retry',
  templateID?: string,
  config?: TemplateConfig,
) {
  busy.value = id
  error.value = ''
  const response = await fetch(`/api/clips/${id}/${path}`, {
    method: 'POST',
    ...(path === 'process'
      ? {
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ templateId: templateID, config }),
        }
      : {}),
  })
  if (!response.ok) {
    error.value = await readError(
      response,
      'Не удалось поставить задачу в очередь',
    )
  } else if (path === 'process') {
    queuedProcessIDs.value = new Set([...queuedProcessIDs.value, id])
  }
  busy.value = ''
  await load()
}

async function openTemplateChooser(id: string, clipIDs: string[] = [id]) {
  error.value = ''
  templatesLoading.value = true
  processClipID.value = id
  processClipIDs.value = clipIDs
  processStage.value = 'choose'
  selectedTemplate.value = null
  sourceURL.value = ''
  timelineTime.value = 0
  try {
    const [templateResponse, assetResponse, ...sourceResponses] =
      await Promise.all([
        fetch('/api/templates'),
        fetch('/api/assets'),
        ...clipIDs.map((clipID) => fetch(`/api/clips/${clipID}/source`)),
      ])
    if (!templateResponse.ok)
      throw new Error(
        await readError(templateResponse, 'Не удалось загрузить шаблоны'),
      )
    if (!assetResponse.ok)
      throw new Error(
        await readError(assetResponse, 'Не удалось загрузить ассеты'),
      )
    if (sourceResponses.some((response) => !response.ok))
      throw new Error('Не удалось открыть один из скачанных фрагментов')
    templates.value = await readData<VideoTemplate[]>(templateResponse)
    const library = await readData<AssetLibrary>(assetResponse)
    assets.value = library.assets
    folders.value = library.folders
    const sourceItems = await Promise.all(
      sourceResponses.map((response) => readData<{ url: string }>(response)),
    )
    sourceURLs.value = Object.fromEntries(
      clipIDs.map((clipID, index) => [clipID, sourceItems[index]?.url ?? '']),
    )
    sourceURL.value = sourceURLs.value[id] ?? ''
    if (!templates.value.length) {
      error.value = 'Сначала создайте хотя бы один шаблон видео.'
      processClipID.value = null
    }
  } catch (cause) {
    error.value =
      cause instanceof Error ? cause.message : 'Не удалось загрузить шаблоны'
    processClipID.value = null
  } finally {
    templatesLoading.value = false
  }
}

async function chooseTemplate(template: VideoTemplate) {
  selectedTemplate.value = template
  renderEditor.replace(normalizeConfig(template.config))
  const selectedClips = processClipIDs.value
    .map((id) => clips.value.find((clip) => clip.id === id))
    .filter((clip): clip is Clip => Boolean(clip))
  const transitionIDs = template.config.train?.enabled
    ? (template.config.train.transitionAssetIds ?? [])
    : []
  const transitionDurations = new Map<string, number>()
  await Promise.all(
    transitionIDs.map(async (assetID) => {
      const asset = assets.value.find((item) => item.id === assetID)
      transitionDurations.set(assetID, await videoDuration(asset?.url))
    }),
  )
  const segments = selectedClips.flatMap((clip, clipIndex) => {
    const sourceSegments = clip.editTimeline?.segments?.length
      ? clip.editTimeline.segments
      : [
          {
            id: `fragment-${crypto.randomUUID().slice(0, 8)}`,
            source: 'clip' as const,
            start: 0,
            end: Math.max(0.1, clip.duration),
            sourceDuration: Math.max(0.1, clip.duration),
          },
        ]
    const items: TimelineSegment[] = sourceSegments.map((segment) => ({
      ...segment,
      id: `segment-${crypto.randomUUID().slice(0, 8)}`,
      source: 'clip' as const,
      clipId: clip.id,
      assetId: undefined,
    }))
    if (clipIndex < selectedClips.length - 1 && transitionIDs.length) {
      const assetID = transitionIDs[clipIndex % transitionIDs.length]
      if (assetID) {
        const duration = transitionDurations.get(assetID) ?? 2
        items.push({
          id: `transition-${crypto.randomUUID().slice(0, 8)}`,
          source: 'asset' as const,
          assetId: assetID,
          clipId: undefined,
          start: 0,
          end: duration,
          sourceDuration: duration,
        })
      }
    }
    return items
  })
  let outputCursor = 0
  const transitionLayers: Layer[] = []
  for (const segment of segments) {
    const duration = Math.max(0, segment.end - segment.start)
    if (segment.source === 'asset' && segment.assetId) {
      const asset = assets.value.find((item) => item.id === segment.assetId)
      transitionLayers.push({
        id: `transition-layer-${crypto.randomUUID().slice(0, 8)}`,
        trackId: 'train-transitions',
        timelineSegmentId: segment.id,
        name: `Перебивка · ${asset?.name ?? 'Видео'}`,
        type: 'video',
        source: 'asset',
        assetId: segment.assetId,
        x: 0,
        y: 0,
        width: renderEditor.draft.value.canvas.width,
        height: renderEditor.draft.value.canvas.height,
        visible: true,
        opacity: 1,
        fit: 'cover',
        startTime: outputCursor,
        endTime: outputCursor + duration,
      })
    }
    outputCursor += duration
  }
  renderEditor.updateConfig({
    timeline: { segments },
    layers: [...renderEditor.draft.value.layers, ...transitionLayers],
  })
  timelineTime.value = 0
  selectedTimelineSegmentID.value = null
  processStage.value = 'edit'
}

function selectEditorLayer(id: string | null) {
  selectedTimelineSegmentID.value = null
  const source = renderEditor.draft.value.layers.find((item) => item.id === id)
  if (!source) {
    renderEditor.selectedLayerID.value = null
    return
  }
  if (source.timelineSegmentId) {
    let cursor = 0
    for (const segment of renderEditor.draft.value.timeline?.segments ?? []) {
      const duration = Math.max(0, segment.end - segment.start)
      if (segment.id === source.timelineSegmentId) {
        timelineTime.value = cursor + Math.min(0.05, duration / 2)
        break
      }
      cursor += duration
    }
  }
  const trackID = source.trackId ?? source.id
  const active = renderEditor.draft.value.layers.find(
    (layer) =>
      (layer.trackId ?? layer.id) === trackID &&
      timelineTime.value >= (layer.startTime ?? 0) &&
      (layer.endTime === undefined || timelineTime.value <= layer.endTime),
  )
  renderEditor.selectedLayerID.value = active?.id ?? source.id
}

function updateEditorLayerTrack(
  id: string,
  patch: Partial<TemplateConfig['layers'][number]>,
) {
  const source = renderEditor.draft.value.layers.find(
    (layer) => layer.id === id,
  )
  if (!source) return
  const trackID = source.trackId ?? source.id
  renderEditor.updateConfig({
    layers: renderEditor.draft.value.layers.map((layer) =>
      (layer.trackId ?? layer.id) === trackID ? { ...layer, ...patch } : layer,
    ),
  })
}

function addEditorLayer(type: Parameters<typeof renderEditor.addLayer>[0]) {
  selectedTimelineSegmentID.value = null
  renderEditor.addLayer(type)
}

function selectTimelineSegment(id: string) {
  selectedTimelineSegmentID.value = id
  renderEditor.selectedLayerID.value = null
}

function removeEditorLayer(id: string) {
  const layer = renderEditor.draft.value.layers.find((item) => item.id === id)
  if (!layer?.timelineSegmentId) {
    renderEditor.removeLayer(id)
    return
  }
  const timeline = renderEditor.draft.value.timeline
  const trackID = layer.trackId ?? layer.id
  const removedSegmentIDs = new Set(
    renderEditor.draft.value.layers
      .filter((item) => (item.trackId ?? item.id) === trackID)
      .map((item) => item.timelineSegmentId)
      .filter((id): id is string => Boolean(id)),
  )
  const remainingLayers = renderEditor.draft.value.layers.filter(
    (item) => (item.trackId ?? item.id) !== trackID,
  )
  const remainingLinkedSegmentIDs = new Set(
    remainingLayers
      .map((item) => item.timelineSegmentId)
      .filter((id): id is string => Boolean(id)),
  )
  renderEditor.updateConfig({
    layers: remainingLayers,
    ...(timeline
      ? {
          timeline: {
            segments: timeline.segments.filter(
              (segment) =>
                !removedSegmentIDs.has(segment.id) ||
                remainingLinkedSegmentIDs.has(segment.id),
            ),
          },
        }
      : {}),
  })
  renderEditor.selectedLayerID.value = null
}

async function sendToRender() {
  const id = processClipID.value
  const template = selectedTemplate.value
  if (!id || !template) return
  const missingAsset = renderEditor.draft.value.layers.find(
    (layer) =>
      (layer.type === 'image' ||
        layer.type === 'gif' ||
        layer.type === 'asset_video' ||
        (layer.type === 'video' && layer.source === 'asset')) &&
      !layer.assetId,
  )
  if (missingAsset) {
    error.value = `Для слоя «${missingAsset.name}» выберите файл.`
    return
  }
  await action(id, 'process', template.id, renderEditor.draft.value)
  if (!error.value) closeProcessDialog()
}

function closeProcessDialog() {
  processClipID.value = null
  selectedTemplate.value = null
  processStage.value = 'choose'
  sourceURL.value = ''
  sourceURLs.value = {}
  processClipIDs.value = []
  timelineTime.value = 0
  selectedTimelineSegmentID.value = null
}

function videoDuration(url?: string) {
  if (!url) return Promise.resolve(2)
  return new Promise<number>((resolve) => {
    const video = document.createElement('video')
    const finish = (duration: number) => {
      window.clearTimeout(timer)
      video.removeAttribute('src')
      resolve(duration)
    }
    const timer = window.setTimeout(() => finish(2), 5000)
    video.preload = 'metadata'
    video.onloadedmetadata = () =>
      finish(Number.isFinite(video.duration) ? video.duration : 2)
    video.onerror = () => finish(2)
    video.src = url
  })
}

async function updateFragment(
  clip: Clip,
  ready: boolean,
  segments = clip.editTimeline?.segments,
) {
  busy.value = clip.id
  error.value = ''
  const response = await fetch(`/api/clips/${clip.id}/fragment`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      ready,
      timeline: segments?.length ? { segments } : null,
    }),
  })
  if (!response.ok)
    error.value = await readError(response, 'Не удалось сохранить фрагмент')
  busy.value = ''
  await load()
}

async function openFragmentEditor(clip: Clip) {
  busy.value = clip.id
  error.value = ''
  try {
    const response = await fetch(`/api/clips/${clip.id}/source`)
    fragmentSourceURL.value = (await readData<{ url: string }>(response)).url
    fragmentClip.value = clip
    fragmentSegments.value = (
      clip.editTimeline?.segments?.length
        ? clip.editTimeline.segments
        : [
            {
              id: `fragment-${crypto.randomUUID().slice(0, 8)}`,
              start: 0,
              end: Math.max(0.1, clip.duration),
              sourceDuration: Math.max(0.1, clip.duration),
            },
          ]
    ).map((segment) => ({
      ...segment,
      source: 'clip' as const,
      sourceDuration: segment.sourceDuration ?? clip.duration,
    }))
    fragmentSelectedSegmentID.value = fragmentSegments.value[0]?.id ?? null
    fragmentPreviewTime.value = 0
    fragmentPreviewSegmentIndex.value = 0
  } catch (cause) {
    error.value =
      cause instanceof Error ? cause.message : 'Не удалось открыть исходник'
  } finally {
    busy.value = ''
  }
}

function setFragmentTimelineTime(time: number) {
  const duration = fragmentOutputDuration()
  const target = Math.min(Math.max(0, time), duration)
  let outputStart = 0
  for (const [index, segment] of fragmentSegments.value.entries()) {
    const duration = Math.max(0, segment.end - segment.start)
    if (
      target <= outputStart + duration ||
      index === fragmentSegments.value.length - 1
    ) {
      fragmentPreviewSegmentIndex.value = index
      fragmentPreviewTime.value = target
      if (fragmentVideo.value) {
        fragmentVideo.value.currentTime = Math.max(
          0,
          Math.min(segment.end, segment.start + target - outputStart),
        )
      }
      return
    }
    outputStart += duration
  }
}

function fragmentOutputDuration() {
  return fragmentSegments.value.reduce(
    (total, segment) => total + Math.max(0, segment.end - segment.start),
    0,
  )
}

function fragmentSegmentOutputStart(index: number) {
  return fragmentSegments.value
    .slice(0, index)
    .reduce(
      (total, segment) => total + Math.max(0, segment.end - segment.start),
      0,
    )
}

function updateFragmentSegments(segments: TimelineSegment[]) {
  fragmentSegments.value = segments
  setFragmentTimelineTime(
    Math.min(fragmentPreviewTime.value, fragmentOutputDuration()),
  )
}

function startFragmentPreview() {
  const video = fragmentVideo.value
  if (!video || !fragmentSegments.value.length) return
  if (fragmentPreviewTime.value >= fragmentOutputDuration() - 0.05) {
    setFragmentTimelineTime(0)
  } else {
    setFragmentTimelineTime(fragmentPreviewTime.value)
  }
}

function updateFragmentPreview() {
  const video = fragmentVideo.value
  const index = fragmentPreviewSegmentIndex.value
  const segment = fragmentSegments.value[index]
  if (!video || !segment) return
  if (video.currentTime >= segment.end - 0.04) {
    const next = fragmentSegments.value[index + 1]
    if (!next) {
      fragmentPreviewTime.value = fragmentOutputDuration()
      video.pause()
      return
    }
    fragmentPreviewSegmentIndex.value = index + 1
    fragmentPreviewTime.value = fragmentSegmentOutputStart(index + 1)
    video.currentTime = next.start
    void video.play().catch(() => undefined)
    return
  }
  if (video.currentTime < segment.start) {
    video.currentTime = segment.start
    return
  }
  fragmentPreviewTime.value =
    fragmentSegmentOutputStart(index) + video.currentTime - segment.start
}

async function saveFragmentEdit() {
  const clip = fragmentClip.value
  if (!clip) return
  const segments = fragmentSegments.value
    .map((segment) => ({ ...segment, source: 'clip' as const }))
    .filter((segment) => segment.end > segment.start)
  await updateFragment(clip, true, segments)
  if (!error.value) fragmentClip.value = null
}

function toggleTrainClip(id: string) {
  const next = new Set(selectedTrainClipIDs.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  selectedTrainClipIDs.value = next
}

function startTrain() {
  const ids = [...selectedTrainClipIDs.value]
  if (ids.length < 2) {
    error.value = 'Выберите минимум два готовых фрагмента.'
    return
  }
  void openTemplateChooser(ids[0] as string, ids)
}

async function remove(clip: Clip) {
  busy.value = clip.id
  error.value = ''
  const response = await fetch(`/api/clips/${clip.id}`, { method: 'DELETE' })
  if (!response.ok) {
    error.value = await readError(response, 'Не удалось удалить клип')
  }
  busy.value = ''
  await load()
}

async function removeRenderedVideo(video: Video) {
  busy.value = video.id
  error.value = ''
  const response = await fetch(`/api/videos/${video.id}`, { method: 'DELETE' })
  if (!response.ok) {
    error.value = await readError(response, 'Не удалось удалить готовое видео')
  }
  busy.value = ''
  await load()
}

function twitchEmbedURL(clip: Clip) {
  const query = new URLSearchParams({
    clip: clip.twitchClipId,
    parent: window.location.hostname || 'localhost',
    autoplay: 'true',
    muted: 'false',
  })
  return `https://clips.twitch.tv/embed?${query}`
}

async function openClipPreview(clip: Clip) {
  error.value = ''
  if (!clip.hasSource) {
    previewVideo.value = {
      key: `twitch-${clip.id}`,
      title: clip.title,
      streamer: clip.streamerName,
      url: twitchEmbedURL(clip),
      mode: 'twitch',
    }
    return
  }

  busy.value = clip.id
  try {
    const response = await fetch(`/api/clips/${clip.id}/source`)
    if (!response.ok) {
      throw new Error(
        await readError(response, 'Не удалось открыть скачанное видео'),
      )
    }
    const source = await readData<{ url: string }>(response)
    previewVideo.value = {
      key: `source-${clip.id}`,
      title: clip.title,
      streamer: clip.streamerName,
      url: source.url,
      mode: 'video',
    }
  } catch (cause) {
    error.value =
      cause instanceof Error
        ? cause.message
        : 'Не удалось открыть скачанное видео'
  } finally {
    busy.value = ''
  }
}

function openRenderedVideo(video: Video) {
  previewVideo.value = {
    key: `render-${video.id}`,
    title: video.title,
    streamer: video.streamer,
    templateName: video.templateName,
    url: video.url,
    mode: 'video',
  }
}

function openInstagramDialog(video: Video) {
  instagramVideo.value = video
  instagramMessage.value = ''
  instagramForm.value = {
    tunnelUrl: '',
    caption: defaultInstagramCaption,
    shareToFeed: true,
    collaborators: '',
    coverUrl: '',
    audioName: '',
    locationId: '',
    thumbOffset: '',
  }
}

function closeInstagramDialog() {
  if (instagramBusy.value) return
  instagramVideo.value = null
  instagramMessage.value = ''
}

function publicVideoURL(video: Video) {
  return `${instagramForm.value.tunnelUrl.trim().replace(/\/$/, '')}/api/videos/${video.id}/content`
}

function wait(milliseconds: number) {
  return new Promise((resolve) => window.setTimeout(resolve, milliseconds))
}

async function publishToInstagram() {
  const video = instagramVideo.value
  if (!video) return
  instagramBusy.value = true
  instagramMessage.value = 'Instagram скачивает и обрабатывает видео…'
  error.value = ''
  try {
    const thumbOffset = instagramForm.value.thumbOffset.trim()
    const response = await fetch(`/api/videos/${video.id}/instagram`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        videoUrl: publicVideoURL(video),
        caption: instagramForm.value.caption,
        shareToFeed: instagramForm.value.shareToFeed,
        collaborators: instagramForm.value.collaborators
          .split(',')
          .map((value) => value.trim())
          .filter(Boolean),
        coverUrl: instagramForm.value.coverUrl.trim(),
        audioName: instagramForm.value.audioName.trim(),
        locationId: instagramForm.value.locationId.trim(),
        thumbOffset: thumbOffset ? Number(thumbOffset) : null,
      }),
    })
    const container = await readData<InstagramContainer>(response)
    let status = container
    for (
      let attempt = 0;
      attempt < 60 && status.status === 'IN_PROGRESS';
      attempt += 1
    ) {
      await wait(5000)
      status = await readData<InstagramContainer>(
        await fetch(`/api/videos/${video.id}/instagram/${container.id}`),
      )
    }
    if (status.status !== 'FINISHED') {
      throw new Error(
        status.error ||
          (status.status === 'IN_PROGRESS'
            ? 'Instagram не обработал видео за 5 минут'
            : `Instagram вернул статус ${status.status}`),
      )
    }
    await readData<{ id: string }>(
      await fetch(`/api/videos/${video.id}/instagram/${container.id}/publish`, {
        method: 'POST',
      }),
    )
    instagramMessage.value = 'Видео опубликовано в Instagram.'
  } catch (cause) {
    instagramMessage.value = ''
    error.value =
      cause instanceof Error ? cause.message : 'Не удалось опубликовать видео'
  } finally {
    instagramBusy.value = false
  }
}

function canDelete(clip: Clip) {
  return (
    ['saved', 'downloaded', 'failed', 'completed'].includes(clip.status) &&
    !isProcessQueued(clip)
  )
}

function isProcessQueued(clip: Clip) {
  return (
    queuedProcessIDs.value.has(clip.id) ||
    (clip.lastJobType === 'process' &&
      ['pending', 'running'].includes(clip.lastJobStatus))
  )
}

function drop(column: Column) {
  const clip = dragged.value
  dragged.value = null
  if (!clip) return

  if (column === 'fragments' && clip.hasSource && !clip.isReadyFragment) {
    void updateFragment(clip, true)
  }
  if (column === 'renders' && clip.isReadyFragment && !isProcessQueued(clip)) {
    void openTemplateChooser(clip.id)
  }
}

function statusText(clip: Clip) {
  if (clip.status === 'saved') return 'Ожидает скачивания'
  if (isProcessQueued(clip)) {
    return clip.lastJobStatus === 'running'
      ? `${clip.currentStep || 'Запуск обработки'} · ${clip.progress}%`
      : 'В очереди на обработку'
  }
  if (clip.status === 'downloaded') return 'Готов к обработке'
  if (clip.status === 'completed') return 'Есть готовый рендер'
  if (clip.status === 'failed') return 'Ошибка'
  return `${clip.currentStep || clip.status} · ${clip.progress}%`
}

function trainJobStatus(job: Job) {
  if (job.status === 'pending') return 'В очереди'
  if (job.status === 'running') return `Рендерится · ${job.progress}%`
  if (job.status === 'completed') return 'Готово'
  return 'Ошибка'
}

function trainJobStatusClass(job: Job) {
  if (job.status === 'completed') return 'bg-emerald-100 text-emerald-700'
  if (job.status === 'failed') return 'bg-red-100 text-red-700'
  if (job.status === 'running') return 'bg-violet-100 text-violet-700'
  return 'bg-amber-100 text-amber-700'
}

function videoForJob(job: Job) {
  return videos.value.find((video) => video.processingJobId === job.id)
}

function openTrainJobVideo(job: Job) {
  const video = videoForJob(job)
  if (video) openRenderedVideo(video)
}

let timer: number | undefined
onMounted(() => {
  void load()
  timer = window.setInterval(load, 3000)
})
onBeforeUnmount(() => {
  if (timer) window.clearInterval(timer)
})
</script>

<template>
  <section>
    <ErrorState v-if="error" class="mt-4" :message="error" />

    <div class="mt-6 rounded-xl border border-slate-200 bg-white p-3">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 class="font-semibold">Доска монтажа</h2>
          <p class="text-sm text-slate-500">
            Скачайте клип, подготовьте фрагмент и отправьте его на рендер
            отдельно или в общей сборке.
          </p>
        </div>
        <div class="flex flex-wrap gap-2">
          <AppButton
            variant="secondary"
            @click="trainSelectionMode = !trainSelectionMode; selectedTrainClipIDs = new Set()"
          >
            {{ trainSelectionMode ? 'Отменить паровозик' : 'Паровозик' }}
          </AppButton>
          <AppButton
            v-if="trainSelectionMode"
            :disabled="selectedTrainClipIDs.size < 2"
            @click="startTrain"
          >
            Собрать выбранные · {{ selectedTrainClipIDs.size }}
          </AppButton>
        </div>
      </div>

      <div class="mt-3 border-t border-slate-200 pt-3">
        <div class="flex items-center justify-between gap-3">
          <h3 class="text-sm font-semibold">Рендеры паровозика</h3>
          <span class="text-xs text-slate-500"
            >Последние {{ trainJobs.length }}</span
          >
        </div>
        <p v-if="!trainJobs.length" class="mt-2 text-sm text-slate-500">
          Сборок пока нет. Выберите минимум два готовых фрагмента и нажмите
          «Собрать выбранные».
        </p>
        <ul v-else class="mt-2 grid gap-2 lg:grid-cols-2 xl:grid-cols-3">
          <li
            v-for="job in trainJobs"
            :key="job.id"
            class="rounded-lg border border-slate-200 p-3"
          >
            <div class="flex items-start justify-between gap-2">
              <div class="min-w-0">
                <p class="truncate text-sm font-medium">
                  Паровозик · {{ job.fragmentCount }} фрагм.
                </p>
                <p class="truncate text-xs text-slate-500">
                  {{ job.templateName || 'Шаблон' }} ·
                  {{ new Date(job.createdAt).toLocaleString() }}
                </p>
              </div>
              <span
                class="shrink-0 rounded-full px-2 py-1 text-xs font-medium"
                :class="trainJobStatusClass(job)"
              >
                {{ trainJobStatus(job) }}
              </span>
            </div>
            <div
              v-if="job.status === 'pending' || job.status === 'running'"
              class="mt-2 h-1.5 overflow-hidden rounded-full bg-slate-100"
            >
              <div
                class="h-full rounded-full bg-violet-600 transition-all"
                :style="{ width: `${Math.max(job.status === 'pending' ? 4 : job.progress, 4)}%` }"
              />
            </div>
            <p
              v-if="job.status === 'failed' && job.error"
              class="mt-2 line-clamp-2 text-xs text-red-600"
            >
              {{ job.error.split('\n')[0] }}
            </p>
            <AppButton
              v-if="job.status === 'completed' && videoForJob(job)"
              class="mt-2 w-full"
              variant="secondary"
              @click="openTrainJobVideo(job)"
            >
              <Play class="mr-1 inline size-4" />Посмотреть результат
            </AppButton>
          </li>
        </ul>
      </div>
    </div>

    <div class="mt-6 grid gap-4 xl:grid-cols-4">
      <section class="min-w-0">
        <h3 class="font-semibold">Скачанные · {{ downloaded.length }}</h3>
        <div class="mt-3 space-y-3">
          <p v-if="!downloaded.length" class="text-sm text-slate-500">
            Пока пусто.
          </p>
          <article
            v-for="clip in downloaded"
            :key="clip.id"
            draggable="true"
            class="relative cursor-grab overflow-hidden rounded-xl active:cursor-grabbing"
            @dragstart="dragged = clip"
            @dragend="dragged = null"
          >
            <img
              v-if="clip.thumbnailUrl"
              :src="clip.thumbnailUrl"
              draggable="false"
              :alt="`Превью клипа: ${clip.title}`"
              class="aspect-video w-full object-cover"
            >

            <div
              class="absolute top-0 px-3 py-1 text-white [-webkit-text-stroke:2px_black] [paint-order:stroke_fill]"
            >
              <b>{{ clip.title }}</b>
              <p>{{ clip.streamerName }} · {{ statusText(clip) }}</p>
              <p v-if="clip.error" class="mt-1 text-sm text-red-600">
                {{ clip.error }}
              </p>
            </div>

            <div class="absolute right-0 bottom-0 flex gap-2 p-3">
              <AppButton
                v-if="clip.hasSource"
                class="grid h-8 w-8 place-content-center"
                :disabled="busy === clip.id"
                title="Посмотреть скачанное видео"
                :aria-label="`Посмотреть скачанное видео «${clip.title}»`"
                @click="openClipPreview(clip)"
              >
                <Play :size="16" />
              </AppButton>
              <AppButton
                v-if="['downloaded', 'completed'].includes(clip.status) && !isProcessQueued(clip)"
                :disabled="busy === clip.id"
                class="grid place-content-center w-8 h-8"
                title="Обрезать и подготовить фрагмент"
                @click="openFragmentEditor(clip)"
              >
                <Scissors :size="16" />
              </AppButton>
              <AppButton
                v-if="clip.hasSource && !isProcessQueued(clip)"
                :disabled="busy === clip.id"
                class="grid h-8 w-8 place-content-center"
                title="Отправить в готовые фрагменты без редактирования"
                @click="updateFragment(clip, true)"
              >
                <Check :size="16" />
              </AppButton>
              <AppButton
                v-if="clip.status === 'failed'"
                class="grid place-content-center w-8 h-8"
                :disabled="busy === clip.id"
                @click="action(clip.id, 'retry')"
              >
                <RotateCcw :size="16" />
              </AppButton>
              <AppButton
                v-if="canDelete(clip)"
                variant="danger"
                class="grid place-content-center w-8 h-8"
                :disabled="busy === clip.id"
                @click="remove(clip)"
              >
                <Trash :size="16" />
              </AppButton>
            </div>
          </article>
        </div>
      </section>

      <!-- biome-ignore lint/a11y/noStaticElementInteractions: native drop target for the drag-and-drop board -->
      <section
        class="xl:border-l xl:border-dashed xl:border-slate-300 xl:pl-4"
        @dragover.prevent
        @drop="drop('fragments')"
      >
        <h3 class="font-semibold">
          Готовые фрагменты · {{ readyFragments.length }}
        </h3>
        <div class="mt-3 space-y-3">
          <p v-if="!readyFragments.length" class="text-sm text-slate-500">
            Пока пусто.
          </p>
          <article
            v-for="clip in readyFragments"
            :key="clip.id"
            draggable="true"
            class="relative cursor-grab overflow-hidden rounded-xl ring-offset-2 active:cursor-grabbing"
            :class="selectedTrainClipIDs.has(clip.id) ? 'ring-2 ring-violet-500' : ''"
            :tabindex="trainSelectionMode ? 0 : undefined"
            @dragstart="dragged = clip"
            @dragend="dragged = null"
            @click="trainSelectionMode && toggleTrainClip(clip.id)"
            @keydown.enter="trainSelectionMode && toggleTrainClip(clip.id)"
          >
            <img
              v-if="clip.thumbnailUrl"
              :src="clip.thumbnailUrl"
              draggable="false"
              :alt="`Превью фрагмента: ${clip.title}`"
              class="aspect-video w-full object-cover"
            >
            <div
              class="absolute top-0 px-3 py-1 text-white [-webkit-text-stroke:2px_black] [paint-order:stroke_fill]"
            >
              <b>{{ clip.title }}</b>
              <p>{{ clip.streamerName }}</p>
            </div>
            <span
              v-if="trainSelectionMode"
              class="absolute left-3 bottom-3 grid size-7 place-content-center rounded-full bg-white text-violet-700"
            >
              <Check v-if="selectedTrainClipIDs.has(clip.id)" :size="17" />
            </span>
            <div v-else class="absolute right-0 bottom-0 flex gap-2 p-3">
              <AppButton
                class="grid h-8 w-8 place-content-center"
                @click.stop="openClipPreview(clip)"
              >
                <Play :size="16" />
              </AppButton>
              <AppButton
                class="grid h-8 w-8 place-content-center"
                title="Изменить монтаж"
                @click.stop="openFragmentEditor(clip)"
              >
                <Scissors :size="16" />
              </AppButton>
              <AppButton
                class="grid h-8 w-8 place-content-center"
                title="Отправить на рендер"
                @click.stop="openTemplateChooser(clip.id)"
              >
                <Plus :size="16" />
              </AppButton>
              <AppButton
                variant="secondary"
                class="grid h-8 w-8 place-content-center"
                title="Вернуть в скачанные"
                @click.stop="updateFragment(clip, false)"
              >
                <ArrowLeft :size="16" />
              </AppButton>
            </div>
          </article>
        </div>
      </section>

      <!-- biome-ignore lint/a11y/noStaticElementInteractions: native drop target for the drag-and-drop board -->
      <section
        class="col-span-2 xl:border-l xl:border-dashed xl:border-slate-300 xl:pl-4"
        @dragover.prevent
        @drop="drop('renders')"
      >
        <h3 class="font-semibold">Готовые видео · {{ videos.length }}</h3>
        <div class="grid grid-cols-2 gap-3 mt-3">
          <p v-if="!videos.length" class="text-sm text-slate-500">
            Пока пусто.
          </p>
          <article
            v-for="video in videos"
            :key="video.id"
            class="relative overflow-hidden rounded-xl"
          >
            <img
              v-if="video.thumbnailUrl"
              :src="video.thumbnailUrl"
              :alt="`Превью клипа: ${video.title}`"
              class="aspect-video w-full object-cover"
            >
            <div
              v-else
              class="grid aspect-video w-full place-content-center bg-slate-100 text-sm text-slate-500"
            >
              Нет превью
            </div>

            <div
              class="absolute top-0 px-3 py-1 text-white [-webkit-text-stroke:2px_black] [paint-order:stroke_fill]"
            >
              <b>{{ video.title }}</b>
              <p>
                {{ video.streamer }}
                <template v-if="video.templateName">
                  · {{ video.templateName }}</template
                >
                · {{ new Date(video.createdAt).toLocaleString() }}
              </p>
            </div>

            <div class="absolute right-0 bottom-0 flex gap-2 p-3">
              <a
                :href="video.twitchUrl"
                target="_blank"
                rel="noopener noreferrer"
                class="app-button app-button--secondary grid h-8 w-8 place-content-center"
                title="Открыть оригинал на Twitch"
                :aria-label="`Открыть оригинал «${video.title}» на Twitch`"
              >
                <ExternalLink :size="16" />
              </a>
              <AppButton
                class="grid h-8 w-8 place-content-center"
                title="Открыть готовое видео"
                :aria-label="`Открыть готовое видео «${video.title}»`"
                @click="openRenderedVideo(video)"
              >
                <Play :size="16" />
              </AppButton>
              <AppButton
                variant="secondary"
                class="grid h-8 w-8 place-content-center"
                title="Опубликовать в Instagram"
                :aria-label="`Опубликовать «${video.title}» в Instagram`"
                @click="openInstagramDialog(video)"
              >
                <Send :size="16" />
              </AppButton>
              <a
                :href="`/api/videos/${video.id}/download`"
                class="app-button app-button--primary grid h-8 w-8 place-content-center"
                title="Скачать готовое видео"
                :aria-label="`Скачать готовое видео «${video.title}»`"
              >
                <Download :size="16" />
              </a>
              <AppButton
                variant="danger"
                class="grid h-8 w-8 place-content-center"
                :disabled="busy === video.id"
                title="Удалить готовое видео"
                :aria-label="`Удалить готовое видео «${video.title}»`"
                @click="removeRenderedVideo(video)"
              >
                <Trash :size="16" />
              </AppButton>
            </div>
          </article>
        </div>
      </section>
    </div>

    <div
      v-if="previewVideo"
      class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/70 p-4"
      role="dialog"
      aria-modal="true"
      aria-labelledby="video-preview-title"
    >
      <section
        class="w-full max-w-4xl overflow-hidden rounded-2xl bg-white shadow-2xl"
      >
        <header
          class="flex items-start justify-between gap-4 border-b border-slate-200 px-5 py-4"
        >
          <div>
            <h2 id="video-preview-title" class="text-lg font-semibold">
              {{ previewVideo.title }}
            </h2>
            <p class="mt-1 text-sm text-slate-600">
              {{ previewVideo.streamer }}
              <template v-if="previewVideo.templateName">
                · {{ previewVideo.templateName }}</template
              >
            </p>
          </div>
          <button
            type="button"
            class="rounded-lg p-2 text-slate-600 hover:bg-slate-100"
            aria-label="Закрыть просмотр видео"
            @click="previewVideo = null"
          >
            <X class="size-5" />
          </button>
        </header>
        <div class="flex justify-center bg-slate-950 p-4">
          <video
            v-if="previewVideo.mode === 'video'"
            :key="previewVideo.key"
            :src="previewVideo.url"
            controls
            autoplay
            class="max-h-[75vh] max-w-full"
          />
          <iframe
            v-else
            :key="previewVideo.key"
            :src="previewVideo.url"
            :title="`Twitch-клип: ${previewVideo.title}`"
            class="aspect-video max-h-[75vh] w-full border-0"
            allow="autoplay; fullscreen"
            allowfullscreen
          />
        </div>
      </section>
    </div>

    <div
      v-if="instagramVideo"
      class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/70 p-4"
      role="dialog"
      aria-modal="true"
      aria-labelledby="instagram-dialog-title"
    >
      <section
        class="max-h-[92vh] w-full max-w-3xl overflow-auto rounded-2xl bg-white shadow-2xl"
      >
        <header
          class="flex items-start justify-between gap-4 border-b border-slate-200 px-5 py-4"
        >
          <div>
            <h2 id="instagram-dialog-title" class="text-lg font-semibold">
              Публикация Reels
            </h2>
            <p class="mt-1 text-sm text-slate-600">
              {{ instagramVideo.title }}
            </p>
          </div>
          <button
            type="button"
            class="rounded-lg p-2 text-slate-600 hover:bg-slate-100 disabled:opacity-50"
            :disabled="instagramBusy"
            aria-label="Закрыть публикацию в Instagram"
            @click="closeInstagramDialog"
          >
            <X class="size-5" />
          </button>
        </header>

        <form class="space-y-4 p-5" @submit.prevent="publishToInstagram">
          <label class="block text-sm font-medium">
            HTTPS URL Cloudflare Tunnel
            <input
              v-model="instagramForm.tunnelUrl"
              type="url"
              required
              placeholder="https://example.trycloudflare.com"
              class="mt-1 w-full rounded-lg border border-slate-300 px-3 py-2 font-normal"
            >
          </label>
          <p
            class="break-all rounded-lg bg-slate-100 p-3 text-xs text-slate-600"
          >
            Instagram получит:
            <a
              :href="publicVideoURL(instagramVideo)"
              target="_blank"
              rel="noopener noreferrer"
            >
              {{ publicVideoURL(instagramVideo) }}
            </a>
          </p>
          <label class="block text-sm font-medium">
            Подпись
            <textarea
              v-model="instagramForm.caption"
              rows="4"
              class="mt-1 w-full h-50 rounded-lg border border-slate-300 px-3 py-2 font-normal"
            />
          </label>
          <label class="flex items-center gap-2 text-sm font-medium">
            <input v-model="instagramForm.shareToFeed" type="checkbox">
            Также показать в ленте
          </label>

          <details class="rounded-lg border border-slate-200 p-4">
            <summary class="cursor-pointer text-sm font-semibold">
              Дополнительные настройки
            </summary>
            <div class="mt-4 grid gap-4 sm:grid-cols-2">
              <label class="block text-sm font-medium">
                Соавторы через запятую
                <input
                  v-model="instagramForm.collaborators"
                  placeholder="username, another_user"
                  class="mt-1 w-full rounded-lg border border-slate-300 px-3 py-2 font-normal"
                >
              </label>
              <label class="block text-sm font-medium">
                Название аудио
                <input
                  v-model="instagramForm.audioName"
                  class="mt-1 w-full rounded-lg border border-slate-300 px-3 py-2 font-normal"
                >
              </label>
              <label class="block text-sm font-medium sm:col-span-2">
                HTTPS URL обложки
                <input
                  v-model="instagramForm.coverUrl"
                  type="url"
                  placeholder="https://…/cover.jpg"
                  class="mt-1 w-full rounded-lg border border-slate-300 px-3 py-2 font-normal"
                >
              </label>
              <label class="block text-sm font-medium">
                Кадр обложки, мс
                <input
                  v-model="instagramForm.thumbOffset"
                  type="number"
                  min="0"
                  placeholder="0"
                  class="mt-1 w-full rounded-lg border border-slate-300 px-3 py-2 font-normal"
                >
              </label>
              <label class="block text-sm font-medium">
                Instagram location ID
                <input
                  v-model="instagramForm.locationId"
                  class="mt-1 w-full rounded-lg border border-slate-300 px-3 py-2 font-normal"
                >
              </label>
            </div>
            <p class="mt-3 text-xs text-slate-500">
              Если задан URL обложки, Instagram игнорирует смещение кадра.
            </p>
          </details>

          <p v-if="instagramMessage" class="text-sm text-emerald-700">
            {{ instagramMessage }}
          </p>
          <footer class="flex justify-end gap-2 border-t border-slate-200 pt-4">
            <AppButton
              variant="secondary"
              :disabled="instagramBusy"
              @click="closeInstagramDialog"
            >
              Закрыть
            </AppButton>
            <AppButton type="submit" :disabled="instagramBusy">
              {{ instagramBusy ? 'Публикуем…' : 'Опубликовать' }}
            </AppButton>
          </footer>
        </form>
      </section>
    </div>

    <div
      v-if="fragmentClip"
      class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/55 p-4"
      role="dialog"
      aria-modal="true"
      aria-labelledby="fragment-editor-title"
    >
      <section
        class="max-h-[94vh] w-full max-w-6xl overflow-auto rounded-2xl bg-slate-50 shadow-2xl"
      >
        <header
          class="sticky top-0 z-10 flex items-start justify-between gap-4 border-b border-slate-200 bg-white px-5 py-4"
        >
          <div>
            <h2 id="fragment-editor-title" class="text-lg font-semibold">
              Монтаж скачанного видео
            </h2>
            <p class="mt-1 text-sm text-slate-500">
              Оставьте нужные интервалы. Разделите диапазон и удалите часть,
              чтобы вырезать паузу или неудачный момент.
            </p>
          </div>
          <button
            type="button"
            class="rounded-lg p-2 hover:bg-slate-100"
            aria-label="Закрыть"
            @click="fragmentClip = null"
          >
            <X class="size-5" />
          </button>
        </header>
        <div class="relative z-0 p-5 pb-24">
          <video
            ref="fragmentVideo"
            :src="fragmentSourceURL"
            controls
            class="mx-auto aspect-video w-full max-w-3xl rounded-xl bg-black"
            @play="startFragmentPreview"
            @timeupdate="updateFragmentPreview"
          />
          <p class="mx-auto my-2 max-w-3xl text-sm text-slate-500">
            Предпросмотр результата: {{ fragmentPreviewTime.toFixed(1) }} /
            {{ fragmentOutputDuration().toFixed(1) }} сек.
          </p>
          <VideoTimeline
            class="mt-5"
            source-only
            :segments="fragmentSegments"
            :layers="[]"
            :assets="[]"
            :folders="[]"
            :source-duration="Math.max(0.1, fragmentClip.duration)"
            :selected-layer-id="null"
            :selected-segment-id="fragmentSelectedSegmentID"
            :playhead-time="fragmentPreviewTime"
            @update-segments="updateFragmentSegments"
            @select-segment="fragmentSelectedSegmentID = $event"
            @update-time="setFragmentTimelineTime"
          />
        </div>
        <footer
          class="sticky bottom-0 z-30 flex justify-end gap-2 border-t border-slate-200 bg-white px-5 py-4 shadow-[0_-8px_20px_rgba(15,23,42,0.08)]"
        >
          <AppButton variant="secondary" @click="fragmentClip = null"
            >Отмена</AppButton
          >
          <AppButton
            :disabled="busy === fragmentClip.id"
            @click="saveFragmentEdit"
          >
            Сохранить в готовые фрагменты
          </AppButton>
        </footer>
      </section>
    </div>

    <div
      v-if="processClipID"
      class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/55 p-4"
      role="dialog"
      aria-modal="true"
      aria-labelledby="template-dialog-title"
    >
      <section
        class="max-h-[94vh] w-full overflow-auto rounded-2xl bg-slate-50 shadow-2xl"
        :class="processStage === 'edit' ? 'max-w-[1500px]' : 'max-w-3xl'"
      >
        <header
          class="sticky top-0 z-10 flex items-start justify-between gap-4 border-b border-slate-200 bg-white px-5 py-4"
        >
          <div>
            <h2 id="template-dialog-title" class="text-lg font-semibold">
              {{
                processStage === 'choose' ? 'Выберите шаблон' : 'Подготовьте видео к рендеру'
              }}
            </h2>
            <p class="mt-1 text-sm text-slate-600">
              <template v-if="processStage === 'choose'">
                {{
                  processClipIDs.length > 1
                    ? 'Показаны только шаблоны с включённой галочкой «Паровозик».'
                    : 'Шаблон по умолчанию отмечен первым, но можно выбрать любой.'
                }}
              </template>
              <template v-else>
                {{
                  processClipIDs.length > 1 ? `Паровозик из ${processClipIDs.length} фрагментов` : processClip?.title
                }}
                · {{ selectedTemplate?.name }}. Изменения применятся только к
                этому рендеру.
              </template>
            </p>
          </div>
          <button
            type="button"
            class="rounded-lg p-2 text-slate-600 hover:bg-slate-100"
            aria-label="Закрыть редактор рендера"
            @click="closeProcessDialog"
          >
            <X class="size-5" />
          </button>
        </header>
        <p v-if="templatesLoading" class="p-5 text-slate-500">
          Загружаем шаблоны…
        </p>
        <div
          v-else-if="processStage === 'choose'"
          class="grid gap-3 p-5 sm:grid-cols-2"
        >
          <div
            v-if="!availableTemplates.length"
            class="rounded-xl border border-dashed border-slate-300 bg-white p-5 text-sm text-slate-600 sm:col-span-2"
          >
            Нет шаблонов для «Паровозика». Откройте редактор шаблона, включите
            галочку «Паровозик» и сохраните шаблон.
          </div>
          <button
            v-for="template in availableTemplates"
            :key="template.id"
            type="button"
            class="relative overflow-hidden rounded-xl border bg-white text-left hover:border-violet-500 hover:ring-2 hover:ring-violet-100"
            :class="template.isDefault ? 'border-violet-400' : 'border-slate-200'"
            @click="chooseTemplate(template)"
          >
            <span
              v-if="template.isDefault"
              class="absolute right-2 top-2 z-10 rounded-full bg-violet-600 px-2 py-1 text-xs font-medium text-white"
              >По умолчанию</span
            >
            <img
              v-if="template.previewUrl"
              :src="template.previewUrl"
              :alt="`Превью шаблона ${template.name}`"
              class="aspect-video w-full object-cover"
            >
            <div
              v-else
              class="flex aspect-video items-center justify-center bg-gradient-to-br from-violet-950 to-slate-900 text-sm text-violet-100"
            >
              9:16 · {{ template.config.layers.length }} слоёв
            </div>
            <div class="p-3">
              <b>{{ template.name }}</b>
              <p class="mt-1 text-sm text-slate-600">
                {{ template.description || 'Без описания' }}
              </p>
              <p class="mt-3 text-sm font-medium text-violet-700">
                Выбрать и настроить →
              </p>
            </div>
          </button>
        </div>

        <template v-else>
          <div
            class="flex flex-wrap items-center justify-between gap-3 border-b border-slate-200 bg-white px-5 py-3"
          >
            <button
              type="button"
              class="inline-flex items-center gap-2 text-sm font-medium text-violet-700"
              @click="processStage = 'choose'"
            >
              <ArrowLeft class="size-4" />Другой шаблон
            </button>
            <div class="flex gap-2">
              <AppButton
                variant="secondary"
                :disabled="!renderEditor.canUndo.value"
                @click="renderEditor.undo"
              >
                <Undo2 class="mr-1 inline size-4" />Назад
              </AppButton>
              <AppButton
                variant="secondary"
                :disabled="!renderEditor.canRedo.value"
                @click="renderEditor.redo"
              >
                <Redo2 class="mr-1 inline size-4" />Вперёд
              </AppButton>
            </div>
          </div>

          <div
            class="grid gap-4 p-4 xl:grid-cols-[250px_minmax(360px,1fr)_300px]"
          >
            <LayerPanel
              :layers="renderEditor.draft.value.layers"
              :selected-layer-id="renderEditor.selectedLayerID.value"
              @select="selectEditorLayer"
              @add="addEditorLayer"
              @update="updateEditorLayerTrack"
              @duplicate="renderEditor.duplicateLayer"
              @remove="removeEditorLayer"
              @move="renderEditor.moveLayer"
            />
            <TemplateCanvas
              :config="renderEditor.draft.value"
              :selected-layer-id="renderEditor.selectedLayerID.value"
              :assets="assets"
              :source-url="sourceURL"
              :source-urls="sourceURLs"
              :timeline-time="timelineTime"
              :thumbnail-url="processClip?.thumbnailUrl"
              :streamer-name="processClip?.streamerName"
              @select="selectEditorLayer"
              @update="updateEditorLayerTrack"
            />
            <PropertiesPanel
              :layer="renderEditor.selectedLayer.value"
              :assets="assets"
              :folders="folders"
              @update="renderEditor.selectedLayer.value && updateEditorLayerTrack(renderEditor.selectedLayer.value.id, $event)"
            />
          </div>

          <div class="px-4 pb-4">
            <VideoTimeline
              v-if="renderEditor.draft.value.timeline && processClip"
              :segments="renderEditor.draft.value.timeline.segments"
              :layers="renderEditor.draft.value.layers"
              :assets="assets"
              :folders="folders"
              :source-duration="Math.max(0.1, processClip.duration)"
              :selected-layer-id="renderEditor.selectedLayerID.value"
              :selected-segment-id="selectedTimelineSegmentID"
              @update-segments="renderEditor.updateConfig({ timeline: { segments: $event } })"
              @update-layers="renderEditor.updateConfig({ layers: $event })"
              @update-layer="renderEditor.updateLayer($event.id, $event.patch)"
              @select-layer="selectEditorLayer"
              @select-segment="selectTimelineSegment"
              @update-time="timelineTime = $event"
            />
          </div>

          <footer
            class="sticky bottom-0 z-10 flex flex-wrap items-center justify-between gap-3 border-t border-slate-200 bg-white px-5 py-4"
          >
            <p class="text-sm text-slate-500">
              Исходный клип останется в «Скачанных» после завершения.
            </p>
            <div class="flex gap-2">
              <AppButton variant="secondary" @click="closeProcessDialog"
                >Отмена</AppButton
              >
              <AppButton
                :disabled="busy === processClipID"
                @click="sendToRender"
              >
                {{
                  busy === processClipID ? 'Отправляем…' : 'Отправить на рендер'
                }}
              </AppButton>
            </div>
          </footer>
        </template>
      </section>
    </div>
  </section>
</template>

import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import type {
  Asset,
  AssetFolder,
  AssetLibrary,
} from '@/entities/asset/model/types'
import type {
  InstagramContainer,
  InstagramPublishForm,
  PipelineClip,
  PipelineVideoPreview,
  RenderedVideo,
} from '@/entities/pipeline/model/types'
import {
  createDefaultConfig,
  type Layer,
  normalizeConfig,
  type TemplateConfig,
  type TimelineSegment,
  type VideoTemplate,
} from '@/entities/template/model/types'
import { usePipelineBoard } from '@/features/pipeline-board/model/usePipelineBoard'
import { useTemplateEditor } from '@/features/template-editor/model/useTemplateEditor'
import { readData, readError } from '@/shared/api/http'

export function usePipelineWorkspace() {
  const defaultInstagramCaption = `Лучшие моменты со стримов в коротком формате 🎬
  
  Подписывайся, чтобы не пропустить новые клипы!
  
  #стрим #стример #твич #twitch #twitchclips #клипы #нарезка #моменты #приколы #reels`

  const {
    action,
    busy,
    canDelete,
    clips,
    downloaded,
    error,
    isProcessQueued,
    load,
    readyFragments,
    removeClip: remove,
    removeRenderedVideo,
    statusText,
    renderJobs,
    renderJobStatus,
    renderJobStatusClass,
    usedFragmentIDs,
    updateFragment,
    videos,
  } = usePipelineBoard()
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
  const previewVideo = ref<PipelineVideoPreview | null>(null)
  const instagramVideo = ref<RenderedVideo | null>(null)
  const instagramBusy = ref(false)
  const instagramMessage = ref('')
  const trainSelectionMode = ref(false)
  const selectedTrainClipIDs = ref(new Set<string>())
  const fragmentClip = ref<PipelineClip | null>(null)
  const fragmentSourceURL = ref('')
  const fragmentSegments = ref<TimelineSegment[]>([])
  const fragmentSelectedSegmentID = ref<string | null>(null)
  const fragmentVideo = ref<HTMLVideoElement | null>(null)
  const fragmentPreviewTime = ref(0)
  const fragmentPreviewSegmentIndex = ref(0)
  const instagramForm = ref<InstagramPublishForm>({
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

  const availableTemplates = computed(() =>
    processClipIDs.value.length > 1
      ? templates.value.filter((template) => template.config.train?.enabled)
      : templates.value.filter((template) => !template.config.train?.enabled),
  )
  const processClip = computed(
    () => clips.value.find((clip) => clip.id === processClipID.value) ?? null,
  )
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
      .filter((clip): clip is PipelineClip => Boolean(clip))
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
    const source = renderEditor.draft.value.layers.find(
      (item) => item.id === id,
    )
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
        (layer.trackId ?? layer.id) === trackID
          ? { ...layer, ...patch }
          : layer,
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
    if (!error.value) {
      selectedTrainClipIDs.value = new Set()
      trainSelectionMode.value = false
      closeProcessDialog()
    }
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

  async function openFragmentEditor(clip: PipelineClip) {
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
    if (usedFragmentIDs.value.has(id)) return
    const next = new Set(selectedTrainClipIDs.value)
    if (next.has(id)) next.delete(id)
    else next.add(id)
    selectedTrainClipIDs.value = next
  }

  function startTrain() {
    const ids = [...selectedTrainClipIDs.value].filter(
      (id) => !usedFragmentIDs.value.has(id),
    )
    if (ids.length < 2) {
      error.value = 'Выберите минимум два готовых фрагмента.'
      return
    }
    void openTemplateChooser(ids[0] as string, ids)
  }

  function twitchEmbedURL(clip: PipelineClip) {
    const query = new URLSearchParams({
      clip: clip.twitchClipId,
      parent: window.location.hostname || 'localhost',
      autoplay: 'true',
      muted: 'false',
    })
    return `https://clips.twitch.tv/embed?${query}`
  }

  async function openClipPreview(clip: PipelineClip) {
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

  function openRenderedVideo(video: RenderedVideo) {
    previewVideo.value = {
      key: `render-${video.id}`,
      title: video.title,
      streamer: video.streamer,
      templateName: video.templateName,
      url: video.url,
      mode: 'video',
    }
  }

  function openInstagramDialog(video: RenderedVideo) {
    instagramVideo.value = video
    instagramMessage.value = ''
    instagramForm.value = {
      tunnelUrl: "https://" + window.location.hostname,
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

  function publicVideoURL(video: RenderedVideo) {
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
        await fetch(
          `/api/videos/${video.id}/instagram/${container.id}/publish`,
          {
            method: 'POST',
          },
        ),
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

  let timer: number | undefined
  onMounted(() => {
    void load()
    timer = window.setInterval(load, 3000)
  })
  onBeforeUnmount(() => {
    if (timer) window.clearInterval(timer)
  })

  return {
    action,
    assets,
    availableTemplates,
    busy,
    canDelete,
    chooseTemplate,
    closeInstagramDialog,
    closeProcessDialog,
    downloaded,
    error,
    folders,
    fragmentClip,
    fragmentOutputDuration,
    fragmentPreviewTime,
    fragmentSegments,
    fragmentSelectedSegmentID,
    fragmentSourceURL,
    fragmentVideo,
    instagramBusy,
    instagramForm,
    instagramMessage,
    instagramVideo,
    isProcessQueued,
    openClipPreview,
    openFragmentEditor,
    openInstagramDialog,
    openRenderedVideo,
    openTemplateChooser,
    previewVideo,
    processClip,
    processClipID,
    processClipIDs,
    processStage,
    publicVideoURL,
    publishToInstagram,
    readyFragments,
    remove,
    removeRenderedVideo,
    renderEditor,
    saveFragmentEdit,
    selectedTemplate,
    selectedTimelineSegmentID,
    selectedTrainClipIDs,
    sendToRender,
    sourceURL,
    sourceURLs,
    startFragmentPreview,
    startTrain,
    statusText,
    templatesLoading,
    timelineTime,
    toggleTrainClip,
    renderJobs,
    renderJobStatus,
    renderJobStatusClass,
    trainSelectionMode,
    usedFragmentIDs,
    updateEditorLayerTrack,
    updateFragment,
    updateFragmentPreview,
    updateFragmentSegments,
    videos,
    setFragmentTimelineTime,
    selectEditorLayer,
    selectTimelineSegment,
    addEditorLayer,
    removeEditorLayer,
  }
}

export type PipelineWorkspaceModel = ReturnType<typeof usePipelineWorkspace>

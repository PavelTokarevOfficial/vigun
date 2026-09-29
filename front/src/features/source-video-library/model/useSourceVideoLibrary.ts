import { computed, ref } from 'vue'
import type {
  SourceVideo,
  SourceVideoFolder,
  SourceVideoLibrary,
} from '@/entities/source-video/model/types'
import type { TimelineSegment } from '@/entities/template/model/types'
import { readData, readError } from '@/shared/api/http'

export function useSourceVideoLibrary() {
  const videos = ref<SourceVideo[]>([])
  const folders = ref<SourceVideoFolder[]>([])
  const folderId = ref<string | null>(null)
  const busy = ref(false)
  const error = ref('')
  const editorVideo = ref<SourceVideo | null>(null)
  const editorElement = ref<HTMLVideoElement | null>(null)
  const title = ref('')
  const segments = ref<TimelineSegment[]>([])
  const selectedSegmentID = ref<string | null>(null)
  const playhead = ref(0)
  const isPreviewPlaying = ref(false)
  const savedMessage = ref('')
  const segmentHistory = ref<TimelineSegment[][]>([])

  const canUndo = computed(() => segmentHistory.value.length > 0)

  const selectedSegment = computed(
    () =>
      segments.value.find(
        (segment) => segment.id === selectedSegmentID.value,
      ) ?? null,
  )

  const outputDuration = computed(() =>
    segments.value.reduce(
      (total, segment) => total + Math.max(0, segment.end - segment.start),
      0,
    ),
  )

  async function load() {
    try {
      const library = await readData<SourceVideoLibrary>(
        await fetch('/api/source-videos'),
      )
      videos.value = library.videos
      folders.value = library.folders
      folderId.value = library.folderId
    } catch (cause) {
      error.value =
        cause instanceof Error ? cause.message : 'Не удалось загрузить видео'
    }
  }

  async function pinFolder(value: string) {
    if (busy.value) return
    busy.value = true
    error.value = ''
    try {
      await request(
        await fetch('/api/source-videos/folder', {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ folderId: value || null }),
        }),
        'Не удалось закрепить папку',
      )
      await load()
    } catch (cause) {
      error.value =
        cause instanceof Error ? cause.message : 'Не удалось закрепить папку'
    } finally {
      busy.value = false
    }
  }

  async function request(response: Response, fallback: string) {
    if (!response.ok) throw new Error(await readError(response, fallback))
  }

  async function upload(files: FileList | File[]) {
    if (!files.length || busy.value) return
    busy.value = true
    error.value = ''
    try {
      for (const file of Array.from(files)) {
        const form = new FormData()
        form.append('file', file)
        await request(
          await fetch('/api/source-videos', { method: 'POST', body: form }),
          `Не удалось загрузить ${file.name}`,
        )
      }
      await load()
    } catch (cause) {
      error.value =
        cause instanceof Error ? cause.message : 'Не удалось загрузить видео'
    } finally {
      busy.value = false
    }
  }

  async function rememberDuration(video: SourceVideo, duration: number) {
    if (
      !Number.isFinite(duration) ||
      duration <= 0 ||
      Math.abs(video.duration - duration) < 0.25
    )
      return
    video.duration = duration
    try {
      await request(
        await fetch(`/api/source-videos/${video.id}`, {
          method: 'PATCH',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ duration }),
        }),
        'Не удалось сохранить длительность видео',
      )
    } catch (cause) {
      error.value =
        cause instanceof Error
          ? cause.message
          : 'Не удалось сохранить длительность видео'
    }
  }

  function openEditor(video: SourceVideo) {
    if (video.duration <= 0) {
      error.value = 'Дождитесь загрузки метаданных видео и попробуйте снова.'
      return
    }
    editorVideo.value = video
    title.value = `${video.name.replace(/\.[^.]+$/, '')} — фрагмент ${video.cuts + 1}`
    savedMessage.value = ''
    segmentHistory.value = []
    const segmentID = `cut-${crypto.randomUUID().slice(0, 8)}`
    segments.value = [
      {
        id: segmentID,
        source: 'clip',
        start: 0,
        end: Math.min(30, video.duration),
        sourceDuration: video.duration,
      },
    ]
    selectedSegmentID.value = segmentID
    playhead.value = 0
    isPreviewPlaying.value = false
  }

  function closeEditor() {
    editorElement.value?.pause()
    isPreviewPlaying.value = false
    editorElement.value = null
    editorVideo.value = null
    savedMessage.value = ''
    segmentHistory.value = []
  }

  function setTimelineTime(time: number) {
    const target = Math.min(
      Math.max(0, time),
      Math.max(0, editorVideo.value?.duration ?? 0),
    )
    playhead.value = target
    if (editorElement.value) editorElement.value.currentTime = target
  }

  function updateSelection(segment: TimelineSegment) {
    const current = selectedSegment.value
    if (current && JSON.stringify(current) === JSON.stringify(segment)) return
    segmentHistory.value.push(segments.value.map((item) => ({ ...item })))
    if (segmentHistory.value.length > 50) segmentHistory.value.shift()
    segments.value = [{ ...segment, source: 'clip' }]
    selectedSegmentID.value = segment.id
    savedMessage.value = ''
    if (playhead.value < segment.start || playhead.value > segment.end) {
      setTimelineTime(segment.start)
    }
  }

  function updateSegments(value: TimelineSegment[]) {
    if (JSON.stringify(value) === JSON.stringify(segments.value)) return
    const segmentCountChanged = value.length !== segments.value.length
    segmentHistory.value.push(segments.value.map((segment) => ({ ...segment })))
    if (segmentHistory.value.length > 50) segmentHistory.value.shift()
    segments.value = value.map((segment) => ({
      ...segment,
      source: 'clip' as const,
    }))
    if (segmentCountChanged) selectedSegmentID.value = null
    setTimelineTime(Math.min(playhead.value, outputDuration.value))
  }

  function undoLastAction() {
    const previous = segmentHistory.value.pop()
    if (!previous) return
    segments.value = previous.map((segment) => ({ ...segment }))
    selectedSegmentID.value = segments.value[0]?.id ?? null
    savedMessage.value = ''
    if (segments.value[0]) setTimelineTime(segments.value[0].start)
  }

  function startPreview() {
    const element = editorElement.value
    const segment = selectedSegment.value
    if (!element || !segment) return
    if (element.currentTime < segment.start || element.currentTime >= segment.end - 0.04) {
      setTimelineTime(segment.start)
    }
    isPreviewPlaying.value = true
  }

  function updatePreview() {
    const element = editorElement.value
    const segment = selectedSegment.value
    if (!element || !segment) return
    if (element.currentTime >= segment.end - 0.04) {
      playhead.value = segment.end
      element.pause()
      isPreviewPlaying.value = false
      return
    }
    if (element.currentTime < segment.start) {
      element.currentTime = segment.start
      return
    }
    playhead.value = element.currentTime
  }

  async function togglePreview() {
    const element = editorElement.value
    const segment = selectedSegment.value
    if (!element || !segment) return
    if (!element.paused) {
      element.pause()
      isPreviewPlaying.value = false
      return
    }
    if (element.currentTime < segment.start || element.currentTime >= segment.end - 0.04) {
      setTimelineTime(segment.start)
    }
    await element.play().catch(() => undefined)
  }

  function stopPreview() {
    isPreviewPlaying.value = false
  }

  async function saveCut() {
    const video = editorVideo.value
    const segment = selectedSegment.value
    if (!video || !segment || !title.value.trim() || busy.value) return
    busy.value = true
    error.value = ''
    try {
      await request(
        await fetch(`/api/source-videos/${video.id}/cuts`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            title: title.value,
            timeline: { segments: [segment] },
          }),
        }),
        'Не удалось сохранить фрагмент',
      )
      video.cuts += 1
      savedMessage.value = `Фрагмент ${segment.start.toFixed(1)}–${segment.end.toFixed(1)} сек. отправлен на создание. Прогресс виден в Pipeline.`
      title.value = `${video.name.replace(/\.[^.]+$/, '')} — фрагмент ${video.cuts + 1}`
      await load()
    } catch (cause) {
      error.value =
        cause instanceof Error ? cause.message : 'Не удалось сохранить фрагмент'
    } finally {
      busy.value = false
    }
  }

  async function remove(video: SourceVideo) {
    if (busy.value || !window.confirm(`Удалить «${video.name}»?`)) return
    busy.value = true
    error.value = ''
    try {
      await request(
        await fetch(`/api/source-videos/${video.id}`, { method: 'DELETE' }),
        'Не удалось удалить видео',
      )
      await load()
    } catch (cause) {
      error.value =
        cause instanceof Error ? cause.message : 'Не удалось удалить видео'
    } finally {
      busy.value = false
    }
  }

  return {
    busy,
    canUndo,
    closeEditor,
    editorElement,
    editorVideo,
    error,
    folderId,
    folders,
    isPreviewPlaying,
    load,
    openEditor,
    outputDuration,
    pinFolder,
    playhead,
    rememberDuration,
    remove,
    saveCut,
    savedMessage,
    segments,
    selectedSegment,
    selectedSegmentID,
    setTimelineTime,
    startPreview,
    stopPreview,
    title,
    togglePreview,
    updatePreview,
    updateSegments,
    updateSelection,
    undoLastAction,
    upload,
    videos,
  }
}

export type SourceVideoLibraryModel = ReturnType<typeof useSourceVideoLibrary>

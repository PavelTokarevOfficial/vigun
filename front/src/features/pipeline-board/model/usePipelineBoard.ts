import { computed, ref } from 'vue'
import type {
  PipelineClip,
  ProcessingJob,
  RenderedVideo,
} from '@/entities/pipeline/model/types'
import type {
  TemplateConfig,
  TimelineSegment,
} from '@/entities/template/model/types'
import { readError } from '@/shared/api/http'

export function usePipelineBoard() {
  const clips = ref<PipelineClip[]>([])
  const videos = ref<RenderedVideo[]>([])
  const jobs = ref<ProcessingJob[]>([])
  const error = ref('')
  const busy = ref('')
  const queuedProcessIDs = ref(new Set<string>())

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
  const renderJobs = computed(() =>
    jobs.value
      .filter((job) => job.type === 'process' && job.status !== 'completed')
      .slice(0, 6),
  )
  const usedFragmentIDs = computed(() => {
    const ids = new Set<string>()
    for (const job of jobs.value) {
      const hasRenderedVideo = videos.value.some(
        (video) => video.processingJobId === job.id,
      )
      const reservesFragments =
        ['pending', 'running'].includes(job.status) ||
        (job.status === 'completed' && hasRenderedVideo)
      if (job.type !== 'process' || !reservesFragments) continue
      for (const id of job.fragmentClipIds || []) ids.add(id)
    }
    return ids
  })

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
              clip.hasSource &&
              ['downloaded', 'completed'].includes(clip.status),
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

  async function updateFragment(
    clip: PipelineClip,
    ready: boolean,
    segments: TimelineSegment[] | undefined = clip.editTimeline?.segments,
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
    if (!response.ok) {
      error.value = await readError(response, 'Не удалось сохранить фрагмент')
    }
    busy.value = ''
    await load()
  }

  async function removeClip(clip: PipelineClip) {
    busy.value = clip.id
    error.value = ''
    const response = await fetch(`/api/clips/${clip.id}`, { method: 'DELETE' })
    if (!response.ok) {
      error.value = await readError(response, 'Не удалось удалить клип')
    }
    busy.value = ''
    await load()
  }

  async function removeRenderedVideo(video: RenderedVideo) {
    busy.value = video.id
    error.value = ''
    const response = await fetch(`/api/videos/${video.id}`, {
      method: 'DELETE',
    })
    if (!response.ok) {
      error.value = await readError(
        response,
        'Не удалось удалить готовое видео',
      )
    }
    busy.value = ''
    await load()
  }

  function isProcessQueued(clip: PipelineClip) {
    return (
      queuedProcessIDs.value.has(clip.id) ||
      (clip.lastJobType === 'process' &&
        ['pending', 'running'].includes(clip.lastJobStatus))
    )
  }

  function canDelete(clip: PipelineClip) {
    return (
      ['saved', 'downloaded', 'failed', 'completed'].includes(clip.status) &&
      !isProcessQueued(clip)
    )
  }

  function statusText(clip: PipelineClip) {
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

  function renderJobStatus(job: ProcessingJob) {
    if (job.status === 'pending') return 'В очереди'
    if (job.status === 'running') return `Рендерится · ${job.progress}%`
    if (job.status === 'completed') return 'Готово'
    return 'Ошибка'
  }

  function renderJobStatusClass(job: ProcessingJob) {
    if (job.status === 'completed') return 'bg-emerald-100 text-emerald-700'
    if (job.status === 'failed') return 'bg-red-100 text-red-700'
    if (job.status === 'running') return 'bg-violet-100 text-violet-700'
    return 'bg-amber-100 text-amber-700'
  }

  return {
    clips,
    videos,
    jobs,
    error,
    busy,
    downloaded,
    readyFragments,
    renderJobs,
    usedFragmentIDs,
    load,
    action,
    updateFragment,
    removeClip,
    removeRenderedVideo,
    isProcessQueued,
    canDelete,
    statusText,
    renderJobStatus,
    renderJobStatusClass,
  }
}

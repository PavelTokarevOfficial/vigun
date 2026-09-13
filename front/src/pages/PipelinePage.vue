<script setup lang="ts">
import {
  ArrowLeft,
  Redo2,
  Undo2,
  X,
  Download,
  ExternalLink,
  Play,
  Trash,
  RotateCcw,
  Plus,
} from '@lucide/vue'
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import AppButton from '../shared/ui/AppButton.vue'
import ErrorState from '../shared/ui/ErrorState.vue'
import { readData, readError } from '../shared/api/http'
import type {
  Asset,
  AssetFolder,
  AssetLibrary,
} from '../entities/asset/model/types'
import {
  createDefaultConfig,
  normalizeConfig,
  type TemplateConfig,
  type VideoTemplate,
} from '../entities/template/model/types'
import { useTemplateEditor } from '../features/template-editor/model/useTemplateEditor'
import LayerPanel from '../features/template-editor/ui/LayerPanel.vue'
import PropertiesPanel from '../features/template-editor/ui/PropertiesPanel.vue'
import TemplateCanvas from '../features/template-editor/ui/TemplateCanvas.vue'
import VideoTimeline from '../features/template-editor/ui/VideoTimeline.vue'

type Clip = {
  id: string
  title: string
  streamerName: string
  thumbnailUrl: string
  duration: number
  hasSource: boolean
  status: string
  error: string
  currentStep: string
  progress: number
  lastJobType: string
  lastJobStatus: string
}

type Video = {
  id: string
  clipId: string
  title: string
  streamer: string
  twitchUrl: string
  thumbnailUrl: string
  templateName: string
  url: string
  createdAt: string
}

type Column = 'downloaded' | 'ready'

const clips = ref<Clip[]>([])
const videos = ref<Video[]>([])
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
const timelineTime = ref(0)
const selectedTimelineSegmentID = ref<string | null>(null)
const previewVideo = ref<Video | null>(null)
const renderEditor = useTemplateEditor(createDefaultConfig())

const favorites = computed(() =>
  clips.value.filter(
    (clip) =>
      clip.status === 'saved' ||
      clip.status === 'downloading' ||
      (clip.status === 'failed' && clip.lastJobType === 'download'),
  ),
)
const downloaded = computed(() =>
  clips.value.filter(
    (clip) =>
      clip.hasSource &&
      ([
        'downloaded',
        'transcribing',
        'ready_to_render',
        'rendering',
        'completed',
      ].includes(clip.status) ||
        (clip.status === 'failed' && clip.lastJobType === 'process')),
  ),
)
const processClip = computed(
  () => clips.value.find((clip) => clip.id === processClipID.value) ?? null,
)
async function load() {
  try {
    const [clipResponse, videoResponse] = await Promise.all([
      fetch('/api/clips'),
      fetch('/api/videos'),
    ])
    if (!clipResponse.ok || !videoResponse.ok) {
      throw new Error('Не удалось загрузить доску')
    }
    clips.value = (await clipResponse.json()).data || []
    videos.value = (await videoResponse.json()).data || []
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

async function openTemplateChooser(id: string) {
  error.value = ''
  templatesLoading.value = true
  processClipID.value = id
  processStage.value = 'choose'
  selectedTemplate.value = null
  sourceURL.value = ''
  timelineTime.value = 0
  try {
    const [templateResponse, assetResponse, sourceResponse] = await Promise.all(
      [
        fetch('/api/templates'),
        fetch('/api/assets'),
        fetch(`/api/clips/${id}/source`),
      ],
    )
    if (!templateResponse.ok)
      throw new Error(
        await readError(templateResponse, 'Не удалось загрузить шаблоны'),
      )
    if (!assetResponse.ok)
      throw new Error(
        await readError(assetResponse, 'Не удалось загрузить ассеты'),
      )
    if (!sourceResponse.ok)
      throw new Error(
        await readError(sourceResponse, 'Не удалось открыть скачанное видео'),
      )
    templates.value = await readData<VideoTemplate[]>(templateResponse)
    const library = await readData<AssetLibrary>(assetResponse)
    assets.value = library.assets
    folders.value = library.folders
    sourceURL.value = (await readData<{ url: string }>(sourceResponse)).url
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

function chooseTemplate(template: VideoTemplate) {
  selectedTemplate.value = template
  renderEditor.replace(normalizeConfig(template.config))
  renderEditor.updateConfig({
    timeline: {
      segments: [
        {
          id: `segment-${crypto.randomUUID().slice(0, 8)}`,
          source: 'clip',
          start: 0,
          end: Math.max(0.1, processClip.value?.duration ?? 0.1),
          sourceDuration: Math.max(0.1, processClip.value?.duration ?? 0.1),
        },
      ],
    },
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
  const remainingLayers = renderEditor.draft.value.layers.filter(
    (item) => (item.trackId ?? item.id) !== trackID,
  )
  const hasAnotherLinkedLayer = remainingLayers.some(
    (item) => item.timelineSegmentId === layer.timelineSegmentId,
  )
  renderEditor.updateConfig({
    layers: remainingLayers,
    ...(timeline
      ? {
          timeline: {
            segments: hasAnotherLinkedLayer
              ? timeline.segments
              : timeline.segments.filter(
                  (segment) => segment.id !== layer.timelineSegmentId,
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
  timelineTime.value = 0
  selectedTimelineSegmentID.value = null
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
  if (
    !window.confirm(
      `Удалить готовое видео «${video.title}»? Исходник останется в «Скачанных».`,
    )
  )
    return
  busy.value = video.id
  error.value = ''
  const response = await fetch(`/api/videos/${video.id}`, { method: 'DELETE' })
  if (!response.ok) {
    error.value = await readError(response, 'Не удалось удалить готовое видео')
  }
  busy.value = ''
  await load()
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

  if (column === 'downloaded' && clip.status === 'saved') {
    void action(clip.id, 'download')
  }
  if (
    column === 'ready' &&
    ['downloaded', 'completed'].includes(clip.status) &&
    !isProcessQueued(clip)
  ) {
    void openTemplateChooser(clip.id)
  }
}

function statusText(clip: Clip) {
  if (clip.status === 'saved') return 'В избранном'
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

    <div class="mt-6 grid gap-4 xl:grid-cols-4">
      <section>
        <h3 class="font-semibold">Избранное · {{ favorites.length }}</h3>
        <div class="mt-3 space-y-3">
          <p v-if="!favorites.length" class="text-sm text-slate-500">
            Пока пусто.
          </p>
          <article
            v-for="clip in favorites"
            :key="clip.id"
            draggable="true"
            class="relative cursor-grab rounded-xl overflow-hidden active:cursor-grabbing"
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
              class="absolute top-0 py-1 px-3 text-white [-webkit-text-stroke:2px_black] [paint-order:stroke_fill]"
            >
              <b>{{ clip.title }}</b>
              <p> {{ clip.streamerName }} · {{ statusText(clip) }} </p>
              <p v-if="clip.error" class="mt-1 text-sm text-red-600">
                {{ clip.error }}
              </p>
            </div>

            <div class="flex gap-2 absolute bottom-0 right-0 p-3">
              <AppButton
                v-if="clip.status === 'saved'"
                :disabled="busy === clip.id"
                class="grid place-content-center w-8 h-8"
                @click="action(clip.id, 'download')"
              >
                <Download :size="16" />
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
        @drop="drop('downloaded')"
      >
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
                v-if="['downloaded', 'completed'].includes(clip.status) && !isProcessQueued(clip)"
                :disabled="busy === clip.id"
                class="grid place-content-center w-8 h-8"
                @click="openTemplateChooser(clip.id)"
              >
                <Plus :size="16" />
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
        class="col-span-2 xl:border-l xl:border-dashed xl:border-slate-300 xl:pl-4"
        @dragover.prevent
        @drop="drop('ready')"
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
                @click="previewVideo = video"
              >
                <Play :size="16" />
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
            :src="previewVideo.url"
            controls
            autoplay
            class="max-h-[75vh] max-w-full"
          />
        </div>
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
                Шаблон по умолчанию отмечен первым, но можно выбрать любой.
              </template>
              <template v-else>
                {{ processClip?.title }}
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
          <button
            v-for="template in templates"
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

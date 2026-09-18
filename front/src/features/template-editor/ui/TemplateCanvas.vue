<script setup lang="ts">
import { Volume2, VolumeX } from '@lucide/vue'
import {
  computed,
  nextTick,
  onBeforeUnmount,
  ref,
  shallowRef,
  watch,
} from 'vue'
import type { Asset } from '@/entities/asset/model/types'
import type { Layer, TemplateConfig } from '@/entities/template/model/types'

const props = withDefaults(
  defineProps<{
    config: TemplateConfig
    selectedLayerId: string | null
    assets?: Asset[]
    sourceUrl?: string
    timelineTime?: number
    thumbnailUrl?: string
    streamerName?: string
  }>(),
  {
    assets: () => [],
    sourceUrl: '',
    timelineTime: 0,
    thumbnailUrl: '',
    streamerName: '',
  },
)
const emit = defineEmits<{
  select: [id: string]
  update: [id: string, patch: Partial<Layer>]
}>()

const scale = 0.3
const stageRef = ref()
const transformerRef = ref()
const media = shallowRef<Record<string, HTMLImageElement | HTMLVideoElement>>(
  {},
)
const previewMuted = ref(true)
const previewVolume = ref(1)
let animationFrame = 0

const assetByID = computed(
  () => new Map(props.assets.map((asset) => [asset.id, asset])),
)
const timelinePosition = computed(() => {
  let outputStart = 0
  for (const segment of props.config.timeline?.segments ?? []) {
    const duration = Math.max(0, segment.end - segment.start)
    if (props.timelineTime <= outputStart + duration) {
      return {
        segment,
        outputStart,
        outputEnd: outputStart + duration,
        sourceTime:
          segment.start + Math.max(0, props.timelineTime - outputStart),
      }
    }
    outputStart += duration
  }
  return null
})
const hasVideoPreview = computed(() =>
  Object.values(media.value).some(
    (element) => element instanceof HTMLVideoElement,
  ),
)

function isVideo(layer: Layer) {
  return ['video', 'input_video', 'asset_video'].includes(layer.type)
}
function usesClip(layer: Layer) {
  return (
    layer.type === 'input_video' ||
    (layer.type === 'video' && (layer.source ?? 'clip') === 'clip')
  )
}
function sourceFor(layer: Layer) {
  if (usesClip(layer)) {
    const active = timelinePosition.value?.segment
    if (
      active?.source === 'asset' &&
      !props.config.layers.some((item) => item.timelineSegmentId === active.id)
    ) {
      const asset = active.assetId
        ? assetByID.value.get(active.assetId)
        : undefined
      return { url: asset?.url ?? '', video: asset?.kind === 'video' }
    }
    return {
      url: props.sourceUrl || props.thumbnailUrl,
      video: Boolean(props.sourceUrl),
    }
  }
  const asset = layer.assetId ? assetByID.value.get(layer.assetId) : undefined
  return { url: asset?.url ?? '', video: asset?.kind === 'video' }
}

function activeLayer(layer: Layer) {
  if (!layer.visible) return false
  if (
    props.timelineTime < (layer.startTime ?? 0) ||
    (layer.endTime !== undefined && props.timelineTime > layer.endTime)
  )
    return false
  const activeSegment = timelinePosition.value?.segment
  if (layer.timelineSegmentId)
    return activeSegment?.id === layer.timelineSegmentId
  if (
    usesClip(layer) &&
    activeSegment?.source === 'asset' &&
    props.config.layers.some(
      (item) => item.timelineSegmentId === activeSegment.id,
    )
  )
    return false
  return true
}
function loadMedia() {
  const next: Record<string, HTMLImageElement | HTMLVideoElement> = {}
  for (const layer of props.config.layers) {
    if (!isVideo(layer) && layer.type !== 'image' && layer.type !== 'gif')
      continue
    const source = sourceFor(layer)
    const previous = media.value[layer.id]
    if (previous?.getAttribute('data-source') === source.url) {
      next[layer.id] = previous
      if (previous instanceof HTMLVideoElement) seekVideo(layer, previous)
      continue
    }
    if (!source.url) continue
    if (source.video) {
      const element = document.createElement('video')
      element.setAttribute('data-source', source.url)
      element.src = source.url
      element.muted = true
      element.loop = true
      element.playsInline = true
      element.preload = 'auto'
      element.addEventListener('loadeddata', () => {
        media.value = { ...media.value }
        seekVideo(layer, element)
        syncPreviewAudio()
        void element.play().catch(() => undefined)
      })
      next[layer.id] = element
    } else {
      const element = new window.Image()
      element.setAttribute('data-source', source.url)
      element.crossOrigin = 'anonymous'
      element.addEventListener('load', () => {
        media.value = { ...media.value }
      })
      element.src = source.url
      next[layer.id] = element
    }
  }
  for (const [id, element] of Object.entries(media.value)) {
    if (!next[id] && element instanceof HTMLVideoElement) element.pause()
  }
  media.value = next
  syncPreviewAudio()
}

function syncPreviewAudio() {
  const activeVideos = props.config.layers
    .filter((layer) => isVideo(layer) && activeLayer(layer))
    .map((layer) => ({ layer, element: media.value[layer.id] }))
    .filter(
      (item): item is { layer: Layer; element: HTMLVideoElement } =>
        item.element instanceof HTMLVideoElement,
    )
  const audible =
    activeVideos.find(({ layer }) => usesClip(layer))?.element ??
    activeVideos[0]?.element
  for (const element of Object.values(media.value)) {
    if (element instanceof HTMLVideoElement) {
      element.volume = previewVolume.value
      element.muted =
        previewMuted.value || previewVolume.value === 0 || element !== audible
    }
  }
  if (!previewMuted.value && audible) {
    void audible.play().catch(() => undefined)
  }
}

function togglePreviewSound() {
  if (previewMuted.value && previewVolume.value === 0) {
    previewVolume.value = 1
  }
  previewMuted.value = !previewMuted.value
  syncPreviewAudio()
}

function updatePreviewVolume(event: Event) {
  const value = Number((event.target as HTMLInputElement).value)
  previewVolume.value = Math.min(1, Math.max(0, value))
  previewMuted.value = previewVolume.value === 0
  syncPreviewAudio()
}

function seekVideo(layer: Layer, element: HTMLVideoElement) {
  if (!Number.isFinite(element.duration) || element.duration <= 0) return
  const position = timelinePosition.value
  let sourceTime = Math.max(0, props.timelineTime - (layer.startTime ?? 0))
  if (
    position &&
    (usesClip(layer) || layer.timelineSegmentId === position.segment.id)
  ) {
    sourceTime = position.sourceTime
  }
  element.currentTime = Math.min(
    Math.max(0, sourceTime),
    Math.max(0, element.duration - 0.01),
  )
}

function redraw() {
  stageRef.value?.getNode?.()?.batchDraw()
  animationFrame = window.requestAnimationFrame(redraw)
}

function shape(layer: Layer) {
  return {
    id: `layer-${layer.id}`,
    x: layer.x * scale,
    y: layer.y * scale,
    width: Math.max(layer.width * scale, 1),
    height: Math.max(layer.height * scale, 1),
    opacity: layer.opacity,
    draggable: true,
    clipX: 0,
    clipY: 0,
    clipWidth: Math.max(layer.width * scale, 1),
    clipHeight: Math.max(layer.height * scale, 1),
  }
}

function mediaConfig(layer: Layer) {
  const element = media.value[layer.id]
  const targetWidth = Math.max(layer.width * scale, 1)
  const targetHeight = Math.max(layer.height * scale, 1)
  if (!element || layer.fit === 'stretch') {
    return {
      image: element,
      x: 0,
      y: 0,
      width: targetWidth,
      height: targetHeight,
    }
  }
  const sourceWidth =
    element instanceof HTMLVideoElement
      ? element.videoWidth
      : element.naturalWidth
  const sourceHeight =
    element instanceof HTMLVideoElement
      ? element.videoHeight
      : element.naturalHeight
  if (!sourceWidth || !sourceHeight) {
    return {
      image: element,
      x: 0,
      y: 0,
      width: targetWidth,
      height: targetHeight,
    }
  }
  const ratio =
    layer.fit === 'cover'
      ? Math.max(targetWidth / sourceWidth, targetHeight / sourceHeight)
      : Math.min(targetWidth / sourceWidth, targetHeight / sourceHeight)
  const width = sourceWidth * ratio
  const height = sourceHeight * ratio
  return {
    image: element,
    x: (targetWidth - width) / 2,
    y: (targetHeight - height) / 2,
    width,
    height,
  }
}

function label(layer: Layer) {
  if (layer.type === 'subtitles') return 'Пример субтитров'
  if (layer.type === 'text') {
    return layer.textSource === 'streamer_name'
      ? props.streamerName || 'Название Twitch-канала'
      : layer.text || 'Текст'
  }
  if (layer.type === 'blur') return 'Блюр'
  if (layer.type === 'audio') return 'Аудио'
  return layer.name
}

function placeholderColor(layer: Layer) {
  if (layer.type === 'subtitles') return '#18181b'
  if (layer.type === 'text') return '#312e81'
  if (layer.type === 'blur') {
    const darkness = Math.abs(layer.filters?.brightness ?? -0.2)
    return `rgba(15, 23, 42, ${Math.min(0.85, 0.12 + darkness * 0.7)})`
  }
  if (layer.type === 'color') return layer.color || '#111827'
  return '#ede9fe'
}

function updateTransformer() {
  void nextTick(() => {
    const stage = stageRef.value?.getNode?.()
    const transformer = transformerRef.value?.getNode?.()
    if (!stage || !transformer) return
    const node = props.selectedLayerId
      ? stage.findOne(`#layer-${props.selectedLayerId}`)
      : null
    transformer.nodes(node ? [node] : [])
    transformer.getLayer()?.batchDraw()
  })
}

function dragged(
  layer: Layer,
  event: { target: { x: () => number; y: () => number } },
) {
  emit('update', layer.id, {
    x: Math.round(event.target.x() / scale),
    y: Math.round(event.target.y() / scale),
  })
}

function transformed(
  layer: Layer,
  event: {
    target: {
      x: () => number
      y: () => number
      width: () => number
      height: () => number
      scaleX: (value?: number) => number
      scaleY: (value?: number) => number
    }
  },
) {
  const node = event.target
  const width = Math.max(1, Math.round((node.width() * node.scaleX()) / scale))
  const height = Math.max(
    1,
    Math.round((node.height() * node.scaleY()) / scale),
  )
  node.scaleX(1)
  node.scaleY(1)
  emit('update', layer.id, {
    x: Math.round(node.x() / scale),
    y: Math.round(node.y() / scale),
    width,
    height,
  })
}

watch(
  () => [
    props.config.layers,
    props.assets,
    props.sourceUrl,
    props.thumbnailUrl,
    props.timelineTime,
  ],
  loadMedia,
  { deep: true, immediate: true },
)
watch(
  () => props.timelineTime,
  () => {
    for (const layer of props.config.layers) {
      const element = media.value[layer.id]
      if (element instanceof HTMLVideoElement) seekVideo(layer, element)
    }
    syncPreviewAudio()
  },
)
watch(() => [props.selectedLayerId, props.config.layers], updateTransformer, {
  deep: true,
})

animationFrame = window.requestAnimationFrame(redraw)
onBeforeUnmount(() => {
  window.cancelAnimationFrame(animationFrame)
  for (const element of Object.values(media.value)) {
    if (element instanceof HTMLVideoElement) element.pause()
  }
})
</script>

<template>
  <div
    class="relative flex min-h-[620px] items-center justify-center overflow-auto rounded-xl border border-slate-200 bg-white p-5"
  >
    <div
      v-if="hasVideoPreview"
      class="absolute right-3 top-3 z-10 flex items-center gap-2 rounded-full border border-slate-200 bg-white/95 p-1.5 pr-3 text-slate-700 shadow-sm"
    >
      <button
        type="button"
        class="grid size-8 place-items-center rounded-full transition hover:bg-violet-50 hover:text-violet-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-violet-500"
        :title="previewMuted ? 'Включить звук предпросмотра' : 'Выключить звук предпросмотра'"
        :aria-label="previewMuted ? 'Включить звук предпросмотра' : 'Выключить звук предпросмотра'"
        @click="togglePreviewSound"
      >
        <VolumeX v-if="previewMuted || previewVolume === 0" class="size-5" />
        <Volume2 v-else class="size-5" />
      </button>
      <input
        type="range"
        min="0"
        max="1"
        step="0.05"
        :value="previewVolume"
        class="h-1.5 w-24 cursor-pointer accent-violet-600"
        :aria-label="`Громкость предпросмотра ${Math.round(previewVolume * 100)}%`"
        :title="`Громкость ${Math.round(previewVolume * 100)}%`"
        @input="updatePreviewVolume"
      >
    </div>
    <div class="overflow-hidden rounded-lg border border-slate-300 shadow-xl">
      <v-stage
        ref="stageRef"
        :config="{ width: config.canvas.width * scale, height: config.canvas.height * scale }"
        @mousedown="updateTransformer"
      >
        <v-layer>
          <v-rect
            :config="{
              x: 0,
              y: 0,
              width: config.canvas.width * scale,
              height: config.canvas.height * scale,
              fill: config.canvas.background,
            }"
          />
          <template v-for="layer in config.layers" :key="layer.id">
            <v-group
              v-if="activeLayer(layer) && layer.type !== 'audio'"
              :config="shape(layer)"
              @click="emit('select', layer.id)"
              @tap="emit('select', layer.id)"
              @dragend="dragged(layer, $event)"
              @transformend="transformed(layer, $event)"
            >
              <v-rect
                :config="{
                  width: Math.max(layer.width * scale, 1),
                  height: Math.max(layer.height * scale, 1),
                  fill: media[layer.id] ? 'transparent' : placeholderColor(layer),
                  stroke: selectedLayerId === layer.id ? '#7c3aed' : '#cbd5e1',
                  strokeWidth: selectedLayerId === layer.id ? 3 : 1,
                }"
              />
              <v-image v-if="media[layer.id]" :config="mediaConfig(layer)" />
              <v-text
                v-if="!media[layer.id] || layer.type === 'text' || layer.type === 'subtitles' || layer.type === 'blur'"
                :config="{
                  text: label(layer),
                  width: Math.max(layer.width * scale - 12, 1),
                  height: Math.max(layer.height * scale - 12, 1),
                  x: 6,
                  y: 6,
                  fontSize: Math.min(18, Math.max(10, layer.width * scale * 0.07)),
                  fill: layer.type === 'video' || layer.type === 'image' ? '#475569' : '#fff',
                  align: 'center',
                  verticalAlign: layer.type === 'subtitles' ? 'top' : 'middle',
                  wrap: 'word',
                }"
              />
            </v-group>
          </template>
          <v-transformer
            ref="transformerRef"
            :config="{ rotateEnabled: false, keepRatio: false, borderStroke: '#7c3aed', anchorStroke: '#7c3aed', anchorFill: '#fff' }"
          />
        </v-layer>
      </v-stage>
    </div>
  </div>
</template>

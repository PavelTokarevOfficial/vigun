<script setup lang="ts">
import { Maximize2, Pause, Play, Undo2, ZoomIn, ZoomOut } from '@lucide/vue'
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import TimelineEditor, {
  type TimelineAction,
  type TimelineEffect,
  type TimelineOptions,
  type TimelineRow,
} from 'vue-timeline-editor'
import 'vue-timeline-editor/style.css'
import type { TimelineSegment } from '@/entities/template/model/types'
import AppButton from '@/shared/ui/AppButton.vue'

const props = defineProps<{
  duration: number
  segment: TimelineSegment
  playhead: number
  playing: boolean
  canUndo: boolean
}>()

const emit = defineEmits<{
  updateSegment: [segment: TimelineSegment]
  updateTime: [time: number]
  togglePreview: []
  undo: []
}>()

const effects: Record<string, TimelineEffect> = {
  source: { id: 'source', name: 'Сериал' },
  selection: { id: 'selection', name: 'Фрагмент' },
}
const rows = ref<TimelineRow[]>([])
const timelineEditor = ref<{ setTime: (time: number) => void } | null>(null)
const viewport = ref<HTMLElement | null>(null)
const viewportWidth = ref(960)
const manualScale = ref<number | null>(null)
let resizeObserver: ResizeObserver | null = null
let syncingPlayhead = false

const fitScale = computed(() =>
  Math.max(0.05, props.duration / Math.max(1, (viewportWidth.value - 18) / 120)),
)
const scale = computed(() => manualScale.value ?? fitScale.value)
const fragmentDuration = computed(() =>
  Math.max(0.1, props.segment.end - props.segment.start),
)
const options = computed<TimelineOptions>(() => ({
  scale: scale.value,
  scaleWidth: 120,
  scaleSplitCount: 5,
  minScaleCount: 1,
  startLeft: 18,
  rowHeight: 34,
  duration: Math.max(1, props.duration),
  gridSnap: true,
  dragLine: true,
  enableRowDrag: false,
  backgroundColor: '#f8fafc',
  contentBackgroundColor: '#ffffff',
  borderColor: '#e2e8f0',
  gridColor: '#e2e8f0',
  cursorColor: '#7c3aed',
  snapLineColor: '#f59e0b',
}))

function rebuildRows() {
  rows.value = [
    {
      id: 'source-row',
      data: { label: 'Сериал' },
      actions: [
        {
          id: 'source-video',
          start: 0,
          end: Math.max(0.1, props.duration),
          effectId: 'source',
          flexible: false,
          movable: false,
          data: { kind: 'source', label: 'Сериал', color: '#475569' },
        },
      ],
    },
    {
      id: 'selection-row',
      data: { label: 'Фрагмент' },
      actions: [
        {
          id: 'fragment-range',
          start: props.segment.start,
          end: props.segment.end,
          effectId: 'selection',
          flexible: true,
          movable: true,
          selected: true,
          data: {
            kind: 'selection',
            label: 'Фрагмент',
            detail: `${formatTime(props.segment.start)}–${formatTime(props.segment.end)}`,
            color: '#7c3aed',
          },
        },
      ],
    },
  ]
}

watch([() => props.segment, () => props.duration], rebuildRows, {
  deep: true,
  immediate: true,
})
watch(
  () => props.playhead,
  (time) => {
    const next = clamp(time, 0, props.duration)
    syncingPlayhead = true
    timelineEditor.value?.setTime(next)
    syncingPlayhead = false
  },
)

onMounted(() => {
  if (!viewport.value) return
  const updateWidth = () => {
    viewportWidth.value = Math.max(120, viewport.value?.clientWidth ?? 960)
  }
  updateWidth()
  resizeObserver = new ResizeObserver(updateWidth)
  resizeObserver.observe(viewport.value)
})
onBeforeUnmount(() => resizeObserver?.disconnect())

function setTime(time: number) {
  if (!syncingPlayhead) emit('updateTime', clamp(time, 0, props.duration))
}

function moveRange(params: { action: TimelineAction; start: number; end: number }) {
  if (params.action.id !== 'fragment-range') return
  const duration = fragmentDuration.value
  const start = clamp(params.start, 0, Math.max(0, props.duration - duration))
  emit('updateSegment', {
    ...props.segment,
    start: roundTime(start),
    end: roundTime(start + duration),
  })
}

function resizeRange(params: { action: TimelineAction; start: number; end: number }) {
  if (params.action.id !== 'fragment-range') return
  const start = clamp(params.start, 0, Math.max(0, props.duration - 0.1))
  const end = clamp(params.end, start + 0.1, props.duration)
  emit('updateSegment', {
    ...props.segment,
    start: roundTime(start),
    end: roundTime(end),
  })
}

function setDuration(value: number) {
  if (!Number.isFinite(value)) return
  const duration = clamp(value, 0.1, props.duration)
  const start = Math.min(props.segment.start, Math.max(0, props.duration - duration))
  emit('updateSegment', {
    ...props.segment,
    start: roundTime(start),
    end: roundTime(start + duration),
  })
}

function zoomIn() {
  manualScale.value = Math.max(0.05, scale.value / 1.5)
}
function zoomOut() {
  manualScale.value = Math.min(Math.max(1, props.duration), scale.value * 1.5)
}
function fitTimeline() {
  manualScale.value = null
}
function clamp(value: number, min: number, max: number) {
  return Math.min(max, Math.max(min, value))
}
function roundTime(value: number) {
  return Math.round(value * 1000) / 1000
}
function formatTime(seconds: number) {
  const safe = Math.max(0, seconds)
  const minutes = Math.floor(safe / 60)
  const rest = safe - minutes * 60
  return `${minutes}:${rest.toFixed(1).padStart(4, '0')}`
}
</script>

<template>
  <section class="rounded-xl border border-slate-200 bg-white p-1">
    <div class="grid grid-cols-[1fr_auto_1fr] items-center gap-2">
      <div>
        <AppButton
          variant="secondary"
          class="grid size-8 place-content-center !p-0"
          :disabled="!canUndo"
          title="Отменить последнее действие"
          @click="emit('undo')"
        >
          <Undo2 class="size-4" />
        </AppButton>
      </div>

      <div class="flex items-center justify-center gap-2">
        <AppButton
          variant="secondary"
          class="grid size-8 place-content-center !p-0"
          :title="playing ? 'Пауза' : 'Воспроизвести фрагмент'"
          @click="emit('togglePreview')"
        >
          <Pause v-if="playing" class="size-4" />
          <Play v-else class="size-4" />
        </AppButton>
        <label class="flex items-center gap-1.5 text-xs font-medium text-slate-600">
          Размер фрагмента
          <input
            :value="fragmentDuration.toFixed(1)"
            type="number"
            min="0.1"
            :max="duration"
            step="0.1"
            class="w-20 rounded-md border border-slate-300 bg-white px-2 py-1 text-sm text-slate-900 outline-none focus:border-violet-500"
            @change="setDuration(Number(($event.target as HTMLInputElement).value))"
          >
          <span>сек.</span>
        </label>
      </div>

      <div class="flex justify-end gap-1">
        <button type="button" class="rounded p-1.5 hover:bg-slate-100" title="Приблизить" @click="zoomIn">
          <ZoomIn class="size-4" />
        </button>
        <button type="button" class="rounded p-1.5 hover:bg-slate-100" title="Отдалить" @click="zoomOut">
          <ZoomOut class="size-4" />
        </button>
        <button type="button" class="rounded p-1.5 hover:bg-slate-100" title="Вместить целиком" @click="fitTimeline">
          <Maximize2 class="size-4" />
        </button>
      </div>
    </div>

    <div ref="viewport" class="mt-1 overflow-hidden rounded-lg border border-slate-200">
      <TimelineEditor
        ref="timelineEditor"
        v-model="rows"
        :effects="effects"
        :options="options"
        :auto-scroll="true"
        @time-update="setTime"
        @click-time-area="setTime"
        @action-move-end="moveRange"
        @action-resize-end="resizeRange"
      >
        <template #action="{ action }">
          <div
            class="flex h-full w-full min-w-0 items-center justify-between gap-2 overflow-hidden px-2 text-xs font-medium text-white"
            :class="action.data?.kind === 'selection' ? 'cursor-grab border-2 border-dashed border-amber-200 active:cursor-grabbing' : ''"
            :style="{ backgroundColor: action.data?.color }"
          >
            <span class="truncate">{{ action.data?.label }}</span>
            <span v-if="action.data?.detail" class="shrink-0 opacity-80">{{ action.data.detail }}</span>
          </div>
        </template>
      </TimelineEditor>
    </div>
  </section>
</template>

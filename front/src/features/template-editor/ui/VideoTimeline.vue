<script setup lang="ts">
import {
  ChevronLeft,
  ChevronRight,
  Eye,
  EyeOff,
  Maximize2,
  Scissors,
  Trash2,
  Video,
  ZoomIn,
  ZoomOut,
} from '@lucide/vue'
import { computed, nextTick, ref, watch } from 'vue'
import TimelineEditor, {
  type TimelineAction,
  type TimelineEffect,
  type TimelineOptions,
  type TimelineRow,
} from 'vue-timeline-editor'
import 'vue-timeline-editor/style.css'
import type { Asset, AssetFolder } from '@/entities/asset/model/types'
import type { Layer, TimelineSegment } from '@/entities/template/model/types'
import AppButton from '@/shared/ui/AppButton.vue'
import AssetPickerDialog from './AssetPickerDialog.vue'

const props = defineProps<{
  segments: TimelineSegment[]
  layers: Layer[]
  assets: Asset[]
  folders: AssetFolder[]
  sourceDuration: number
  selectedLayerId: string | null
  selectedSegmentId: string | null
}>()
const emit = defineEmits<{
  updateSegments: [segments: TimelineSegment[]]
  updateLayers: [layers: Layer[]]
  updateLayer: [value: { id: string; patch: Partial<Layer> }]
  selectLayer: [id: string | null]
  selectSegment: [id: string]
  updateTime: [time: number]
}>()

const currentTime = ref(0)
const selectedID = ref<string | null>(
  props.segments[0] ? segmentActionID(props.segments[0].id) : null,
)
const selectedLayerIDs = ref<Set<string>>(new Set())
const rows = ref<TimelineRow[]>([])
const pickerOpen = ref(false)
const readingAsset = ref(false)
const assetDurationWarning = ref('')
const draggedSegmentID = ref<string | null>(null)
const manualScale = ref<number | null>(null)
let groupMoveSnapshot = new Map<string, { start: number; end: number }>()
const suppressActionClick = ref(false)

const effects: Record<string, TimelineEffect> = {
  sequence: { id: 'sequence', name: 'Монтаж видео' },
  layer: { id: 'layer', name: 'Слой' },
}

const outputDuration = computed(() =>
  props.segments.reduce(
    (sum, segment) => sum + Math.max(0, segment.end - segment.start),
    0,
  ),
)
const timelineDuration = computed(() =>
  Math.max(
    1,
    outputDuration.value,
    ...props.layers.map((layer) => layer.endTime ?? 0),
  ),
)
const fitScale = computed(() => Math.max(0.1, timelineDuration.value / 8))
const timelineScale = computed(() => manualScale.value ?? fitScale.value)
const options = computed<TimelineOptions>(() => ({
  scale: timelineScale.value,
  scaleWidth: 120,
  scaleSplitCount: 5,
  minScaleCount: 1,
  startLeft: 18,
  rowHeight: 46,
  duration: timelineDuration.value,
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

const selectedSegmentIndex = computed(() => {
  if (!selectedID.value?.startsWith('segment:')) return -1
  const id = selectedID.value.slice('segment:'.length)
  return props.segments.findIndex((segment) => segment.id === id)
})
const selectedLayer = computed(() => {
  if (!selectedID.value?.startsWith('layer:')) return null
  const id = selectedID.value.slice('layer:'.length)
  return props.layers.find((layer) => layer.id === id) ?? null
})
const canSplitSelected = computed(() => {
  if (selectedSegmentIndex.value >= 0) {
    const layout = segmentLayout(selectedSegmentIndex.value)
    return Boolean(
      layout &&
        currentTime.value > layout.start + 0.1 &&
        currentTime.value < layout.end - 0.1,
    )
  }
  if (!selectedLayer.value) return false
  const start = selectedLayer.value.startTime ?? 0
  const end = selectedLayer.value.endTime ?? outputDuration.value
  return currentTime.value > start + 0.1 && currentTime.value < end - 0.1
})
const canRemoveSelected = computed(
  () =>
    Boolean(selectedLayer.value) ||
    (selectedSegmentIndex.value >= 0 && props.segments.length > 1),
)
const canSplitAllLayers = computed(
  () =>
    Boolean(segmentAtOutputTime(props.segments, currentTime.value)) ||
    props.layers.some((layer) => layerContainsTime(layer, currentTime.value)),
)

watch([() => props.segments, () => props.layers, outputDuration], rebuildRows, {
  deep: true,
  immediate: true,
})
watch(
  [() => props.selectedLayerId, () => props.selectedSegmentId],
  ([layerID, segmentID]) => {
    selectedID.value = segmentID
      ? segmentActionID(segmentID)
      : layerID
        ? layerActionID(layerID)
        : null
    if (segmentID) {
      selectedLayerIDs.value = new Set()
    } else if (layerID) {
      if (!selectedLayerIDs.value.has(layerID)) {
        selectedLayerIDs.value = new Set([layerID])
      }
    } else {
      selectedLayerIDs.value = new Set()
    }
    rebuildRows()
  },
  { immediate: true },
)

function rebuildRows() {
  let cursor = 0
  const sequenceActions = props.segments.map((segment, index) => {
    const duration = segmentDuration(segment)
    const start = cursor
    const end = start + duration
    cursor = end
    const source = segment.source ?? 'clip'
    return {
      id: segmentActionID(segment.id),
      start,
      end,
      effectId: 'sequence',
      flexible: true,
      movable: true,
      selected: selectedID.value === segmentActionID(segment.id),
      data: {
        kind: 'segment',
        segmentId: segment.id,
        label:
          source === 'asset'
            ? `Монтаж · ${assetName(segment.assetId) || `вставка ${index + 1}`}`
            : `Монтаж · исходник ${index + 1}`,
        detail: `${formatTime(segment.start)}–${formatTime(segment.end)}`,
        color: source === 'asset' ? '#d97706' : '#7c3aed',
      },
    } satisfies TimelineAction
  })

  const layerRows: TimelineRow[] = []
  const rowsByTrack = new Map<string, TimelineRow>()
  for (const layer of [...props.layers].reverse()) {
    const start = Math.max(0, layer.startTime ?? 0)
    const end = Math.max(
      start + 0.1,
      Math.min(layer.endTime ?? outputDuration.value, timelineDuration.value),
    )
    const trackID = layer.trackId ?? layer.id
    let row = rowsByTrack.get(trackID)
    if (!row) {
      row = {
        id: `layer-row:${trackID}`,
        data: { label: layer.name },
        actions: [],
      }
      rowsByTrack.set(trackID, row)
      layerRows.push(row)
    }
    row.actions.push({
      id: layerActionID(layer.id),
      start,
      end,
      effectId: 'layer',
      flexible: true,
      movable: true,
      selected: selectedLayerIDs.value.has(layer.id),
      data: {
        kind: 'layer',
        layerId: layer.id,
        trackId: trackID,
        label: layer.name,
        detail: layerTypeName(layer.type),
        color: layer.visible ? layerColor(layer.type) : '#94a3b8',
      },
    })
  }
  for (const row of layerRows) {
    row.actions.sort((left, right) => left.start - right.start)
  }

  rows.value = [
    {
      id: 'sequence-row',
      data: { label: 'Монтаж: источники видео и звук' },
      actions: sequenceActions,
    },
    ...layerRows,
  ]
}

function selectAction(
  event: MouseEvent,
  params: { action: TimelineAction; row: TimelineRow; time: number },
) {
  if (suppressActionClick.value) {
    suppressActionClick.value = false
    return
  }
  selectedID.value = params.action.id
  const layerID = params.action.data?.layerId as string | undefined
  const segmentID = params.action.data?.segmentId as string | undefined
  if (layerID) {
    const layer = props.layers.find((item) => item.id === layerID)
    if (!layer) return
    if (event.shiftKey) {
      const next = new Set(selectedLayerIDs.value)
      if (next.has(layerID) && next.size > 1) next.delete(layerID)
      else next.add(layerID)
      selectedLayerIDs.value = next
    } else {
      selectedLayerIDs.value = new Set([layerID])
    }
    emit('selectLayer', layerID)
  }
  if (segmentID) {
    selectedLayerIDs.value = new Set()
    emit('selectSegment', segmentID)
  }
  rebuildRows()
}

function setCurrentTime(time: number) {
  currentTime.value = time
  emit('updateTime', time)
}

function moveAction(params: {
  action: TimelineAction
  row: TimelineRow
  start: number
  end: number
}) {
  const kind = params.action.data?.kind
  if (kind === 'segment') {
    const actions = rows.value
      .flatMap((row) => row.actions)
      .filter((action) => action.data?.kind === 'segment')
      .sort((left, right) => left.start - right.start)
    const byID = new Map(props.segments.map((segment) => [segment.id, segment]))
    const next = actions
      .map((action) => byID.get(action.data?.segmentId as string))
      .filter((segment): segment is TimelineSegment => Boolean(segment))
    if (next.length === props.segments.length) emitSegments(next)
    return
  }
  const layerID = params.action.data?.layerId as string | undefined
  if (!layerID) return
  const draggedLayer = props.layers.find((layer) => layer.id === layerID)
  if (!draggedLayer) return
  const activeLayers = selectedLayerIDs.value.has(draggedLayer.id)
    ? selectedLayerIDs.value
    : new Set([draggedLayer.id])
  const selectedLayers = props.layers.filter((layer) =>
    activeLayers.has(layer.id),
  )
  if (!selectedLayers.length) return
  const originalStart = draggedLayer.startTime ?? 0
  const requestedDelta = params.start - originalStart
  const firstStart = Math.min(
    ...selectedLayers.map((layer) => layer.startTime ?? 0),
  )
  const lastEnd = Math.max(
    ...selectedLayers.map((layer) => layer.endTime ?? outputDuration.value),
  )
  const delta = clamp(
    requestedDelta,
    -firstStart,
    outputDuration.value - lastEnd,
  )
  emit(
    'updateLayers',
    props.layers.map((layer) => {
      if (!activeLayers.has(layer.id)) return layer
      return {
        ...layer,
        startTime: roundTime((layer.startTime ?? 0) + delta),
        endTime: roundTime((layer.endTime ?? outputDuration.value) + delta),
      }
    }),
  )
  groupMoveSnapshot.clear()
  nextTick(rebuildRows)
}

function resizeAction(params: {
  action: TimelineAction
  row: TimelineRow
  start: number
  end: number
  dir: 'left' | 'right'
}) {
  const segmentID = params.action.data?.segmentId as string | undefined
  if (segmentID) {
    const index = props.segments.findIndex(
      (segment) => segment.id === segmentID,
    )
    const segment = props.segments[index]
    const layout = segmentLayout(index)
    if (!segment || !layout) return
    const maxDuration =
      segment.sourceDuration ??
      ((segment.source ?? 'clip') === 'clip'
        ? props.sourceDuration
        : segment.end)
    let start = segment.start
    let end = segment.end
    if (params.dir === 'left') {
      start = clamp(
        segment.start + params.start - layout.start,
        0,
        segment.end - 0.1,
      )
    } else {
      end = clamp(
        segment.end + params.end - layout.end,
        segment.start + 0.1,
        Math.max(segment.start + 0.1, maxDuration),
      )
    }
    emitSegments(
      props.segments.map((item) =>
        item.id === segmentID ? { ...item, start, end } : item,
      ),
    )
    return
  }

  const layerID = params.action.data?.layerId as string | undefined
  if (layerID) {
    const groupedRanges = selectedResizeRanges(params)
    if (groupedRanges.size > 1) {
      emit(
        'updateLayers',
        props.layers.map((layer) => {
          const range = groupedRanges.get(layer.id)
          return range
            ? {
                ...layer,
                startTime: range.start,
                endTime: range.end,
              }
            : layer
        }),
      )
      window.setTimeout(() => {
        suppressActionClick.value = false
      }, 0)
      nextTick(rebuildRows)
      return
    }
    emitLayerRange(
      layerID,
      clamp(params.start, 0, Math.max(0, outputDuration.value - 0.1)),
      clamp(params.end, 0.1, outputDuration.value),
    )
  }
}

function startResizeAction(params: {
  action: TimelineAction
  row: TimelineRow
  dir: 'left' | 'right'
}) {
  const layerID = params.action.data?.layerId as string | undefined
  if (
    layerID &&
    selectedLayerIDs.value.size > 1 &&
    selectedLayerIDs.value.has(layerID)
  ) {
    suppressActionClick.value = true
  }
}

function previewResizeAction(params: {
  action: TimelineAction
  row: TimelineRow
  start: number
  end: number
  dir: 'left' | 'right'
}) {
  const ranges = selectedResizeRanges(params)
  if (ranges.size < 2) return
  nextTick(() => {
    rows.value = rows.value.map((row) => ({
      ...row,
      actions: row.actions.map((action) => {
        const layerID = action.data?.layerId as string | undefined
        const range = layerID ? ranges.get(layerID) : undefined
        return range
          ? { ...action, start: range.start, end: range.end }
          : action
      }),
    }))
  })
}

function selectedResizeRanges(params: {
  action: TimelineAction
  start: number
  end: number
  dir: 'left' | 'right'
}) {
  const layerID = params.action.data?.layerId as string | undefined
  if (
    !layerID ||
    selectedLayerIDs.value.size < 2 ||
    !selectedLayerIDs.value.has(layerID)
  ) {
    return new Map<string, { start: number; end: number }>()
  }
  const selectedLayers = props.layers.filter((layer) =>
    selectedLayerIDs.value.has(layer.id),
  )
  const draggedLayer = selectedLayers.find((layer) => layer.id === layerID)
  if (!draggedLayer || selectedLayers.length < 2) {
    return new Map<string, { start: number; end: number }>()
  }
  const original = {
    start: draggedLayer.startTime ?? 0,
    end: draggedLayer.endTime ?? outputDuration.value,
  }
  const ranges = new Map(
    selectedLayers.map((layer) => [
      layer.id,
      {
        start: layer.startTime ?? 0,
        end: layer.endTime ?? outputDuration.value,
      },
    ]),
  )
  const minDuration = 0.1
  if (params.dir === 'left') {
    const requestedDelta = params.start - original.start
    const minDelta = -Math.min(
      ...[...ranges.values()].map((item) => item.start),
    )
    const maxDelta = Math.min(
      ...[...ranges.values()].map(
        (item) => item.end - item.start - minDuration,
      ),
    )
    const delta = clamp(requestedDelta, minDelta, maxDelta)
    for (const [id, range] of ranges) {
      ranges.set(id, {
        start: roundTime(range.start + delta),
        end: roundTime(range.end),
      })
    }
  } else {
    const requestedDelta = params.end - original.end
    const minDelta = -Math.min(
      ...[...ranges.values()].map(
        (item) => item.end - item.start - minDuration,
      ),
    )
    const maxDelta =
      outputDuration.value -
      Math.max(...[...ranges.values()].map((item) => item.end))
    const delta = clamp(requestedDelta, minDelta, maxDelta)
    for (const [id, range] of ranges) {
      ranges.set(id, {
        start: roundTime(range.start),
        end: roundTime(range.end + delta),
      })
    }
  }
  return ranges
}

function split() {
  if (!canSplitSelected.value) return
  const segmentIndex = selectedSegmentIndex.value
  if (segmentIndex >= 0) {
    const segment = props.segments[segmentIndex]
    const layout = segmentLayout(segmentIndex)
    if (!segment || !layout) return
    const point = segment.start + currentTime.value - layout.start
    const left = { ...segment, end: point }
    const right = {
      ...segment,
      id: `segment-${crypto.randomUUID().slice(0, 8)}`,
      start: point,
    }
    const nextLayers = props.layers.flatMap((layer) => {
      if (
        layer.timelineSegmentId !== segment.id ||
        !layerContainsTime(layer, currentTime.value)
      )
        return [layer]
      const trackID = layerTrackID(layer)
      return [
        {
          ...layer,
          trackId: trackID,
          endTime: roundTime(currentTime.value),
        },
        {
          ...layer,
          id: `${layer.type}-${crypto.randomUUID().slice(0, 8)}`,
          name: `${layer.name} · часть 2`,
          trackId: trackID,
          timelineSegmentId: right.id,
          startTime: roundTime(currentTime.value),
        },
      ]
    })
    selectedID.value = segmentActionID(right.id)
    emit('selectSegment', right.id)
    emitSegments(
      props.segments.flatMap((item) =>
        item.id === segment.id ? [left, right] : [item],
      ),
      nextLayers,
    )
    return
  }

  const layer = selectedLayer.value
  if (!layer) return
  const end = layer.endTime ?? outputDuration.value
  const trackID = layer.trackId ?? layer.id
  const left = {
    ...layer,
    trackId: trackID,
    endTime: roundTime(currentTime.value),
  }
  const right = {
    ...layer,
    id: `${layer.type}-${crypto.randomUUID().slice(0, 8)}`,
    name: `${layer.name} · часть 2`,
    trackId: trackID,
    startTime: roundTime(currentTime.value),
    endTime: roundTime(end),
  }
  selectedID.value = layerActionID(right.id)
  emit(
    'updateLayers',
    props.layers.flatMap((item) =>
      item.id === layer.id ? [left, right] : [item],
    ),
  )
  emit('selectLayer', right.id)
  nextTick(rebuildRows)
}

function splitAllLayers() {
  if (!canSplitAllLayers.value) return
  const point = roundTime(currentTime.value)
  const activeSegment = segmentAtOutputTime(props.segments, point)
  let rightSegmentID = ''
  const nextSegments = props.segments.flatMap((segment) => {
    if (!activeSegment || segment.id !== activeSegment.segment.id)
      return [segment]
    const sourcePoint = segment.start + point - activeSegment.start
    const right = {
      ...segment,
      id: `segment-${crypto.randomUUID().slice(0, 8)}`,
      start: sourcePoint,
    }
    rightSegmentID = right.id
    return [{ ...segment, end: sourcePoint }, right]
  })
  const selectedRightLayers = new Set<string>()
  const next = props.layers.flatMap((layer) => {
    if (!layerContainsTime(layer, point)) return [layer]
    const trackID = layerTrackID(layer)
    const rightID = `${layer.type}-${crypto.randomUUID().slice(0, 8)}`
    selectedRightLayers.add(rightID)
    return [
      { ...layer, trackId: trackID, endTime: point },
      {
        ...layer,
        id: rightID,
        name: `${layer.name} · часть 2`,
        trackId: trackID,
        timelineSegmentId:
          layer.timelineSegmentId === activeSegment?.segment.id &&
          rightSegmentID
            ? rightSegmentID
            : layer.timelineSegmentId,
        startTime: point,
      },
    ]
  })
  selectedLayerIDs.value = selectedRightLayers
  emitSegments(nextSegments, next)
  const selectedRight = [...next]
    .reverse()
    .find(
      (layer) => layer.startTime === point && selectedRightLayers.has(layer.id),
    )
  if (selectedRight) {
    selectedID.value = layerActionID(selectedRight.id)
    emit('selectLayer', selectedRight.id)
  }
  nextTick(rebuildRows)
}

function removeSelected() {
  const index = selectedSegmentIndex.value
  if (index >= 0) {
    if (props.segments.length <= 1) return
    const removed = props.segments[index]
    const next = props.segments.filter((_, itemIndex) => itemIndex !== index)
    const selected = next[Math.min(index, next.length - 1)]
    selectedID.value = selected ? segmentActionID(selected.id) : null
    if (selected) emit('selectSegment', selected.id)
    const nextLayers = removed
      ? props.layers.filter((layer) => layer.timelineSegmentId !== removed.id)
      : props.layers
    emitSegments(next, nextLayers)
    return
  }
  const layer = selectedLayer.value
  if (!layer) return
  selectedID.value = null
  emit(
    'updateLayers',
    props.layers.filter((item) => item.id !== layer.id),
  )
  emit('selectLayer', null)
  nextTick(rebuildRows)
}

function toggleSelectedLayer() {
  const layer = selectedLayer.value
  if (!layer) return
  emit('updateLayer', { id: layer.id, patch: { visible: !layer.visible } })
}

function moveSelected(direction: -1 | 1) {
  const index = selectedSegmentIndex.value
  const target = index + direction
  if (index < 0 || target < 0 || target >= props.segments.length) return
  const next = [...props.segments]
  ;[next[index], next[target]] = [next[target], next[index]]
  emitSegments(next)
}

function startSegmentDrag(event: DragEvent, action: TimelineAction) {
  if (action.data?.kind !== 'segment') return
  const id = action.data.segmentId as string
  draggedSegmentID.value = id
  selectedID.value = segmentActionID(id)
  emit('selectSegment', id)
  if (event.dataTransfer) {
    event.dataTransfer.effectAllowed = 'move'
    event.dataTransfer.setData('text/plain', id)
  }
}

function dropSegment(event: DragEvent, action: TimelineAction) {
  if (action.data?.kind !== 'segment') return
  event.preventDefault()
  const sourceID =
    draggedSegmentID.value || event.dataTransfer?.getData('text/plain')
  const targetID = action.data.segmentId as string
  if (!sourceID || sourceID === targetID) return
  const sourceIndex = props.segments.findIndex((item) => item.id === sourceID)
  const targetIndex = props.segments.findIndex((item) => item.id === targetID)
  if (sourceIndex < 0 || targetIndex < 0) return
  const next = [...props.segments]
  const [moved] = next.splice(sourceIndex, 1)
  if (!moved) return
  next.splice(targetIndex, 0, moved)
  selectedID.value = segmentActionID(moved.id)
  emit('selectSegment', moved.id)
  draggedSegmentID.value = null
  emitSegments(next)
}

function stopTimelineMove(event: MouseEvent, action: TimelineAction) {
  if (action.data?.kind === 'segment') {
    event.stopPropagation()
    return
  }
  const layerID = action.data?.layerId as string | undefined
  if (
    !layerID ||
    selectedLayerIDs.value.size < 2 ||
    !selectedLayerIDs.value.has(layerID)
  )
    return
  event.stopPropagation()
  event.preventDefault()
  suppressActionClick.value = true
  groupMoveSnapshot = new Map(
    props.layers
      .filter((layer) => selectedLayerIDs.value.has(layer.id))
      .map((layer) => [
        layer.id,
        {
          start: layer.startTime ?? 0,
          end: layer.endTime ?? outputDuration.value,
        },
      ]),
  )
  const startX = event.clientX
  const firstStart = Math.min(
    ...[...groupMoveSnapshot.values()].map((item) => item.start),
  )
  const lastEnd = Math.max(
    ...[...groupMoveSnapshot.values()].map((item) => item.end),
  )
  let delta = 0
  const move = (moveEvent: MouseEvent) => {
    const rawDelta = ((moveEvent.clientX - startX) / 120) * timelineScale.value
    const step = timelineScale.value / 5
    delta = clamp(
      Math.round(rawDelta / step) * step,
      -firstStart,
      outputDuration.value - lastEnd,
    )
    rows.value = rows.value.map((row) => ({
      ...row,
      actions: row.actions.map((item) => {
        const id = item.data?.layerId as string | undefined
        const original = id ? groupMoveSnapshot.get(id) : undefined
        return original
          ? {
              ...item,
              start: roundTime(original.start + delta),
              end: roundTime(original.end + delta),
            }
          : item
      }),
    }))
  }
  const finish = () => {
    window.removeEventListener('mousemove', move)
    window.removeEventListener('mouseup', finish)
    window.setTimeout(() => {
      suppressActionClick.value = false
    }, 0)
    if (Math.abs(delta) >= 0.001) {
      emit(
        'updateLayers',
        props.layers.map((layer) => {
          const original = groupMoveSnapshot.get(layer.id)
          return original
            ? {
                ...layer,
                startTime: roundTime(original.start + delta),
                endTime: roundTime(original.end + delta),
              }
            : layer
        }),
      )
    }
    groupMoveSnapshot.clear()
    nextTick(rebuildRows)
  }
  window.addEventListener('mousemove', move)
  window.addEventListener('mouseup', finish)
}

async function insertAsset(asset: Asset) {
  pickerOpen.value = false
  readingAsset.value = true
  assetDurationWarning.value = ''
  const measured = await readVideoDuration(asset.url)
  readingAsset.value = false
  const duration = measured ?? 5
  if (measured === null) {
    assetDurationWarning.value =
      'Не удалось определить длительность ассета: добавлено 5 секунд, края можно подправить.'
  }
  const inserted: TimelineSegment = {
    id: `segment-${crypto.randomUUID().slice(0, 8)}`,
    source: 'asset',
    assetId: asset.id,
    start: 0,
    end: duration,
    sourceDuration: duration,
  }
  const next = insertAtTime(props.segments, inserted, currentTime.value)
  const outputStart = segmentOutputStart(next, inserted.id)
  const reference = [...props.layers]
    .reverse()
    .find(
      (layer) =>
        layer.visible &&
        (layer.type === 'input_video' ||
          (layer.type === 'video' && (layer.source ?? 'clip') === 'clip')),
    )
  const layerID = `video-${crypto.randomUUID().slice(0, 8)}`
  const importedLayer: Layer = {
    id: layerID,
    name: asset.name,
    type: 'video',
    source: 'asset',
    assetId: asset.id,
    timelineSegmentId: inserted.id,
    trackId: layerID,
    x: reference?.x ?? 0,
    y: reference?.y ?? 0,
    width: reference?.width ?? 1080,
    height: reference?.height ?? 1920,
    visible: true,
    opacity: reference?.opacity ?? 1,
    fit: reference?.fit ?? 'contain',
    filters: reference?.filters ? { ...reference.filters } : undefined,
    startTime: roundTime(outputStart),
    endTime: roundTime(outputStart + duration),
  }
  const referenceIndex = reference
    ? props.layers.findIndex((layer) => layer.id === reference.id)
    : props.layers.length - 1
  const nextLayers = [...props.layers]
  nextLayers.splice(referenceIndex + 1, 0, importedLayer)
  selectedID.value = layerActionID(layerID)
  selectedLayerIDs.value = new Set([layerID])
  emitSegments(next, nextLayers)
  emit('selectLayer', layerID)
}

function zoomIn() {
  manualScale.value = Math.max(0.05, timelineScale.value / 1.5)
}

function zoomOut() {
  manualScale.value = Math.min(600, timelineScale.value * 1.5)
}

function fitTimeline() {
  manualScale.value = null
}

function insertAtTime(
  source: TimelineSegment[],
  inserted: TimelineSegment,
  time: number,
) {
  const next: TimelineSegment[] = []
  let cursor = 0
  let added = false
  for (const segment of source) {
    const duration = segmentDuration(segment)
    const offset = time - cursor
    if (!added && offset > 0.1 && offset < duration - 0.1) {
      const point = segment.start + offset
      next.push({ ...segment, end: point }, inserted, {
        ...segment,
        id: `segment-${crypto.randomUUID().slice(0, 8)}`,
        start: point,
      })
      added = true
    } else {
      if (!added && time <= cursor + 0.1) {
        next.push(inserted)
        added = true
      }
      next.push(segment)
      if (!added && time <= cursor + duration + 0.1) {
        next.push(inserted)
        added = true
      }
    }
    cursor += duration
  }
  if (!added) next.push(inserted)
  return next
}

function emitSegments(segments: TimelineSegment[], baseLayers = props.layers) {
  const before = segmentLayouts(props.segments)
  const after = segmentLayouts(segments)
  const syncedLayers = baseLayers.map((layer) => {
    if (!layer.timelineSegmentId) return layer
    const previous = before.get(layer.timelineSegmentId)
    const current = after.get(layer.timelineSegmentId)
    if (!previous || !current) return layer
    const startOffset = (layer.startTime ?? previous.start) - previous.start
    const endOffset = previous.end - (layer.endTime ?? previous.end)
    const start = clamp(
      current.start + startOffset,
      current.start,
      Math.max(current.start, current.end - 0.1),
    )
    const end = clamp(current.end - endOffset, start + 0.1, current.end)
    return {
      ...layer,
      startTime: roundTime(start),
      endTime: roundTime(end),
    }
  })
  emit('updateSegments', segments)
  if (JSON.stringify(syncedLayers) !== JSON.stringify(props.layers)) {
    emit('updateLayers', syncedLayers)
  }
  nextTick(rebuildRows)
}

function segmentLayouts(segments: TimelineSegment[]) {
  const layouts = new Map<string, { start: number; end: number }>()
  let cursor = 0
  for (const segment of segments) {
    const duration = segmentDuration(segment)
    layouts.set(segment.id, { start: cursor, end: cursor + duration })
    cursor += duration
  }
  return layouts
}

function segmentOutputStart(segments: TimelineSegment[], id: string) {
  return segmentLayouts(segments).get(id)?.start ?? 0
}

function segmentAtOutputTime(segments: TimelineSegment[], time: number) {
  let cursor = 0
  for (const segment of segments) {
    const duration = segmentDuration(segment)
    const end = cursor + duration
    if (time > cursor + 0.1 && time < end - 0.1) {
      return { segment, start: cursor, end }
    }
    cursor = end
  }
  return null
}

function emitLayerRange(id: string, start: number, end: number) {
  if (end <= start) return
  emit('updateLayer', {
    id,
    patch: { startTime: roundTime(start), endTime: roundTime(end) },
  })
  nextTick(rebuildRows)
}

function segmentLayout(index: number) {
  const segment = props.segments[index]
  if (!segment) return null
  const start = props.segments
    .slice(0, index)
    .reduce((sum, item) => sum + segmentDuration(item), 0)
  return { start, end: start + segmentDuration(segment) }
}

function segmentDuration(segment: TimelineSegment) {
  return Math.max(0.1, segment.end - segment.start)
}

function segmentActionID(id: string) {
  return `segment:${id}`
}

function layerActionID(id: string) {
  return `layer:${id}`
}

function layerTrackID(layer: Layer) {
  return layer.trackId ?? layer.id
}

function layerContainsTime(layer: Layer, time: number) {
  const start = layer.startTime ?? 0
  const end = layer.endTime ?? outputDuration.value
  return time > start + 0.1 && time < end - 0.1
}

function assetName(id?: string) {
  return props.assets.find((asset) => asset.id === id)?.name ?? ''
}

function layerTypeName(type: Layer['type']) {
  return {
    video: 'Видео',
    input_video: 'Видео',
    asset_video: 'Видео',
    image: 'Картинка',
    gif: 'GIF',
    subtitles: 'Субтитры',
    text: 'Текст',
    blur: 'Блюр',
    audio: 'Аудио',
    color: 'Блок',
  }[type]
}

function layerColor(type: Layer['type']) {
  if (['video', 'input_video', 'asset_video'].includes(type)) return '#2563eb'
  if (type === 'subtitles') return '#059669'
  if (type === 'image' || type === 'gif') return '#db2777'
  if (type === 'text') return '#0891b2'
  if (type === 'audio') return '#16a34a'
  return '#475569'
}

function formatTime(value: number) {
  const minutes = Math.floor(value / 60)
  const seconds = (value % 60).toFixed(1).padStart(4, '0')
  return `${minutes}:${seconds}`
}

function clamp(value: number, min: number, max: number) {
  return Math.min(max, Math.max(min, value))
}

function roundTime(value: number) {
  return Math.round(value * 1000) / 1000
}

function readVideoDuration(url?: string) {
  if (!url) return Promise.resolve<number | null>(null)
  return new Promise<number | null>((resolve) => {
    const video = document.createElement('video')
    let settled = false
    const finish = (value: number | null) => {
      if (settled) return
      settled = true
      window.clearTimeout(timer)
      video.removeAttribute('src')
      video.load()
      resolve(value)
    }
    const timer = window.setTimeout(() => finish(null), 8000)
    video.preload = 'metadata'
    video.onloadedmetadata = () =>
      finish(
        Number.isFinite(video.duration) && video.duration > 0
          ? roundTime(video.duration)
          : null,
      )
    video.onerror = () => finish(null)
    video.src = url
  })
}
</script>

<template>
  <section class="rounded-xl border border-slate-200 bg-white p-4">
    <div class="flex flex-wrap items-start justify-between gap-4">
      <div>
        <h3 class="font-semibold">Монтажный таймлайн</h3>
        <p class="mt-1 max-w-3xl text-sm text-slate-500">
          Выберите фрагмент или слой — разделение, удаление и скрытие применятся
          именно к нему. Видео-ассет вставляется в позицию курсора; если курсор
          стоит внутри Twitch-клипа, он разделится автоматически. Удерживайте
          Shift, чтобы выбрать несколько фрагментов и перетащить их вместе.
        </p>
      </div>
      <div class="text-right text-sm text-slate-500">
        <p>Курсор: {{ formatTime(currentTime) }}</p>
        <p>Результат: {{ formatTime(outputDuration) }}</p>
      </div>
    </div>

    <div class="mt-4 flex flex-wrap items-center gap-2">
      <AppButton :disabled="readingAsset" @click="pickerOpen = true">
        <Video class="mr-1 inline size-4" />
        {{ readingAsset ? 'Читаем видео…' : 'Вставить видео' }}
      </AppButton>
      <AppButton
        variant="secondary"
        :disabled="!canSplitSelected"
        @click="split"
      >
        <Scissors class="mr-1 inline size-4" />Разделить по курсору
      </AppButton>
      <AppButton
        variant="secondary"
        :disabled="!canSplitAllLayers"
        @click="splitAllLayers"
      >
        <Scissors class="mr-1 inline size-4" />Разделить монтаж и все слои
      </AppButton>
      <AppButton
        variant="secondary"
        :disabled="selectedSegmentIndex <= 0"
        title="Переместить выбранный фрагмент левее"
        @click="moveSelected(-1)"
      >
        <ChevronLeft class="size-4" />
      </AppButton>
      <AppButton
        variant="secondary"
        :disabled="selectedSegmentIndex < 0 || selectedSegmentIndex >= segments.length - 1"
        title="Переместить выбранный фрагмент правее"
        @click="moveSelected(1)"
      >
        <ChevronRight class="size-4" />
      </AppButton>
      <AppButton
        v-if="selectedLayer"
        variant="secondary"
        @click="toggleSelectedLayer"
      >
        <EyeOff v-if="selectedLayer.visible" class="mr-1 inline size-4" />
        <Eye v-else class="mr-1 inline size-4" />
        {{ selectedLayer.visible ? 'Скрыть слой' : 'Показать слой' }}
      </AppButton>
      <AppButton
        variant="danger"
        :disabled="!canRemoveSelected"
        @click="removeSelected"
      >
        <Trash2 class="mr-1 inline size-4" />
        {{ selectedLayer ? 'Удалить слой' : 'Удалить фрагмент' }}
      </AppButton>
      <div
        class="ml-auto flex items-center gap-1 rounded-lg border border-slate-200 bg-slate-50 p-1"
      >
        <button
          type="button"
          class="rounded p-1.5 hover:bg-white"
          title="Приблизить таймлайн"
          aria-label="Приблизить таймлайн"
          @click="zoomIn"
        >
          <ZoomIn class="size-4" />
        </button>
        <button
          type="button"
          class="rounded p-1.5 hover:bg-white"
          title="Отдалить таймлайн"
          aria-label="Отдалить таймлайн"
          @click="zoomOut"
        >
          <ZoomOut class="size-4" />
        </button>
        <button
          type="button"
          class="rounded p-1.5 hover:bg-white"
          title="Вместить таймлайн целиком"
          aria-label="Вместить таймлайн целиком"
          @click="fitTimeline"
        >
          <Maximize2 class="size-4" />
        </button>
        <span class="px-1 text-xs text-slate-500">
          деление {{ formatTime(timelineScale) }}
        </span>
      </div>
    </div>
    <p v-if="assetDurationWarning" class="mt-2 text-sm text-amber-700">
      {{ assetDurationWarning }}
    </p>
    <div class="mt-4 overflow-hidden rounded-lg border border-slate-200">
      <TimelineEditor
        v-model="rows"
        :effects="effects"
        :options="options"
        :auto-scroll="true"
        @time-update="setCurrentTime"
        @click-time-area="setCurrentTime"
        @click-action="selectAction"
        @action-move-end="moveAction"
        @action-resize-start="startResizeAction"
        @action-resizing="previewResizeAction"
        @action-resize-end="resizeAction"
      >
        <template #action="{ action }">
          <button
            type="button"
            class="flex h-full w-full min-w-0 cursor-grab items-center justify-between gap-2 overflow-hidden px-3 text-xs font-medium text-white active:cursor-grabbing"
            :class="action.selected ? 'border-2 border-dashed border-amber-200 shadow-[inset_0_0_0_1px_rgba(15,23,42,0.35)]' : 'border-0'"
            :style="{ backgroundColor: action.data?.color }"
            :draggable="action.data?.kind === 'segment'"
            @mousedown="stopTimelineMove($event, action)"
            @dragstart="startSegmentDrag($event, action)"
            @dragover.prevent
            @drop="dropSegment($event, action)"
            @dragend="draggedSegmentID = null"
          >
            <span class="truncate">{{ action.data?.label }}</span>
            <span class="shrink-0 opacity-75">{{ action.data?.detail }}</span>
          </button>
        </template>
      </TimelineEditor>
    </div>

    <div class="mt-3 flex flex-wrap gap-x-4 gap-y-2 text-xs text-slate-500">
      <span
        ><i
          class="mr-1 inline-block size-2 rounded-full bg-violet-600"
        />Монтаж</span
      >
      <span
        ><i
          class="mr-1 inline-block size-2 rounded-full bg-blue-600"
        />Видео-слои</span
      >
      <span
        ><i
          class="mr-1 inline-block size-2 rounded-full bg-emerald-600"
        />Субтитры</span
      >
      <span
        ><i
          class="mr-1 inline-block size-2 rounded-full bg-pink-600"
        />Картинки</span
      >
      <span
        ><i
          class="mr-1 inline-block size-2 rounded-full bg-cyan-600"
        />Текст</span
      >
    </div>
  </section>

  <AssetPickerDialog
    :open="pickerOpen"
    :assets="assets"
    :folders="folders"
    :kinds="['video']"
    @close="pickerOpen = false"
    @select="insertAsset"
  />
</template>

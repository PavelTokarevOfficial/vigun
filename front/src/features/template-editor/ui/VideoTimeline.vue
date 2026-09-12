<script setup lang="ts">
import { Scissors, Trash2 } from '@lucide/vue'
import { computed, ref } from 'vue'
import TimelineEditor, {
  type TimelineAction,
  type TimelineEffect,
  type TimelineOptions,
  type TimelineRow,
} from 'vue-timeline-editor'
import 'vue-timeline-editor/style.css'
import type { TimelineSegment } from '@/entities/template/model/types'
import AppButton from '@/shared/ui/AppButton.vue'

const props = defineProps<{ segments: TimelineSegment[]; duration: number }>()
const emit = defineEmits<{ update: [segments: TimelineSegment[]] }>()
const currentTime = ref(0)
const selectedID = ref<string | null>(props.segments[0]?.id ?? null)

const effects: Record<string, TimelineEffect> = {
  clip: { id: 'clip', name: 'Twitch-клип' },
}
const options = computed<TimelineOptions>(() => ({
  scale: Math.max(0.25, Math.min(2, props.duration / 30)),
  scaleWidth: 120,
  scaleSplitCount: 5,
  startLeft: 18,
  rowHeight: 44,
  duration: Math.max(1, props.duration),
  gridSnap: true,
  dragLine: true,
  enableRowDrag: false,
}))
const rows = computed<TimelineRow[]>({
  get: () => [
    {
      id: 'source-video',
      actions: props.segments.map((segment, index) => ({
        ...segment,
        effectId: 'clip',
        flexible: true,
        movable: false,
        minStart: props.segments[index - 1]?.end ?? 0,
        maxEnd: props.segments[index + 1]?.start ?? props.duration,
        selected: segment.id === selectedID.value,
        data: { label: `Фрагмент ${index + 1}` },
      })),
    },
  ],
  set: (value) => {
    const segments = (value[0]?.actions ?? [])
      .map(({ id, start, end }) => ({ id, start, end }))
      .sort((left, right) => left.start - right.start)
    emit('update', segments)
  },
})
const outputDuration = computed(() =>
  props.segments.reduce((sum, segment) => sum + segment.end - segment.start, 0),
)

function selectAction(
  _event: MouseEvent,
  params: { action: TimelineAction; row: TimelineRow; time: number },
) {
  selectedID.value = params.action.id
  currentTime.value = params.time
}
function split() {
  const segment =
    props.segments.find(
      (item) => currentTime.value > item.start && currentTime.value < item.end,
    ) ?? props.segments.find((item) => item.id === selectedID.value)
  if (!segment) return
  const point = Math.max(
    segment.start + 0.1,
    Math.min(segment.end - 0.1, currentTime.value),
  )
  if (point <= segment.start || point >= segment.end) return
  const left = { ...segment, end: point }
  const right = {
    id: `segment-${crypto.randomUUID().slice(0, 8)}`,
    start: point,
    end: segment.end,
  }
  selectedID.value = right.id
  emit(
    'update',
    props.segments.flatMap((item) =>
      item.id === segment.id ? [left, right] : [item],
    ),
  )
}
function removeSelected() {
  if (!selectedID.value || props.segments.length <= 1) return
  const next = props.segments.filter((item) => item.id !== selectedID.value)
  selectedID.value = next[0]?.id ?? null
  emit('update', next)
}
function formatTime(value: number) {
  return `${value.toFixed(1)} сек.`
}
</script>

<template>
  <section class="rounded-xl border border-slate-200 bg-white p-4">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <h3 class="font-semibold">Таймлайн</h3>
        <p class="mt-1 text-sm text-slate-500">
          Тяните края фрагмента для обрезки. Поставьте курсор и разделите видео,
          затем удалите ненужный фрагмент.
        </p>
      </div>
      <div class="flex flex-wrap items-center gap-2">
        <span class="text-sm text-slate-500"
          >Курсор: {{ formatTime(currentTime) }} · Результат:
          {{ formatTime(outputDuration) }}</span
        >
        <AppButton variant="secondary" @click="split">
          <Scissors class="mr-1 inline size-4" />Разделить
        </AppButton>
        <AppButton
          variant="danger"
          :disabled="segments.length <= 1 || !selectedID"
          @click="removeSelected"
        >
          <Trash2 class="mr-1 inline size-4" />Удалить фрагмент
        </AppButton>
      </div>
    </div>
    <div class="mt-4 overflow-hidden rounded-lg border border-slate-200">
      <TimelineEditor
        v-model="rows"
        :effects="effects"
        :options="options"
        :auto-scroll="true"
        @time-update="currentTime = $event"
        @click-time-area="currentTime = $event"
        @click-action="selectAction"
      >
        <template #action="{ action }">
          <div
            class="flex h-full items-center overflow-hidden px-3 text-xs font-medium text-white"
          >
            {{ action.data?.label }} · {{ action.start.toFixed(1) }}–{{
              action.end.toFixed(1)
            }}
          </div>
        </template>
      </TimelineEditor>
    </div>
  </section>
</template>

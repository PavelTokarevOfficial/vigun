<script setup lang="ts">
import { X } from '@lucide/vue'
import VideoTimeline from '@/features/template-editor/ui/VideoTimeline.vue'
import AppButton from '@/shared/ui/AppButton.vue'
import type { PipelineWorkspaceModel } from '../model/usePipelineWorkspace'

const props = defineProps<{ workspace: PipelineWorkspaceModel }>()
const {
  busy,
  fragmentClip,
  fragmentOutputDuration,
  fragmentPreviewTime,
  fragmentSegments,
  fragmentSelectedSegmentID,
  fragmentSourceURL,
  fragmentVideo,
  saveFragmentEdit,
  setFragmentTimelineTime,
  startFragmentPreview,
  updateFragmentPreview,
  updateFragmentSegments,
} = props.workspace

function setFragmentVideo(element: unknown) {
  fragmentVideo.value = element instanceof HTMLVideoElement ? element : null
}
</script>

<template>
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
            Оставьте нужные интервалы. Разделите диапазон и удалите часть, чтобы
            вырезать паузу или неудачный момент.
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
          :ref="setFragmentVideo"
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
</template>

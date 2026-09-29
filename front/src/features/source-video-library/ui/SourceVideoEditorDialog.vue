<script setup lang="ts">
import { X } from '@lucide/vue'
import AppButton from '@/shared/ui/AppButton.vue'
import type { SourceVideoLibraryModel } from '../model/useSourceVideoLibrary'
import SourceVideoRangeTimeline from './SourceVideoRangeTimeline.vue'

const props = defineProps<{ library: SourceVideoLibraryModel }>()
const {
  busy,
  canUndo,
  closeEditor,
  editorElement,
  editorVideo,
  isPreviewPlaying,
  playhead,
  saveCut,
  savedMessage,
  selectedSegment,
  setTimelineTime,
  startPreview,
  stopPreview,
  title,
  togglePreview,
  updatePreview,
  updateSelection,
  undoLastAction,
} = props.library
function setVideoElement(value: unknown) {
  editorElement.value = value instanceof HTMLVideoElement ? value : null
}
</script>

<template>
  <div
    v-if="editorVideo"
    class="fixed inset-0 z-50 bg-slate-950"
    role="dialog"
    aria-modal="true"
  >
    <section class="flex h-dvh w-full flex-col overflow-hidden bg-slate-50">
      <header
        class="z-20 flex shrink-0 items-center justify-between gap-4 border-b border-slate-200 bg-white px-4 py-2"
      >
        <div class="min-w-0">
          <h2 class="truncate text-lg font-semibold">
            {{ editorVideo.name }}
          </h2>
          <p class="text-xs text-slate-500">Просмотр и монтаж фрагмента</p>
        </div>
        <button
          type="button"
          class="rounded-lg p-2 hover:bg-slate-100"
          aria-label="Закрыть"
          @click="closeEditor"
        >
          <X class="size-5" />
        </button>
      </header>
      <div class="flex min-h-0 flex-1 flex-col">
        <div
          class="flex min-h-0 flex-1 items-center justify-center bg-slate-950 p-2"
        >
          <video
            :ref="setVideoElement"
            :src="editorVideo.url"
            controls
            autoplay
            preload="metadata"
            class="h-full w-full object-contain"
            @play="startPreview"
            @pause="stopPreview"
            @timeupdate="updatePreview"
          />
        </div>
        <div class="shrink-0 border-t border-slate-200 bg-slate-50 p-1.5">
          <SourceVideoRangeTimeline
            v-if="selectedSegment"
            :can-undo="canUndo"
            :duration="Math.max(0.1, editorVideo.duration)"
            :segment="selectedSegment"
            :playhead="playhead"
            :playing="isPreviewPlaying"
            @update-segment="updateSelection"
            @update-time="setTimelineTime"
            @toggle-preview="togglePreview"
            @undo="undoLastAction"
          />
        </div>
      </div>
      <footer
        class="z-30 flex shrink-0 items-center gap-2 border-t border-slate-200 bg-white px-3 py-2 shadow-[0_-8px_20px_rgba(15,23,42,0.08)]"
      >
        <input
          v-model="title"
          class="min-w-0 flex-1 rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm outline-none focus:border-violet-500"
          maxlength="180"
          aria-label="Название фрагмента"
          placeholder="Название фрагмента"
        >
        <span
          v-if="savedMessage"
          class="hidden shrink-0 text-xs font-medium text-emerald-700 lg:block"
          >{{
            savedMessage
          }}</span
        >
        <div class="flex shrink-0 gap-2">
          <AppButton variant="secondary" @click="closeEditor"
            >Закрыть</AppButton
          >
          <AppButton
            :disabled="busy || !title.trim() || !selectedSegment"
            @click="saveCut"
            >Отправить во фрагменты</AppButton
          >
        </div>
      </footer>
    </section>
  </div>
</template>

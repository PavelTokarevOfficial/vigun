<script setup lang="ts">
import { Film, Play, Trash2 } from '@lucide/vue'
import { onMounted } from 'vue'
import AppButton from '@/shared/ui/AppButton.vue'
import EmptyState from '@/shared/ui/EmptyState.vue'
import ErrorState from '@/shared/ui/ErrorState.vue'
import { useSourceVideoLibrary } from '../model/useSourceVideoLibrary'
import SourceVideoEditorDialog from './SourceVideoEditorDialog.vue'

const library = useSourceVideoLibrary()
const { busy, error, folderId, folders, videos } = library
function formatSize(value: number) {
  if (value >= 1024 ** 3) return `${(value / 1024 ** 3).toFixed(1)} ГБ`
  return `${(value / 1024 ** 2).toFixed(0)} МБ`
}
function formatDuration(value: number) {
  const hours = Math.floor(value / 3600)
  const minutes = Math.floor((value % 3600) / 60)
  const seconds = Math.floor(value % 60)
  return hours
    ? `${hours}:${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}`
    : `${minutes}:${String(seconds).padStart(2, '0')}`
}
onMounted(() => void library.load())
</script>

<template>
  <section>
    <div
      class="w-full max-w-md rounded-xl border border-slate-200 bg-white p-3"
    >
      <label class="block text-sm font-medium">
        Закреплённая папка из Ассетов
        <select
          :value="folderId ?? ''"
          class="mt-1 w-full rounded-lg border border-slate-300 bg-white px-3 py-2"
          :disabled="busy"
          @change="library.pinFolder(($event.target as HTMLSelectElement).value)"
        >
          <option value="">Не выбрана</option>
          <option v-for="folder in folders" :key="folder.id" :value="folder.id">
            {{ folder.name }}
          </option>
        </select>
      </label>
    </div>
    <ErrorState v-if="error" class="mt-5" :message="error" />
    <EmptyState
      v-if="!videos.length && !busy"
      class="mt-6"
      message="Здесь появятся загруженные фильмы и эпизоды."
    />
    <div v-else class="mt-6 grid gap-5 sm:grid-cols-2 xl:grid-cols-3">
      <article
        v-for="video in videos"
        :key="video.id"
        class="group relative overflow-hidden rounded-xl border border-slate-200 bg-white shadow-sm transition hover:-translate-y-0.5 hover:border-violet-300 hover:shadow-md focus-within:ring-2 focus-within:ring-violet-500"
      >
        <button
          type="button"
          class="absolute inset-0 z-10 cursor-pointer"
          :aria-label="`Открыть видео ${video.name}`"
          @click="library.openEditor(video)"
        />
        <div class="relative bg-black">
          <video
            :src="video.url"
            muted
            preload="metadata"
            class="pointer-events-none aspect-video w-full object-contain"
            @loadedmetadata="library.rememberDuration(video, ($event.currentTarget as HTMLVideoElement).duration)"
          />
          <span
            class="absolute inset-0 grid place-content-center bg-slate-950/10 transition group-hover:bg-slate-950/25"
          >
            <span
              class="grid size-14 place-content-center rounded-full bg-white/90 text-violet-700 shadow-lg transition group-hover:scale-105"
            >
              <Play class="ml-0.5 size-6" fill="currentColor" />
            </span>
          </span>
        </div>
        <div class="p-4">
          <div class="flex items-start gap-3">
            <Film class="mt-0.5 size-5 shrink-0 text-violet-600" />
            <div class="min-w-0 flex-1">
              <h2 class="truncate font-semibold" :title="video.name">
                {{ video.name }}
              </h2>
              <p class="mt-1 text-sm text-slate-500">
                {{ formatDuration(video.duration) }} ·
                {{ formatSize(video.size) }} · фрагментов: {{ video.cuts }}
              </p>
            </div>
          </div>
          <div class="relative z-20 mt-4 flex justify-end gap-2">
            <AppButton
              variant="danger"
              class="grid size-9 place-content-center"
              :disabled="busy"
              title="Удалить исходник"
              @click.stop="library.remove(video)"
              ><Trash2 :size="16" /></AppButton
            >
          </div>
        </div>
      </article>
    </div>
    <SourceVideoEditorDialog :library="library" />
  </section>
</template>

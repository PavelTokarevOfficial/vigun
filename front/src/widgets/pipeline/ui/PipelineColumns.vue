<script setup lang="ts">
import {
  ArrowLeft,
  Check,
  Download,
  ExternalLink,
  Play,
  Plus,
  RotateCcw,
  Scissors,
  Send,
  Trash,
  X,
} from '@lucide/vue'
import AppButton from '@/shared/ui/AppButton.vue'
import type { PipelineWorkspaceModel } from '../model/usePipelineWorkspace'

const props = defineProps<{ workspace: PipelineWorkspaceModel }>()
const {
  action,
  busy,
  canDelete,
  downloaded,
  isProcessQueued,
  openClipPreview,
  openFragmentEditor,
  openInstagramDialog,
  openRenderedVideo,
  openTemplateChooser,
  readyFragments,
  remove,
  removeRenderedVideo,
  selectedTrainClipIDs,
  statusText,
  startTrain,
  toggleTrainClip,
  renderJobs,
  renderJobStatus,
  renderJobStatusClass,
  trainSelectionMode,
  usedFragmentIDs,
  updateFragment,
  videos,
} = props.workspace

function toggleTrainSelection() {
  trainSelectionMode.value = !trainSelectionMode.value
  selectedTrainClipIDs.value = new Set()
}
</script>

<template>
  <div class="mt-6 grid gap-4 xl:grid-cols-4">
    <section class="min-w-0">
      <h3 class="font-semibold">Скачанные · {{ downloaded.length }}</h3>
      <div class="mt-3 space-y-3">
        <p v-if="!downloaded.length" class="text-sm text-slate-500">
          Пока пусто.
        </p>
        <article
          v-for="clip in downloaded"
          :key="clip.id"
          class="relative overflow-hidden rounded-xl"
        >
          <img
            v-if="clip.thumbnailUrl"
            :src="clip.thumbnailUrl"
            :alt="`Превью клипа: ${clip.title}`"
            class="aspect-video w-full object-cover"
            draggable="false"
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
              v-if="clip.hasSource"
              class="grid h-8 w-8 place-content-center"
              :disabled="busy === clip.id"
              title="Посмотреть скачанное видео"
              :aria-label="`Посмотреть скачанное видео «${clip.title}»`"
              @click="openClipPreview(clip)"
            >
              <Play :size="16" />
            </AppButton>
            <AppButton
              v-if="['downloaded', 'completed'].includes(clip.status) && !isProcessQueued(clip)"
              :disabled="busy === clip.id"
              class="grid place-content-center w-8 h-8"
              title="Обрезать и подготовить фрагмент"
              @click="openFragmentEditor(clip)"
            >
              <Scissors :size="16" />
            </AppButton>
            <AppButton
              v-if="clip.hasSource && !isProcessQueued(clip)"
              :disabled="busy === clip.id"
              class="grid h-8 w-8 place-content-center"
              title="Отправить в готовые фрагменты без редактирования"
              @click="updateFragment(clip, true)"
            >
              <Check :size="16" />
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

    <section class="xl:border-l xl:border-dashed xl:border-slate-300 xl:pl-4">
      <div class="flex items-center justify-between gap-2">
        <h3 class="font-semibold">
          Фрагменты · {{ readyFragments.length }}
        </h3>
        <div class="flex items-center gap-2">
          <button
            v-if="trainSelectionMode && selectedTrainClipIDs.size >= 2"
            type="button"
            class="rounded-lg bg-violet-600 px-3 py-1.5 text-xs font-semibold text-white hover:bg-violet-700"
            @click="startTrain"
          >
            Собрать · {{ selectedTrainClipIDs.size }}
          </button>
          <button
            type="button"
            class="grid size-8 place-content-center rounded-lg border text-slate-700 transition hover:bg-slate-100"
            :class="trainSelectionMode ? 'border-violet-300 bg-violet-50 text-violet-700' : 'border-slate-200 bg-white'"
            :title="trainSelectionMode ? 'Отменить выбор' : 'Собрать несколько фрагментов'"
            :aria-label="trainSelectionMode ? 'Отменить выбор фрагментов' : 'Собрать несколько фрагментов'"
            @click="toggleTrainSelection"
          >
            <X v-if="trainSelectionMode" :size="17" />
            <Plus v-else :size="17" />
          </button>
        </div>
      </div>
      <div class="mt-3 space-y-3">
        <p v-if="!readyFragments.length" class="text-sm text-slate-500">
          Пока пусто.
        </p>
        <article
          v-for="clip in readyFragments"
          :key="clip.id"
          class="relative overflow-hidden rounded-xl ring-offset-2"
          :class="[selectedTrainClipIDs.has(clip.id) ? 'ring-2 ring-violet-500' : '']"
          :tabindex="trainSelectionMode && !usedFragmentIDs.has(clip.id) ? 0 : undefined"
          @click="trainSelectionMode && toggleTrainClip(clip.id)"
          @keydown.enter="trainSelectionMode && toggleTrainClip(clip.id)"
        >
          <img
            v-if="clip.thumbnailUrl"
            :src="clip.thumbnailUrl"
            :alt="`Превью фрагмента: ${clip.title}`"
            :class="[
              'aspect-video w-full object-cover',
              usedFragmentIDs.has(clip.id) ? 'brightness-50' : '',
            ]"
            draggable="false"
          >
          <div
            class="absolute top-0 px-3 py-1 text-white [-webkit-text-stroke:2px_black] [paint-order:stroke_fill]"
          >
            <b>{{ clip.title }}</b>
            <p>{{ clip.streamerName }}</p>
          </div>
          <span
            v-if="usedFragmentIDs.has(clip.id)"
            class="absolute top-3 right-3 grid size-7 place-content-center rounded-full bg-emerald-500 text-white shadow"
            title="Фрагмент уже использован в рендере"
          >
            <Check :size="17" :stroke-width="3" />
          </span>
          <span
            v-if="trainSelectionMode && !usedFragmentIDs.has(clip.id)"
            class="absolute left-3 bottom-3 grid size-7 place-content-center rounded-full bg-white text-violet-700"
          >
            <Check v-if="selectedTrainClipIDs.has(clip.id)" :size="17" />
          </span>
          <div v-else class="absolute right-0 bottom-0 flex gap-2 p-3">
            <AppButton
              class="grid h-8 w-8 place-content-center"
              @click.stop="openClipPreview(clip)"
            >
              <Play :size="16" />
            </AppButton>
            <AppButton
              class="grid h-8 w-8 place-content-center"
              title="Изменить монтаж"
              @click.stop="openFragmentEditor(clip)"
            >
              <Scissors :size="16" />
            </AppButton>
            <AppButton
              v-if="!usedFragmentIDs.has(clip.id)"
              class="grid h-8 w-8 place-content-center"
              title="Отправить на рендер"
              @click.stop="openTemplateChooser(clip.id)"
            >
              <Plus :size="16" />
            </AppButton>
            <AppButton
              variant="secondary"
              class="grid h-8 w-8 place-content-center"
              title="Вернуть в скачанные"
              @click.stop="updateFragment(clip, false)"
            >
              <ArrowLeft :size="16" />
            </AppButton>
          </div>
        </article>
      </div>
    </section>

    <section
      class="col-span-2 xl:border-l xl:border-dashed xl:border-slate-300 xl:pl-4"
    >
      <h3 class="font-semibold">Готовые · {{ videos.length }}</h3>
      <div class="grid grid-cols-2 gap-3 mt-3">
        <p
          v-if="!videos.length && !renderJobs.length"
          class="text-sm text-slate-500"
        >
          Пока пусто.
        </p>
        <article
          v-for="job in renderJobs"
          :key="`rendering-${job.id}`"
          class="relative overflow-hidden rounded-xl border border-violet-200 bg-violet-50"
        >
          <div
            class="grid aspect-video w-full place-content-center bg-gradient-to-br from-violet-100 to-slate-200"
          >
            <span class="text-sm font-semibold text-violet-800">
              <template v-if="job.isTrain">
                Паровозик · {{ job.fragmentCount }} фрагм.
              </template>
              <template v-else>
                {{ job.clipTitle || 'Видео' }}
              </template>
            </span>
          </div>
          <div class="absolute inset-x-0 top-0 p-3 text-slate-900">
            <b>{{ job.templateName || 'Рендер видео' }}</b>
            <p class="text-xs">{{ renderJobStatus(job) }}</p>
          </div>
          <div class="absolute inset-x-3 bottom-3">
            <div class="h-1.5 overflow-hidden rounded-full bg-white/80">
              <div
                class="h-full rounded-full bg-violet-600 transition-all"
                :class="renderJobStatusClass(job)"
                :style="{ width: `${Math.max(job.status === 'pending' ? 4 : job.progress, 4)}%` }"
              />
            </div>
            <p
              v-if="job.status === 'failed' && job.error"
              class="mt-1 line-clamp-1 text-xs text-red-700"
            >
              {{ job.error.split('\n')[0] }}
            </p>
          </div>
        </article>
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
            draggable="false"
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
              @click="openRenderedVideo(video)"
            >
              <Play :size="16" />
            </AppButton>
            <AppButton
              variant="secondary"
              class="grid h-8 w-8 place-content-center"
              title="Опубликовать в Instagram"
              :aria-label="`Опубликовать «${video.title}» в Instagram`"
              @click="openInstagramDialog(video)"
            >
              <Send :size="16" />
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
</template>

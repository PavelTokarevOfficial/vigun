<script setup lang="ts">
import {
  Check,
  Download,
  ExternalLink,
  Film,
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
  cancelJob,
  downloaded,
  isProcessQueued,
  openClipPreview,
  openFragmentEditor,
  openInstagramDialog,
  openRenderedVideo,
  openTemplateChooser,
  readyFragments,
  fragmentJobs,
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

function formatRenderDuration(seconds: number) {
  const total = Math.max(1, Math.round(seconds))
  const hours = Math.floor(total / 3600)
  const minutes = Math.floor((total % 3600) / 60)
  const rest = total % 60
  return [
    hours ? `${hours} ч` : '',
    minutes ? `${minutes} мин` : '',
    rest || (!hours && !minutes) ? `${rest} сек` : '',
  ]
    .filter(Boolean)
    .join(' ')
}
</script>

<template>
  <div class="mt-6 grid gap-4 xl:grid-cols-4 xl:gap-8">
    <section class="min-w-0">
      <h3 class="min-h-10 font-semibold">
        Скачанные · {{ downloaded.length }}
      </h3>
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
            :class="clip.hasFragment ? 'brightness-50' : ''"
            draggable="false"
          >
          <div
            v-else
            class="grid aspect-video w-full place-content-center bg-gradient-to-br from-violet-700 to-slate-900 text-white"
            :class="clip.hasFragment ? 'brightness-50' : ''"
          >
            <Film class="size-10 opacity-70" />
          </div>

          <span
            v-if="clip.hasFragment"
            class="absolute top-3 right-3 grid size-7 place-content-center rounded-full bg-emerald-500 text-white shadow"
            title="Из исходника уже создан фрагмент"
          >
            <Check :size="17" :stroke-width="3" />
          </span>

          <div
            class="absolute inset-x-0 top-0 min-w-0 px-3 py-1 pr-12 text-white [-webkit-text-stroke:2px_black] [paint-order:stroke_fill]"
          >
            <b class="block truncate" :title="clip.title">{{ clip.title }}</b>
            <p>{{ clip.streamerName }} · {{ statusText(clip) }}</p>
            <p v-if="clip.error" class="mt-1 text-sm text-red-600">
              {{ clip.error }}
            </p>
          </div>

          <div class="absolute right-0 bottom-0 flex gap-2 p-3">
            <AppButton
              v-if="!clip.hasSource && !['pending', 'running'].includes(clip.lastJobStatus)"
              class="grid h-8 w-8 place-content-center"
              :disabled="busy === clip.id"
              title="Скачать исходное видео заново"
              :aria-label="`Скачать исходное видео «${clip.title}» заново`"
              @click="action(clip.id, 'download')"
            >
              <Download :size="16" />
            </AppButton>
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
              v-if="clip.hasSource && ['downloaded', 'completed'].includes(clip.status) && !isProcessQueued(clip)"
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

    <section
      class="relative min-w-0 xl:before:absolute xl:before:inset-y-0 xl:before:-left-4 xl:before:border-l xl:before:border-dashed xl:before:border-slate-300 xl:before:content-['']"
    >
      <div class="flex items-center justify-between gap-2">
        <h3 class="min-h-10 font-semibold">
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
        <p
          v-if="!readyFragments.length && !fragmentJobs.length"
          class="text-sm text-slate-500"
        >
          Пока пусто.
        </p>
        <article
          v-for="job in fragmentJobs"
          :key="`fragment-job-${job.id}`"
          class="relative overflow-hidden rounded-xl border border-violet-200 bg-violet-50"
        >
          <div
            class="grid aspect-video w-full place-content-center bg-gradient-to-br from-violet-100 to-slate-200 px-4 text-center"
          >
            <span
              class="block max-w-full truncate text-sm font-semibold text-violet-800"
              :title="job.clipTitle || 'Новый фрагмент'"
              >{{
                job.clipTitle || 'Новый фрагмент'
              }}</span
            >
          </div>
          <div class="absolute inset-x-0 top-0 p-3 text-slate-900">
            <b>Создание фрагмента</b>
            <p class="text-xs">{{ renderJobStatus(job) }}</p>
          </div>
          <div class="absolute inset-x-3 bottom-3 flex items-end gap-2">
            <div class="min-w-0 flex-1">
              <div class="h-1.5 overflow-hidden rounded-full bg-white/80">
                <div
                  class="h-full rounded-full bg-violet-600 transition-all"
                  :style="{ width: `${Math.max(job.status === 'pending' ? 4 : job.progress, 4)}%` }"
                />
              </div>
            </div>
            <AppButton
              v-if="['pending', 'running'].includes(job.status)"
              variant="danger"
              class="grid size-8 place-content-center"
              :disabled="busy === job.id"
              title="Остановить создание фрагмента"
              @click="cancelJob(job.id)"
            >
              <X :size="16" />
            </AppButton>
          </div>
        </article>
        <article
          v-for="clip in readyFragments"
          :key="clip.id"
          class="relative overflow-hidden rounded-xl ring-offset-2"
          :class="[selectedTrainClipIDs.has(clip.id) ? 'ring-2 ring-violet-500' : '']"
          :tabindex="trainSelectionMode ? 0 : undefined"
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
            v-else
            class="grid aspect-video w-full place-content-center bg-gradient-to-br from-violet-700 to-slate-900 text-white"
            :class="usedFragmentIDs.has(clip.id) ? 'brightness-50' : ''"
          >
            <Film class="size-10 opacity-70" />
          </div>
          <div
            class="absolute inset-x-0 top-0 min-w-0 px-3 py-1 pr-12 text-white [-webkit-text-stroke:2px_black] [paint-order:stroke_fill]"
          >
            <b class="block truncate" :title="clip.title">{{ clip.title }}</b>
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
            v-if="trainSelectionMode"
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
              v-if="!isProcessQueued(clip)"
              class="grid h-8 w-8 place-content-center"
              title="Отправить на рендер"
              @click.stop="openTemplateChooser(clip.id)"
            >
              <Plus :size="16" />
            </AppButton>
            <AppButton
              variant="secondary"
              class="grid h-8 w-8 place-content-center"
              title="Удалить фрагмент"
              @click.stop="remove(clip)"
            >
              <Trash :size="16" />
            </AppButton>
          </div>
        </article>
      </div>
    </section>

    <section
      class="relative min-w-0 xl:col-span-2 xl:before:absolute xl:before:inset-y-0 xl:before:-left-4 xl:before:border-l xl:before:border-dashed xl:before:border-slate-300 xl:before:content-['']"
    >
      <h3 class="min-h-10 font-semibold">Готовые · {{ videos.length }}</h3>
      <div class="grid xl:grid-cols-2 gap-4 xl:gap-8 mt-3">
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
            <span
              class="block max-w-full truncate px-3 text-sm font-semibold text-violet-800"
              :title="job.isTrain ? `Паровозик · ${job.fragmentCount} фрагм.` : (job.clipTitle || 'Видео')"
            >
              <template v-if="job.isTrain">
                Паровозик · {{ job.fragmentCount }} фрагм.
              </template>
              <template v-else>
                {{ job.clipTitle || 'Видео' }}
              </template>
            </span>
          </div>
          <div class="absolute inset-x-0 top-0 p-3 text-slate-900">
            <b
              class="block truncate"
              :title="job.templateName || 'Рендер видео'"
              >{{
                job.templateName || 'Рендер видео'
              }}</b
            >
            <p class="text-xs">{{ renderJobStatus(job) }}</p>
          </div>
          <div class="absolute inset-x-3 bottom-3 flex items-end gap-2">
            <div class="min-w-0 flex-1">
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
            <AppButton
              v-if="['pending', 'running'].includes(job.status)"
              variant="danger"
              class="grid size-8 place-content-center"
              :disabled="busy === job.id"
              title="Остановить рендер"
              @click="cancelJob(job.id)"
            >
              <X :size="16" />
            </AppButton>
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
            class="grid aspect-video w-full place-content-center bg-gradient-to-br from-violet-700 to-slate-900 text-white"
          >
            <Film class="size-10 opacity-70" />
          </div>

          <div
            class="absolute inset-x-0 top-0 min-w-0 px-3 py-1 text-white [-webkit-text-stroke:2px_black] [paint-order:stroke_fill]"
          >
            <b class="block truncate" :title="video.title">{{ video.title }}</b>
            <p>
              {{ video.streamer }}
              <template v-if="video.templateName">
                · {{ video.templateName }}</template
              >
              · {{ new Date(video.createdAt).toLocaleString() }}
              <template v-if="video.renderDurationSeconds > 0">
                · рендер {{ formatRenderDuration(video.renderDurationSeconds) }}
              </template>
              <template v-if="video.whisperModel">
                · Whisper {{ video.whisperModel }}
              </template>
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

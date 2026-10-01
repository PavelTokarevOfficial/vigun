<script setup lang="ts">
import { X } from '@lucide/vue'
import { computed, nextTick, ref, watch } from 'vue'
import type { PipelineWorkspaceModel } from '../model/usePipelineWorkspace'

const props = defineProps<{ workspace: PipelineWorkspaceModel }>()
const { previewVideo } = props.workspace

type SubtitleCue = { start: number; end: number; text: string }

const video = ref<HTMLVideoElement | null>(null)
const subtitleList = ref<HTMLElement | null>(null)
const cues = ref<SubtitleCue[]>([])
const currentTime = ref(0)
const subtitlesLoading = ref(false)
const subtitlesError = ref('')
const activeCueIndex = computed(() =>
  cues.value.findIndex(
    (cue) => currentTime.value >= cue.start && currentTime.value < cue.end,
  ),
)

watch(
  () => previewVideo.value?.key,
  async () => {
    cues.value = []
    currentTime.value = 0
    subtitlesError.value = ''
    const videoId = previewVideo.value?.renderedVideoId
    if (!videoId) return
    subtitlesLoading.value = true
    try {
      const response = await fetch(`/api/videos/${videoId}/subtitles`)
      if (!response.ok) {
        subtitlesError.value = 'Для этого видео субтитры не сохранились'
        return
      }
      cues.value = parseSRT(await response.text())
      if (!cues.value.length) {
        subtitlesError.value = 'В этом видео нет распознанных фраз'
      }
    } catch {
      subtitlesError.value = 'Не удалось загрузить субтитры'
    } finally {
      subtitlesLoading.value = false
    }
  },
  { immediate: true },
)

watch(activeCueIndex, async (index) => {
  if (index < 0) return
  await nextTick()
  const list = subtitleList.value
  const cue = list?.querySelector<HTMLElement>(`[data-cue-index="${index}"]`)
  if (!list || !cue) return
  const listBounds = list.getBoundingClientRect()
  const cueBounds = cue.getBoundingClientRect()
  list.scrollTo({
    top:
      list.scrollTop +
      cueBounds.top -
      listBounds.top -
      list.clientHeight / 2 +
      cueBounds.height / 2,
    behavior: 'smooth',
  })
})

function parseSRT(raw: string): SubtitleCue[] {
  return raw
    .replaceAll('\r\n', '\n')
    .trim()
    .split(/\n{2,}/)
    .flatMap((block) => {
      const lines = block.split('\n')
      const timingIndex = lines.findIndex((line) => line.includes('-->'))
      if (timingIndex < 0) return []
      const [start, end] = lines[timingIndex].split('-->').map(parseSRTTime)
      const text = lines
        .slice(timingIndex + 1)
        .join(' ')
        .replace(/<[^>]+>/g, '')
        .trim()
      return Number.isFinite(start) && Number.isFinite(end) && text
        ? [{ start, end, text }]
        : []
    })
}

function parseSRTTime(value: string) {
  const parts = value.trim().replace(',', '.').split(':').map(Number)
  if (parts.length !== 3 || parts.some((part) => !Number.isFinite(part))) {
    return Number.NaN
  }
  return parts[0] * 3600 + parts[1] * 60 + parts[2]
}

function seekToCue(cue: SubtitleCue) {
  if (!video.value) return
  video.value.currentTime = cue.start
  void video.value.play()
}
</script>

<template>
  <div
    v-if="previewVideo"
    class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/70 p-4"
    role="dialog"
    aria-modal="true"
    aria-labelledby="video-preview-title"
  >
    <section
      class="w-full max-w-6xl overflow-hidden rounded-2xl bg-white shadow-2xl"
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
      <div
        class="grid h-[min(80vh,52rem)] min-h-0 bg-slate-950 lg:grid-cols-[minmax(0,1fr)_minmax(20rem,0.8fr)]"
      >
        <div
          class="flex min-h-0 min-w-0 items-center justify-center overflow-hidden p-4"
        >
          <video
            v-if="previewVideo.mode === 'video'"
            :key="previewVideo.key"
            ref="video"
            :src="previewVideo.url"
            controls
            autoplay
            playsinline
            class="block h-full max-h-full w-full max-w-full object-contain"
            @timeupdate="currentTime = video?.currentTime ?? 0"
          />
          <iframe
            v-else
            :key="previewVideo.key"
            :src="previewVideo.url"
            :title="`Twitch-клип: ${previewVideo.title}`"
            class="aspect-video max-h-[75vh] w-full border-0"
            allow="autoplay; fullscreen"
            allowfullscreen
          />
        </div>
        <aside
          v-if="previewVideo.renderedVideoId"
          class="flex min-h-0 flex-col border-t border-white/10 bg-slate-900 lg:border-t-0 lg:border-l"
        >
          <div class="border-b border-white/10 px-5 py-4">
            <h3 class="font-semibold text-white">Субтитры</h3>
            <p class="mt-1 text-xs text-slate-400">
              Фразы прокручиваются вместе с видео
            </p>
          </div>
          <div
            ref="subtitleList"
            class="min-h-48 flex-1 overflow-y-auto px-4 py-[30vh] lg:min-h-0"
          >
            <p v-if="subtitlesLoading" class="text-sm text-slate-400">
              Загружаем субтитры…
            </p>
            <p v-else-if="subtitlesError" class="text-sm text-slate-400">
              {{ subtitlesError }}
            </p>
            <button
              v-for="(cue, index) in cues"
              v-else
              :key="`${cue.start}-${index}`"
              type="button"
              :data-cue-index="index"
              class="block w-full rounded-xl px-4 py-3 text-left text-base leading-7 transition"
              :class="
                index === activeCueIndex
                  ? 'bg-white/10 font-semibold text-white'
                  : index < activeCueIndex
                    ? 'text-slate-500'
                    : 'text-slate-300 hover:bg-white/5 hover:text-white'
              "
              @click="seekToCue(cue)"
            >
              {{ cue.text }}
            </button>
          </div>
        </aside>
      </div>
    </section>
  </div>
</template>

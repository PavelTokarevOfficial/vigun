<script setup lang="ts">
import { Check, ChevronLeft, ChevronRight, ExternalLink, X } from '@lucide/vue'
import type { Swiper as SwiperInstance } from 'swiper'
import { Keyboard } from 'swiper/modules'
import { Swiper, SwiperSlide } from 'swiper/vue'
import {
  computed,
  onBeforeUnmount,
  onMounted,
  ref,
  shallowRef,
  watch,
} from 'vue'
import 'swiper/css'
import type { TwitchClip } from '@/entities/clip/model/types'

const props = defineProps<{
  open: boolean
  clips: TwitchClip[]
  streamerName: string
  initialIndex?: number
}>()
const emit = defineEmits<{
  close: []
  view: [clip: TwitchClip]
}>()
defineSlots<{
  action(props: { clip: TwitchClip }): unknown
}>()

const swiper = shallowRef<SwiperInstance | null>(null)
const currentIndex = ref(0)
const modules = [Keyboard]

const normalizedInitialIndex = computed(() =>
  Math.min(
    Math.max(props.initialIndex ?? 0, 0),
    Math.max(0, props.clips.length - 1),
  ),
)
const currentClip = computed(() => props.clips[currentIndex.value] ?? null)
const canGoBack = computed(() => currentIndex.value > 0)
const canGoForward = computed(() => currentIndex.value < props.clips.length - 1)
const iframeAllowed = computed(
  () =>
    window.location.protocol === 'https:' ||
    ['localhost', '127.0.0.1', '[::1]'].includes(window.location.hostname),
)

watch(
  () => props.open,
  (open) => {
    if (!open) {
      swiper.value = null
      return
    }
    currentIndex.value = normalizedInitialIndex.value
  },
)
watch(
  () => props.clips.length,
  (length) => {
    currentIndex.value = Math.min(currentIndex.value, Math.max(0, length - 1))
  },
)

function iframeURL(clip: TwitchClip) {
  const query = new URLSearchParams({
    clip: clip.id,
    parent: window.location.hostname || 'localhost',
    autoplay: 'true',
    muted: 'false',
  })
  return `https://clips.twitch.tv/embed?${query}`
}

function setSwiper(instance: SwiperInstance) {
  swiper.value = instance
  currentIndex.value = instance.activeIndex
  if (currentClip.value) emit('view', currentClip.value)
}

function slideChanged(instance: SwiperInstance) {
  currentIndex.value = instance.activeIndex
  if (currentClip.value) emit('view', currentClip.value)
}

function navigate(delta: number) {
  if (delta < 0) swiper.value?.slidePrev()
  else swiper.value?.slideNext()
}

function keydown(event: KeyboardEvent) {
  if (props.open && event.key === 'Escape') emit('close')
}

onMounted(() => window.addEventListener('keydown', keydown))
onBeforeUnmount(() => window.removeEventListener('keydown', keydown))
</script>

<template>
  <Teleport to="body">
    <div
      v-if="open"
      class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/80 p-3 backdrop-blur-sm sm:p-6"
      role="dialog"
      aria-modal="true"
      aria-label="Предпросмотр Twitch-клипов"
    >
      <button
        type="button"
        class="absolute inset-0 cursor-default"
        aria-label="Закрыть предпросмотр"
        @click="emit('close')"
      />
      <section class="relative z-10 w-full max-w-3xl">
        <div class="mb-3 flex items-center justify-between text-white">
          <div>
            <p class="text-sm text-white/65">{{ streamerName }}</p>
            <p class="font-medium">
              {{
                clips.length
                  ? `${currentIndex + 1} из ${clips.length}`
                  : 'Клипов нет'
              }}
            </p>
          </div>
          <button
            type="button"
            class="grid size-11 place-items-center rounded-full bg-white/10 transition hover:bg-white/20 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-white"
            aria-label="Закрыть предпросмотр"
            @click="emit('close')"
          >
            <X class="size-6" />
          </button>
        </div>

        <Swiper
          v-if="clips.length"
          class="preview-swiper"
          :modules="modules"
          :initial-slide="normalizedInitialIndex"
          :keyboard="{ enabled: true }"
          :space-between="24"
          @swiper="setSwiper"
          @slide-change="slideChanged"
        >
          <SwiperSlide v-for="(clip, index) in clips" :key="clip.id">
            <article class="overflow-hidden rounded-2xl bg-white shadow-2xl">
              <div class="relative aspect-video bg-black">
                <iframe
                  v-if="iframeAllowed && index === currentIndex"
                  :key="clip.id"
                  :src="iframeURL(clip)+ '&autoplay=true&muted=false'"
                  :title="`Twitch-клип: ${clip.title}`"
                  class="size-full border-0"
                  allow="autoplay; fullscreen"
                  allowfullscreen
                />
                <img
                  v-else
                  :src="clip.thumbnail_url"
                  :alt="`Превью клипа: ${clip.title}`"
                  class="size-full object-cover"
                >
                <div
                  v-if="!iframeAllowed && index === currentIndex"
                  class="absolute inset-x-0 bottom-0 bg-slate-950/85 p-4 text-center text-sm text-white"
                >
                  <p>
                    Twitch разрешает iframe только через HTTPS или localhost.
                  </p>
                  <a
                    :href="clip.url"
                    target="_blank"
                    rel="noopener noreferrer"
                    class="mt-2 inline-flex items-center gap-1 font-medium text-violet-300"
                  >
                    <ExternalLink class="size-4" />Открыть клип на Twitch
                  </a>
                </div>
              </div>

              <div class="p-4 sm:p-5">
                <h3 class="text-lg font-semibold">{{ clip.title }}</h3>
                <div
                  class="mt-1 flex flex-wrap items-center gap-2 text-sm text-slate-500"
                >
                  <span>
                    {{ new Date(clip.created_at).toLocaleDateString() }}
                    ·
                    {{ Math.round(clip.duration) }}
                    сек.
                  </span>
                  <span
                    v-if="clip.viewed"
                    class="inline-flex items-center gap-1 rounded-full bg-emerald-50 px-2 py-0.5 text-xs font-medium text-emerald-700"
                  >
                    <Check class="size-3" />Просмотрено
                  </span>
                </div>
                <div class="mt-3 flex items-center justify-between gap-3">
                  <a
                    :href="clip.url"
                    target="_blank"
                    rel="noopener noreferrer"
                    class="inline-flex items-center gap-1 text-sm text-violet-700"
                  >
                    <ExternalLink class="size-4" />Twitch
                  </a>
                  <slot name="action" :clip="clip" />
                </div>
              </div>
            </article>
          </SwiperSlide>
        </Swiper>

        <div class="mt-4 flex justify-center gap-4">
          <button
            type="button"
            class="grid size-12 place-items-center rounded-full bg-white text-slate-700 shadow-lg transition enabled:hover:scale-105 disabled:cursor-not-allowed disabled:opacity-35"
            :disabled="!canGoBack"
            aria-label="Предыдущий клип"
            @click="navigate(-1)"
          >
            <ChevronLeft class="size-7" />
          </button>
          <button
            type="button"
            class="grid size-12 place-items-center rounded-full bg-violet-600 text-white shadow-lg transition enabled:hover:scale-105 disabled:cursor-not-allowed disabled:opacity-35"
            :disabled="!canGoForward"
            aria-label="Следующий клип"
            @click="navigate(1)"
          >
            <ChevronRight class="size-7" />
          </button>
        </div>
      </section>
    </div>
  </Teleport>
</template>

<style scoped>
.preview-swiper :deep(.swiper-slide) {
  height: auto;
}
</style>

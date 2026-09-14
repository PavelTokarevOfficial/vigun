<script setup lang="ts">
import { Check, ExternalLink } from '@lucide/vue'
import { Navigation } from 'swiper/modules'
import { Swiper, SwiperSlide } from 'swiper/vue'
import 'swiper/css'
import 'swiper/css/navigation'
import type { SubscriptionFeed, TwitchClip } from '@/entities/clip/model/types'
import ClipImportButton from '@/features/clip-import/ClipImportButton.vue'

defineProps<{
  feed: SubscriptionFeed
  busyClipId: string
}>()
const emit = defineEmits<{
  open: [index: number]
  save: [clip: TwitchClip]
}>()

const modules = [Navigation]
</script>

<template>
  <div class="subscription-swiper relative">
    <Swiper
      :modules="modules"
      :navigation="true"
      :space-between="14"
      :slides-per-view="1.12"
      :breakpoints="{
        520: { slidesPerView: 2.1 },
        850: { slidesPerView: 3.1 },
        1180: { slidesPerView: 4.1 },
      }"
    >
      <SwiperSlide
        v-for="(clip, index) in feed.clips"
        :key="clip.id"
        class="h-auto"
      >
        <article
          class="flex h-full flex-col overflow-hidden rounded-xl border border-slate-200 bg-slate-50 transition hover:border-violet-300 hover:shadow-md"
        >
          <button
            type="button"
            class="block flex-1 text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-violet-600"
            :aria-label="`Смотреть клип: ${clip.title}`"
            @click="emit('open', index)"
          >
            <div class="relative aspect-video overflow-hidden bg-slate-200">
              <img
                :src="clip.thumbnail_url"
                :alt="`Превью клипа: ${clip.title}`"
                class="size-full object-cover"
              >
              <span
                v-if="clip.viewed"
                class="absolute left-2 top-2 inline-flex items-center gap-1 rounded-full bg-emerald-600 px-2 py-1 text-xs font-semibold text-white shadow"
              >
                <Check class="size-3" />Просмотрено
              </span>
              <span
                class="absolute inset-0 grid place-items-center bg-slate-950/0 text-white opacity-0 transition hover:bg-slate-950/25 hover:opacity-100"
              >
                <span
                  class="rounded-full bg-slate-950/75 px-3 py-1.5 text-sm font-semibold"
                >
                  Смотреть
                </span>
              </span>
            </div>
            <div class="p-3 pb-1">
              <h3 class="min-h-12 line-clamp-2 font-semibold">
                {{ clip.title }}
              </h3>
              <p class="mt-1 text-xs text-slate-500">
                {{ new Date(clip.created_at).toLocaleDateString() }} ·
                {{ Math.round(clip.duration) }} сек.
              </p>
            </div>
          </button>
          <div class="flex items-center justify-between gap-2 p-3 pt-2">
            <a
              :href="clip.url"
              target="_blank"
              rel="noopener noreferrer"
              class="inline-flex items-center gap-1 text-sm text-violet-700"
            >
              <ExternalLink class="size-4" />Twitch
            </a>
            <ClipImportButton
              :saved="clip.saved"
              :busy="busyClipId === clip.id"
              @save="emit('save', clip)"
            />
          </div>
        </article>
      </SwiperSlide>
    </Swiper>
  </div>
</template>

<style scoped>
.subscription-swiper :deep(.swiper-button-prev),
.subscription-swiper :deep(.swiper-button-next) {
  width: 2.5rem;
  height: 2.5rem;
  border-radius: 9999px;
  background: rgb(255 255 255 / 0.94);
  color: #6d28d9;
  box-shadow: 0 4px 14px rgb(15 23 42 / 0.18);
}
.subscription-swiper :deep(.swiper-button-prev::after),
.subscription-swiper :deep(.swiper-button-next::after) {
  font-size: 1rem;
  font-weight: 800;
}
@media (max-width: 639px) {
  .subscription-swiper :deep(.swiper-button-prev),
  .subscription-swiper :deep(.swiper-button-next) {
    display: none;
  }
}
</style>

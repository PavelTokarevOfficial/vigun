<script setup lang="ts">
import { Check, ChevronDown, ChevronUp, ExternalLink } from '@lucide/vue'
import { ref } from 'vue'
import type { SubscriptionFeed, TwitchClip } from '@/entities/clip/model/types'
import ClipImportButton from '@/features/clip-import/ClipImportButton.vue'

const props = defineProps<{
  feed: SubscriptionFeed
  busyClipId: string
}>()
const emit = defineEmits<{
  open: [index: number]
  save: [clip: TwitchClip]
}>()

const expanded = ref(false)

function toggleVisibilityClass() {
  const count = props.feed.clips.length
  if (count <= 1) return 'subscription-toggle--hidden'
  if (count === 2) return 'subscription-toggle--hide-from-sm'
  if (count === 3) return 'subscription-toggle--hide-from-lg'
  if (count === 4) return 'subscription-toggle--hide-from-xl'
  if (count === 5) return 'subscription-toggle--hide-from-2xl'
  return ''
}
</script>

<template>
  <div>
    <div
      class="subscription-clip-grid grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 2xl:grid-cols-5"
      :class="{ 'subscription-clip-grid--collapsed': !expanded }"
    >
      <article
        v-for="(clip, index) in feed.clips"
        :key="clip.id"
        class="flex h-full min-w-0 flex-col justify-between overflow-hidden rounded-xl border border-slate-200 bg-slate-50 transition hover:border-violet-300 hover:shadow-md"
      >
        <button
          type="button"
          class="block text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-violet-600"
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
          <div class="p-2 pb-1">
            <h3 class="line-clamp-2 font-semibold">
              {{ clip.title }}
            </h3>
            <p class="mt-1 text-xs text-slate-500">
              {{ new Date(clip.created_at).toLocaleDateString() }} ·
              {{ Math.round(clip.duration) }} сек.
            </p>
          </div>
        </button>
        <div class="flex items-center justify-between gap-2 p-2">
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
    </div>

    <button
      type="button"
      class="mx-auto mt-3 flex min-h-10 items-center justify-center gap-2 rounded-xl px-4 text-sm font-semibold text-violet-700 transition hover:bg-violet-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-violet-600"
      :class="toggleVisibilityClass()"
      :aria-expanded="expanded"
      @click="expanded = !expanded"
    >
      <ChevronUp v-if="expanded" class="size-4" />
      <ChevronDown v-else class="size-4" />
      {{ expanded ? 'Скрыть' : 'Показать все' }}
    </button>
  </div>
</template>

<style scoped>
.subscription-clip-grid--collapsed > article {
  display: none;
}

.subscription-clip-grid--collapsed > article:nth-child(-n + 1) {
  display: flex;
}

@media (min-width: 640px) {
  .subscription-clip-grid--collapsed > article:nth-child(-n + 2) {
    display: flex;
  }

  .subscription-toggle--hide-from-sm {
    display: none;
  }
}

@media (min-width: 1024px) {
  .subscription-clip-grid--collapsed > article:nth-child(-n + 3) {
    display: flex;
  }

  .subscription-toggle--hide-from-lg {
    display: none;
  }
}

@media (min-width: 1280px) {
  .subscription-clip-grid--collapsed > article:nth-child(-n + 4) {
    display: flex;
  }

  .subscription-toggle--hide-from-xl {
    display: none;
  }
}

@media (min-width: 1536px) {
  .subscription-clip-grid--collapsed > article:nth-child(-n + 5) {
    display: flex;
  }

  .subscription-toggle--hide-from-2xl {
    display: none;
  }
}

.subscription-toggle--hidden {
  display: none;
}
</style>

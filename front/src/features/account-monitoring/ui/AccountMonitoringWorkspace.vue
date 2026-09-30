<script setup lang="ts">
import { ExternalLink, Play, X } from '@lucide/vue'
import { ref } from 'vue'
import { useAccountMonitoring } from '@/features/account-monitoring/model/useAccountMonitoring'
import AppButton from '@/shared/ui/AppButton.vue'
import EmptyState from '@/shared/ui/EmptyState.vue'
import ErrorState from '@/shared/ui/ErrorState.vue'

type Platform = 'instagram' | 'youtube' | 'tiktok'

const activePlatform = ref<Platform>('instagram')
const model = useAccountMonitoring()
const tabs: { id: Platform; label: string }[] = [
  { id: 'instagram', label: 'Instagram' },
  { id: 'youtube', label: 'YouTube' },
  { id: 'tiktok', label: 'TikTok' },
]

function formatDate(value: string) {
  return value
    ? new Date(value).toLocaleString([], {
        dateStyle: 'short',
        timeStyle: 'short',
      })
    : ''
}
</script>

<template>
  <section>
    <div class="mb-6">
      <h1 class="text-3xl font-semibold sm:text-4xl">Мониторинг аккаунтов</h1>
      <p class="mt-2 text-sm text-slate-500">
        Опубликованные видео подключённых социальных аккаунтов.
      </p>
    </div>

    <div class="mb-6 flex gap-1 border-b border-slate-200">
      <button
        v-for="tab in tabs"
        :key="tab.id"
        type="button"
        class="border-b-2 px-4 py-3 text-sm font-semibold transition"
        :class="activePlatform === tab.id ? 'border-violet-600 text-violet-700' : 'border-transparent text-slate-500 hover:text-slate-900'"
        @click="activePlatform = tab.id"
      >
        {{ tab.label }}
      </button>
    </div>

    <template v-if="activePlatform === 'instagram'">
      <ErrorState v-if="model.error.value" :message="model.error.value" />
      <p v-if="model.loading.value" class="py-8 text-center text-slate-500">
        Загружаем Reels…
      </p>
      <EmptyState
        v-else-if="!model.groups.value.length"
        message="Instagram-аккаунты ещё не добавлены."
      >
        <RouterLink
          to="/accounts"
          class="mt-3 inline-block font-medium text-violet-700"
        >
          Открыть менеджер аккаунтов
        </RouterLink>
      </EmptyState>

      <section
        v-for="group in model.groups.value"
        v-else
        :key="group.account.id"
        class="mb-10"
      >
        <div class="mb-4 flex items-center justify-between gap-3">
          <div class="min-w-0">
            <h2 class="truncate text-xl font-semibold">
              @{{ group.account.verifiedUsername || group.account.nickname }}
            </h2>
            <p class="text-sm text-slate-500">
              Reels · {{ group.items.length }}
            </p>
          </div>
          <AppButton
            variant="secondary"
            :disabled="group.loading"
            @click="model.loadGroup(group)"
          >
            {{ group.loading ? 'Обновляем…' : 'Обновить' }}
          </AppButton>
        </div>

        <ErrorState v-if="group.error" :message="group.error" />
        <p v-if="group.loading" class="py-8 text-center text-slate-500">
          Загружаем видео…
        </p>
        <EmptyState
          v-else-if="!group.items.length"
          message="У этого аккаунта Reels пока не найдены."
        />
        <div
          v-else
          class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4"
        >
          <article
            v-for="reel in group.items"
            :key="reel.id"
            class="min-w-0 overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm"
          >
            <button
              type="button"
              class="group relative block aspect-[9/16] w-full overflow-hidden bg-slate-950"
              :disabled="!reel.mediaUrl"
              :aria-label="`Смотреть Reel ${reel.caption || reel.id}`"
              @click="model.preview.value = reel"
            >
              <img
                v-if="reel.thumbnailUrl"
                :src="reel.thumbnailUrl"
                alt=""
                class="size-full object-cover transition group-hover:scale-[1.02]"
              >
              <video
                v-else-if="reel.mediaUrl"
                :src="reel.mediaUrl"
                muted
                preload="metadata"
                class="size-full object-cover"
              />
              <div
                class="absolute inset-0 grid place-content-center bg-slate-950/15"
              >
                <span
                  class="grid size-12 place-content-center rounded-full bg-violet-600 text-white shadow-lg"
                >
                  <Play class="size-5 fill-current" />
                </span>
              </div>
            </button>
            <div class="p-3">
              <p class="line-clamp-2 min-h-10 text-sm font-medium">
                {{ reel.caption || 'Без подписи' }}
              </p>
              <div
                class="mt-3 flex items-center justify-between gap-3 text-xs text-slate-500"
              >
                <span>{{ formatDate(reel.timestamp) }}</span>
                <a
                  v-if="reel.permalink"
                  :href="reel.permalink"
                  target="_blank"
                  rel="noopener noreferrer"
                  class="inline-flex items-center gap-1 text-violet-700 hover:text-violet-900"
                  @click.stop
                >
                  Instagram <ExternalLink class="size-3.5" />
                </a>
              </div>
            </div>
          </article>
        </div>
        <div v-if="group.nextCursor" class="mt-5 flex justify-center">
          <AppButton
            variant="secondary"
            :disabled="group.loadingMore"
            @click="model.loadGroup(group, true)"
          >
            {{ group.loadingMore ? 'Загружаем…' : 'Показать ещё' }}
          </AppButton>
        </div>
      </section>
    </template>

    <EmptyState
      v-else
      :message="`${tabs.find((tab) => tab.id === activePlatform)?.label}: мониторинг пока не подключён.`"
    />

    <div
      v-if="model.preview.value"
      class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/80 p-4"
      role="dialog"
      aria-modal="true"
      aria-label="Просмотр Instagram Reel"
      tabindex="-1"
      @click.self="model.preview.value = null"
      @keydown.esc="model.preview.value = null"
    >
      <div
        class="relative flex max-h-[94vh] max-w-full flex-col overflow-hidden rounded-2xl bg-black shadow-2xl"
      >
        <button
          type="button"
          class="absolute top-3 right-3 z-10 grid size-9 place-content-center rounded-full bg-black/60 text-white hover:bg-black/80"
          aria-label="Закрыть видео"
          @click="model.preview.value = null"
        >
          <X class="size-5" />
        </button>
        <video
          :key="model.preview.value.id"
          :src="model.preview.value.mediaUrl"
          controls
          autoplay
          class="max-h-[94vh] max-w-[min(100vw-2rem,720px)]"
        />
      </div>
    </div>
  </section>
</template>

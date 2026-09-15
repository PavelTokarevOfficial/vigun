<script setup lang="ts">
import { Play } from '@lucide/vue'
import { inject, onBeforeUnmount, onMounted } from 'vue'
import ClipImportButton from '@/features/clip-import/ClipImportButton.vue'
import ClipSwipePreview from '@/features/clip-preview/ui/ClipSwipePreview.vue'
import { subscriptionSyncControlsKey } from '@/features/clip-subscriptions/model/syncControls'
import { useClipSubscriptions } from '@/features/clip-subscriptions/model/useClipSubscriptions'
import EmptyState from '@/shared/ui/EmptyState.vue'
import ErrorState from '@/shared/ui/ErrorState.vue'
import SubscriptionClipGrid from './SubscriptionClipGrid.vue'

const syncControls = inject(subscriptionSyncControlsKey)
if (!syncControls) throw new Error('Subscription sync controls are unavailable')

const model = useClipSubscriptions(syncControls)
const {
  busyClipId,
  error,
  feeds,
  loading,
  notice,
  previewFeed,
  previewInitialIndex,
  previewOpen,
} = model
const unregisterSync = syncControls.registerSync(() => void model.sync())

function formatSynced(value: string | null) {
  return value
    ? new Date(value).toLocaleString([], {
        dateStyle: 'short',
        timeStyle: 'short',
      })
    : 'ещё не запускалась'
}

onMounted(() => void model.load())
onBeforeUnmount(() => {
  unregisterSync()
  model.dispose()
})
</script>

<template>
  <div>
    <ErrorState v-if="error" :message="error" />
    <p v-if="notice" class="mb-4 text-sm text-emerald-700">{{ notice }}</p>
    <p v-if="loading" class="py-8 text-center text-slate-500">
      Загружаем подписки…
    </p>

    <EmptyState v-else-if="feeds.length === 0" message="Подписок пока нет.">
      <RouterLink
        to="/streamers"
        class="mt-3 inline-block font-medium text-violet-700"
      >
        Выбрать стримеров
      </RouterLink>
    </EmptyState>

    <section
      v-for="feed in feeds"
      v-else
      :key="feed.streamerId"
      class="mb-8 overflow-hidden rounded-2xl border border-slate-200 bg-white p-4 pb-2 shadow-sm"
    >
      <div
        class="mb-4 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between"
      >
        <div class="min-w-0">
          <h2 class="truncate text-xl font-semibold">{{ feed.displayName }}</h2>
          <p class="truncate text-sm text-slate-500">
            @{{ feed.twitchLogin }} · синхронизация:
            {{ formatSynced(feed.lastSyncedAt) }}
          </p>
        </div>
        <button
          type="button"
          class="inline-flex min-h-10 shrink-0 items-center justify-center gap-2 rounded-xl bg-slate-900 px-4 text-sm font-semibold text-white disabled:cursor-not-allowed disabled:opacity-40"
          :disabled="feed.clips.length === 0"
          @click="model.openPreview(feed)"
        >
          <Play class="size-4" />Смотреть клипы
        </button>
      </div>

      <SubscriptionClipGrid
        v-if="feed.clips.length"
        :feed="feed"
        :busy-clip-id="busyClipId"
        @open="model.openPreview(feed, $event)"
        @save="model.save(feed, $event)"
      />
      <p
        v-else
        class="rounded-xl border border-dashed border-slate-300 p-5 text-sm text-slate-500"
      >
        Свежих клипов пока нет. Запустите синхронизацию.
      </p>
    </section>

    <ClipSwipePreview
      v-if="previewFeed"
      :open="previewOpen"
      :clips="previewFeed.clips"
      :streamer-name="previewFeed.displayName"
      :initial-index="previewInitialIndex"
      @close="model.closePreview"
      @view="model.markViewed"
    >
      <template #action="{ clip }">
        <ClipImportButton
          :saved="clip.saved"
          :busy="busyClipId === clip.id"
          @save="model.saveFromPreview(clip)"
        />
      </template>
    </ClipSwipePreview>
  </div>
</template>

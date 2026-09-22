<script setup lang="ts">
import { BellPlus, Check, Play } from '@lucide/vue'
import ClipImportButton from '@/features/clip-import/ClipImportButton.vue'
import ClipSwipePreview from '@/features/clip-preview/ui/ClipSwipePreview.vue'
import { useClipSearch } from '@/features/clip-search/model/useClipSearch'
import StreamerSearchInput from '@/features/clip-search/ui/StreamerSearchInput.vue'
import { dateInputValue } from '@/shared/lib/dateRange'
import AppButton from '@/shared/ui/AppButton.vue'
import AppDateRangePicker from '@/shared/ui/AppDateRangePicker.vue'
import ErrorState from '@/shared/ui/ErrorState.vue'

const search = useClipSearch()

function updateDateRange(range: { start: string; end: string }) {
  search.startedAt.value = range.start
  search.endedAt.value = range.end
}
</script>

<template>
  <section>
    <h1 class="text-2xl font-semibold sm:text-3xl">Поиск клипов</h1>
    <p class="mt-1 text-slate-600">
      Найдите клипы по нику стримера и добавьте подходящие в избранное.
    </p>
    <div class="mt-5 flex flex-wrap gap-3">
      <StreamerSearchInput
        v-model="search.query.value"
        :busy="search.resolving.value || search.loading.value"
        @search="search.search"
      />
      <div class="block min-w-72 text-sm">
        <span class="mb-1 block">Период</span>
        <AppDateRangePicker
          :model-value="{
            start: search.startedAt.value,
            end: search.endedAt.value,
          }"
          :max="dateInputValue()"
          @update:model-value="updateDateRange"
        />
      </div>
      <div v-if="search.selected.value" class="flex items-end">
        <AppButton
          :disabled="search.loading.value || search.clips.value.length === 0"
          @click="search.openPreview()"
        >
          <Play class="mr-1 inline size-4" />
          {{ search.loading.value ? 'Загружаем…' : 'Смотреть клипы' }}
        </AppButton>
      </div>
      <div v-if="search.foundStreamer.value" class="flex items-end">
        <AppButton
          v-if="!search.foundStreamer.value.subscribed"
          variant="secondary"
          :disabled="search.subscribing.value"
          @click="search.subscribe"
        >
          <BellPlus class="mr-1 inline size-4" />
          {{ search.subscribing.value ? 'Подписываем…' : 'Подписаться' }}
        </AppButton>
        <span
          v-else
          class="inline-flex min-h-10 items-center gap-2 rounded-xl bg-emerald-50 px-3 py-2 text-sm font-medium text-emerald-700"
        >
          <Check class="size-4" />
          Вы подписаны
        </span>
      </div>
    </div>
    <p v-if="search.notice.value" class="mt-3 text-sm text-emerald-700">
      {{ search.notice.value }}
    </p>
    <ErrorState v-if="search.error.value" :message="search.error.value" />
    <div class="mt-6 grid gap-4 md:grid-cols-3">
      <article
        v-for="(clip, index) in search.clips.value"
        :key="clip.id"
        class="overflow-hidden rounded-xl bg-white shadow-sm"
      >
        <button
          type="button"
          class="block w-full text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-violet-600"
          :aria-label="`Смотреть клип: ${clip.title}`"
          @click="search.openPreview(index)"
        >
          <img
            :src="clip.thumbnail_url"
            :alt="`Превью клипа: ${clip.title}`"
            class="aspect-video w-full object-cover"
          >
          <div class="px-3 pt-3">
            <b>{{ clip.title }}</b>
            <p class="my-2 text-sm text-slate-600">
              {{ search.streamerName.value }} ·
              {{ new Date(clip.created_at).toLocaleDateString() }} ·
              {{ Math.round(clip.duration) }} сек.
            </p>
          </div>
        </button>
        <div class="flex justify-between px-3 pb-3">
          <a
            :href="clip.url"
            target="_blank"
            rel="noopener noreferrer"
            class="text-violet-700"
          >
            Twitch
          </a>
          <ClipImportButton
            :saved="clip.saved"
            :busy="search.busyClipId.value === clip.id"
            @save="search.save(clip)"
          />
        </div>
      </article>
    </div>
    <ClipSwipePreview
      :open="search.previewOpen.value"
      :clips="search.clips.value"
      :streamer-name="search.streamerName.value"
      :initial-index="search.previewInitialIndex.value"
      @close="search.previewOpen.value = false"
    >
      <template #action="{ clip }">
        <ClipImportButton
          :saved="clip.saved"
          :busy="search.busyClipId.value === clip.id"
          @save="search.save(clip)"
        />
      </template>
    </ClipSwipePreview>
  </section>
</template>

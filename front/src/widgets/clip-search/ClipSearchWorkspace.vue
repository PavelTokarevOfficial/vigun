<script setup lang="ts">
import { Play } from '@lucide/vue'
import ClipImportButton from '@/features/clip-import/ClipImportButton.vue'
import ClipSwipePreview from '@/features/clip-preview/ui/ClipSwipePreview.vue'
import { useClipSearch } from '@/features/clip-search/model/useClipSearch'
import AppButton from '@/shared/ui/AppButton.vue'
import ErrorState from '@/shared/ui/ErrorState.vue'

const search = useClipSearch()
</script>

<template>
  <section>
    <h1 class="text-2xl font-semibold sm:text-3xl">Поиск клипов</h1>
    <p class="mt-1 text-slate-600">
      Найдите клипы по нику стримера и добавьте подходящие в избранное.
    </p>
    <div class="mt-5 flex flex-wrap gap-3">
      <label class="text-sm"
        >Стример<select v-model="search.selected.value" class="mt-1 block">
          <option value="">Выберите стримера</option>
          <option
            v-for="streamer in search.streamers.value"
            :key="streamer.id"
            :value="streamer.id"
          >
            {{ streamer.displayName }}
          </option>
        </select></label
      >
      <label class="text-sm"
        >С<input
          v-model="search.startedAt.value"
          type="date"
          :max="search.endedAt.value"
          class="mt-1 block"
        ></label
      ><label class="text-sm"
        >По<input
          v-model="search.endedAt.value"
          type="date"
          :min="search.startedAt.value"
          :max="search.dateInputValue()"
          class="mt-1 block"
        ></label
      >
      <div v-if="search.selected.value" class="flex items-end">
        <AppButton
          :disabled="search.loading.value || search.clips.value.length === 0"
          @click="search.openPreview()"
        >
          <Play class="mr-1 inline size-4" />
          {{ search.loading.value ? 'Загружаем…' : 'Смотреть клипы' }}
        </AppButton>
      </div>
    </div>
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

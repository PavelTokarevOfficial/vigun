<script setup lang="ts">
import { X } from '@lucide/vue'
import type { PipelineWorkspaceModel } from '../model/usePipelineWorkspace'

const props = defineProps<{ workspace: PipelineWorkspaceModel }>()
const { previewVideo } = props.workspace
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
      class="w-full max-w-4xl overflow-hidden rounded-2xl bg-white shadow-2xl"
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
      <div class="flex justify-center bg-slate-950 p-4">
        <video
          v-if="previewVideo.mode === 'video'"
          :key="previewVideo.key"
          :src="previewVideo.url"
          controls
          autoplay
          class="max-h-[75vh] max-w-full"
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
    </section>
  </div>
</template>

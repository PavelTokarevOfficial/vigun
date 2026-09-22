<script setup lang="ts">
import { Play } from '@lucide/vue'
import AppButton from '@/shared/ui/AppButton.vue'
import type { PipelineWorkspaceModel } from '../model/usePipelineWorkspace'

const props = defineProps<{ workspace: PipelineWorkspaceModel }>()
const {
  trainSelectionMode,
  selectedTrainClipIDs,
  startTrain,
  trainJobs,
  trainJobStatusClass,
  trainJobStatus,
  videoForJob,
  openTrainJobVideo,
} = props.workspace
</script>

<template>
  <div class="mt-6 rounded-xl border border-slate-200 bg-white p-3">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <h2 class="font-semibold">Доска монтажа</h2>
        <p class="text-sm text-slate-500">
          Скачайте клип, подготовьте фрагмент и отправьте его на рендер отдельно
          или в общей сборке.
        </p>
      </div>
      <div class="flex flex-wrap gap-2">
        <AppButton
          variant="secondary"
          @click="trainSelectionMode = !trainSelectionMode; selectedTrainClipIDs = new Set()"
        >
          {{ trainSelectionMode ? 'Отменить паровозик' : 'Паровозик' }}
        </AppButton>
        <AppButton
          v-if="trainSelectionMode"
          :disabled="selectedTrainClipIDs.size < 2"
          @click="startTrain"
        >
          Собрать выбранные · {{ selectedTrainClipIDs.size }}
        </AppButton>
      </div>
    </div>

    <div class="mt-3 border-t border-slate-200 pt-3">
      <div class="flex items-center justify-between gap-3">
        <h3 class="text-sm font-semibold">Рендеры паровозика</h3>
        <span class="text-xs text-slate-500"
          >Последние {{ trainJobs.length }}</span
        >
      </div>
      <p v-if="!trainJobs.length" class="mt-2 text-sm text-slate-500">
        Сборок пока нет. Выберите минимум два готовых фрагмента и нажмите
        «Собрать выбранные».
      </p>
      <ul v-else class="mt-2 grid gap-2 lg:grid-cols-2 xl:grid-cols-3">
        <li
          v-for="job in trainJobs"
          :key="job.id"
          class="rounded-lg border border-slate-200 p-3"
        >
          <div class="flex items-start justify-between gap-2">
            <div class="min-w-0">
              <p class="truncate text-sm font-medium">
                Паровозик · {{ job.fragmentCount }} фрагм.
              </p>
              <p class="truncate text-xs text-slate-500">
                {{ job.templateName || 'Шаблон' }} ·
                {{ new Date(job.createdAt).toLocaleString() }}
              </p>
            </div>
            <span
              class="shrink-0 rounded-full px-2 py-1 text-xs font-medium"
              :class="trainJobStatusClass(job)"
            >
              {{ trainJobStatus(job) }}
            </span>
          </div>
          <div
            v-if="job.status === 'pending' || job.status === 'running'"
            class="mt-2 h-1.5 overflow-hidden rounded-full bg-slate-100"
          >
            <div
              class="h-full rounded-full bg-violet-600 transition-all"
              :style="{ width: `${Math.max(job.status === 'pending' ? 4 : job.progress, 4)}%` }"
            />
          </div>
          <p
            v-if="job.status === 'failed' && job.error"
            class="mt-2 line-clamp-2 text-xs text-red-600"
          >
            {{ job.error.split('\n')[0] }}
          </p>
          <AppButton
            v-if="job.status === 'completed' && videoForJob(job)"
            class="mt-2 w-full"
            variant="secondary"
            @click="openTrainJobVideo(job)"
          >
            <Play class="mr-1 inline size-4" />Посмотреть результат
          </AppButton>
        </li>
      </ul>
    </div>
  </div>
</template>

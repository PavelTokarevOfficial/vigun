<script setup lang="ts">
import { ArrowLeft, Redo2, Undo2, X } from '@lucide/vue'
import LayerPanel from '@/features/template-editor/ui/LayerPanel.vue'
import PropertiesPanel from '@/features/template-editor/ui/PropertiesPanel.vue'
import TemplateCanvas from '@/features/template-editor/ui/TemplateCanvas.vue'
import VideoTimeline from '@/features/template-editor/ui/VideoTimeline.vue'
import AppButton from '@/shared/ui/AppButton.vue'
import type { PipelineWorkspaceModel } from '../model/usePipelineWorkspace'

const props = defineProps<{ workspace: PipelineWorkspaceModel }>()
const {
  addEditorLayer,
  assets,
  availableTemplates,
  busy,
  chooseTemplate,
  closeProcessDialog,
  folders,
  processClip,
  processClipID,
  processClipIDs,
  processStage,
  removeEditorLayer,
  renderEditor,
  selectedTemplate,
  selectedTimelineSegmentID,
  selectEditorLayer,
  selectTimelineSegment,
  sendToRender,
  sourceURL,
  sourceURLs,
  templatesLoading,
  timelineTime,
  updateEditorLayerTrack,
} = props.workspace
</script>

<template>
  <div
    v-if="processClipID"
    class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/55 p-4"
    role="dialog"
    aria-modal="true"
    aria-labelledby="template-dialog-title"
  >
    <section
      class="max-h-[94vh] w-full overflow-auto rounded-2xl bg-slate-50 shadow-2xl"
      :class="processStage === 'edit' ? 'max-w-[1500px]' : 'max-w-3xl'"
    >
      <header
        class="sticky top-0 z-10 flex items-start justify-between gap-4 border-b border-slate-200 bg-white px-5 py-4"
      >
        <div>
          <h2 id="template-dialog-title" class="text-lg font-semibold">
            {{
              processStage === 'choose' ? 'Выберите шаблон' : 'Подготовьте видео к рендеру'
            }}
          </h2>
          <p class="mt-1 text-sm text-slate-600">
            <template v-if="processStage === 'choose'">
              {{
                processClipIDs.length > 1
                    ? 'Показаны только шаблоны с включённой галочкой «Паровозик».'
                    : 'Шаблон по умолчанию отмечен первым, но можно выбрать любой.'
              }}
            </template>
            <template v-else>
              {{
                processClipIDs.length > 1 ? `Паровозик из ${processClipIDs.length} фрагментов` : processClip?.title
              }}
              · {{ selectedTemplate?.name }}. Изменения применятся только к
              этому рендеру.
            </template>
          </p>
        </div>
        <button
          type="button"
          class="rounded-lg p-2 text-slate-600 hover:bg-slate-100"
          aria-label="Закрыть редактор рендера"
          @click="closeProcessDialog"
        >
          <X class="size-5" />
        </button>
      </header>
      <p v-if="templatesLoading" class="p-5 text-slate-500">
        Загружаем шаблоны…
      </p>
      <div
        v-else-if="processStage === 'choose'"
        class="grid gap-3 p-5 sm:grid-cols-2"
      >
        <div
          v-if="!availableTemplates.length"
          class="rounded-xl border border-dashed border-slate-300 bg-white p-5 text-sm text-slate-600 sm:col-span-2"
        >
          Нет шаблонов для «Паровозика». Откройте редактор шаблона, включите
          галочку «Паровозик» и сохраните шаблон.
        </div>
        <button
          v-for="template in availableTemplates"
          :key="template.id"
          type="button"
          class="relative overflow-hidden rounded-xl border bg-white text-left hover:border-violet-500 hover:ring-2 hover:ring-violet-100"
          :class="template.isDefault ? 'border-violet-400' : 'border-slate-200'"
          @click="chooseTemplate(template)"
        >
          <span
            v-if="template.isDefault"
            class="absolute right-2 top-2 z-10 rounded-full bg-violet-600 px-2 py-1 text-xs font-medium text-white"
            >По умолчанию</span
          >
          <img
            v-if="template.previewUrl"
            :src="template.previewUrl"
            :alt="`Превью шаблона ${template.name}`"
            class="aspect-video w-full object-cover"
          >
          <div
            v-else
            class="flex aspect-video items-center justify-center bg-gradient-to-br from-violet-950 to-slate-900 text-sm text-violet-100"
          >
            9:16 · {{ template.config.layers.length }} слоёв
          </div>
          <div class="p-3">
            <b>{{ template.name }}</b>
            <p class="mt-1 text-sm text-slate-600">
              {{ template.description || 'Без описания' }}
            </p>
            <p class="mt-3 text-sm font-medium text-violet-700">
              Выбрать и настроить →
            </p>
          </div>
        </button>
      </div>

      <template v-else>
        <div
          class="flex flex-wrap items-center justify-between gap-3 border-b border-slate-200 bg-white px-5 py-3"
        >
          <button
            type="button"
            class="inline-flex items-center gap-2 text-sm font-medium text-violet-700"
            @click="processStage = 'choose'"
          >
            <ArrowLeft class="size-4" />Другой шаблон
          </button>
          <div class="flex gap-2">
            <AppButton
              variant="secondary"
              :disabled="!renderEditor.canUndo.value"
              @click="renderEditor.undo"
            >
              <Undo2 class="mr-1 inline size-4" />Назад
            </AppButton>
            <AppButton
              variant="secondary"
              :disabled="!renderEditor.canRedo.value"
              @click="renderEditor.redo"
            >
              <Redo2 class="mr-1 inline size-4" />Вперёд
            </AppButton>
          </div>
        </div>

        <div
          class="grid gap-4 p-4 xl:grid-cols-[250px_minmax(360px,1fr)_300px]"
        >
          <LayerPanel
            :layers="renderEditor.draft.value.layers"
            :selected-layer-id="renderEditor.selectedLayerID.value"
            @select="selectEditorLayer"
            @add="addEditorLayer"
            @update="updateEditorLayerTrack"
            @duplicate="renderEditor.duplicateLayer"
            @remove="removeEditorLayer"
            @move="renderEditor.moveLayer"
          />
          <TemplateCanvas
            :config="renderEditor.draft.value"
            :selected-layer-id="renderEditor.selectedLayerID.value"
            :assets="assets"
            :source-url="sourceURL"
            :source-urls="sourceURLs"
            :timeline-time="timelineTime"
            :thumbnail-url="processClip?.thumbnailUrl"
            :streamer-name="processClip?.streamerName"
            @select="selectEditorLayer"
            @update="updateEditorLayerTrack"
          />
          <PropertiesPanel
            :layer="renderEditor.selectedLayer.value"
            :assets="assets"
            :folders="folders"
            @update="renderEditor.selectedLayer.value && updateEditorLayerTrack(renderEditor.selectedLayer.value.id, $event)"
          />
        </div>

        <div class="px-4 pb-4">
          <VideoTimeline
            v-if="renderEditor.draft.value.timeline && processClip"
            :segments="renderEditor.draft.value.timeline.segments"
            :layers="renderEditor.draft.value.layers"
            :assets="assets"
            :folders="folders"
            :source-duration="Math.max(0.1, processClip.duration)"
            :selected-layer-id="renderEditor.selectedLayerID.value"
            :selected-segment-id="selectedTimelineSegmentID"
            @update-segments="renderEditor.updateConfig({ timeline: { segments: $event } })"
            @update-layers="renderEditor.updateConfig({ layers: $event })"
            @update-layer="renderEditor.updateLayer($event.id, $event.patch)"
            @select-layer="selectEditorLayer"
            @select-segment="selectTimelineSegment"
            @update-time="timelineTime = $event"
          />
        </div>

        <footer
          class="sticky bottom-0 z-10 flex flex-wrap items-center justify-between gap-3 border-t border-slate-200 bg-white px-5 py-4"
        >
          <p class="text-sm text-slate-500">
            Исходный клип останется в «Скачанных» после завершения.
          </p>
          <div class="flex gap-2">
            <AppButton variant="secondary" @click="closeProcessDialog"
              >Отмена</AppButton
            >
            <AppButton :disabled="busy === processClipID" @click="sendToRender">
              {{
                busy === processClipID ? 'Отправляем…' : 'Отправить на рендер'
              }}
            </AppButton>
          </div>
        </footer>
      </template>
    </section>
  </div>
</template>

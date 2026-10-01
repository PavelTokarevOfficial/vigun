<script setup lang="ts">
import { Check, Download, HardDrive, Trash2 } from '@lucide/vue'
import AppButton from '@/shared/ui/AppButton.vue'
import ErrorState from '@/shared/ui/ErrorState.vue'
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from '@/shared/ui/shadcn/accordion'
import { useWhisperSettings } from '../model/useWhisperSettings'

const model = useWhisperSettings()
const presets = [
  {
    id: 'fast',
    name: 'Fast',
    description: 'Минимальная задержка, меньше вариантов декодирования.',
  },
  {
    id: 'balanced',
    name: 'Balanced',
    description: 'Текущее поведение: короткие сегменты и разделение по словам.',
  },
  {
    id: 'maximum',
    name: 'Maximum quality',
    description: 'Расширенный beam search с приоритетом качества.',
  },
  {
    id: 'subtitles',
    name: 'Subtitles',
    description: 'Фразы до 42 символов и разделение по границам слов.',
  },
]
const formatSize = (bytes: number) =>
  bytes >= 1024 ** 3
    ? `~${(bytes / 1024 ** 3).toFixed(1)} GB`
    : `~${Math.round(bytes / 1024 ** 2)} MB`
const progress = (item: {
  status: { downloadedBytes: number; totalBytes: number }
}) =>
  item.status.totalBytes
    ? Math.min(
        100,
        Math.round(
          (item.status.downloadedBytes / item.status.totalBytes) * 100,
        ),
      )
    : 0
function setSplitOnWord(event: Event) {
  const value = (event.target as HTMLSelectElement).value
  model.settings.value.splitOnWord = value === 'preset' ? null : value === 'yes'
}
</script>

<template>
  <section>
    <ErrorState v-if="model.error.value" :message="model.error.value" />
    <p v-if="model.loading.value" class="py-12 text-center text-slate-500">
      Загружаем модели…
    </p>
    <template v-else>
      <section id="whisper-models" class="scroll-mt-6">
        <div class="mb-5">
          <h2 class="text-2xl font-semibold">Модели Whisper</h2>
          <p class="mt-1 text-sm text-slate-500">
            Модели скачиваются только по вашему запросу и сохраняются между
            перезапусками.
          </p>
        </div>
        <Accordion
          type="single"
          collapsible
          :default-value="model.settings.value.activeModel"
          class="overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm"
        >
          <AccordionItem
            v-for="item in model.models.value"
            :key="item.id"
            :value="item.id"
            class="border-slate-200 px-4 transition-colors duration-200 data-[state=open]:bg-violet-50"
          >
            <AccordionTrigger
              class="gap-3 py-3 hover:no-underline"
              :class="item.selected?'bg-violet-300 -mx-4 px-4 rounded-none':''"
            >
              <div class="min-w-0 flex-1 sm:flex sm:items-center sm:gap-3">
                <h3 class="truncate font-semibold">{{ item.name }}</h3>
                <span
                  class="mt-1 inline-flex rounded-full bg-slate-100 px-2 py-0.5 text-xs font-semibold text-slate-600 sm:mt-0"
                  >{{
                    item.type
                  }}</span
                >
              </div>
              <span class="hidden text-sm text-slate-500 sm:block">{{
                formatSize(item.sizeBytes)
              }}</span>
              <span
                v-if="item.selected"
                class="inline-flex items-center gap-1 rounded-full bg-violet-100 px-2 py-1 text-xs font-semibold text-violet-700"
                ><Check class="size-3" />Используется</span
              >
              <span
                v-else-if="item.status.state==='downloading'"
                class="text-xs font-semibold text-violet-700"
                >Скачивается · {{ progress(item) }}%</span
              >
              <span
                v-else-if="item.installed"
                class="text-xs font-medium text-emerald-700"
                >Скачана</span
              >
              <span
                v-else-if="item.status.state==='error'"
                class="text-xs font-medium text-red-600"
                >Ошибка</span
              >
              <span v-else class="text-xs text-slate-400">Не скачана</span>
            </AccordionTrigger>
            <AccordionContent class="border-t border-slate-100 pt-3">
              <div
                class="grid gap-4 lg:grid-cols-[minmax(0,1fr)_auto] lg:items-end"
              >
                <div>
                  <dl
                    class="grid grid-cols-2 gap-x-4 gap-y-2 text-sm sm:max-w-2xl sm:grid-cols-4"
                  >
                    <div>
                      <dt class="text-slate-500">Размер</dt>
                      <dd class="mt-0.5">{{ formatSize(item.sizeBytes) }}</dd>
                    </div>
                    <div>
                      <dt class="text-slate-500">Качество</dt>
                      <dd class="mt-0.5 tracking-wide text-amber-500">
                        {{ '★'.repeat(item.quality)
                        }}<span class="text-slate-200">{{
                          '★'.repeat(5-item.quality)
                        }}</span>
                      </dd>
                    </div>
                    <div>
                      <dt class="text-slate-500">Скорость</dt>
                      <dd class="mt-0.5">{{ item.speed }}</dd>
                    </div>
                    <div>
                      <dt class="text-slate-500">Память</dt>
                      <dd class="mt-0.5">{{ item.memory }}</dd>
                    </div>
                  </dl>
                  <p class="mt-3 text-sm leading-6 text-slate-600">
                    {{ item.description }}
                  </p>
                </div>
                <div class="flex gap-2 lg:min-w-48 lg:justify-end">
                  <AppButton
                    v-if="!item.installed"
                    class="flex-1"
                    :disabled="item.status.state==='downloading'||model.busy.value===item.id"
                    @click="model.download(item.id)"
                    ><Download class="mr-2 inline size-4" />
                    {{
                      item.status.state==='downloading'?'Скачивается…':'Скачать'
                    }}</AppButton
                  >
                  <AppButton
                    v-if="item.installed&&!item.selected"
                    class="flex-1"
                    :disabled="model.busy.value===item.id"
                    @click="model.select(item.id)"
                    >Выбрать</AppButton
                  >
                  <AppButton
                    v-if="item.installed&&!item.selected"
                    variant="danger"
                    :disabled="model.busy.value===item.id"
                    aria-label="Удалить модель"
                    @click="model.remove(item.id)"
                    ><Trash2 class="size-4" /></AppButton
                  >
                  <div
                    v-if="item.selected"
                    class="flex items-center gap-2 text-sm font-medium text-violet-700"
                  >
                    <HardDrive class="size-4" />Файл установлен
                  </div>
                </div>
              </div>
              <div v-if="item.status.state==='downloading'" class="mt-4">
                <div class="mb-1 flex justify-between text-xs text-slate-500">
                  <span>Скачивается</span
                  ><span
                    >{{ formatSize(item.status.downloadedBytes) }}
                    / {{ formatSize(item.status.totalBytes) }}</span
                  >
                </div>
                <div class="h-2 overflow-hidden rounded-full bg-slate-100">
                  <div
                    class="h-full rounded-full bg-violet-600 transition-all"
                    :style="{width:`${progress(item)}%`}"
                  ></div>
                </div>
              </div>
              <p
                v-if="item.status.state==='error'"
                class="mt-3 text-sm text-red-600"
              >
                {{ item.status.error }}
              </p>
            </AccordionContent>
          </AccordionItem>
        </Accordion>
      </section>
      <section
        id="transcription-preset"
        class="mt-10 scroll-mt-6 rounded-2xl border border-slate-200 bg-white p-5 shadow-sm"
      >
        <h2 class="text-2xl font-semibold">Preset транскрибации</h2>
        <div class="mt-4 grid gap-3 md:grid-cols-2 xl:grid-cols-4">
          <label
            v-for="preset in presets"
            :key="preset.id"
            class="cursor-pointer rounded-xl border p-4"
            :class="model.settings.value.preset===preset.id?'border-violet-500 bg-violet-50':'border-slate-200'"
            ><input
              v-model="model.settings.value.preset"
              type="radio"
              :value="preset.id"
              class="sr-only"
            ><span class="font-semibold">{{ preset.name }}</span
            ><span class="mt-1 block text-sm text-slate-500">{{
              preset.description
            }}</span></label
          >
        </div>
        <Accordion
          type="single"
          collapsible
          class="mt-6 border-t border-slate-200"
        >
          <AccordionItem value="advanced" class="border-b-0">
            <AccordionTrigger class="py-5 hover:no-underline">
              Advanced settings
            </AccordionTrigger>
            <AccordionContent class="grid gap-4 md:grid-cols-2">
              <label class="text-sm font-medium"
                >Beam size<input
                  v-model.number="model.settings.value.beamSize"
                  type="number"
                  min="1"
                  max="20"
                  placeholder="Из preset"
                  class="mt-1 w-full rounded-lg border border-slate-300 px-3 py-2"
                ></label
              ><label class="text-sm font-medium"
                >Temperature<input
                  v-model.number="model.settings.value.temperature"
                  type="number"
                  min="0"
                  max="1"
                  step="0.05"
                  placeholder="Из preset"
                  class="mt-1 w-full rounded-lg border border-slate-300 px-3 py-2"
                ></label
              ><label class="text-sm font-medium"
                >Максимальная длина сегмента<input
                  v-model.number="model.settings.value.maxSegmentLength"
                  type="number"
                  min="1"
                  max="200"
                  placeholder="Из preset"
                  class="mt-1 w-full rounded-lg border border-slate-300 px-3 py-2"
                ></label
              ><label class="text-sm font-medium"
                >Split on word<select
                  :value="model.settings.value.splitOnWord===null?'preset':model.settings.value.splitOnWord?'yes':'no'"
                  class="mt-1 w-full rounded-lg border border-slate-300 px-3 py-2"
                  @change="setSplitOnWord"
                >
                  <option value="preset">Из preset</option>
                  <option value="yes">Включено</option>
                  <option value="no">Выключено</option>
                </select></label
              ><label class="text-sm font-medium md:col-span-2"
                >Initial prompt<textarea
                  v-model="model.settings.value.initialPrompt"
                  rows="3"
                  placeholder="Имена, термины или контекст для распознавания"
                  class="mt-1 w-full rounded-lg border border-slate-300 px-3 py-2"
                ></textarea></label
              >
            </AccordionContent>
          </AccordionItem>
        </Accordion>
        <div class="mt-5">
          <AppButton
            :disabled="model.busy.value==='settings'"
            @click="model.save"
            >{{
              model.busy.value==='settings'?'Сохраняем…':'Сохранить параметры'
            }}</AppButton
          >
        </div>
      </section>
    </template>
  </section>
</template>

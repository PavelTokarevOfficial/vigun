<script setup lang="ts">
import {
  Captions,
  Copy,
  Eye,
  EyeOff,
  Image,
  Layers,
  Plus,
  SquareDashed,
  Trash2,
  Type,
  Video,
} from '@lucide/vue'
import { computed } from 'vue'
import type { Layer, LayerType } from '@/entities/template/model/types'

const props = defineProps<{ layers: Layer[]; selectedLayerId: string | null }>()
const emit = defineEmits<{
  select: [id: string]
  add: [type: LayerType]
  update: [id: string, patch: Partial<Layer>]
  duplicate: [id: string]
  remove: [id: string]
  move: [id: string, direction: -1 | 1]
}>()

const addable: { type: LayerType; label: string }[] = [
  { type: 'video', label: 'Видео' },
  { type: 'image', label: 'Картинка или GIF' },
  { type: 'subtitles', label: 'Субтитры' },
  { type: 'text', label: 'Текст' },
  { type: 'blur', label: 'Блюр' },
]

const displayedLayers = computed(() => [...props.layers].reverse())

function layerLabel(type: LayerType) {
  return {
    video: 'Видео',
    image: 'Картинка',
    subtitles: 'Субтитры',
    text: 'Текст',
    blur: 'Блюр',
    input_video: 'Видео',
    asset_video: 'Видео',
    gif: 'Картинка',
    audio: 'Аудио',
    color: 'Фон',
  }[type]
}

function addLayer(event: Event) {
  const select = event.target as HTMLSelectElement
  emit('add', select.value as LayerType)
  select.value = ''
}
</script>

<template>
  <aside class="rounded-xl border border-slate-200 bg-white p-3">
    <div class="flex items-center justify-between gap-2">
      <h2 class="flex items-center gap-2 font-semibold">
        <Layers class="size-4" />
        Слои
      </h2>
      <label class="relative">
        <span class="sr-only">Добавить слой</span>
        <Plus class="pointer-events-none absolute left-2 top-2.5 size-4" />
        <select class="max-w-40 py-2 pl-7 pr-2 text-sm" @change="addLayer">
          <option value="" selected disabled>Добавить</option>
          <option v-for="item in addable" :key="item.type" :value="item.type">
            {{ item.label }}
          </option>
        </select>
      </label>
    </div>
    <p class="mt-2 text-xs text-slate-500">
      Верхний слой находится ближе к зрителю.
    </p>
    <ol class="mt-3 space-y-2">
      <li
        v-for="layer in displayedLayers"
        :key="layer.id"
        class="rounded-lg border p-2 transition"
        :class="selectedLayerId === layer.id ? 'border-violet-500 bg-violet-50' : 'border-slate-200'"
      >
        <button
          type="button"
          class="flex w-full items-center justify-between gap-2 text-left"
          @click="emit('select', layer.id)"
        >
          <span class="flex min-w-0 items-center gap-2">
            <Video
              v-if="layerLabel(layer.type) === 'Видео'"
              class="size-4 shrink-0 text-violet-600"
            />
            <Image
              v-else-if="layerLabel(layer.type) === 'Картинка'"
              class="size-4 shrink-0 text-violet-600"
            />
            <Captions
              v-else-if="layer.type === 'subtitles'"
              class="size-4 shrink-0 text-violet-600"
            />
            <Type
              v-else-if="layer.type === 'text'"
              class="size-4 shrink-0 text-violet-600"
            />
            <SquareDashed v-else class="size-4 shrink-0 text-violet-600" />
            <span class="min-w-0 truncate font-medium">{{ layer.name }}</span>
          </span>
          <span class="shrink-0 text-xs text-slate-500">{{
            layerLabel(layer.type)
          }}</span>
        </button>
        <div class="mt-2 flex items-center gap-1 text-xs">
          <button
            type="button"
            class="rounded p-1 hover:bg-white"
            :aria-label="layer.visible ? 'Скрыть слой' : 'Показать слой'"
            @click="emit('update', layer.id, { visible: !layer.visible })"
          >
            <Eye v-if="layer.visible" class="size-4" />
            <EyeOff v-else class="size-4" />
          </button>
          <button
            type="button"
            class="rounded px-1 hover:bg-white"
            :disabled="layers.indexOf(layer) === 0"
            @click="emit('move', layer.id, -1)"
          >
            ↓
          </button>
          <button
            type="button"
            class="rounded px-1 hover:bg-white"
            :disabled="layers.indexOf(layer) === layers.length - 1"
            @click="emit('move', layer.id, 1)"
          >
            ↑
          </button>
          <button
            type="button"
            class="rounded p-1 hover:bg-white"
            aria-label="Создать копию слоя"
            @click="emit('duplicate', layer.id)"
          >
            <Copy class="size-4" />
          </button>
          <button
            type="button"
            class="ml-auto rounded p-1 text-red-600 hover:bg-red-50"
            aria-label="Удалить слой"
            @click="emit('remove', layer.id)"
          >
            <Trash2 class="size-4" />
          </button>
        </div>
      </li>
    </ol>
  </aside>
</template>

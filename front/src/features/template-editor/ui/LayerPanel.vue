<script setup lang="ts">
import {
  Captions,
  Copy,
  Eye,
  EyeOff,
  Film,
  FolderOpen,
  Image,
  Layers,
  SquareDashed,
  Trash2,
  Type,
  Video,
} from '@lucide/vue'
import { computed, ref } from 'vue'
import type { Asset, AssetFolder } from '@/entities/asset/model/types'
import type {
  Layer,
  LayerType,
  TemplateConfig,
} from '@/entities/template/model/types'
import AssetPickerDialog from './AssetPickerDialog.vue'

const props = defineProps<{
  layers: Layer[]
  selectedLayerId: string | null
  showTrainControls?: boolean
  train?: TemplateConfig['train']
  assets?: Asset[]
  folders?: AssetFolder[]
}>()
const emit = defineEmits<{
  select: [id: string]
  add: [type: LayerType]
  update: [id: string, patch: Partial<Layer>]
  duplicate: [id: string]
  remove: [id: string]
  move: [id: string, direction: -1 | 1]
  setTrainMode: [enabled: boolean]
  addTransitionAsset: [id: string]
  removeTransitionAsset: [id: string]
}>()

const transitionPickerOpen = ref(false)
const selectedTransitionAssets = computed(() => {
  const byID = new Map((props.assets ?? []).map((asset) => [asset.id, asset]))
  return (props.train?.transitionAssetIds ?? [])
    .map((id) => byID.get(id))
    .filter((asset): asset is Asset => asset?.kind === 'video')
})

const addable: { type: LayerType; label: string }[] = [
  { type: 'video', label: 'Видео' },
  { type: 'image', label: 'Картинка или GIF' },
  { type: 'subtitles', label: 'Субтитры' },
  { type: 'text', label: 'Текст' },
  { type: 'blur', label: 'Блюр' },
]

const layerTracks = computed(() => {
  const tracks = new Map<
    string,
    { id: string; layer: Layer; layers: Layer[]; order: number }
  >()
  props.layers.forEach((layer) => {
    const id = layer.trackId ?? layer.id
    const track = tracks.get(id)
    if (track) track.layers.push(layer)
    else tracks.set(id, { id, layer, layers: [layer], order: tracks.size })
  })
  return [...tracks.values()].reverse()
})

function isSelected(track: { id: string }) {
  const selected = props.layers.find(
    (layer) => layer.id === props.selectedLayerId,
  )
  return selected ? (selected.trackId ?? selected.id) === track.id : false
}

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

function displayedLayerLabel(layer: Layer) {
  if (layer.type === 'video' && layer.source === 'asset')
    return 'Импортированный контент'
  return layerLabel(layer.type)
}

function displayedLayerName(layer: Layer) {
  return layer.name.replace(/ · часть \d+$/, '')
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
        <span
          class="grid size-9 place-items-center rounded-lg border border-slate-300 bg-white text-xl leading-none text-slate-700 hover:border-violet-400 hover:text-violet-700"
          aria-hidden="true"
          >+</span
        >
        <select
          class="absolute inset-0 size-full cursor-pointer opacity-0"
          aria-label="Добавить слой"
          @change="addLayer"
        >
          <option value="" selected disabled>Выберите слой</option>
          <option v-for="item in addable" :key="item.type" :value="item.type">
            {{ item.label }}
          </option>
        </select>
      </label>
    </div>
    <div
      v-if="showTrainControls"
      class="mt-3 rounded-lg border border-slate-200 p-2.5 text-sm"
    >
      <label class="flex cursor-pointer items-center gap-2 font-medium">
        <input
          type="checkbox"
          :checked="train?.enabled ?? false"
          @change="emit('setTrainMode', ($event.target as HTMLInputElement).checked)"
        >
        Паровозик
      </label>
      <template v-if="train?.enabled">
        <div v-if="selectedTransitionAssets.length" class="mt-2 space-y-1.5">
          <div
            v-for="asset in selectedTransitionAssets"
            :key="asset.id"
            class="flex items-center gap-2 rounded-md bg-slate-50 p-2"
          >
            <Film class="size-4 shrink-0 text-violet-600" />
            <span class="min-w-0 flex-1 truncate">{{ asset.name }}</span>
            <button
              type="button"
              class="rounded p-1 text-red-600 hover:bg-red-50"
              :aria-label="`Убрать ${asset.name}`"
              @click="emit('removeTransitionAsset', asset.id)"
            >
              <Trash2 class="size-4" />
            </button>
          </div>
        </div>
        <p v-else class="mt-2 text-xs text-slate-500">
          Видео для перебивки не выбрано.
        </p>
        <button
          type="button"
          class="mt-2 flex w-full items-center justify-center gap-1.5 rounded-lg border border-slate-300 px-2 py-2 font-medium hover:border-violet-400 hover:bg-violet-50"
          @click="transitionPickerOpen = true"
        >
          <FolderOpen class="size-4" />Выбрать видео
        </button>
      </template>
    </div>
    <ol class="mt-3 space-y-2">
      <li
        v-for="track in layerTracks"
        :key="track.id"
        class="rounded-lg border p-2 transition"
        :class="isSelected(track) ? 'border-violet-500 bg-violet-50' : 'border-slate-200'"
      >
        <button
          type="button"
          class="flex w-full items-center justify-between gap-2 text-left"
          @click="emit('select', track.layer.id)"
        >
          <span class="flex min-w-0 items-center gap-2">
            <Film
              v-if="track.layer.type === 'video' && track.layer.source === 'asset'"
              class="size-4 shrink-0 text-amber-600"
            />
            <Video
              v-else-if="layerLabel(track.layer.type) === 'Видео'"
              class="size-4 shrink-0 text-violet-600"
            />
            <Image
              v-else-if="layerLabel(track.layer.type) === 'Картинка'"
              class="size-4 shrink-0 text-violet-600"
            />
            <Captions
              v-else-if="track.layer.type === 'subtitles'"
              class="size-4 shrink-0 text-violet-600"
            />
            <Type
              v-else-if="track.layer.type === 'text'"
              class="size-4 shrink-0 text-violet-600"
            />
            <SquareDashed v-else class="size-4 shrink-0 text-violet-600" />
            <span class="min-w-0 truncate font-medium">{{
              displayedLayerName(track.layer)
            }}</span>
          </span>
          <span class="shrink-0 text-xs text-slate-500">{{
            displayedLayerLabel(track.layer)
          }}</span>
        </button>
        <div class="mt-2 flex items-center gap-1 text-xs">
          <button
            type="button"
            class="rounded p-1 hover:bg-white"
            :aria-label="track.layers.some((layer) => layer.visible) ? 'Скрыть слой' : 'Показать слой'"
            @click="emit('update', track.layer.id, { visible: !track.layers.some((layer) => layer.visible) })"
          >
            <Eye
              v-if="track.layers.some((layer) => layer.visible)"
              class="size-4"
            />
            <EyeOff v-else class="size-4" />
          </button>
          <button
            type="button"
            class="rounded px-1 hover:bg-white"
            :disabled="track.order === 0"
            @click="emit('move', track.layer.id, -1)"
          >
            ↓
          </button>
          <button
            type="button"
            class="rounded px-1 hover:bg-white"
            :disabled="track.order === layerTracks.length - 1"
            @click="emit('move', track.layer.id, 1)"
          >
            ↑
          </button>
          <button
            type="button"
            class="rounded p-1 hover:bg-white"
            aria-label="Создать копию слоя"
            @click="emit('duplicate', track.layer.id)"
          >
            <Copy class="size-4" />
          </button>
          <button
            type="button"
            class="ml-auto rounded p-1 text-red-600 hover:bg-red-50"
            aria-label="Удалить слой"
            @click="emit('remove', track.layer.id)"
          >
            <Trash2 class="size-4" />
          </button>
        </div>
      </li>
    </ol>
  </aside>

  <AssetPickerDialog
    :open="transitionPickerOpen"
    :assets="assets ?? []"
    :folders="folders ?? []"
    :kinds="['video']"
    @close="transitionPickerOpen = false"
    @select="emit('addTransitionAsset', $event.id); transitionPickerOpen = false"
  />
</template>

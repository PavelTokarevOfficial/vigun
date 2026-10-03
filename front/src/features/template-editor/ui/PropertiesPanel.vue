<script setup lang="ts">
import { FolderOpen, Trash2 } from '@lucide/vue'
import { computed, ref } from 'vue'
import type {
  Asset,
  AssetFolder,
  AssetKind,
} from '@/entities/asset/model/types'
import type { Layer } from '@/entities/template/model/types'
import AppButton from '@/shared/ui/AppButton.vue'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/shared/ui/shadcn/select'
import AssetPickerDialog from './AssetPickerDialog.vue'

const props = defineProps<{
  layer: Layer | null
  assets: Asset[]
  folders: AssetFolder[]
  whisperModels?: { id: string; name: string; type: string }[]
  defaultWhisperModelId?: string
}>()
const emit = defineEmits<{ update: [patch: Partial<Layer>] }>()
const pickerOpen = ref(false)

const isVideo = computed(() =>
  ['video', 'input_video', 'asset_video'].includes(props.layer?.type ?? ''),
)
const videoSource = computed(() => {
  if (props.layer?.type === 'input_video') return 'clip'
  if (props.layer?.type === 'asset_video') return 'asset'
  return props.layer?.source ?? 'clip'
})
const needsAsset = computed(
  () =>
    props.layer?.type === 'image' ||
    props.layer?.type === 'gif' ||
    (isVideo.value && videoSource.value === 'asset'),
)
const pickerKinds = computed<AssetKind[]>(() =>
  isVideo.value ? ['video'] : ['image', 'gif'],
)
const selectedAsset = computed(() =>
  props.assets.find((asset) => asset.id === props.layer?.assetId),
)

function numberValue(event: Event) {
  return Number((event.target as HTMLInputElement).value)
}
function textColor(value: string | undefined, fallback: string) {
  if (value?.startsWith('#') && value.length === 7) return value
  const match = value?.match(
    /^&H[0-9a-f]{2}([0-9a-f]{2})([0-9a-f]{2})([0-9a-f]{2})$/i,
  )
  return match ? `#${match[3]}${match[2]}${match[1]}` : fallback
}
function setVideoSource(source: 'clip' | 'asset') {
  emit('update', {
    type: 'video',
    source,
    ...(source === 'clip' ? { assetId: undefined } : {}),
  })
}
</script>

<template>
  <aside class="rounded-xl border border-slate-200 bg-white p-4">
    <h2 class="font-semibold">Свойства</h2>
    <p v-if="!layer" class="mt-3 text-sm text-slate-500">
      Выберите слой на рабочей зоне или в списке.
    </p>
    <div v-else class="mt-4 space-y-4">
      <label class="block text-sm font-medium"
        >Название<input
          class="mt-1 w-full font-normal"
          :value="layer.name"
          @change="emit('update', { name: ($event.target as HTMLInputElement).value })"
        ></label
      >

      <fieldset v-if="isVideo" class="space-y-2">
        <legend class="text-sm font-medium">Источник видео</legend>
        <label
          class="flex cursor-pointer items-center gap-2 rounded-lg border border-slate-200 p-2 text-sm"
        >
          <input
            type="radio"
            name="video-source"
            :checked="videoSource === 'clip'"
            @change="setVideoSource('clip')"
          >
          Twitch-клип
        </label>
        <label
          class="flex cursor-pointer items-center gap-2 rounded-lg border border-slate-200 p-2 text-sm"
        >
          <input
            type="radio"
            name="video-source"
            :checked="videoSource === 'asset'"
            @change="setVideoSource('asset')"
          >
          Видео из хранилища
        </label>
      </fieldset>

      <div v-if="needsAsset" class="rounded-xl border border-slate-200 p-3">
        <p class="text-sm font-medium">Файл</p>
        <div
          v-if="selectedAsset"
          class="mt-2 flex items-center gap-3 rounded-lg bg-slate-50 p-2"
        >
          <img
            v-if="selectedAsset.kind === 'image' || selectedAsset.kind === 'gif'"
            :src="selectedAsset.url"
            :alt="selectedAsset.name"
            class="size-11 rounded object-cover"
          >
          <div
            v-else
            class="flex size-11 items-center justify-center rounded bg-violet-100 text-xs text-violet-700"
          >
            MP4
          </div>
          <span class="min-w-0 flex-1 truncate text-sm">{{
            selectedAsset.name
          }}</span>
          <button
            type="button"
            class="rounded p-2 text-red-600 hover:bg-red-50"
            aria-label="Убрать выбранный файл"
            @click="emit('update', { assetId: undefined })"
          >
            <Trash2 class="size-4" />
          </button>
        </div>
        <p v-else class="mt-2 text-sm text-slate-500">Файл не выбран</p>
        <AppButton
          class="mt-3 w-full"
          variant="secondary"
          @click="pickerOpen = true"
        >
          <FolderOpen class="mr-1 inline size-4" />Выбрать
        </AppButton>
      </div>

      <template v-if="layer.type === 'text'">
        <label class="flex items-center gap-2 text-sm">
          <input
            type="checkbox"
            :checked="layer.textSource === 'streamer_name'"
            @change="emit('update', { textSource: ($event.target as HTMLInputElement).checked ? 'streamer_name' : 'custom' })"
          >
          Показывать название Twitch-канала
        </label>
        <label
          v-if="layer.textSource !== 'streamer_name'"
          class="block text-sm font-medium"
          >Текст<textarea
            class="mt-1 w-full rounded border border-slate-300 px-3 py-2 font-normal"
            :value="layer.text"
            @change="emit('update', { text: ($event.target as HTMLTextAreaElement).value })"
          /></label
        >
        <div class="block text-sm font-medium">
          <span>Выравнивание</span>
          <Select
            :model-value="layer.style?.textAlign || 'center'"
            @update:model-value="emit('update', { style: { ...layer.style, textAlign: String($event) as 'left' | 'center' | 'right' } })"
          >
            <SelectTrigger class="mt-1 w-full font-normal">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="left">По левому краю</SelectItem>
              <SelectItem value="center">По центру</SelectItem>
              <SelectItem value="right">По правому краю</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <label class="block text-sm"
          >Размер шрифта<input
            class="mt-1 w-full"
            type="number"
            min="1"
            :value="layer.style?.fontSize || 64"
            @change="emit('update', { style: { ...layer.style, fontSize: numberValue($event) } })"
          ></label
        >
        <label class="flex items-center justify-between gap-3 text-sm"
          ><span>Цвет текста</span>
          <input
            type="color"
            :value="textColor(layer.style?.primaryColor, '#ffffff')"
            @input="emit('update', { style: { ...layer.style, primaryColor: ($event.target as HTMLInputElement).value } })"
          ></label
        >
        <label class="block text-sm"
          >Толщина контура<input
            class="mt-1 w-full"
            type="number"
            min="0"
            :value="layer.style?.outline ?? 2"
            @change="emit('update', { style: { ...layer.style, outline: numberValue($event) } })"
          ></label
        >
        <label class="flex items-center justify-between gap-3 text-sm"
          ><span>Цвет контура</span>
          <input
            type="color"
            :value="textColor(layer.style?.outlineColor, '#000000')"
            @input="emit('update', { style: { ...layer.style, outlineColor: ($event.target as HTMLInputElement).value } })"
          ></label
        >
      </template>

      <template v-if="layer.type === 'blur'">
        <label class="block text-sm font-medium"
          >Сила блюра {{ layer.filters?.blur ?? 18
          }}<input
            class="mt-1 w-full"
            type="range"
            min="0"
            max="50"
            step="1"
            :value="layer.filters?.blur ?? 18"
            @input="emit('update', { filters: { ...layer.filters, blur: numberValue($event) } })"
          ></label
        >
        <label class="block text-sm font-medium"
          >Затемнение
          {{
            Math.round(Math.abs(layer.filters?.brightness ?? -0.2) * 100)
          }}%<input
            class="mt-1 w-full"
            type="range"
            min="0"
            max="1"
            step="0.05"
            :value="Math.abs(layer.filters?.brightness ?? -0.2)"
            @input="emit('update', { filters: { ...layer.filters, brightness: -numberValue($event) } })"
          ></label
        >
      </template>

      <div v-if="layer.type !== 'audio'" class="grid grid-cols-2 gap-2">
        <label class="text-sm"
          >X<input
            class="mt-1 w-full"
            type="number"
            :value="layer.x"
            @change="emit('update', { x: numberValue($event) })"
          ></label
        >
        <label class="text-sm"
          >Y<input
            class="mt-1 w-full"
            type="number"
            :value="layer.y"
            @change="emit('update', { y: numberValue($event) })"
          ></label
        >
        <label class="text-sm"
          >Ширина<input
            class="mt-1 w-full"
            type="number"
            min="1"
            :value="layer.width"
            @change="emit('update', { width: numberValue($event) })"
          ></label
        >
        <label class="text-sm"
          >Высота<input
            class="mt-1 w-full"
            type="number"
            min="1"
            :value="layer.height"
            @change="emit('update', { height: numberValue($event) })"
          ></label
        >
      </div>

      <label class="block text-sm font-medium"
        >Прозрачность {{ Math.round(layer.opacity * 100) }}%<input
          class="mt-1 w-full"
          type="range"
          min="0"
          max="1"
          step="0.05"
          :value="layer.opacity"
          @input="emit('update', { opacity: numberValue($event) })"
        ></label
      >

      <label
        v-if="isVideo || layer.type === 'image' || layer.type === 'gif'"
        class="block text-sm font-medium"
        >Вписывание<select
          class="mt-1 w-full font-normal"
          :value="layer.fit || 'contain'"
          @change="emit('update', { fit: ($event.target as HTMLSelectElement).value as Layer['fit'] })"
        >
          <option value="contain">Вписать</option>
          <option value="cover">Заполнить</option>
          <option value="stretch">Растянуть</option>
        </select></label
      >

      <template v-if="layer.type === 'subtitles'">
        <div class="block text-sm font-medium">
          <span>Модель Whisper</span>
          <Select
            :model-value="layer.whisperModelId || defaultWhisperModelId"
            @update:model-value="emit('update', { whisperModelId: String($event) })"
          >
            <SelectTrigger class="mt-1 w-full font-normal">
              <SelectValue placeholder="Выберите скачанную модель" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem
                v-for="model in whisperModels"
                :key="model.id"
                :value="model.id"
              >
                {{ model.name }}
                · {{ model.type }}
              </SelectItem>
            </SelectContent>
          </Select>
          <span
            v-if="!whisperModels?.length"
            class="mt-1 block text-xs font-normal text-amber-700"
          >
            Сначала скачайте модель в настройках Whisper.
          </span>
        </div>
        <div class="block text-sm font-medium">
          <span>Выравнивание</span>
          <Select
            :model-value="layer.style?.textAlign || 'center'"
            @update:model-value="emit('update', { style: { ...layer.style, textAlign: String($event) as 'left' | 'center' | 'right' } })"
          >
            <SelectTrigger class="mt-1 w-full font-normal">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="left">По левому краю</SelectItem>
              <SelectItem value="center">По центру</SelectItem>
              <SelectItem value="right">По правому краю</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <label class="block text-sm"
          >Размер шрифта<input
            class="mt-1 w-full"
            type="number"
            min="1"
            :value="layer.style?.fontSize || 53"
            @change="emit('update', { style: { ...layer.style, fontSize: numberValue($event) } })"
          ></label
        >
        <label class="flex items-center justify-between gap-3 text-sm"
          ><span>Цвет текста</span>
          <input
            type="color"
            :value="textColor(layer.style?.primaryColor, '#ffffff')"
            @input="emit('update', { style: { ...layer.style, primaryColor: ($event.target as HTMLInputElement).value } })"
          ></label
        >
        <label class="block text-sm"
          >Толщина контура<input
            class="mt-1 w-full"
            type="number"
            min="0"
            :value="layer.style?.outline ?? 2"
            @change="emit('update', { style: { ...layer.style, outline: numberValue($event) } })"
          ></label
        >
        <label class="flex items-center justify-between gap-3 text-sm"
          ><span>Цвет контура</span>
          <input
            type="color"
            :value="textColor(layer.style?.outlineColor, '#000000')"
            @input="emit('update', { style: { ...layer.style, outlineColor: ($event.target as HTMLInputElement).value } })"
          ></label
        >
      </template>
    </div>
  </aside>

  <AssetPickerDialog
    :open="pickerOpen"
    :assets="assets"
    :folders="folders"
    :kinds="pickerKinds"
    :selected-id="layer?.assetId"
    @close="pickerOpen = false"
    @select="emit('update', { assetId: $event.id }); pickerOpen = false"
  />
</template>

<script setup lang="ts">
import { ArrowLeft, FileImage, Folder, X } from '@lucide/vue'
import { computed, ref, watch } from 'vue'
import type {
  Asset,
  AssetFolder,
  AssetKind,
} from '@/entities/asset/model/types'
import AppButton from '@/shared/ui/AppButton.vue'

const props = defineProps<{
  open: boolean
  assets: Asset[]
  folders: AssetFolder[]
  kinds: AssetKind[]
  selectedId?: string
}>()
const emit = defineEmits<{
  close: []
  select: [asset: Asset]
}>()

const currentFolderID = ref<string | null>(null)
const currentFolder = computed(
  () =>
    props.folders.find((folder) => folder.id === currentFolderID.value) ?? null,
)
const visibleFolders = computed(() =>
  props.folders.filter(
    (folder) => (folder.parentId ?? null) === currentFolderID.value,
  ),
)
const visibleAssets = computed(() =>
  props.assets.filter(
    (asset) =>
      (asset.folderId ?? null) === currentFolderID.value &&
      props.kinds.includes(asset.kind),
  ),
)

function goUp() {
  currentFolderID.value = currentFolder.value?.parentId ?? null
}

watch(
  () => props.open,
  (open) => {
    if (open) currentFolderID.value = null
  },
)
</script>

<template>
  <div
    v-if="open"
    class="fixed inset-0 z-[70] flex items-center justify-center bg-slate-950/55 p-4"
    role="dialog"
    aria-modal="true"
    aria-labelledby="asset-picker-title"
    @mousedown.self="emit('close')"
  >
    <section
      class="flex max-h-[82vh] w-full max-w-4xl flex-col overflow-hidden rounded-2xl bg-white shadow-2xl"
    >
      <header
        class="flex items-center justify-between border-b border-slate-200 px-5 py-4"
      >
        <div>
          <h2 id="asset-picker-title" class="text-lg font-semibold">
            Выберите файл
          </h2>
          <p class="mt-1 text-sm text-slate-500">
            {{ currentFolder?.name || 'Ассеты' }}
          </p>
        </div>
        <button
          type="button"
          class="rounded-lg p-2 hover:bg-slate-100"
          aria-label="Закрыть"
          @click="emit('close')"
        >
          <X class="size-5" />
        </button>
      </header>

      <div class="flex-1 overflow-auto p-5">
        <button
          v-if="currentFolderID"
          type="button"
          class="mb-4 inline-flex items-center gap-2 rounded-lg px-3 py-2 text-sm text-slate-600 hover:bg-slate-100"
          @click="goUp"
        >
          <ArrowLeft class="size-4" />Назад
        </button>

        <div class="grid gap-3 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4">
          <button
            v-for="folder in visibleFolders"
            :key="folder.id"
            type="button"
            class="flex min-h-32 flex-col items-start justify-center rounded-xl border border-slate-200 p-4 text-left hover:border-violet-400 hover:bg-violet-50"
            @click="currentFolderID = folder.id"
          >
            <Folder class="size-8 text-violet-600" />
            <span class="mt-3 w-full truncate font-medium">{{
              folder.name
            }}</span>
            <span class="mt-1 text-xs text-slate-500">Папка</span>
          </button>

          <button
            v-for="asset in visibleAssets"
            :key="asset.id"
            type="button"
            class="overflow-hidden rounded-xl border bg-white text-left hover:border-violet-500 hover:ring-2 hover:ring-violet-100"
            :class="selectedId === asset.id ? 'border-violet-600 ring-2 ring-violet-200' : 'border-slate-200'"
            @click="emit('select', asset)"
          >
            <video
              v-if="asset.kind === 'video'"
              :src="asset.url"
              muted
              preload="metadata"
              class="aspect-video w-full bg-slate-100 object-contain"
            />
            <img
              v-else
              :src="asset.url"
              :alt="asset.name"
              class="aspect-video w-full bg-slate-100 object-contain"
            >
            <div class="p-3">
              <p class="truncate font-medium">{{ asset.name }}</p>
              <p class="mt-1 text-xs text-slate-500">{{ asset.kind }}</p>
            </div>
          </button>
        </div>

        <div
          v-if="!visibleFolders.length && !visibleAssets.length"
          class="flex min-h-48 flex-col items-center justify-center rounded-xl border border-dashed border-slate-300 text-slate-500"
        >
          <FileImage class="mb-2 size-8" />
          В этой папке нет подходящих файлов.
        </div>
      </div>

      <footer class="flex justify-end border-t border-slate-200 px-5 py-3">
        <AppButton variant="secondary" @click="emit('close')">Отмена</AppButton>
      </footer>
    </section>
  </div>
</template>

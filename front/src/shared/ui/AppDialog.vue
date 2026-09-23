<script setup lang="ts">
import { X } from '@lucide/vue'
import {
  DialogContent,
  DialogDescription,
  DialogOverlay,
  DialogPortal,
  DialogRoot,
  DialogTitle,
} from 'reka-ui'

defineProps<{
  open: boolean
  title: string
  description?: string
}>()

const emit = defineEmits<{
  'update:open': [value: boolean]
}>()
</script>

<template>
  <DialogRoot :open="open" @update:open="emit('update:open', $event)">
    <DialogPortal>
      <DialogOverlay class="fixed inset-0 z-[70] bg-slate-950/55" />
      <DialogContent
        class="fixed left-1/2 top-1/2 z-[71] max-h-[calc(100vh-2rem)] w-[calc(100%-2rem)] max-w-lg -translate-x-1/2 -translate-y-1/2 overflow-y-auto rounded-2xl bg-white shadow-2xl focus:outline-none"
      >
        <header
          class="flex items-start justify-between gap-4 border-b border-slate-200 px-5 py-4"
        >
          <div>
            <DialogTitle class="text-lg font-semibold">{{ title }}</DialogTitle>
            <DialogDescription
              v-if="description"
              class="mt-1 text-sm text-slate-500"
            >
              {{ description }}
            </DialogDescription>
          </div>
          <button
            type="button"
            class="rounded-lg p-2 text-slate-500 hover:bg-slate-100"
            aria-label="Закрыть"
            @click="emit('update:open', false)"
          >
            <X class="size-5" />
          </button>
        </header>
        <div class="p-5"><slot /></div>
        <footer
          v-if="$slots.footer"
          class="flex justify-end gap-2 border-t border-slate-200 bg-slate-50 px-5 py-4"
        >
          <slot name="footer" />
        </footer>
      </DialogContent>
    </DialogPortal>
  </DialogRoot>
</template>

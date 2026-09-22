<script setup lang="ts">
import { LoaderCircle, Search } from '@lucide/vue'

defineProps<{
  modelValue: string
  busy: boolean
}>()
const emit = defineEmits<{
  'update:modelValue': [value: string]
  search: []
}>()
</script>

<template>
  <form class="block min-w-60 text-sm" @submit.prevent="emit('search')">
    <span>Стример</span>
    <span class="relative mt-1 block">
      <input
        :value="modelValue"
        type="text"
        autocomplete="off"
        autocapitalize="none"
        spellcheck="false"
        placeholder="Введите Twitch-ник"
        class="block w-full pr-11"
        style="padding-right: 3rem"
        @input="emit('update:modelValue', ($event.target as HTMLInputElement).value)"
      >
      <button
        type="submit"
        :disabled="busy || !modelValue.trim()"
        class="absolute right-1 top-1/2 flex size-8 -translate-y-1/2 items-center justify-center rounded-lg text-slate-500 transition hover:bg-slate-100 hover:text-violet-700 disabled:cursor-not-allowed disabled:opacity-40"
        aria-label="Найти стримера"
      >
        <LoaderCircle v-if="busy" class="size-4 animate-spin" />
        <Search v-else class="size-4" />
      </button>
    </span>
  </form>
</template>

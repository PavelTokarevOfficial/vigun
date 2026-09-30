<script setup lang="ts">
import 'vue-sonner/style.css'
import { Toaster } from '@/shared/ui/shadcn/sonner'
import AppHeader from '../widgets/AppHeader.vue'

const persistentToastDuration = Number.POSITIVE_INFINITY
</script>

<template>
  <main class="mx-auto p-3 sm:p-6">
    <AppHeader />
    <RouterView />
    <Toaster
      position="top-center"
      rich-colors
      close-button
      close-button-position="top-right"
      :duration="persistentToastDuration"
    />
  </main>
</template>

<script setup lang="ts">
defineProps<{ message: string }>()
</script>

<template>
  <div class="empty-state">
    <p>{{ message }}</p>
    <slot />
  </div>
</template>

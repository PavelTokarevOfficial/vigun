<script setup lang="ts">
import { Trash2 } from '@lucide/vue'
import AppButton from '@/shared/ui/AppButton.vue'
import AppDialog from '@/shared/ui/AppDialog.vue'

defineProps<{ open: boolean; busy: boolean }>()
const emit = defineEmits<{
  'update:open': [value: boolean]
  confirm: []
}>()
</script>

<template>
  <AppDialog
    :open="open"
    title="Очистить синхронизированные клипы?"
    description="Лента подписок станет пустой до следующей синхронизации."
    @update:open="emit('update:open', $event)"
  >
    <p class="text-sm text-slate-600">
      Скачанные клипы, исходники и готовые видео в Pipeline удалены не будут.
    </p>
    <template #footer>
      <AppButton variant="secondary" @click="emit('update:open', false)">
        Отмена
      </AppButton>
      <AppButton :disabled="busy" variant="danger" @click="emit('confirm')">
        <Trash2 class="mr-1 inline size-4" />
        {{ busy ? 'Очищаем…' : 'Очистить клипы' }}
      </AppButton>
    </template>
  </AppDialog>
</template>

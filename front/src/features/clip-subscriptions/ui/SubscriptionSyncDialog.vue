<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import {
  type DateRangeValue,
  dateInputValue,
  formatDateRange,
} from '@/shared/lib/dateRange'
import AppButton from '@/shared/ui/AppButton.vue'
import AppDateRangePicker from '@/shared/ui/AppDateRangePicker.vue'
import AppDialog from '@/shared/ui/AppDialog.vue'

type Preset = 'today' | 'three-days' | 'week' | 'all-time' | 'custom'

const props = defineProps<{ open: boolean; syncing: boolean }>()
const emit = defineEmits<{
  'update:open': [value: boolean]
  sync: [range: DateRangeValue]
}>()

const preset = ref<Preset>('week')
const customRange = ref<DateRangeValue>({
  start: dateInputValue(6),
  end: dateInputValue(),
})

const options: Array<{ value: Preset; label: string }> = [
  { value: 'today', label: 'За сегодня' },
  { value: 'three-days', label: 'За 3 дня' },
  { value: 'week', label: 'За неделю' },
  { value: 'all-time', label: 'За всё время' },
  { value: 'custom', label: 'Свой период' },
]

const selectedRange = computed<DateRangeValue>(() => {
  if (preset.value === 'custom') return customRange.value
  if (preset.value === 'all-time') {
    return { start: '2011-06-06', end: dateInputValue() }
  }
  const daysAgo =
    preset.value === 'today' ? 0 : preset.value === 'three-days' ? 2 : 6
  return { start: dateInputValue(daysAgo), end: dateInputValue() }
})

watch(
  () => props.open,
  (open) => {
    if (open) preset.value = 'week'
  },
)
</script>

<template>
  <AppDialog
    :open="open"
    title="Синхронизация клипов"
    description="Выберите период, за который нужно получить клипы подписок."
    @update:open="emit('update:open', $event)"
  >
    <div class="flex flex-wrap gap-2">
      <button
        v-for="option in options"
        :key="option.value"
        type="button"
        class="rounded-xl border px-3 py-2 text-sm font-semibold transition"
        :class="preset === option.value
          ? 'border-violet-600 bg-violet-600 text-white'
          : 'border-slate-200 bg-white text-slate-700 hover:bg-slate-50'"
        @click="preset = option.value"
      >
        {{ option.label }}
      </button>
    </div>
    <div class="mt-5">
      <div v-if="preset === 'custom'" class="block">
        <span class="mb-2 block text-sm font-medium text-slate-700"
          >Период</span
        >
        <AppDateRangePicker
          v-model="customRange"
          :max="dateInputValue()"
          :disabled="syncing"
        />
      </div>
      <p v-else class="rounded-xl bg-slate-50 px-4 py-3 text-sm text-slate-600">
        Будут загружены клипы за период:
        <b class="text-slate-900">{{ formatDateRange(selectedRange) }}</b>
      </p>
    </div>
    <template #footer>
      <AppButton variant="secondary" @click="emit('update:open', false)">
        Отмена
      </AppButton>
      <AppButton :disabled="syncing" @click="emit('sync', selectedRange)">
        {{ syncing ? 'Синхронизация…' : 'Синхронизировать' }}
      </AppButton>
    </template>
  </AppDialog>
</template>

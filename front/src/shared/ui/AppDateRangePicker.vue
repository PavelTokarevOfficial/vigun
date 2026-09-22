<script setup lang="ts">
import { parseDate } from '@internationalized/date'
import { CalendarDays, ChevronLeft, ChevronRight } from '@lucide/vue'
import {
  type DateRange,
  DateRangePickerCalendar,
  DateRangePickerCell,
  DateRangePickerCellTrigger,
  DateRangePickerContent,
  DateRangePickerGrid,
  DateRangePickerGridBody,
  DateRangePickerGridHead,
  DateRangePickerGridRow,
  DateRangePickerHeadCell,
  DateRangePickerHeader,
  DateRangePickerHeading,
  DateRangePickerNext,
  DateRangePickerPrev,
  DateRangePickerRoot,
  DateRangePickerTrigger,
  PopoverPortal,
} from 'reka-ui'
import { shallowRef, watch } from 'vue'
import { type DateRangeValue, formatDateRange } from '@/shared/lib/dateRange'

const props = withDefaults(
  defineProps<{
    modelValue: DateRangeValue
    max?: string
    disabled?: boolean
  }>(),
  { max: undefined, disabled: false },
)

const emit = defineEmits<{
  'update:modelValue': [value: DateRangeValue]
}>()

const value = shallowRef<DateRange>({
  start: parseDate(props.modelValue.start),
  end: parseDate(props.modelValue.end),
})

watch(
  () => props.modelValue,
  (next) => {
    if (
      value.value.start?.toString() === next.start &&
      value.value.end?.toString() === next.end
    )
      return
    value.value = { start: parseDate(next.start), end: parseDate(next.end) }
  },
  { deep: true },
)

function update(next: DateRange) {
  value.value = next
  if (next.start && next.end) {
    emit('update:modelValue', {
      start: next.start.toString(),
      end: next.end.toString(),
    })
  }
}

const maxValue = props.max ? parseDate(props.max) : undefined
</script>

<template>
  <DateRangePickerRoot
    :model-value="value"
    :max-value="maxValue"
    :disabled="disabled"
    locale="ru-RU"
    :week-starts-on="1"
    :close-on-select="true"
    @update:model-value="update"
  >
    <DateRangePickerTrigger
      class="flex min-h-10 w-full items-center justify-between gap-3 rounded-xl border border-slate-300 bg-white px-3 py-2 text-left text-sm font-medium text-slate-800 outline-none transition hover:border-slate-400 focus-visible:ring-2 focus-visible:ring-violet-500 disabled:opacity-50"
    >
      <span>{{ formatDateRange(modelValue) }}</span>
      <CalendarDays class="size-4 shrink-0 text-slate-500" />
    </DateRangePickerTrigger>
    <PopoverPortal>
      <DateRangePickerContent
        position="popper"
        :side-offset="8"
        class="z-[90] rounded-2xl border border-slate-200 bg-white p-4 shadow-2xl outline-none"
      >
        <DateRangePickerCalendar v-slot="{ grid, weekDays }">
          <DateRangePickerHeader class="mb-3 flex items-center justify-between">
            <DateRangePickerPrev
              class="rounded-lg p-2 text-slate-600 hover:bg-slate-100"
            >
              <ChevronLeft class="size-4" />
            </DateRangePickerPrev>
            <DateRangePickerHeading class="text-sm font-semibold" />
            <DateRangePickerNext
              class="rounded-lg p-2 text-slate-600 hover:bg-slate-100"
            >
              <ChevronRight class="size-4" />
            </DateRangePickerNext>
          </DateRangePickerHeader>
          <DateRangePickerGrid
            v-for="month in grid"
            :key="month.value.toString()"
            class="w-full border-collapse"
          >
            <DateRangePickerGridHead>
              <DateRangePickerGridRow class="grid grid-cols-7">
                <DateRangePickerHeadCell
                  v-for="day in weekDays"
                  :key="day"
                  class="flex size-9 items-center justify-center text-xs font-medium text-slate-400"
                >
                  {{ day }}
                </DateRangePickerHeadCell>
              </DateRangePickerGridRow>
            </DateRangePickerGridHead>
            <DateRangePickerGridBody>
              <DateRangePickerGridRow
                v-for="(week, index) in month.rows"
                :key="index"
                class="mt-1 grid grid-cols-7"
              >
                <DateRangePickerCell
                  v-for="day in week"
                  :key="day.toString()"
                  :date="day"
                  class="relative flex size-9 items-center justify-center"
                >
                  <DateRangePickerCellTrigger
                    :day="day"
                    :month="month.value"
                    class="flex size-9 items-center justify-center rounded-lg text-sm outline-none transition hover:bg-violet-50 focus-visible:ring-2 focus-visible:ring-violet-500 data-[outside-view]:text-slate-300 data-[disabled]:pointer-events-none data-[disabled]:opacity-30 data-[highlighted]:bg-violet-100 data-[selected]:bg-violet-100 data-[selection-start]:bg-violet-600 data-[selection-start]:text-white data-[selection-end]:bg-violet-600 data-[selection-end]:text-white data-[today]:font-bold"
                  />
                </DateRangePickerCell>
              </DateRangePickerGridRow>
            </DateRangePickerGridBody>
          </DateRangePickerGrid>
        </DateRangePickerCalendar>
      </DateRangePickerContent>
    </PopoverPortal>
  </DateRangePickerRoot>
</template>

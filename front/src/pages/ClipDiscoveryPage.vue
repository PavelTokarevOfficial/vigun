<script setup lang="ts">
import { Eye, EyeOff, RefreshCw, Trash2 } from '@lucide/vue'
import { computed, provide, ref } from 'vue'
import { useRoute } from 'vue-router'
import {
  createSubscriptionSyncControls,
  subscriptionSyncControlsKey,
} from '@/features/clip-subscriptions/model/syncControls'
import ClearSubscriptionClipsDialog from '@/features/clip-subscriptions/ui/ClearSubscriptionClipsDialog.vue'
import SubscriptionSyncDialog from '@/features/clip-subscriptions/ui/SubscriptionSyncDialog.vue'
import type { DateRangeValue } from '@/shared/lib/dateRange'

const route = useRoute()
const onSubscriptions = computed(() => route.name === 'clip-subscriptions')
const syncControls = createSubscriptionSyncControls()
const clearDialogOpen = ref(false)
const syncDialogOpen = ref(false)
const {
  canSync,
  canClear,
  clearing,
  hideViewed,
  progress: syncProgress,
  requestSync,
  requestClear,
  syncing,
} = syncControls

provide(subscriptionSyncControlsKey, syncControls)

function confirmClear() {
  clearDialogOpen.value = false
  requestClear()
}

function startSync(range: DateRangeValue) {
  syncDialogOpen.value = false
  requestSync(range)
}
</script>

<template>
  <section>
    <div
      class="mb-6 flex items-center justify-between gap-2 border-b border-slate-200"
    >
      <nav
        class="-mb-px flex gap-3 overflow-x-auto sm:gap-5"
        aria-label="Разделы клипов"
      >
        <RouterLink
          :to="{ name: 'clip-subscriptions' }"
          class="shrink-0 border-b-2 px-1 pb-3 text-sm font-semibold transition"
          :class="onSubscriptions ? 'border-violet-600 text-violet-700' : 'border-transparent text-slate-500 hover:text-slate-900'"
        >
          Подписки
        </RouterLink>
        <RouterLink
          :to="{ name: 'clip-search' }"
          class="shrink-0 border-b-2 px-1 pb-3 text-sm font-semibold transition"
          :class="!onSubscriptions ? 'border-violet-600 text-violet-700' : 'border-transparent text-slate-500 hover:text-slate-900'"
        >
          Поиск
        </RouterLink>
      </nav>
      <div v-if="onSubscriptions" class="mb-3 flex shrink-0 items-center gap-2">
        <button
          type="button"
          class="inline-flex min-h-10 shrink-0 items-center justify-center gap-2 rounded-xl border border-red-200 bg-white px-3 text-xs font-semibold text-red-700 transition hover:bg-red-50 disabled:cursor-not-allowed disabled:opacity-40 sm:px-4 sm:text-sm"
          :disabled="syncing || clearing || !canClear"
          title="Удалить все синхронизированные клипы"
          @click="clearDialogOpen = true"
        >
          <Trash2 class="size-4" />
          <span class="hidden lg:inline">{{
            clearing ? 'Очищаем…' : 'Очистить клипы'
          }}</span>
        </button>
        <button
          type="button"
          class="inline-flex min-h-10 shrink-0 items-center justify-center gap-2 rounded-xl border px-3 text-xs font-semibold transition sm:px-4 sm:text-sm"
          :class="hideViewed
            ? 'border-violet-200 bg-violet-50 text-violet-700 hover:bg-violet-100'
            : 'border-slate-200 bg-white text-slate-700 hover:bg-slate-50'"
          :aria-pressed="hideViewed"
          :title="hideViewed ? 'Показать просмотренные' : 'Скрыть просмотренные'"
          @click="hideViewed = !hideViewed"
        >
          <Eye v-if="hideViewed" class="size-4" />
          <EyeOff v-else class="size-4" />
          <span class="hidden md:inline">
            {{ hideViewed ? 'Показать просмотренные' : 'Скрыть просмотренные' }}
          </span>
        </button>
        <button
          type="button"
          class="inline-flex min-h-10 shrink-0 items-center justify-center gap-2 rounded-xl bg-violet-600 px-3 text-xs font-semibold text-white transition hover:bg-violet-700 disabled:cursor-not-allowed disabled:opacity-50 sm:px-4 sm:text-sm"
          :disabled="syncing || clearing || !canSync"
          @click="syncDialogOpen = true"
        >
          <RefreshCw class="size-4" :class="syncing ? 'animate-spin' : ''" />
          <span class="sm:hidden">{{
            syncing ? syncProgress : 'Обновить'
          }}</span>
          <span class="hidden sm:inline">
            {{ syncing ? `Синхронизация ${syncProgress}` : 'Синхронизировать' }}
          </span>
        </button>
      </div>
    </div>
    <RouterView />

    <SubscriptionSyncDialog
      v-model:open="syncDialogOpen"
      :syncing="syncing"
      @sync="startSync"
    />
    <ClearSubscriptionClipsDialog
      v-model:open="clearDialogOpen"
      :busy="clearing"
      @confirm="confirmClear"
    />
  </section>
</template>

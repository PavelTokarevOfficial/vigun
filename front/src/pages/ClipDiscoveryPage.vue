<script setup lang="ts">
import { RefreshCw } from '@lucide/vue'
import { computed, provide } from 'vue'
import { useRoute } from 'vue-router'
import {
  createSubscriptionSyncControls,
  subscriptionSyncControlsKey,
} from '@/features/clip-subscriptions/model/syncControls'

const route = useRoute()
const onSubscriptions = computed(() => route.name === 'clip-subscriptions')
const syncControls = createSubscriptionSyncControls()
const { canSync, progress: syncProgress, requestSync, syncing } = syncControls

provide(subscriptionSyncControlsKey, syncControls)
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
      <button
        v-if="onSubscriptions"
        type="button"
        class="mb-3 inline-flex min-h-10 shrink-0 items-center justify-center gap-2 rounded-xl bg-violet-600 px-3 text-xs font-semibold text-white transition hover:bg-violet-700 disabled:cursor-not-allowed disabled:opacity-50 sm:px-4 sm:text-sm"
        :disabled="syncing || !canSync"
        @click="requestSync"
      >
        <RefreshCw class="size-4" :class="syncing ? 'animate-spin' : ''" />
        <span class="sm:hidden">{{ syncing ? syncProgress : 'Обновить' }}</span>
        <span class="hidden sm:inline">
          {{ syncing ? `Синхронизация ${syncProgress}` : 'Синхронизировать' }}
        </span>
      </button>
    </div>
    <RouterView />
  </section>
</template>

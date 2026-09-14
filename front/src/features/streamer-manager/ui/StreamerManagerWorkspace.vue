<script setup lang="ts">
import { Bell, Pencil, Plus, Trash2 } from '@lucide/vue'
import { onMounted } from 'vue'
import { useStreamerManager } from '@/features/streamer-manager/model/useStreamerManager'
import AppButton from '@/shared/ui/AppButton.vue'
import EmptyState from '@/shared/ui/EmptyState.vue'
import ErrorState from '@/shared/ui/ErrorState.vue'
import AddStreamersDialog from './AddStreamersDialog.vue'

const manager = useStreamerManager()
const { busy, editNickname, editing, error, notice, rows } = manager

function updatePriority(event: Event, id: string) {
  const priority = Number((event.target as HTMLInputElement).value)
  const streamer = rows.value.find((item) => item.id === id)
  if (streamer) void manager.setPriority(streamer, priority)
}

function updateSubscription(event: Event, id: string) {
  const streamer = rows.value.find((item) => item.id === id)
  if (streamer) {
    void manager.setSubscribed(
      streamer,
      (event.target as HTMLInputElement).checked,
    )
  }
}

onMounted(() => void manager.load())
</script>

<template>
  <section>
    <div class="mb-6 flex items-center justify-between gap-3 sm:mb-10">
      <h1 class="text-3xl font-semibold sm:text-4xl">Стримеры</h1>
      <AppButton type="button" @click="manager.openAddDialog"
        ><Plus class="mr-1 inline size-4" />Добавить</AppButton
      >
    </div>
    <ErrorState v-if="error" :message="error" />
    <p v-if="notice" class="mb-4 text-sm text-emerald-700">{{ notice }}</p>
    <EmptyState v-if="!rows.length" message="Стримеров пока нет." />
    <article
      v-for="row in rows"
      :key="row.id"
      class="mb-3 rounded-2xl border border-slate-200 bg-white p-4 shadow-sm"
    >
      <form
        v-if="editing?.id === row.id"
        class="grid gap-2 sm:grid-cols-[minmax(0,1fr)_auto_auto]"
        @submit.prevent="manager.saveEdit"
      >
        <input v-model="editNickname" class="min-w-0 w-full" required>
        <AppButton type="submit" :disabled="busy">Сохранить</AppButton>
        <AppButton type="button" variant="secondary" @click="manager.cancelEdit"
          >Отмена</AppButton
        >
      </form>
      <div
        v-else
        class="flex min-w-0 flex-col gap-4 lg:flex-row lg:items-center lg:justify-between"
      >
        <div class="min-w-0">
          <b class="block truncate">{{ row.displayName }}</b>
          <span class="block truncate text-sm text-slate-500">
            @{{ row.twitchLogin }}
          </span>
        </div>
        <div
          class="grid gap-3 sm:grid-cols-[auto_auto] lg:flex lg:shrink-0 lg:items-center"
        >
          <div class="grid gap-3 sm:contents">
            <label
              class="flex min-h-11 items-center gap-2 rounded-xl border border-slate-200 px-3 text-sm text-slate-700"
            >
              <input
                type="checkbox"
                class="size-4 accent-violet-600"
                :checked="row.subscribed"
                :disabled="busy"
                @change="updateSubscription($event, row.id)"
              >
              <Bell class="size-4 text-violet-600" />
              <span>Подписаться</span>
            </label>
            <label
              class="flex min-h-11 items-center justify-between gap-2 text-sm text-slate-600"
              ><span>Приоритет</span>
              <input
                type="number"
                min="0"
                step="1"
                class="w-18 rounded border border-slate-300 px-2 py-1 text-right text-slate-900"
                :value="row.priority"
                :disabled="busy"
                @change="updatePriority($event, row.id)"
              ></label
            >
          </div>
          <div class="grid grid-cols-2 gap-2">
            <AppButton
              variant="secondary"
              :disabled="busy"
              @click="manager.startEdit(row)"
              ><Pencil class="mr-1 inline size-4" />Изменить</AppButton
            >
            <AppButton
              variant="danger"
              :disabled="busy"
              @click="manager.remove(row)"
              ><Trash2 class="mr-1 inline size-4" />Удалить</AppButton
            >
          </div>
        </div>
      </div>
    </article>
    <AddStreamersDialog :manager="manager" />
  </section>
</template>

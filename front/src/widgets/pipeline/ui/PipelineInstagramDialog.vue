<script setup lang="ts">
import { X } from '@lucide/vue'
import AppButton from '@/shared/ui/AppButton.vue'
import type { PipelineWorkspaceModel } from '../model/usePipelineWorkspace'

const props = defineProps<{ workspace: PipelineWorkspaceModel }>()
const {
  closeInstagramDialog,
  instagramAccounts,
  instagramAccountsLoading,
  instagramBusy,
  instagramForm,
  instagramMessage,
  instagramVideo,
  publicVideoURL,
  publishToInstagram,
} = props.workspace
</script>

<template>
  <div
    v-if="instagramVideo"
    class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/70 p-4"
    role="dialog"
    aria-modal="true"
    aria-labelledby="instagram-dialog-title"
  >
    <section
      class="max-h-[92vh] w-full max-w-3xl overflow-auto rounded-2xl bg-white shadow-2xl"
    >
      <header
        class="flex items-start justify-between gap-4 border-b border-slate-200 px-5 py-4"
      >
        <div>
          <h2 id="instagram-dialog-title" class="text-lg font-semibold">
            Публикация Reels
          </h2>
          <p class="mt-1 text-sm text-slate-600">
            {{ instagramVideo.title }}
          </p>
        </div>
        <button
          type="button"
          class="rounded-lg p-2 text-slate-600 hover:bg-slate-100 disabled:opacity-50"
          :disabled="instagramBusy"
          aria-label="Закрыть публикацию в Instagram"
          @click="closeInstagramDialog"
        >
          <X class="size-5" />
        </button>
      </header>

      <form class="space-y-4 p-5" @submit.prevent="publishToInstagram">
        <label class="block text-sm font-medium">
          Instagram-аккаунт
          <select
            v-model="instagramForm.accountId"
            required
            :disabled="instagramAccountsLoading || instagramBusy"
            class="mt-1 w-full rounded-lg border border-slate-300 bg-white px-3 py-2 font-normal"
          >
            <option value="" disabled>
              {{
                instagramAccountsLoading ? 'Загружаем аккаунты…' : 'Выберите аккаунт'
              }}
            </option>
            <option
              v-for="account in instagramAccounts"
              :key="account.id"
              :value="account.id"
            >
              @{{ account.nickname }} · {{ account.instagramUserId }}
            </option>
          </select>
        </label>
        <p
          v-if="!instagramAccountsLoading && !instagramAccounts.length"
          class="rounded-lg bg-amber-50 p-3 text-sm text-amber-800"
        >
          Сначала добавьте Instagram-аккаунт в
          <RouterLink to="/accounts" class="font-semibold underline">
            менеджере аккаунтов</RouterLink
          >.
        </p>
        <label class="block text-sm font-medium">
          HTTPS URL Cloudflare Tunnel
          <input
            v-model="instagramForm.tunnelUrl"
            type="url"
            required
            placeholder="https://example.trycloudflare.com"
            class="mt-1 w-full rounded-lg border border-slate-300 px-3 py-2 font-normal"
          >
        </label>
        <p class="break-all rounded-lg bg-slate-100 p-3 text-xs text-slate-600">
          Instagram получит:
          <a
            :href="publicVideoURL(instagramVideo)"
            target="_blank"
            rel="noopener noreferrer"
          >
            {{ publicVideoURL(instagramVideo) }}
          </a>
        </p>
        <label class="block text-sm font-medium">
          Подпись
          <textarea
            v-model="instagramForm.caption"
            rows="4"
            class="mt-1 w-full h-50 rounded-lg border border-slate-300 px-3 py-2 font-normal"
          />
        </label>
        <label class="flex items-center gap-2 text-sm font-medium">
          <input v-model="instagramForm.shareToFeed" type="checkbox">
          Также показать в ленте
        </label>

        <details class="rounded-lg border border-slate-200 p-4">
          <summary class="cursor-pointer text-sm font-semibold">
            Дополнительные настройки
          </summary>
          <div class="mt-4 grid gap-4 sm:grid-cols-2">
            <label class="block text-sm font-medium">
              Соавторы через запятую
              <input
                v-model="instagramForm.collaborators"
                placeholder="username, another_user"
                class="mt-1 w-full rounded-lg border border-slate-300 px-3 py-2 font-normal"
              >
            </label>
            <label class="block text-sm font-medium">
              Название аудио
              <input
                v-model="instagramForm.audioName"
                class="mt-1 w-full rounded-lg border border-slate-300 px-3 py-2 font-normal"
              >
            </label>
            <label class="block text-sm font-medium sm:col-span-2">
              HTTPS URL обложки
              <input
                v-model="instagramForm.coverUrl"
                type="url"
                placeholder="https://…/cover.jpg"
                class="mt-1 w-full rounded-lg border border-slate-300 px-3 py-2 font-normal"
              >
            </label>
            <label class="block text-sm font-medium">
              Кадр обложки, мс
              <input
                v-model="instagramForm.thumbOffset"
                type="number"
                min="0"
                placeholder="0"
                class="mt-1 w-full rounded-lg border border-slate-300 px-3 py-2 font-normal"
              >
            </label>
            <label class="block text-sm font-medium">
              Instagram location ID
              <input
                v-model="instagramForm.locationId"
                class="mt-1 w-full rounded-lg border border-slate-300 px-3 py-2 font-normal"
              >
            </label>
          </div>
          <p class="mt-3 text-xs text-slate-500">
            Если задан URL обложки, Instagram игнорирует смещение кадра.
          </p>
        </details>

        <p v-if="instagramMessage" class="text-sm text-emerald-700">
          {{ instagramMessage }}
        </p>
        <footer class="flex justify-end gap-2 border-t border-slate-200 pt-4">
          <AppButton
            variant="secondary"
            :disabled="instagramBusy"
            @click="closeInstagramDialog"
          >
            Закрыть
          </AppButton>
          <AppButton
            type="submit"
            :disabled="instagramBusy || instagramAccountsLoading || !instagramForm.accountId"
          >
            {{ instagramBusy ? 'Публикуем…' : 'Опубликовать' }}
          </AppButton>
        </footer>
      </form>
    </section>
  </div>
</template>

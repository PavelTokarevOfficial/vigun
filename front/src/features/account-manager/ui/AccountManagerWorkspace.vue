<script setup lang="ts">
import { AtSign, Pencil, Plus, Trash2, X } from '@lucide/vue'
import { ref } from 'vue'
import AppButton from '@/shared/ui/AppButton.vue'
import { useInstagramAccounts } from '../model/useInstagramAccounts'

type Platform = 'instagram' | 'youtube' | 'tiktok'

const activePlatform = ref<Platform>('instagram')
const model = useInstagramAccounts()
const {
  accounts,
  busy,
  deleteCandidate,
  editing,
  editorOpen,
  error,
  form,
  loading,
  notice,
  tokenBusy,
} = model

const tabs: { id: Platform; label: string }[] = [
  { id: 'instagram', label: 'Instagram' },
  { id: 'youtube', label: 'YouTube' },
  { id: 'tiktok', label: 'TikTok' },
]

function formatDate(value: string | null) {
  return value ? new Date(value).toLocaleString() : 'нет данных'
}

function tokenLifetime(value: string | null) {
  if (!value) return 'Срок действия неизвестен'
  const milliseconds = new Date(value).getTime() - Date.now()
  if (milliseconds <= 0) return `Истёк ${formatDate(value)}`
  const hours = Math.ceil(milliseconds / 3_600_000)
  const remaining = hours >= 48 ? `${Math.ceil(hours / 24)} дн.` : `${hours} ч.`
  return `До ${formatDate(value)} · осталось ${remaining}`
}
</script>

<template>
  <section>
    <div class="mb-6 flex flex-wrap items-center justify-between gap-3">
      <div>
        <h1 class="text-3xl font-semibold sm:text-4xl">Менеджер аккаунтов</h1>
        <p class="mt-2 text-sm text-slate-500">
          Аккаунты для публикации готовых видео.
        </p>
      </div>
      <AppButton
        v-if="activePlatform === 'instagram'"
        @click="model.openCreate"
      >
        <Plus class="mr-1 inline size-4" />Добавить аккаунт
      </AppButton>
    </div>

    <div class="mb-6 flex gap-1 border-b border-slate-200">
      <button
        v-for="tab in tabs"
        :key="tab.id"
        type="button"
        class="border-b-2 px-4 py-3 text-sm font-semibold transition"
        :class="activePlatform === tab.id ? 'border-violet-600 text-violet-700' : 'border-transparent text-slate-500 hover:text-slate-900'"
        @click="activePlatform = tab.id"
      >
        {{ tab.label }}
      </button>
    </div>

    <p v-if="error" class="mb-4 rounded-xl bg-red-50 p-3 text-sm text-red-700">
      {{ error }}
    </p>
    <p
      v-if="notice"
      class="mb-4 rounded-xl bg-emerald-50 p-3 text-sm text-emerald-700"
    >
      {{ notice }}
    </p>

    <template v-if="activePlatform === 'instagram'">
      <p v-if="loading" class="text-sm text-slate-500">Загружаем аккаунты…</p>
      <div
        v-else-if="accounts.length"
        class="grid gap-3 md:grid-cols-2 xl:grid-cols-3"
      >
        <article
          v-for="account in accounts"
          :key="account.id"
          class="rounded-2xl border border-slate-200 bg-white p-4 shadow-sm"
        >
          <div class="flex items-start justify-between gap-3">
            <div class="min-w-0">
              <div class="flex items-center gap-2">
                <AtSign class="size-5 text-pink-600" />
                <b class="truncate">@{{ account.nickname }}</b>
              </div>
              <p class="mt-2 break-all text-sm text-slate-500">
                ID: {{ account.instagramUserId }}
              </p>
              <p class="mt-1 text-xs text-emerald-700">Токен сохранён</p>
            </div>
            <div class="flex gap-2">
              <button
                type="button"
                class="rounded-lg p-2 text-slate-500 hover:bg-slate-100 hover:text-slate-900"
                aria-label="Изменить аккаунт"
                @click="model.openEdit(account)"
              >
                <Pencil class="size-4" />
              </button>
              <button
                type="button"
                class="rounded-lg p-2 text-red-500 hover:bg-red-50 hover:text-red-700"
                aria-label="Удалить аккаунт"
                @click="deleteCandidate = account"
              >
                <Trash2 class="size-4" />
              </button>
            </div>
          </div>
          <dl
            class="mt-4 space-y-1 rounded-xl bg-slate-50 p-3 text-xs text-slate-600"
          >
            <div class="flex justify-between gap-3">
              <dt>Обновлён</dt>
              <dd class="text-right">
                {{ formatDate(account.tokenUpdatedAt) }}
              </dd>
            </div>
            <div class="flex justify-between gap-3">
              <dt>Срок</dt>
              <dd
                class="text-right"
                :class="account.tokenExpiresAt && new Date(account.tokenExpiresAt).getTime() <= Date.now() ? 'text-red-600' : ''"
              >
                {{ tokenLifetime(account.tokenExpiresAt) }}
              </dd>
            </div>
            <div class="flex justify-between gap-3">
              <dt>Проверен</dt>
              <dd class="text-right">
                {{ formatDate(account.tokenLastCheckedAt) }}
              </dd>
            </div>
            <div
              v-if="account.verifiedUsername"
              class="flex justify-between gap-3"
            >
              <dt>Instagram</dt>
              <dd class="text-right">
                @{{ account.verifiedUsername
                }}<template v-if="account.accountType">
                  · {{ account.accountType }}</template
                >
              </dd>
            </div>
          </dl>
          <div class="mt-4 flex flex-wrap gap-2">
            <AppButton
              variant="secondary"
              :disabled="Boolean(tokenBusy)"
              @click="model.tokenAction(account, 'exchange-token')"
            >
              Долгоживущий
            </AppButton>
            <AppButton
              variant="secondary"
              :disabled="Boolean(tokenBusy)"
              @click="model.tokenAction(account, 'refresh-token')"
            >
              Обновить токен
            </AppButton>
            <AppButton
              :disabled="Boolean(tokenBusy)"
              @click="model.tokenAction(account, 'check-token')"
            >
              {{
                tokenBusy === `${account.id}:check-token` ? 'Проверяем…' : 'Проверить'
              }}
            </AppButton>
          </div>
        </article>
      </div>
      <div
        v-else-if="!loading"
        class="rounded-2xl border border-dashed border-slate-300 p-8 text-center text-sm text-slate-500"
      >
        Instagram-аккаунты ещё не добавлены.
      </div>
    </template>

    <div
      v-else
      class="rounded-2xl border border-dashed border-slate-300 p-8 text-center text-sm text-slate-500"
    >
      Поддержка
      {{ activePlatform === 'youtube' ? 'YouTube' : 'TikTok' }} появится позже.
    </div>

    <div
      v-if="editorOpen"
      class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/60 p-4"
      role="dialog"
      aria-modal="true"
      aria-labelledby="instagram-account-title"
    >
      <form
        class="w-full max-w-lg rounded-2xl bg-white shadow-2xl"
        @submit.prevent="model.save"
      >
        <header
          class="flex items-center justify-between border-b border-slate-200 px-5 py-4"
        >
          <h2 id="instagram-account-title" class="text-lg font-semibold">
            {{
              editing ? 'Изменить Instagram-аккаунт' : 'Добавить Instagram-аккаунт'
            }}
          </h2>
          <button
            type="button"
            class="rounded-lg p-2 text-slate-500 hover:bg-slate-100"
            aria-label="Закрыть"
            :disabled="busy"
            @click="model.closeEditor"
          >
            <X class="size-5" />
          </button>
        </header>
        <div class="space-y-4 p-5">
          <label class="block text-sm font-medium">
            Ник
            <input
              v-model="form.nickname"
              required
              placeholder="my_account"
              class="mt-1 w-full rounded-lg border border-slate-300 px-3 py-2 font-normal"
            >
          </label>
          <label class="block text-sm font-medium">
            INSTAGRAM_USER_ID
            <input
              v-model="form.instagramUserId"
              required
              class="mt-1 w-full rounded-lg border border-slate-300 px-3 py-2 font-normal"
            >
          </label>
          <label class="block text-sm font-medium">
            INSTAGRAM_ACCESS_TOKEN
            <input
              v-model="form.accessToken"
              type="password"
              :required="!editing"
              autocomplete="new-password"
              :placeholder="editing ? 'Оставьте пустым, чтобы не менять' : ''"
              class="mt-1 w-full rounded-lg border border-slate-300 px-3 py-2 font-normal"
            >
          </label>
        </div>
        <footer
          class="flex justify-end gap-2 border-t border-slate-200 px-5 py-4"
        >
          <AppButton
            type="button"
            variant="secondary"
            :disabled="busy"
            @click="model.closeEditor"
          >
            Отмена
          </AppButton>
          <AppButton type="submit" :disabled="busy">
            {{ busy ? 'Сохраняем…' : 'Сохранить' }}
          </AppButton>
        </footer>
      </form>
    </div>

    <div
      v-if="deleteCandidate"
      class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/60 p-4"
      role="dialog"
      aria-modal="true"
      aria-labelledby="delete-instagram-account-title"
    >
      <section class="w-full max-w-md rounded-2xl bg-white p-5 shadow-2xl">
        <h2 id="delete-instagram-account-title" class="text-lg font-semibold">
          Удалить аккаунт?
        </h2>
        <p class="mt-2 text-sm text-slate-600">
          Аккаунт @{{ deleteCandidate.nickname }} больше нельзя будет выбрать
          для публикации.
        </p>
        <div class="mt-5 flex justify-end gap-2">
          <AppButton
            variant="secondary"
            :disabled="busy"
            @click="deleteCandidate = null"
          >
            Отмена
          </AppButton>
          <AppButton variant="danger" :disabled="busy" @click="model.remove">
            {{ busy ? 'Удаляем…' : 'Удалить' }}
          </AppButton>
        </div>
      </section>
    </div>
  </section>
</template>

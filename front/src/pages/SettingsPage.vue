<script setup lang="ts">
import WhisperSettings from '@/features/whisper-settings/ui/WhisperSettings.vue'
</script>

<template>
  <section>
    <div class="grid gap-8 lg:grid-cols-[220px_minmax(0,1fr)] lg:items-start">
      <aside class="lg:sticky lg:top-6">
        <nav
          aria-label="Разделы настроек"
          class="rounded-xl border border-slate-200 bg-white p-2 shadow-sm"
        >
          <p
            class="hidden px-3 pb-2 pt-1 text-xs font-semibold uppercase tracking-wide text-slate-400 lg:block"
          >
            Оглавление
          </p>
          <div class="px-3 py-2 text-sm font-semibold text-slate-900">
            Whisper
          </div>
          <div class="ml-3 border-l border-slate-200 pl-2">
            <a
              href="#whisper-models"
              class="block rounded-lg bg-violet-50 px-3 py-2 text-sm font-semibold text-violet-700 transition hover:bg-violet-100"
            >
              Модели
            </a>
            <a
              href="#transcription-preset"
              class="mt-1 block rounded-lg px-3 py-2 text-sm font-medium text-slate-700 transition hover:bg-slate-100 hover:text-violet-700"
            >
              Preset транскрибации
            </a>
          </div>
        </nav>
      </aside>

      <main class="min-w-0">
        <WhisperSettings />
      </main>
    </div>
  </section>
</template>

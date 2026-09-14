import { computed, onMounted, ref, watch } from 'vue'
import type { TwitchClip } from '@/entities/clip/model/types'
import type { Streamer } from '@/entities/streamer/model/types'
import { readData, readError } from '@/shared/api/http'

function dateInputValue(daysAgo = 0) {
  const date = new Date()
  date.setDate(date.getDate() - daysAgo)
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${date.getFullYear()}-${month}-${day}`
}

export function useClipSearch() {
  const streamers = ref<Streamer[]>([])
  const selected = ref('')
  const startedAt = ref(dateInputValue(6))
  const endedAt = ref(dateInputValue())
  const clips = ref<TwitchClip[]>([])
  const error = ref('')
  const busyClipId = ref('')
  const loading = ref(false)
  const previewOpen = ref(false)
  const previewInitialIndex = ref(0)
  let requestVersion = 0

  const streamerName = computed(
    () =>
      streamers.value.find((streamer) => streamer.id === selected.value)
        ?.displayName ?? '',
  )

  async function loadStreamers() {
    try {
      streamers.value = await readData<Streamer[]>(
        await fetch('/api/streamers'),
      )
    } catch (cause) {
      error.value =
        cause instanceof Error
          ? cause.message
          : 'Не удалось загрузить стримеров'
    }
  }

  async function loadClips(id: string, start: string, end: string) {
    const version = ++requestVersion
    previewOpen.value = false
    if (!id) {
      clips.value = []
      loading.value = false
      return
    }

    error.value = ''
    loading.value = true
    try {
      const query = new URLSearchParams({ startedAt: start, endedAt: end })
      const response = await fetch(`/api/streamers/${id}/clips?${query}`)
      if (!response.ok) {
        throw new Error(
          await readError(response, 'Не удалось получить Twitch clips'),
        )
      }
      const nextClips = await readData<TwitchClip[]>(response)
      if (version === requestVersion) clips.value = nextClips
    } catch (cause) {
      if (version !== requestVersion) return
      clips.value = []
      error.value =
        cause instanceof Error
          ? cause.message
          : 'Не удалось получить Twitch clips'
    } finally {
      if (version === requestVersion) loading.value = false
    }
  }

  function openPreview(index = 0) {
    previewInitialIndex.value = index
    previewOpen.value = true
  }

  async function save(clip: TwitchClip) {
    if (clip.saved || busyClipId.value || !selected.value) return
    busyClipId.value = clip.id
    try {
      const response = await fetch('/api/clips/import', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ streamerId: selected.value, clip }),
      })
      if (!response.ok) {
        error.value = await readError(response, 'Не удалось сохранить клип')
      } else {
        clip.saved = true
      }
    } catch {
      error.value = 'Не удалось сохранить клип'
    } finally {
      busyClipId.value = ''
    }
  }

  watch([selected, startedAt, endedAt], ([id, start, end]) => {
    void loadClips(id, start, end)
  })
  onMounted(() => void loadStreamers())

  return {
    streamers,
    selected,
    startedAt,
    endedAt,
    clips,
    error,
    busyClipId,
    loading,
    previewOpen,
    previewInitialIndex,
    streamerName,
    dateInputValue,
    openPreview,
    save,
  }
}

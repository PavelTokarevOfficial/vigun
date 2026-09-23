import { computed, onMounted, ref, watch } from 'vue'
import type { TwitchClip } from '@/entities/clip/model/types'
import type { Streamer } from '@/entities/streamer/model/types'
import { readData, readError } from '@/shared/api/http'
import { dateInputValue } from '@/shared/lib/dateRange'

export function useClipSearch() {
  const streamers = ref<Streamer[]>([])
  const query = ref('')
  const selected = ref('')
  const startedAt = ref(dateInputValue(6))
  const endedAt = ref(dateInputValue())
  const clips = ref<TwitchClip[]>([])
  const error = ref('')
  const busyClipId = ref('')
  const loading = ref(false)
  const resolving = ref(false)
  const subscribing = ref(false)
  const resolvedStreamerId = ref('')
  const notice = ref('')
  const previewOpen = ref(false)
  const previewInitialIndex = ref(0)
  let requestVersion = 0

  const streamerName = computed(
    () =>
      streamers.value.find((streamer) => streamer.id === selected.value)
        ?.displayName ?? '',
  )
  const foundStreamer = computed(() =>
    resolvedStreamerId.value === selected.value
      ? (streamers.value.find((streamer) => streamer.id === selected.value) ??
        null)
      : null,
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
      if (version === requestVersion) {
        clips.value = nextClips
        resolvedStreamerId.value = id
      }
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

  async function search() {
    if (resolving.value) return
    const login = query.value.trim().replace(/^@/, '').toLowerCase()
    if (!login) return

    resolving.value = true
    error.value = ''
    notice.value = ''
    resolvedStreamerId.value = ''
    try {
      let created = false
      let streamer = streamers.value.find(
        (item) => item.twitchLogin.toLowerCase() === login,
      )
      if (!streamer) {
        streamer = await readData<Streamer>(
          await fetch('/api/streamers', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ twitchLogin: login, displayName: login }),
          }),
        )
        streamers.value.push(streamer)
        created = true
      }
      query.value = streamer.twitchLogin
      selected.value = streamer.id
      await loadClips(streamer.id, startedAt.value, endedAt.value)
      if (resolvedStreamerId.value === streamer.id) {
        await loadStreamers()
      } else if (created) {
        await fetch(`/api/streamers/${streamer.id}`, { method: 'DELETE' })
        streamers.value = streamers.value.filter(
          (item) => item.id !== streamer.id,
        )
        selected.value = ''
      }
    } catch (cause) {
      error.value =
        cause instanceof Error ? cause.message : 'Не удалось найти стримера'
    } finally {
      resolving.value = false
    }
  }

  async function subscribe() {
    const streamer = foundStreamer.value
    if (!streamer || streamer.subscribed || subscribing.value) return
    subscribing.value = true
    error.value = ''
    try {
      const updated = await readData<Streamer>(
        await fetch(`/api/streamers/${streamer.id}/subscription`, {
          method: 'PATCH',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ subscribed: true }),
        }),
      )
      Object.assign(streamer, updated)
      notice.value = `Вы подписались на ${streamer.displayName}.`
    } catch (cause) {
      error.value =
        cause instanceof Error ? cause.message : 'Не удалось оформить подписку'
    } finally {
      subscribing.value = false
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

  watch([startedAt, endedAt], ([start, end]) => {
    void loadClips(selected.value, start, end)
  })
  watch(query, (value) => {
    const selectedLogin = streamers.value.find(
      (streamer) => streamer.id === selected.value,
    )?.twitchLogin
    if (
      selectedLogin &&
      value.trim().replace(/^@/, '').toLowerCase() !==
        selectedLogin.toLowerCase()
    ) {
      requestVersion += 1
      selected.value = ''
      resolvedStreamerId.value = ''
      clips.value = []
      previewOpen.value = false
      loading.value = false
    }
  })
  onMounted(() => void loadStreamers())

  return {
    streamers,
    query,
    selected,
    startedAt,
    endedAt,
    clips,
    error,
    busyClipId,
    loading,
    resolving,
    subscribing,
    notice,
    previewOpen,
    previewInitialIndex,
    streamerName,
    foundStreamer,
    search,
    subscribe,
    openPreview,
    save,
  }
}

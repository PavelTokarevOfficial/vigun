import { ref } from 'vue'
import type { Streamer } from '@/entities/streamer/model/types'
import { readData, readError } from '@/shared/api/http'

export function useStreamerManager() {
  const rows = ref<Streamer[]>([])
  const error = ref('')
  const notice = ref('')
  const busy = ref(false)
  const subscriptionBusyIDs = ref(new Set<string>())
  const addDialogOpen = ref(false)
  const nicknames = ref('')
  const editing = ref<Streamer | null>(null)
  const editNickname = ref('')

  async function load() {
    try {
      rows.value = await readData<Streamer[]>(await fetch('/api/streamers'))
    } catch (cause) {
      error.value =
        cause instanceof Error ? cause.message : 'Не удалось загрузить список'
    }
  }
  async function run(action: () => Promise<void>) {
    if (busy.value) return false
    busy.value = true
    error.value = ''
    try {
      await action()
      await load()
      return true
    } catch (cause) {
      error.value =
        cause instanceof Error ? cause.message : 'Неизвестная ошибка'
      return false
    } finally {
      busy.value = false
    }
  }
  async function request(
    source: Response | Promise<Response>,
    fallback: string,
  ) {
    const response = await source
    if (!response.ok) throw new Error(await readError(response, fallback))
    return response
  }

  function openAddDialog() {
    nicknames.value = ''
    addDialogOpen.value = true
  }
  function closeAddDialog() {
    nicknames.value = ''
    addDialogOpen.value = false
  }
  async function add() {
    const twitchLogins = nicknames.value.split(/\r?\n/)
    if (!twitchLogins.some((item) => item.trim())) return false
    notice.value = ''
    let created = 0
    const added = await run(async () => {
      const response = await request(
        fetch('/api/streamers/bulk', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ twitchLogins }),
        }),
        'Не удалось добавить стримеров',
      )
      created = (await readData<{ created: number }>(response)).created
    })
    if (added) {
      notice.value = created
        ? `Добавлено стримеров: ${created}.`
        : 'Все указанные стримеры уже были добавлены.'
      closeAddDialog()
    }
    return added
  }

  function startEdit(streamer: Streamer) {
    editing.value = streamer
    editNickname.value = streamer.twitchLogin
  }
  function cancelEdit() {
    editing.value = null
    editNickname.value = ''
  }
  async function saveEdit() {
    const streamer = editing.value
    if (!streamer || !editNickname.value.trim()) return false
    const saved = await run(async () => {
      await request(
        fetch(`/api/streamers/${streamer.id}`, {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ twitchLogin: editNickname.value }),
        }),
        'Не удалось изменить стримера',
      )
    })
    if (saved) cancelEdit()
    return saved
  }
  async function remove(streamer: Streamer) {
    return run(async () => {
      await request(
        fetch(`/api/streamers/${streamer.id}`, { method: 'DELETE' }),
        'Не удалось удалить стримера',
      )
    })
  }
  async function setPriority(streamer: Streamer, priority: number) {
    if (!Number.isInteger(priority) || priority < 0) {
      error.value = 'Приоритет должен быть неотрицательным целым числом'
      return false
    }
    return run(async () => {
      await request(
        fetch(`/api/streamers/${streamer.id}/priority`, {
          method: 'PATCH',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ priority }),
        }),
        'Не удалось изменить приоритет',
      )
    })
  }
  async function setSubscribed(streamer: Streamer, subscribed: boolean) {
    if (subscriptionBusyIDs.value.has(streamer.id)) return false
    const previous = streamer.subscribed
    streamer.subscribed = subscribed
    subscriptionBusyIDs.value = new Set([
      ...subscriptionBusyIDs.value,
      streamer.id,
    ])
    error.value = ''
    try {
      await request(
        fetch(`/api/streamers/${streamer.id}/subscription`, {
          method: 'PATCH',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ subscribed }),
        }),
        'Не удалось изменить подписку',
      )
      return true
    } catch (cause) {
      streamer.subscribed = previous
      error.value =
        cause instanceof Error ? cause.message : 'Не удалось изменить подписку'
      return false
    } finally {
      const next = new Set(subscriptionBusyIDs.value)
      next.delete(streamer.id)
      subscriptionBusyIDs.value = next
    }
  }

  return {
    rows,
    error,
    notice,
    busy,
    subscriptionBusyIDs,
    addDialogOpen,
    nicknames,
    editing,
    editNickname,
    load,
    openAddDialog,
    closeAddDialog,
    add,
    startEdit,
    cancelEdit,
    saveEdit,
    remove,
    setPriority,
    setSubscribed,
  }
}

export type StreamerManager = ReturnType<typeof useStreamerManager>

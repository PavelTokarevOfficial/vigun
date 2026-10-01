import { onBeforeUnmount, onMounted, ref } from 'vue'
import { toast } from 'vue-sonner'
import { readData, readError } from '@/shared/api/http'

export type WhisperModel = {
  id: string
  name: string
  filename: string
  type: 'Full' | 'Q5'
  sizeBytes: number
  quality: number
  speed: string
  memory: string
  description: string
  installed: boolean
  selected: boolean
  status: {
    state: 'available' | 'downloading' | 'installed' | 'error'
    downloadedBytes: number
    totalBytes: number
    error?: string
  }
}
export type WhisperSettings = {
  activeModel: string
  preset: string
  beamSize: number | null
  temperature: number | null
  maxSegmentLength: number | null
  splitOnWord: boolean | null
  initialPrompt: string
}
type Payload = { models: WhisperModel[]; settings: WhisperSettings }

export function useWhisperSettings() {
  const models = ref<WhisperModel[]>([])
  const settings = ref<WhisperSettings>({
    activeModel: 'tiny',
    preset: 'balanced',
    beamSize: null,
    temperature: null,
    maxSegmentLength: null,
    splitOnWord: null,
    initialPrompt: '',
  })
  const loading = ref(true),
    busy = ref<string | null>(null),
    error = ref('')
  let socket: WebSocket | null = null,
    reconnectTimer: number | undefined,
    fallbackTimer: number | undefined,
    reconnectDelay = 1000,
    disposed = false
  async function load() {
    try {
      const data = await readData<Payload>(await fetch('/api/settings/whisper'))
      models.value = data.models
      settings.value = data.settings
      error.value = ''
    } catch (cause) {
      error.value =
        cause instanceof Error
          ? cause.message
          : 'Не удалось загрузить настройки Whisper'
    } finally {
      loading.value = false
    }
  }
  async function action(id: string, method: string, suffix = '') {
    busy.value = id
    try {
      const response = await fetch(
        `/api/settings/whisper/models/${id}${suffix}`,
        { method },
      )
      if (!response.ok)
        throw new Error(await readError(response, 'Операция не выполнена'))
      await load()
      return true
    } catch (cause) {
      toast.error(
        cause instanceof Error ? cause.message : 'Операция не выполнена',
      )
      return false
    } finally {
      busy.value = null
    }
  }
  const download = (id: string) => action(id, 'POST', '/download')
  const select = async (id: string) => {
    if (await action(id, 'PUT', '/active'))
      toast.success('Модель Whisper выбрана.')
  }
  const remove = (id: string) => action(id, 'DELETE')
  async function save() {
    busy.value = 'settings'
    try {
      const payload = {
        ...settings.value,
        beamSize:
          typeof settings.value.beamSize === 'number'
            ? settings.value.beamSize
            : null,
        temperature:
          typeof settings.value.temperature === 'number'
            ? settings.value.temperature
            : null,
        maxSegmentLength:
          typeof settings.value.maxSegmentLength === 'number'
            ? settings.value.maxSegmentLength
            : null,
      }
      const response = await fetch('/api/settings/whisper', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      })
      if (!response.ok)
        throw new Error(
          await readError(response, 'Не удалось сохранить настройки'),
        )
      toast.success('Настройки Whisper сохранены.')
      await load()
    } catch (cause) {
      toast.error(
        cause instanceof Error
          ? cause.message
          : 'Не удалось сохранить настройки',
      )
    } finally {
      busy.value = null
    }
  }
  function connect() {
    if (disposed) return
    const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:'
    socket = new WebSocket(`${protocol}//${location.host}/api/events`)
    socket.onopen = () => {
      reconnectDelay = 1000
    }
    socket.onmessage = (event) => {
      try {
        const message = JSON.parse(String(event.data)) as { scope?: string }
        if (message.scope === 'settings') void load()
      } catch {}
    }
    socket.onerror = () => socket?.close()
    socket.onclose = () => {
      socket = null
      if (!disposed) {
        reconnectTimer = window.setTimeout(connect, reconnectDelay)
        reconnectDelay = Math.min(reconnectDelay * 2, 15000)
      }
    }
  }
  onMounted(() => {
    void load()
    connect()
    fallbackTimer = window.setInterval(load, 60000)
  })
  onBeforeUnmount(() => {
    disposed = true
    if (reconnectTimer) clearTimeout(reconnectTimer)
    if (fallbackTimer) clearInterval(fallbackTimer)
    socket?.close()
  })
  return {
    models,
    settings,
    loading,
    busy,
    error,
    download,
    select,
    remove,
    save,
    load,
  }
}

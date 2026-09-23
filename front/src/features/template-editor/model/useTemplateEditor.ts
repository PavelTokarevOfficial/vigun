import { computed, ref } from 'vue'
import type {
  Layer,
  LayerType,
  TemplateConfig,
} from '@/entities/template/model/types'

// Template config is deliberately JSON-only. JSON cloning also unwraps Vue's
// reactive Proxy objects before putting a snapshot into undo/redo history.
const copy = <T>(value: T): T => JSON.parse(JSON.stringify(value)) as T

export function useTemplateEditor(initial: TemplateConfig) {
  const draft = ref(copy(initial))
  const selectedLayerID = ref<string | null>(
    draft.value.layers.at(-1)?.id ?? null,
  )
  const history = ref<TemplateConfig[]>([copy(initial)])
  const historyIndex = ref(0)
  const selectedLayer = computed(
    () =>
      draft.value.layers.find((layer) => layer.id === selectedLayerID.value) ??
      null,
  )
  const isDirty = computed(
    () => JSON.stringify(draft.value) !== JSON.stringify(history.value[0]),
  )
  const canUndo = computed(() => historyIndex.value > 0)
  const canRedo = computed(() => historyIndex.value < history.value.length - 1)

  function apply(next: TemplateConfig) {
    const snapshot = copy(next)
    history.value = history.value.slice(0, historyIndex.value + 1)
    history.value.push(snapshot)
    historyIndex.value = history.value.length - 1
    draft.value = copy(snapshot)
  }

  function updateLayer(id: string, patch: Partial<Layer>) {
    apply({
      ...draft.value,
      layers: draft.value.layers.map((layer) =>
        layer.id === id ? { ...layer, ...patch } : layer,
      ),
    })
  }

  function updateConfig(patch: Partial<TemplateConfig>) {
    apply({ ...draft.value, ...patch })
  }

  function addLayer(type: LayerType) {
    const id = `${type}-${crypto.randomUUID().slice(0, 8)}`
    const layer: Layer = {
      id,
      name: layerName(type),
      type,
      x: 100,
      y: 100,
      width: type === 'audio' ? 0 : 600,
      height: type === 'audio' ? 0 : 300,
      visible: true,
      opacity: 1,
      fit: 'contain',
      ...(type === 'video' ? { source: 'clip' as const } : {}),
      ...(type === 'text' ? { text: 'Новый текст' } : {}),
      ...(type === 'text' ? { textSource: 'custom' as const } : {}),
      ...(type === 'blur' ? { filters: { blur: 18, brightness: -0.2 } } : {}),
      ...(type === 'color' ? { color: '#111827' } : {}),
      ...(type === 'subtitles'
        ? {
            style: {
              fontSize: 8,
              alignment: 2,
              marginV: 100,
              outline: 2,
              primaryColor: '&H00FFFFFF',
              outlineColor: '&H00000000',
            },
          }
        : {}),
    }
    apply({ ...draft.value, layers: [...draft.value.layers, layer] })
    selectedLayerID.value = id
  }

  function removeLayer(id: string) {
    const source = draft.value.layers.find((layer) => layer.id === id)
    const trackID = source ? (source.trackId ?? source.id) : id
    apply({
      ...draft.value,
      layers: draft.value.layers.filter(
        (layer) => (layer.trackId ?? layer.id) !== trackID,
      ),
    })
    selectedLayerID.value = draft.value.layers.at(-1)?.id ?? null
  }

  function duplicateLayer(id: string) {
    const source = draft.value.layers.find((layer) => layer.id === id)
    if (!source) return
    const sourceTrackID = source.trackId ?? source.id
    const duplicateTrackID = `${source.type}-${crypto.randomUUID().slice(0, 8)}`
    const duplicates = draft.value.layers
      .filter((layer) => (layer.trackId ?? layer.id) === sourceTrackID)
      .map((layer, index) => ({
        ...copy(layer),
        id:
          index === 0
            ? duplicateTrackID
            : `${source.type}-${crypto.randomUUID().slice(0, 8)}`,
        trackId: duplicateTrackID,
        name:
          index === 0
            ? `${source.name} — копия`
            : `${source.name} — копия · часть ${index + 1}`,
        x: layer.x + 30,
        y: layer.y + 30,
      }))
    apply({ ...draft.value, layers: [...draft.value.layers, ...duplicates] })
    selectedLayerID.value = duplicates[0]?.id ?? null
  }

  function moveLayer(id: string, direction: -1 | 1) {
    const source = draft.value.layers.find((layer) => layer.id === id)
    if (!source) return
    const sourceTrackID = source.trackId ?? source.id
    const order = [
      ...new Set(draft.value.layers.map((layer) => layer.trackId ?? layer.id)),
    ]
    const index = order.indexOf(sourceTrackID)
    const nextIndex = index + direction
    if (index < 0 || nextIndex < 0 || nextIndex >= order.length) return
    ;[order[index], order[nextIndex]] = [order[nextIndex], order[index]]
    const byTrack = new Map<string, Layer[]>()
    for (const layer of draft.value.layers) {
      const trackID = layer.trackId ?? layer.id
      byTrack.set(trackID, [...(byTrack.get(trackID) ?? []), layer])
    }
    apply({
      ...draft.value,
      layers: order.flatMap((trackID) => byTrack.get(trackID) ?? []),
    })
  }

  function undo() {
    if (!canUndo.value) return
    historyIndex.value -= 1
    draft.value = copy(history.value[historyIndex.value])
  }
  function redo() {
    if (!canRedo.value) return
    historyIndex.value += 1
    draft.value = copy(history.value[historyIndex.value])
  }
  function markSaved() {
    history.value = [copy(draft.value)]
    historyIndex.value = 0
  }
  function replace(config: TemplateConfig) {
    draft.value = copy(config)
    history.value = [copy(config)]
    historyIndex.value = 0
    selectedLayerID.value = config.layers.at(-1)?.id ?? null
  }

  return {
    draft,
    selectedLayerID,
    selectedLayer,
    isDirty,
    canUndo,
    canRedo,
    updateLayer,
    updateConfig,
    addLayer,
    removeLayer,
    duplicateLayer,
    moveLayer,
    undo,
    redo,
    markSaved,
    replace,
  }
}

function layerName(type: LayerType) {
  return {
    video: 'Видео',
    blur: 'Блюр',
    input_video: 'Исходный клип',
    asset_video: 'Видео-ассет',
    image: 'Изображение',
    gif: 'GIF',
    subtitles: 'Субтитры',
    text: 'Текст',
    audio: 'Аудио',
    color: 'Цветной фон',
  }[type]
}

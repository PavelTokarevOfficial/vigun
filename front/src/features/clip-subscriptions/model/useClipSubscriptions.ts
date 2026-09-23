import { computed, ref } from 'vue'
import type { SubscriptionFeed, TwitchClip } from '@/entities/clip/model/types'
import { readData, readError } from '@/shared/api/http'
import type { DateRangeValue } from '@/shared/lib/dateRange'
import type { SubscriptionSyncControls } from './syncControls'

export function useClipSubscriptions(controls: SubscriptionSyncControls) {
  const feeds = ref<SubscriptionFeed[]>([])
  const loading = ref(true)
  const error = ref('')
  const notice = ref('')
  const busyClipId = ref('')
  const previewFeed = ref<SubscriptionFeed | null>(null)
  const previewInitialIndex = ref(0)
  const previewOpen = computed(() => previewFeed.value !== null)

  async function load() {
    loading.value = true
    error.value = ''
    try {
      feeds.value = await readData<SubscriptionFeed[]>(
        await fetch('/api/subscriptions'),
      )
    } catch (cause) {
      error.value =
        cause instanceof Error ? cause.message : 'Не удалось загрузить подписки'
    } finally {
      loading.value = false
      controls.canSync.value = feeds.value.length > 0
      controls.canClear.value = feeds.value.some(
        (feed) => feed.clips.length > 0,
      )
    }
  }

  async function sync(range: DateRangeValue) {
    if (controls.syncing.value || feeds.value.length === 0) return
    controls.syncing.value = true
    error.value = ''
    notice.value = ''
    const failed: string[] = []
    let total = 0

    try {
      for (const [index, feed] of feeds.value.entries()) {
        controls.progress.value = `${index + 1} из ${feeds.value.length}`
        try {
          const response = await fetch(
            `/api/subscriptions/${feed.streamerId}/sync`,
            {
              method: 'POST',
              headers: { 'Content-Type': 'application/json' },
              body: JSON.stringify({
                startedAt: range.start,
                endedAt: range.end,
              }),
            },
          )
          if (!response.ok) {
            failed.push(feed.displayName)
            continue
          }
          total += (await readData<{ clips: number }>(response)).clips
        } catch {
          failed.push(feed.displayName)
        }
      }
      await load()
      notice.value = `Синхронизация завершена. Получено клипов: ${total}.`
      if (failed.length) {
        error.value = `Не удалось синхронизировать: ${failed.join(', ')}`
      }
    } finally {
      controls.syncing.value = false
      controls.progress.value = ''
    }
  }

  function openPreview(feed: SubscriptionFeed, index?: number) {
    const selectedClipId = index === undefined ? '' : feed.clips[index]?.id
    const previewClips = controls.hideViewed.value
      ? feed.clips.filter((clip) => !clip.viewed)
      : feed.clips
    if (!previewClips.length) return

    const selectedIndex = selectedClipId
      ? previewClips.findIndex((clip) => clip.id === selectedClipId)
      : -1
    const firstUnseen = previewClips.findIndex((clip) => !clip.viewed)
    previewInitialIndex.value = Math.max(
      0,
      selectedIndex >= 0 ? selectedIndex : firstUnseen,
    )
    previewFeed.value =
      previewClips === feed.clips ? feed : { ...feed, clips: previewClips }
  }

  function closePreview() {
    previewFeed.value = null
  }

  async function markViewed(clip: TwitchClip) {
    if (clip.viewed) return
    clip.viewed = true
    try {
      const response = await fetch(
        `/api/subscriptions/clips/${encodeURIComponent(clip.id)}/viewed`,
        { method: 'PATCH' },
      )
      if (response.ok) return
      clip.viewed = false
      error.value = await readError(
        response,
        'Не удалось отметить клип просмотренным',
      )
    } catch {
      clip.viewed = false
      error.value = 'Не удалось отметить клип просмотренным'
    }
  }

  async function save(feed: SubscriptionFeed, clip: TwitchClip) {
    if (clip.saved || busyClipId.value) return
    busyClipId.value = clip.id
    try {
      const response = await fetch('/api/clips/import', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ streamerId: feed.streamerId, clip }),
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

  async function clear() {
    if (
      controls.syncing.value ||
      controls.clearing.value ||
      !controls.canClear.value
    )
      return
    controls.clearing.value = true
    error.value = ''
    notice.value = ''
    try {
      const result = await readData<{ deleted: number }>(
        await fetch('/api/subscriptions/clips', { method: 'DELETE' }),
      )
      closePreview()
      await load()
      notice.value = `Синхронизированные клипы удалены: ${result.deleted}.`
    } catch (cause) {
      error.value =
        cause instanceof Error
          ? cause.message
          : 'Не удалось очистить синхронизированные клипы'
    } finally {
      controls.clearing.value = false
    }
  }

  function saveFromPreview(clip: TwitchClip) {
    if (previewFeed.value) void save(previewFeed.value, clip)
  }

  function dispose() {
    controls.syncing.value = false
    controls.progress.value = ''
    controls.canSync.value = false
    controls.canClear.value = false
    controls.clearing.value = false
  }

  return {
    feeds,
    loading,
    error,
    notice,
    busyClipId,
    previewFeed,
    previewInitialIndex,
    previewOpen,
    load,
    sync,
    clear,
    openPreview,
    closePreview,
    markViewed,
    save,
    saveFromPreview,
    dispose,
  }
}

export type ClipSubscriptionsModel = ReturnType<typeof useClipSubscriptions>

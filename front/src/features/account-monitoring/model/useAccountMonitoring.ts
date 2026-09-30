import { onMounted, ref } from 'vue'
import type {
  InstagramAccount,
  InstagramReel,
  InstagramReelPage,
} from '@/entities/social-account/model/types'
import { readData } from '@/shared/api/http'

export type InstagramAccountReels = {
  account: InstagramAccount
  items: InstagramReel[]
  nextCursor: string
  loading: boolean
  loadingMore: boolean
  error: string
}

export function useAccountMonitoring() {
  const groups = ref<InstagramAccountReels[]>([])
  const loading = ref(true)
  const error = ref('')
  const preview = ref<InstagramReel | null>(null)

  async function loadGroup(group: InstagramAccountReels, append = false) {
    if (append) group.loadingMore = true
    else group.loading = true
    group.error = ''
    try {
      const query =
        append && group.nextCursor
          ? `?after=${encodeURIComponent(group.nextCursor)}`
          : ''
      const page = await readData<InstagramReelPage>(
        await fetch(
          `/api/instagram-accounts/${group.account.id}/reels${query}`,
        ),
      )
      group.items = append ? [...group.items, ...page.items] : page.items
      group.nextCursor = page.nextCursor ?? ''
    } catch (cause) {
      group.error =
        cause instanceof Error ? cause.message : 'Не удалось загрузить Reels'
    } finally {
      group.loading = false
      group.loadingMore = false
    }
  }

  async function load() {
    loading.value = true
    error.value = ''
    try {
      const accounts = await readData<InstagramAccount[]>(
        await fetch('/api/instagram-accounts'),
      )
      groups.value = accounts.map((account) => ({
        account,
        items: [],
        nextCursor: '',
        loading: true,
        loadingMore: false,
        error: '',
      }))
      await Promise.all(groups.value.map((group) => loadGroup(group)))
    } catch (cause) {
      error.value =
        cause instanceof Error
          ? cause.message
          : 'Не удалось загрузить Instagram-аккаунты'
    } finally {
      loading.value = false
    }
  }

  onMounted(load)

  return { groups, loading, error, preview, load, loadGroup }
}

export type AccountMonitoringModel = ReturnType<typeof useAccountMonitoring>

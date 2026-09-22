import { type InjectionKey, type Ref, ref, shallowRef } from 'vue'
import type { DateRangeValue } from '@/shared/lib/dateRange'

export type SubscriptionSyncControls = {
  syncing: Ref<boolean>
  progress: Ref<string>
  canSync: Ref<boolean>
  canClear: Ref<boolean>
  clearing: Ref<boolean>
  hideViewed: Ref<boolean>
  requestSync: (range: DateRangeValue) => void
  requestClear: () => void
  registerSync: (handler: (range: DateRangeValue) => void) => () => void
  registerClear: (handler: () => void) => () => void
}

export function createSubscriptionSyncControls(): SubscriptionSyncControls {
  const syncing = ref(false)
  const progress = ref('')
  const canSync = ref(false)
  const canClear = ref(false)
  const clearing = ref(false)
  const hideViewed = ref(false)
  const syncHandler = shallowRef<((range: DateRangeValue) => void) | null>(null)
  const clearHandler = shallowRef<(() => void) | null>(null)

  return {
    syncing,
    progress,
    canSync,
    canClear,
    clearing,
    hideViewed,
    requestSync: (range) => syncHandler.value?.(range),
    requestClear: () => clearHandler.value?.(),
    registerSync: (nextHandler) => {
      syncHandler.value = nextHandler
      return () => {
        if (syncHandler.value === nextHandler) syncHandler.value = null
      }
    },
    registerClear: (nextHandler) => {
      clearHandler.value = nextHandler
      return () => {
        if (clearHandler.value === nextHandler) clearHandler.value = null
      }
    },
  }
}

export const subscriptionSyncControlsKey: InjectionKey<SubscriptionSyncControls> =
  Symbol('subscription-sync-controls')

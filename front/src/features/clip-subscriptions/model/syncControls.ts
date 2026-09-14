import { type InjectionKey, type Ref, ref, shallowRef } from 'vue'

export type SubscriptionSyncControls = {
  syncing: Ref<boolean>
  progress: Ref<string>
  canSync: Ref<boolean>
  requestSync: () => void
  registerSync: (handler: () => void) => () => void
}

export function createSubscriptionSyncControls(): SubscriptionSyncControls {
  const syncing = ref(false)
  const progress = ref('')
  const canSync = ref(false)
  const handler = shallowRef<(() => void) | null>(null)

  return {
    syncing,
    progress,
    canSync,
    requestSync: () => handler.value?.(),
    registerSync: (nextHandler) => {
      handler.value = nextHandler
      return () => {
        if (handler.value === nextHandler) handler.value = null
      }
    },
  }
}

export const subscriptionSyncControlsKey: InjectionKey<SubscriptionSyncControls> =
  Symbol('subscription-sync-controls')

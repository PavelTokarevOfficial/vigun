import { onMounted, ref } from 'vue'
import type {
  InstagramAccount,
  InstagramAccountForm,
} from '@/entities/social-account/model/types'
import { readData, readError } from '@/shared/api/http'

const emptyForm = (): InstagramAccountForm => ({
  nickname: '',
  instagramUserId: '',
  accessToken: '',
})

export function useInstagramAccounts() {
  const accounts = ref<InstagramAccount[]>([])
  const loading = ref(false)
  const busy = ref(false)
  const error = ref('')
  const notice = ref('')
  const tokenBusy = ref('')
  const editorOpen = ref(false)
  const editing = ref<InstagramAccount | null>(null)
  const form = ref<InstagramAccountForm>(emptyForm())
  const deleteCandidate = ref<InstagramAccount | null>(null)

  async function load() {
    loading.value = true
    error.value = ''
    try {
      accounts.value = await readData<InstagramAccount[]>(
        await fetch('/api/instagram-accounts'),
      )
    } catch (cause) {
      error.value =
        cause instanceof Error ? cause.message : 'Не удалось загрузить аккаунты'
    } finally {
      loading.value = false
    }
  }

  async function tokenAction(
    account: InstagramAccount,
    action: 'exchange-token' | 'refresh-token' | 'check-token',
  ) {
    if (tokenBusy.value) return
    tokenBusy.value = `${account.id}:${action}`
    error.value = ''
    notice.value = ''
    try {
      const response = await fetch(
        `/api/instagram-accounts/${account.id}/${action}`,
        { method: 'POST' },
      )
      if (!response.ok) {
        throw new Error(
          await readError(response, 'Операция с токеном не выполнена'),
        )
      }
      const updated = await readData<InstagramAccount>(response)
      accounts.value = accounts.value.map((item) =>
        item.id === updated.id ? updated : item,
      )
      notice.value =
        action === 'check-token'
          ? `Токен @${updated.verifiedUsername || updated.nickname} работает.`
          : action === 'exchange-token'
            ? `Для @${updated.nickname} сохранён долгоживущий токен.`
            : `Токен @${updated.nickname} обновлён.`
    } catch (cause) {
      error.value =
        cause instanceof Error
          ? cause.message
          : 'Операция с токеном не выполнена'
    } finally {
      tokenBusy.value = ''
    }
  }

  function openCreate() {
    editing.value = null
    form.value = emptyForm()
    editorOpen.value = true
  }

  function openEdit(account: InstagramAccount) {
    editing.value = account
    form.value = {
      nickname: account.nickname,
      instagramUserId: account.instagramUserId,
      accessToken: '',
    }
    editorOpen.value = true
  }

  function closeEditor() {
    if (busy.value) return
    editorOpen.value = false
  }

  async function save() {
    if (busy.value) return
    busy.value = true
    error.value = ''
    const account = editing.value
    try {
      const response = await fetch(
        account
          ? `/api/instagram-accounts/${account.id}`
          : '/api/instagram-accounts',
        {
          method: account ? 'PUT' : 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(form.value),
        },
      )
      if (!response.ok) {
        throw new Error(
          await readError(response, 'Не удалось сохранить аккаунт'),
        )
      }
      const saved = await readData<InstagramAccount>(response)
      accounts.value = account
        ? accounts.value.map((item) => (item.id === saved.id ? saved : item))
        : [...accounts.value, saved].sort((left, right) =>
            left.nickname.localeCompare(right.nickname),
          )
      editorOpen.value = false
    } catch (cause) {
      error.value =
        cause instanceof Error ? cause.message : 'Не удалось сохранить аккаунт'
    } finally {
      busy.value = false
    }
  }

  async function remove() {
    const account = deleteCandidate.value
    if (!account || busy.value) return
    busy.value = true
    error.value = ''
    try {
      const response = await fetch(`/api/instagram-accounts/${account.id}`, {
        method: 'DELETE',
      })
      if (!response.ok) {
        throw new Error(await readError(response, 'Не удалось удалить аккаунт'))
      }
      accounts.value = accounts.value.filter((item) => item.id !== account.id)
      deleteCandidate.value = null
    } catch (cause) {
      error.value =
        cause instanceof Error ? cause.message : 'Не удалось удалить аккаунт'
    } finally {
      busy.value = false
    }
  }

  onMounted(load)

  return {
    accounts,
    loading,
    busy,
    error,
    notice,
    tokenBusy,
    editorOpen,
    editing,
    form,
    deleteCandidate,
    openCreate,
    openEdit,
    closeEditor,
    save,
    remove,
    tokenAction,
  }
}

export type InstagramAccountsModel = ReturnType<typeof useInstagramAccounts>

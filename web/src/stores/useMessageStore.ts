import { create } from 'zustand'
import * as apiClient from '../lib/apiClient'
import { ApiClientError, HttpError } from '../lib/apiErrors'
import type { ListMessagesParams, MessageDetail, MessageSummary } from '../lib/apiTypes'
import { useUIStore } from './useUIStore'

type FetchStatus = 'idle' | 'loading' | 'error'
type SelectedStatus = 'idle' | 'loading' | 'error' | 'not_found'

interface MessageState {
  messages: MessageSummary[]
  total: number
  limit: number
  offset: number
  query: ListMessagesParams
  listStatus: FetchStatus
  listError: ApiClientError | null

  selected: MessageDetail | null
  selectedStatus: SelectedStatus
  selectedError: ApiClientError | null

  fetchMessages: () => Promise<void>
  setQuery: (patch: Partial<ListMessagesParams>) => void
  setPage: (offset: number) => void
  fetchMessage: (id: string) => Promise<void>
  markRead: (id: string) => Promise<void>
  deleteMessageOptimistic: (id: string) => Promise<void>
  clearAll: () => Promise<void>
  clearSelected: () => void
}

export const useMessageStore = create<MessageState>((set, get) => ({
  messages: [],
  total: 0,
  limit: 50,
  offset: 0,
  query: {},
  listStatus: 'idle',
  listError: null,

  selected: null,
  selectedStatus: 'idle',
  selectedError: null,

  fetchMessages: async () => {
    const { query, limit, offset } = get()
    set({ listStatus: 'loading', listError: null })
    try {
      const res = await apiClient.listMessages({ ...query, limit, offset })
      set({ messages: res.messages, total: res.total, limit: res.limit, offset: res.offset, listStatus: 'idle' })
    } catch (err) {
      set({ listStatus: 'error', listError: err as ApiClientError })
    }
  },

  setQuery: (patch) => {
    set((state) => ({ query: { ...state.query, ...patch }, offset: 0 }))
    void get().fetchMessages()
  },

  setPage: (offset) => {
    set({ offset })
    void get().fetchMessages()
  },

  fetchMessage: async (id) => {
    set({ selectedStatus: 'loading', selectedError: null, selected: null })
    try {
      const detail = await apiClient.getMessage(id)
      set({ selected: detail, selectedStatus: 'idle' })
    } catch (err) {
      if (err instanceof HttpError && err.code === 'message_not_found') {
        set({ selectedStatus: 'not_found', selectedError: err })
      } else {
        set({ selectedStatus: 'error', selectedError: err as ApiClientError })
      }
    }
  },

  markRead: async (id) => {
    try {
      await apiClient.markRead(id)
      set((state) => ({
        messages: state.messages.map((m) => (m.id === id ? { ...m, read: true } : m)),
        selected: state.selected && state.selected.id === id ? { ...state.selected, read: true } : state.selected,
      }))
    } catch {
      // Non-critical (read state is advisory); silently ignore failures
      // rather than interrupting the user with a toast for this.
    }
  },

  deleteMessageOptimistic: async (id) => {
    const { messages, total } = get()
    const index = messages.findIndex((m) => m.id === id)
    if (index === -1) return
    const removed = messages[index]

    set({
      messages: [...messages.slice(0, index), ...messages.slice(index + 1)],
      total: Math.max(0, total - 1),
    })

    try {
      await apiClient.deleteMessage(id)
    } catch {
      set((state) => {
        const restored = [...state.messages]
        restored.splice(index, 0, removed)
        return { messages: restored, total: state.total + 1 }
      })
      useUIStore.getState().pushToast('danger', 'Failed to delete message')
    }
  },

  clearAll: async () => {
    try {
      await apiClient.clearMessages()
      set({ messages: [], total: 0, offset: 0 })
    } catch {
      useUIStore.getState().pushToast('danger', 'Failed to clear messages')
    }
  },

  clearSelected: () => set({ selected: null, selectedStatus: 'idle', selectedError: null }),
}))

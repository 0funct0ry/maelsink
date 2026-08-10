import { create } from 'zustand'
import * as apiClient from '../lib/apiClient'
import { ApiClientError, HttpError } from '../lib/apiErrors'
import type { ListMessagesParams, MessageDetail, MessageSummary, Stats, TagCount } from '../lib/apiTypes'
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

  /** IDs recently added via a realtime message.created event, for a brief
   * highlight animation in the row (M7.0). Cleared automatically. */
  highlightIds: Set<string>

  /** Sidebar counts/tags (M6.1's Sidebar, kept live here since M7.0 so the
   * mailbox counts and tag list stay in sync with realtime events without
   * the Sidebar owning its own separate fetch-once-on-mount state). */
  sidebarStats: Stats | null
  sidebarTags: TagCount[]

  fetchMessages: () => Promise<void>
  fetchSidebarData: () => Promise<void>
  setQuery: (patch: Partial<ListMessagesParams>) => void
  setPage: (offset: number) => void
  fetchMessage: (id: string) => Promise<void>
  markRead: (id: string, read?: boolean) => Promise<void>
  updateTagsOptimistic: (id: string, add: string[], remove: string[]) => Promise<void>
  deleteMessageOptimistic: (id: string) => Promise<void>
  clearAll: () => Promise<void>
  clearSelected: () => void

  /** Realtime event handlers (M7.0), driven by wsClient via InboxScreen —
   * these never make network requests themselves. */
  applyMessageCreated: (summary: MessageSummary) => void
  applyMessageDeleted: (id: string) => void
  applyMessagesCleared: () => void
  applyMessageTagsUpdated: (payload: { id: string; tags: string[] }) => void
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

  highlightIds: new Set(),

  sidebarStats: null,
  sidebarTags: [],

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

  fetchSidebarData: async () => {
    try {
      const stats = await apiClient.getStats()
      set({ sidebarStats: stats })
    } catch {
      // Non-fatal: the storage card and mailbox counts just don't update.
    }
    try {
      const tags = await apiClient.getTags()
      set({ sidebarTags: tags })
    } catch {
      // Non-fatal: the tags nav group just doesn't update.
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

  markRead: async (id, read = true) => {
    try {
      await apiClient.markRead(id, read)
      set((state) => ({
        messages: state.messages.map((m) => (m.id === id ? { ...m, read } : m)),
        selected: state.selected && state.selected.id === id ? { ...state.selected, read } : state.selected,
      }))
      void get().fetchSidebarData()
    } catch {
      useUIStore.getState().pushToast('danger', `Failed to mark message as ${read ? 'read' : 'unread'}`)
    }
  },

  updateTagsOptimistic: async (id, add, remove) => {
    const { messages, selected } = get()
    const index = messages.findIndex((m) => m.id === id)
    const prevTags = index !== -1 ? messages[index].tags : selected?.id === id ? selected.tags : undefined
    if (prevTags === undefined) return

    const nextTags = prevTags.filter((t) => !remove.includes(t)).concat(add.filter((t) => !prevTags.includes(t)))

    set((state) => ({
      messages: state.messages.map((m) => (m.id === id ? { ...m, tags: nextTags } : m)),
      selected: state.selected && state.selected.id === id ? { ...state.selected, tags: nextTags } : state.selected,
    }))

    try {
      const updated = await apiClient.updateMessageTags(id, { add, remove })
      set((state) => ({
        messages: state.messages.map((m) => (m.id === id ? { ...m, tags: updated.tags } : m)),
        selected: state.selected && state.selected.id === id ? { ...state.selected, tags: updated.tags } : state.selected,
      }))
      void get().fetchSidebarData()
    } catch {
      set((state) => ({
        messages: state.messages.map((m) => (m.id === id ? { ...m, tags: prevTags } : m)),
        selected: state.selected && state.selected.id === id ? { ...state.selected, tags: prevTags } : state.selected,
      }))
      useUIStore.getState().pushToast('danger', 'Failed to update tags')
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
      void get().fetchSidebarData()
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
      useUIStore.getState().pushToast('success', 'All messages cleared')
      void get().fetchSidebarData()
    } catch {
      useUIStore.getState().pushToast('danger', 'Failed to clear messages')
    }
  },

  clearSelected: () => set({ selected: null, selectedStatus: 'idle', selectedError: null }),

  applyMessageCreated: (summary) => {
    set((state) => ({
      messages: [summary, ...state.messages],
      total: state.total + 1,
      highlightIds: new Set(state.highlightIds).add(summary.id),
    }))
    setTimeout(() => {
      set((state) => {
        if (!state.highlightIds.has(summary.id)) return {}
        const next = new Set(state.highlightIds)
        next.delete(summary.id)
        return { highlightIds: next }
      })
    }, HIGHLIGHT_DURATION_MS)
    void get().fetchSidebarData()
  },

  applyMessageDeleted: (id) => {
    set((state) => {
      const index = state.messages.findIndex((m) => m.id === id)
      const messages = index === -1 ? state.messages : [...state.messages.slice(0, index), ...state.messages.slice(index + 1)]
      const total = index === -1 ? state.total : Math.max(0, state.total - 1)

      if (state.selected?.id === id) {
        return { messages, total, selected: null, selectedStatus: 'not_found', selectedError: null }
      }
      return { messages, total }
    })
    void get().fetchSidebarData()
  },

  applyMessageTagsUpdated: (payload) => {
    set((state) => ({
      messages: state.messages.map((m) => (m.id === payload.id ? { ...m, tags: payload.tags } : m)),
      selected: state.selected && state.selected.id === payload.id ? { ...state.selected, tags: payload.tags } : state.selected,
    }))
    void get().fetchSidebarData()
  },

  applyMessagesCleared: () => {
    set((state) => ({
      messages: [],
      total: 0,
      offset: 0,
      ...(state.selected ? { selected: null, selectedStatus: 'not_found' as const, selectedError: null } : {}),
    }))
    void get().fetchSidebarData()
  },
}))

const HIGHLIGHT_DURATION_MS = 2000

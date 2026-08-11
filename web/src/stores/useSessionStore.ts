import { create } from 'zustand'
import * as apiClient from '../lib/apiClient'
import { ApiClientError, HttpError } from '../lib/apiErrors'
import type { ListSessionsParams, SessionDetail, SessionSummary } from '../lib/apiTypes'
import { useUIStore } from './useUIStore'

type FetchStatus = 'idle' | 'loading' | 'error'
type SelectedStatus = 'idle' | 'loading' | 'error' | 'not_found'

interface SessionState {
  sessions: SessionSummary[]
  total: number
  limit: number
  offset: number
  query: ListSessionsParams
  listStatus: FetchStatus
  listError: ApiClientError | null

  selected: SessionDetail | null
  selectedStatus: SelectedStatus
  selectedError: ApiClientError | null

  fetchSessions: () => Promise<void>
  setQuery: (patch: Partial<ListSessionsParams>) => void
  setPage: (offset: number) => void
  fetchSession: (id: string) => Promise<void>
  clearSelected: () => void
  deleteSessionOptimistic: (id: string) => Promise<void>
  clearAll: () => Promise<void>

  /** Realtime event handlers (M8.4/M8.4a), driven by wsClient via AppShell —
   * these never make network requests themselves, mirroring
   * useMessageStore's applyMessageCreated/applyMessageDeleted. */
  applySessionStarted: (payload: { id: string; client_ip: string; started_at: string }) => void
  applySessionCompleted: (payload: { id: string; status: SessionSummary['status']; message_id: string | null }) => void
  /** Live-tails an in-progress session's transcript (M8.4a): appends one
   * line to `selected` as soon as it's captured server-side, so the detail
   * screen doesn't have to wait for the session to finish. A no-op when
   * the event isn't for the currently-open session. */
  applySessionLine: (payload: { session_id: string; direction: 'C' | 'S'; line: string; position: number }) => void
  applySessionDeleted: (id: string) => void
  applySessionsCleared: () => void
}

export const useSessionStore = create<SessionState>((set, get) => ({
  sessions: [],
  total: 0,
  limit: 50,
  offset: 0,
  query: {},
  listStatus: 'idle',
  listError: null,

  selected: null,
  selectedStatus: 'idle',
  selectedError: null,

  fetchSessions: async () => {
    const { query, limit, offset } = get()
    set({ listStatus: 'loading', listError: null })
    try {
      const res = await apiClient.listSessions({ ...query, limit, offset })
      set({ sessions: res.sessions, total: res.total, limit: res.limit, offset: res.offset, listStatus: 'idle' })
    } catch (err) {
      set({ listStatus: 'error', listError: err as ApiClientError })
    }
  },

  setQuery: (patch) => {
    set((state) => ({ query: { ...state.query, ...patch }, offset: 0 }))
    void get().fetchSessions()
  },

  setPage: (offset) => {
    set({ offset })
    void get().fetchSessions()
  },

  fetchSession: async (id) => {
    set({ selectedStatus: 'loading', selectedError: null, selected: null })
    try {
      const detail = await apiClient.getSession(id)
      set({ selected: detail, selectedStatus: 'idle' })
    } catch (err) {
      if (err instanceof HttpError && err.code === 'session_not_found') {
        set({ selectedStatus: 'not_found', selectedError: err })
      } else {
        set({ selectedStatus: 'error', selectedError: err as ApiClientError })
      }
    }
  },

  clearSelected: () => set({ selected: null, selectedStatus: 'idle', selectedError: null }),

  deleteSessionOptimistic: async (id) => {
    const { sessions, total } = get()
    const index = sessions.findIndex((s) => s.id === id)
    if (index === -1) return
    const removed = sessions[index]

    set({
      sessions: [...sessions.slice(0, index), ...sessions.slice(index + 1)],
      total: Math.max(0, total - 1),
    })

    try {
      await apiClient.deleteSession(id)
    } catch {
      set((state) => {
        const restored = [...state.sessions]
        restored.splice(index, 0, removed)
        return { sessions: restored, total: state.total + 1 }
      })
      useUIStore.getState().pushToast('danger', 'Failed to delete session')
    }
  },

  clearAll: async () => {
    try {
      await apiClient.clearSessions()
      set({ sessions: [], total: 0, offset: 0 })
      useUIStore.getState().pushToast('success', 'All sessions cleared')
    } catch {
      useUIStore.getState().pushToast('danger', 'Failed to clear sessions')
    }
  },

  // session.started drives the list's live row — a summary row with no
  // ended_at/status yet, which applySessionCompleted then fills in.
  applySessionStarted: (payload) => {
    set((state) => ({
      sessions: [
        {
          id: payload.id,
          client_ip: payload.client_ip,
          client_helo: '',
          started_at: payload.started_at,
          ended_at: null,
          status: '',
          message_id: null,
        },
        ...state.sessions,
      ],
      total: state.total + 1,
    }))
  },

  applySessionCompleted: (payload) => {
    set((state) => ({
      sessions: state.sessions.map((s) =>
        s.id === payload.id ? { ...s, status: payload.status, message_id: payload.message_id } : s,
      ),
      selected:
        state.selected && state.selected.id === payload.id
          ? { ...state.selected, status: payload.status, message_id: payload.message_id }
          : state.selected,
    }))
  },

  applySessionLine: (payload) => {
    set((state) => {
      if (!state.selected || state.selected.id !== payload.session_id) return {}
      // Dedupe by position rather than assuming strict append order: the
      // initial GET /api/v1/sessions/:id fetch and the live event stream
      // race by nature (the fetch may already include lines that also
      // arrive over the socket, or vice versa), so a line already present
      // at this position is a no-op rather than a duplicate row.
      if (state.selected.transcript.some((l) => l.position === payload.position)) return {}
      const transcript = [
        ...state.selected.transcript,
        { direction: payload.direction, line: payload.line, position: payload.position },
      ].sort((a, b) => a.position - b.position)
      return { selected: { ...state.selected, transcript } }
    })
  },

  applySessionDeleted: (id) => {
    set((state) => {
      const index = state.sessions.findIndex((s) => s.id === id)
      const sessions =
        index === -1 ? state.sessions : [...state.sessions.slice(0, index), ...state.sessions.slice(index + 1)]
      const total = index === -1 ? state.total : Math.max(0, state.total - 1)

      if (state.selected?.id === id) {
        return { sessions, total, selected: null, selectedStatus: 'not_found' as const, selectedError: null }
      }
      return { sessions, total }
    })
  },

  applySessionsCleared: () => {
    set((state) => ({
      sessions: [],
      total: 0,
      offset: 0,
      ...(state.selected ? { selected: null, selectedStatus: 'not_found' as const, selectedError: null } : {}),
    }))
  },
}))

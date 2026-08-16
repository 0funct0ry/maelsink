import { create } from 'zustand'
import { health, type HealthResponse } from '../lib/composeApi'

const POLL_INTERVAL_MS = 5000

export type ConnectionStatus = 'red' | 'yellow' | 'green'

interface ConnectionState {
  status: ConnectionStatus
  lastChecked: number | null
  lastError: string | null
  poll: () => Promise<void>
  startPolling: () => () => void
}

export const useConnectionStore = create<ConnectionState>((set, get) => ({
  status: 'red',
  lastChecked: null,
  lastError: null,

  poll: async () => {
    try {
      const resp: HealthResponse = await health()
      set({ status: resp.status, lastChecked: Date.now(), lastError: resp.error ?? null })
    } catch (err) {
      set({ status: 'red', lastChecked: Date.now(), lastError: err instanceof Error ? err.message : String(err) })
    }
  },

  // Returns a cleanup function so callers (typically a top-level useEffect)
  // can stop polling on unmount.
  startPolling: () => {
    void get().poll()
    const id = window.setInterval(() => {
      void get().poll()
    }, POLL_INTERVAL_MS)
    return () => window.clearInterval(id)
  },
}))

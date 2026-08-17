import { create } from 'zustand'
import { listJobs, openJobStream, type JobSnapshot } from '../lib/composeJobsApi'

// In-memory only — no localStorage persistence, since job history is
// explicitly not meant to survive a compose restart (SPEC.md §7.7.7).
interface JobsState {
  jobs: JobSnapshot[]
  upsert: (tick: JobSnapshot) => void
  refresh: () => Promise<void>
  // trackJob opens jobId's WS progress stream and keeps upserting ticks
  // into the store for the life of the connection. Called from the store
  // itself (not a component's useEffect) so the stream survives the modal
  // that started the job being closed/unmounted — the Recent Jobs list
  // keeps updating in place after the modal auto-closes.
  trackJob: (jobId: string) => void
  clearAll: () => void
}

// Open stream-close functions, keyed by jobId — module-level (not store
// state) since it holds non-serializable callbacks, not data to render.
// clearAll uses this to stop any in-flight streams so a cleared job can't
// resurrect itself in the list on its next tick.
const openStreams = new Map<string, () => void>()

export const useJobsStore = create<JobsState>((set, get) => ({
  jobs: [],

  upsert: (tick) => {
    const jobs = get().jobs
    const idx = jobs.findIndex((j) => j.jobId === tick.jobId)
    if (idx === -1) {
      set({ jobs: [tick, ...jobs] })
    } else {
      const next = jobs.slice()
      next[idx] = tick
      set({ jobs: next })
    }
  },

  refresh: async () => {
    const jobs = await listJobs()
    // Newest first, matching how upsert prepends newly-started jobs.
    set({ jobs: jobs.slice().sort((a, b) => (a.startedAt < b.startedAt ? 1 : -1)) })
  },

  trackJob: (jobId) => {
    const close = openJobStream(jobId, (tick) => {
      get().upsert(tick)
      if (tick.status !== 'running') {
        openStreams.delete(jobId)
      }
    })
    openStreams.set(jobId, close)
  },

  clearAll: () => {
    for (const close of openStreams.values()) close()
    openStreams.clear()
    set({ jobs: [] })
  },
}))

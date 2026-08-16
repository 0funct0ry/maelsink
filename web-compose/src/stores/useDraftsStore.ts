import { create } from 'zustand'
import { readDrafts, writeDrafts, type Draft, type DraftsMap } from '../lib/draftsStorage'

interface DraftsState {
  drafts: DraftsMap
  saveDraft: (name: string, draft: Draft) => void
  deleteDraft: (name: string) => void
}

export const useDraftsStore = create<DraftsState>((set, get) => ({
  drafts: readDrafts(),
  saveDraft: (name, draft) => {
    const next = { ...get().drafts, [name]: draft }
    set({ drafts: next })
    writeDrafts(next)
  },
  deleteDraft: (name) => {
    const next = { ...get().drafts }
    delete next[name]
    set({ drafts: next })
    writeDrafts(next)
  },
}))

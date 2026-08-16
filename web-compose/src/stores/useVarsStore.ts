import { create } from 'zustand'
import { readVars, writeVars, type VarsMap } from '../lib/varsStorage'

interface VarsState {
  vars: VarsMap
  setVar: (key: string, value: string) => void
  deleteVar: (key: string) => void
}

export const useVarsStore = create<VarsState>((set, get) => ({
  vars: readVars(),
  setVar: (key, value) => {
    const next = { ...get().vars, [key]: value }
    set({ vars: next })
    writeVars(next)
  },
  deleteVar: (key) => {
    const next = { ...get().vars }
    delete next[key]
    set({ vars: next })
    writeVars(next)
  },
}))

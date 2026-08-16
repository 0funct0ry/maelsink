import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { useVarsStore } from './useVarsStore'
import { readVars } from '../lib/varsStorage'

// jsdom's localStorage is flaky under this test runner's sandboxing (per
// web/src/stores/useUIStore.test.ts's own comment) — stub it with a plain
// in-memory implementation.
function makeMemoryStorage(): Storage {
  const data = new Map<string, string>()
  return {
    getItem: (key: string) => (data.has(key) ? data.get(key)! : null),
    setItem: (key: string, value: string) => data.set(key, value),
    removeItem: (key: string) => data.delete(key),
    clear: () => data.clear(),
    key: (index: number) => Array.from(data.keys())[index] ?? null,
    get length() {
      return data.size
    },
  } as Storage
}

beforeEach(() => {
  vi.stubGlobal('localStorage', makeMemoryStorage())
  useVarsStore.setState({ vars: {} })
})

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('useVarsStore', () => {
  it('adds a var', () => {
    useVarsStore.getState().setVar('foo', 'bar')
    expect(useVarsStore.getState().vars).toEqual({ foo: 'bar' })
  })

  it('edits a var', () => {
    useVarsStore.getState().setVar('foo', 'bar')
    useVarsStore.getState().setVar('foo', 'baz')
    expect(useVarsStore.getState().vars.foo).toBe('baz')
  })

  it('deletes a var', () => {
    useVarsStore.getState().setVar('foo', 'bar')
    useVarsStore.getState().deleteVar('foo')
    expect(useVarsStore.getState().vars).toEqual({})
  })

  it('persists to localStorage and survives a simulated reload', () => {
    useVarsStore.getState().setVar('foo', 'bar')
    // Simulate a reload: read persisted storage directly rather than
    // relying on in-memory state.
    expect(readVars()).toEqual({ foo: 'bar' })
  })
})

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { useDraftsStore } from './useDraftsStore'
import { readDrafts } from '../lib/draftsStorage'

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
  useDraftsStore.setState({ drafts: {} })
})

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('useDraftsStore', () => {
  it('saves a draft into state and persists it', () => {
    useDraftsStore.getState().saveDraft('welcome', { format: 'eml', content: 'Subject: hi' })
    expect(useDraftsStore.getState().drafts['welcome']).toEqual({ format: 'eml', content: 'Subject: hi' })
    expect(readDrafts()['welcome']).toEqual({ format: 'eml', content: 'Subject: hi' })
  })

  it('overwrites an existing draft with the same name', () => {
    useDraftsStore.getState().saveDraft('welcome', { format: 'eml', content: 'first' })
    useDraftsStore.getState().saveDraft('welcome', { format: 'json', content: 'second' })
    expect(useDraftsStore.getState().drafts['welcome']).toEqual({ format: 'json', content: 'second' })
    expect(Object.keys(useDraftsStore.getState().drafts)).toHaveLength(1)
  })

  it('deletes a draft from state and persistence', () => {
    useDraftsStore.getState().saveDraft('welcome', { format: 'eml', content: 'hi' })
    useDraftsStore.getState().deleteDraft('welcome')
    expect(useDraftsStore.getState().drafts['welcome']).toBeUndefined()
    expect(readDrafts()['welcome']).toBeUndefined()
  })

  it('deleting a nonexistent draft is a no-op', () => {
    useDraftsStore.getState().saveDraft('welcome', { format: 'eml', content: 'hi' })
    useDraftsStore.getState().deleteDraft('missing')
    expect(Object.keys(useDraftsStore.getState().drafts)).toEqual(['welcome'])
  })
})

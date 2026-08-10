import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { deleteSavedSearch, listSavedSearches, saveSearch } from './savedSearches'

// jsdom's localStorage is flaky under this test runner's sandboxing (see
// useUIStore.test.ts), so stub it with a plain in-memory implementation.
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
})

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('savedSearches', () => {
  it('starts empty', () => {
    expect(listSavedSearches()).toEqual([])
  })

  it('saves and lists a search', () => {
    saveSearch('bugs', { subject: 'bug' })
    expect(listSavedSearches()).toEqual([{ name: 'bugs', query: { subject: 'bug' } }])
  })

  it('overwrites an existing search with the same name', () => {
    saveSearch('bugs', { subject: 'bug' })
    saveSearch('bugs', { subject: 'crash' })
    expect(listSavedSearches()).toEqual([{ name: 'bugs', query: { subject: 'crash' } }])
  })

  it('ignores a blank name', () => {
    saveSearch('   ', { subject: 'bug' })
    expect(listSavedSearches()).toEqual([])
  })

  it('deletes a saved search by name', () => {
    saveSearch('bugs', { subject: 'bug' })
    saveSearch('smoke', { tag: ['smoke'] })
    deleteSavedSearch('bugs')
    expect(listSavedSearches()).toEqual([{ name: 'smoke', query: { tag: ['smoke'] } }])
  })

  it('tolerates corrupted storage', () => {
    window.localStorage.setItem('maelsink.saved_searches', 'not json')
    expect(listSavedSearches()).toEqual([])
  })

  it('normalizes a legacy single-tag string into an array', () => {
    window.localStorage.setItem(
      'maelsink.saved_searches',
      JSON.stringify([{ name: 'legacy', query: { tag: 'smoke' } }]),
    )
    expect(listSavedSearches()).toEqual([{ name: 'legacy', query: { tag: ['smoke'] } }])
  })
})

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { useThemeStore } from './useThemeStore'

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
  document.documentElement.removeAttribute('data-theme')
})

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('useThemeStore', () => {
  it('applies a data-theme attribute for an explicit choice', () => {
    useThemeStore.getState().setTheme('dark')
    expect(document.documentElement.getAttribute('data-theme')).toBe('dark')
    expect(useThemeStore.getState().theme).toBe('dark')
  })

  it('removes the data-theme attribute for system', () => {
    useThemeStore.getState().setTheme('light')
    useThemeStore.getState().setTheme('system')
    expect(document.documentElement.hasAttribute('data-theme')).toBe(false)
  })

  it('persists the choice to localStorage', () => {
    useThemeStore.getState().setTheme('dark')
    expect(window.localStorage.getItem('maelsink-compose-theme')).toBe('dark')
  })
})

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { useUIStore } from './useUIStore'

// jsdom's localStorage is flaky under this test runner's sandboxing, so
// stub it with a plain in-memory implementation for these tests.
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
  useUIStore.setState({
    modal: null,
    toasts: [],
    authToken: null,
    authRequired: false,
    pendingRetry: null,
  })
})

describe('modal', () => {
  it('opens and closes a confirm modal', () => {
    const onConfirm = () => {}
    useUIStore.getState().openConfirm({ title: 'Clear all?', body: 'This deletes everything.', onConfirm })
    expect(useUIStore.getState().modal).toMatchObject({ kind: 'confirm', title: 'Clear all?' })

    useUIStore.getState().closeModal()
    expect(useUIStore.getState().modal).toBeNull()
  })
})

describe('toasts', () => {
  it('pushes and dismisses toasts independently', () => {
    useUIStore.getState().pushToast('danger', 'first')
    useUIStore.getState().pushToast('success', 'second')
    const [first, second] = useUIStore.getState().toasts
    expect(useUIStore.getState().toasts).toHaveLength(2)

    useUIStore.getState().dismissToast(first.id)
    expect(useUIStore.getState().toasts).toEqual([second])
  })
})

describe('auth', () => {
  it('setAuthToken persists to localStorage and clears authRequired', () => {
    useUIStore.getState().setAuthRequired(true)
    useUIStore.getState().setAuthToken('secret')

    expect(useUIStore.getState().authToken).toBe('secret')
    expect(useUIStore.getState().authRequired).toBe(false)
    expect(window.localStorage.getItem('maelsink_api_key')).toBe('secret')
  })

  it('setAuthToken invokes the pending retry exactly once', () => {
    let calls = 0
    useUIStore.getState().setAuthRequired(true, () => {
      calls += 1
    })
    useUIStore.getState().setAuthToken('secret')
    expect(calls).toBe(1)
    expect(useUIStore.getState().pendingRetry).toBeNull()
  })

  it('clearAuthToken removes the in-memory token and localStorage', () => {
    useUIStore.getState().setAuthToken('secret')
    useUIStore.getState().clearAuthToken()
    expect(useUIStore.getState().authToken).toBeNull()
    expect(window.localStorage.getItem('maelsink_api_key')).toBeNull()
  })

  it('setAuthRequired(false) clears any pending retry', () => {
    useUIStore.getState().setAuthRequired(true, () => {})
    useUIStore.getState().setAuthRequired(false)
    expect(useUIStore.getState().pendingRetry).toBeNull()
  })
})

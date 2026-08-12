import { render, screen, fireEvent } from '@testing-library/react'
import SecurityCard from './SecurityCard'
import { useUIStore } from '../../stores/useUIStore'

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
  useUIStore.setState({ authToken: null, modal: null })
})

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('SecurityCard', () => {
  it('disables the clear button when no key is stored', () => {
    render(<SecurityCard />)
    expect(screen.getByRole('button', { name: 'Clear stored key' })).toBeDisabled()
  })

  it('opens a confirmation before clearing a stored key', () => {
    useUIStore.getState().setAuthToken('secret-key')
    render(<SecurityCard />)

    fireEvent.click(screen.getByRole('button', { name: 'Clear stored key' }))
    expect(useUIStore.getState().modal).toMatchObject({ kind: 'confirm', danger: true })
    expect(useUIStore.getState().authToken).toBe('secret-key')
  })

  it('removes the stored key once the confirmation is accepted', () => {
    useUIStore.getState().setAuthToken('secret-key')
    render(<SecurityCard />)

    fireEvent.click(screen.getByRole('button', { name: 'Clear stored key' }))
    useUIStore.getState().modal?.onConfirm()

    expect(useUIStore.getState().authToken).toBeNull()
    expect(window.localStorage.getItem('maelsink_api_key')).toBeNull()
  })
})

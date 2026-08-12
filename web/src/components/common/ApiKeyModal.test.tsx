import { render, screen, fireEvent } from '@testing-library/react'
import ApiKeyModal from './ApiKeyModal'
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
})

afterEach(() => {
  vi.unstubAllGlobals()
  useUIStore.setState({ authToken: null, authRequired: false, pendingRetry: null })
})

describe('ApiKeyModal', () => {
  it('is hidden when authRequired is false', () => {
    render(<ApiKeyModal />)
    expect(screen.queryByText('Authentication required')).not.toBeInTheDocument()
  })

  it('is shown when authRequired is true', () => {
    useUIStore.setState({ authRequired: true })
    render(<ApiKeyModal />)
    expect(screen.getByText('Authentication required')).toBeInTheDocument()
    expect(screen.getByLabelText('Enter API Key')).toBeInTheDocument()
  })

  it('shows a validation message and does not set the token on empty submit', () => {
    useUIStore.setState({ authRequired: true })
    render(<ApiKeyModal />)
    fireEvent.click(screen.getByRole('button', { name: 'Submit' }))
    expect(screen.getByText('API key is required.')).toBeInTheDocument()
    expect(useUIStore.getState().authToken).toBeNull()
    expect(useUIStore.getState().authRequired).toBe(true)
  })

  it('sets the token and hides the modal on valid submit', () => {
    useUIStore.setState({ authRequired: true })
    render(<ApiKeyModal />)
    fireEvent.change(screen.getByLabelText('Enter API Key'), { target: { value: 'secret-key' } })
    fireEvent.click(screen.getByRole('button', { name: 'Submit' }))
    expect(useUIStore.getState().authToken).toBe('secret-key')
    expect(useUIStore.getState().authRequired).toBe(false)
  })

  it('disables browser autocomplete on the key input', () => {
    useUIStore.setState({ authRequired: true })
    render(<ApiKeyModal />)
    expect(screen.getByLabelText('Enter API Key')).toHaveAttribute('autocomplete', 'off')
  })

  it('does not close on Escape (non-dismissable)', () => {
    useUIStore.setState({ authRequired: true })
    render(<ApiKeyModal />)
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(useUIStore.getState().authRequired).toBe(true)
    expect(screen.getByText('Authentication required')).toBeInTheDocument()
  })

  it('fires a pending retry after a successful submit', () => {
    const retry = vi.fn()
    useUIStore.getState().setAuthRequired(true, retry)
    render(<ApiKeyModal />)
    fireEvent.change(screen.getByLabelText('Enter API Key'), { target: { value: 'secret-key' } })
    fireEvent.click(screen.getByRole('button', { name: 'Submit' }))
    expect(retry).toHaveBeenCalledTimes(1)
  })
})

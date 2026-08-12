import { render, screen, fireEvent } from '@testing-library/react'
import ThemeToggle from './ThemeToggle'
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
  document.documentElement.removeAttribute('data-theme')
  useUIStore.setState({ theme: 'system' })
})

afterEach(() => {
  vi.unstubAllGlobals()
  document.documentElement.removeAttribute('data-theme')
})

describe('ThemeToggle', () => {
  it('marks the active theme as pressed', () => {
    render(<ThemeToggle />)
    expect(screen.getByRole('button', { name: 'System' })).toHaveAttribute('aria-pressed', 'true')
    expect(screen.getByRole('button', { name: 'Light' })).toHaveAttribute('aria-pressed', 'false')
  })

  it('switching to dark sets data-theme and persists the choice', () => {
    render(<ThemeToggle />)
    fireEvent.click(screen.getByRole('button', { name: 'Dark' }))

    expect(useUIStore.getState().theme).toBe('dark')
    expect(document.documentElement.getAttribute('data-theme')).toBe('dark')
    expect(window.localStorage.getItem('maelsink_theme')).toBe('dark')
  })

  it('switching to system removes the data-theme attribute', () => {
    render(<ThemeToggle />)
    fireEvent.click(screen.getByRole('button', { name: 'Dark' }))
    fireEvent.click(screen.getByRole('button', { name: 'System' }))

    expect(document.documentElement.hasAttribute('data-theme')).toBe(false)
  })
})

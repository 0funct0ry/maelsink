import { fireEvent, render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import Sidebar from './Sidebar'

const SIDEBAR_COLLAPSED_KEY = 'maelsink-compose-sidebar-collapsed'

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
})

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('Sidebar', () => {
  it('renders a nav link for every screen', () => {
    render(
      <MemoryRouter>
        <Sidebar />
      </MemoryRouter>,
    )
    for (const label of ['Message List', 'Vars', 'Composer', 'API Explorer', 'Jobs']) {
      expect(screen.getByText(label)).toBeInTheDocument()
    }
  })

  it('starts expanded by default and shows the product name', () => {
    render(
      <MemoryRouter>
        <Sidebar />
      </MemoryRouter>,
    )
    expect(screen.getByText('maelsink compose')).toBeInTheDocument()
  })

  it('collapses on toggle click and persists the choice', () => {
    render(
      <MemoryRouter>
        <Sidebar />
      </MemoryRouter>,
    )
    fireEvent.click(screen.getByTitle('Collapse sidebar'))
    expect(screen.queryByText('maelsink compose')).not.toBeInTheDocument()
    expect(window.localStorage.getItem(SIDEBAR_COLLAPSED_KEY)).toBe('1')
  })

  it('starts collapsed when a prior choice was persisted', () => {
    window.localStorage.setItem(SIDEBAR_COLLAPSED_KEY, '1')
    render(
      <MemoryRouter>
        <Sidebar />
      </MemoryRouter>,
    )
    expect(screen.queryByText('maelsink compose')).not.toBeInTheDocument()
    fireEvent.click(screen.getByTitle('Expand sidebar'))
    expect(screen.getByText('maelsink compose')).toBeInTheDocument()
    expect(window.localStorage.getItem(SIDEBAR_COLLAPSED_KEY)).toBe('0')
  })
})

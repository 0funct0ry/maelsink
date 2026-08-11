import { render, screen, fireEvent, act } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import SessionsScreen from './SessionsScreen'
import { useSessionStore } from '../../stores/useSessionStore'
import { useUIStore } from '../../stores/useUIStore'
import type { SessionSummary } from '../../lib/apiTypes'

function summary(id: string, overrides: Partial<SessionSummary> = {}): SessionSummary {
  return {
    id,
    client_ip: '10.0.0.1',
    client_helo: 'client.example.com',
    started_at: '2026-01-01T00:00:00Z',
    ended_at: '2026-01-01T00:00:05Z',
    status: 'completed',
    message_id: null,
    ...overrides,
  }
}

function renderScreen() {
  return render(
    <MemoryRouter initialEntries={['/sessions']}>
      <Routes>
        <Route path="/sessions" element={<SessionsScreen />} />
        <Route path="/sessions/:id" element={<div>Session Detail Route</div>} />
        <Route path="/messages/:id" element={<div>Message Detail Route</div>} />
      </Routes>
    </MemoryRouter>,
  )
}

beforeEach(() => {
  useSessionStore.setState({
    sessions: [],
    total: 0,
    limit: 50,
    offset: 0,
    query: {},
    listStatus: 'idle',
    listError: null,
    fetchSessions: vi.fn(),
    setPage: vi.fn(),
  } as Partial<ReturnType<typeof useSessionStore.getState>>)
})

describe('SessionsScreen', () => {
  it('fetches sessions on mount', () => {
    const fetchSessions = vi.fn()
    useSessionStore.setState({ fetchSessions })
    renderScreen()
    expect(fetchSessions).toHaveBeenCalledTimes(1)
  })

  it('shows the "Sessions" header title', () => {
    renderScreen()
    expect(screen.getByRole('heading', { name: 'Sessions' })).toBeInTheDocument()
  })

  it('shows an empty state when there are no sessions', () => {
    renderScreen()
    expect(screen.getByText('No SMTP sessions yet.')).toBeInTheDocument()
  })

  it('renders a row per session with client IP and status', () => {
    useSessionStore.setState({ sessions: [summary('s1'), summary('s2', { status: 'aborted' })], total: 2 })
    renderScreen()
    expect(screen.getByText('completed')).toBeInTheDocument()
    expect(screen.getByText('aborted')).toBeInTheDocument()
    expect(screen.getAllByText(/10\.0\.0\.1/)).toHaveLength(2)
  })

  it('shows "In progress" for a session with no status yet', () => {
    useSessionStore.setState({ sessions: [summary('s1', { status: '' })], total: 1 })
    renderScreen()
    expect(screen.getByText('In progress')).toBeInTheDocument()
  })

  it('navigates to the session detail route on row click', () => {
    useSessionStore.setState({ sessions: [summary('s1')], total: 1 })
    renderScreen()
    fireEvent.click(screen.getByText('completed'))
    expect(screen.getByText('Session Detail Route')).toBeInTheDocument()
  })

  it('links to the produced message when message_id is set, without triggering row navigation', () => {
    useSessionStore.setState({ sessions: [summary('s1', { message_id: 'msg1' })], total: 1 })
    renderScreen()
    fireEvent.click(screen.getByText('View message'))
    expect(screen.getByText('Message Detail Route')).toBeInTheDocument()
  })

  it('opens a confirm dialog when the per-row delete icon is clicked, without navigating', () => {
    useSessionStore.setState({ sessions: [summary('s1')], total: 1 })
    renderScreen()
    fireEvent.click(screen.getByLabelText('Delete session s1'))
    expect(useUIStore.getState().modal).toMatchObject({ kind: 'confirm', danger: true })
    expect(screen.queryByText('Session Detail Route')).not.toBeInTheDocument()
  })

  it('calls deleteSessionOptimistic when the row-delete confirm is invoked', () => {
    useSessionStore.setState({ sessions: [summary('s1')], total: 1 })
    const deleteSessionOptimistic = vi.fn().mockResolvedValue(undefined)
    useSessionStore.setState({ deleteSessionOptimistic })
    renderScreen()
    fireEvent.click(screen.getByLabelText('Delete session s1'))
    act(() => {
      useUIStore.getState().modal?.onConfirm()
    })
    expect(deleteSessionOptimistic).toHaveBeenCalledWith('s1')
  })

  it('opens a confirm dialog when "Clear all sessions" is clicked', () => {
    renderScreen()
    fireEvent.click(screen.getByLabelText('Clear all sessions'))
    expect(useUIStore.getState().modal).toMatchObject({ kind: 'confirm', danger: true })
  })

  it('calls clearAll when the clear-all confirm is invoked', () => {
    const clearAll = vi.fn().mockResolvedValue(undefined)
    useSessionStore.setState({ clearAll })
    renderScreen()
    fireEvent.click(screen.getByLabelText('Clear all sessions'))
    act(() => {
      useUIStore.getState().modal?.onConfirm()
    })
    expect(clearAll).toHaveBeenCalledTimes(1)
  })
})

import { render, screen, fireEvent, act } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import SessionDetailScreen from './SessionDetailScreen'
import { useSessionStore } from '../../stores/useSessionStore'
import type { SessionDetail } from '../../lib/apiTypes'

const session: SessionDetail = {
  id: 's1',
  client_ip: '10.0.0.1',
  client_helo: 'client.example.com',
  started_at: '2026-01-01T00:00:00Z',
  ended_at: '2026-01-01T00:00:05Z',
  status: 'completed',
  message_id: 'm1',
  transcript: [
    { direction: 'S', line: '220 maelsink.test ESMTP maelsink', position: 0 },
    { direction: 'C', line: 'EHLO client.example.com', position: 1 },
    { direction: 'C', line: 'AUTH PLAIN [REDACTED]', position: 2 },
  ],
}

function renderScreen(id = 's1') {
  return render(
    <MemoryRouter initialEntries={[`/sessions/${id}`]}>
      <Routes>
        <Route path="/sessions/:id" element={<SessionDetailScreen />} />
        <Route path="/sessions" element={<div>Sessions List Route</div>} />
        <Route path="/messages/:id" element={<div>Message Detail Route</div>} />
      </Routes>
    </MemoryRouter>,
  )
}

beforeEach(() => {
  useSessionStore.setState({
    selected: null,
    selectedStatus: 'idle',
    selectedError: null,
    fetchSession: vi.fn(),
    clearSelected: vi.fn(),
  } as Partial<ReturnType<typeof useSessionStore.getState>>)
})

describe('SessionDetailScreen', () => {
  it('shows a loading skeleton while fetching', () => {
    useSessionStore.setState({ selectedStatus: 'loading' })
    renderScreen()
    expect(screen.queryByText('SMTP Session')).not.toBeInTheDocument()
  })

  it('shows a not-found state when the session no longer exists', () => {
    useSessionStore.setState({ selectedStatus: 'not_found' })
    renderScreen()
    expect(screen.getByText('Session not found')).toBeInTheDocument()
  })

  it('renders session header fields', () => {
    useSessionStore.setState({ selected: session, selectedStatus: 'idle' })
    renderScreen()
    expect(screen.getByText('10.0.0.1')).toBeInTheDocument()
    expect(screen.getByText('client.example.com')).toBeInTheDocument()
    expect(screen.getByText('completed')).toBeInTheDocument()
  })

  it('renders the transcript with C:/S: prefixes, including a redacted AUTH line rendered verbatim', () => {
    useSessionStore.setState({ selected: session, selectedStatus: 'idle' })
    renderScreen()
    expect(screen.getByText('220 maelsink.test ESMTP maelsink')).toBeInTheDocument()
    expect(screen.getByText('EHLO client.example.com')).toBeInTheDocument()
    expect(screen.getByText('AUTH PLAIN [REDACTED]')).toBeInTheDocument()
    // No un-redaction affordance: the raw stored line is all that's shown.
    expect(screen.queryByText(/PLAIN [^[]REDACTED/)).not.toBeInTheDocument()
  })

  it('links to the produced message via "View message"', () => {
    useSessionStore.setState({ selected: session, selectedStatus: 'idle' })
    renderScreen()
    fireEvent.click(screen.getByText('View message'))
    expect(screen.getByText('Message Detail Route')).toBeInTheDocument()
  })

  it('does not show a "View message" link when no message was produced', () => {
    useSessionStore.setState({ selected: { ...session, message_id: null }, selectedStatus: 'idle' })
    renderScreen()
    expect(screen.queryByText('View message')).not.toBeInTheDocument()
  })

  it('navigates back to the sessions list', () => {
    useSessionStore.setState({ selected: session, selectedStatus: 'idle' })
    renderScreen()
    fireEvent.click(screen.getByText('Back to sessions'))
    expect(screen.getByText('Sessions List Route')).toBeInTheDocument()
  })

  it('shows "In progress" for a session that has not finished yet', () => {
    useSessionStore.setState({ selected: { ...session, status: '', ended_at: null }, selectedStatus: 'idle' })
    renderScreen()
    expect(screen.getByText('In progress')).toBeInTheDocument()
  })

  it('live-tails new transcript lines via a realtime session.line event (M8.4a), without a refetch', () => {
    useSessionStore.setState({ selected: { ...session, status: '', ended_at: null }, selectedStatus: 'idle' })
    renderScreen()
    expect(screen.queryByText('QUIT')).not.toBeInTheDocument()

    act(() => {
      useSessionStore.getState().applySessionLine({ session_id: 's1', direction: 'C', line: 'QUIT', position: 3 })
    })

    expect(screen.getByText('QUIT')).toBeInTheDocument()
  })
})

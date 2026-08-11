import { afterEach, describe, expect, it, vi } from 'vitest'
import { useSessionStore } from './useSessionStore'
import { useUIStore } from './useUIStore'
import { HttpError } from '../lib/apiErrors'
import * as apiClient from '../lib/apiClient'
import type { SessionDetail, SessionSummary } from '../lib/apiTypes'

vi.mock('../lib/apiClient')

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

afterEach(() => {
  vi.clearAllMocks()
  useSessionStore.setState({
    sessions: [],
    total: 0,
    limit: 50,
    offset: 0,
    query: {},
    listStatus: 'idle',
    listError: null,
    selected: null,
    selectedStatus: 'idle',
    selectedError: null,
  })
  useUIStore.setState({ toasts: [] })
})

describe('fetchSessions', () => {
  it('populates sessions on success', async () => {
    vi.mocked(apiClient.listSessions).mockResolvedValue({
      sessions: [summary('a'), summary('b')],
      total: 2,
      limit: 50,
      offset: 0,
    })

    await useSessionStore.getState().fetchSessions()

    expect(useSessionStore.getState().sessions).toHaveLength(2)
    expect(useSessionStore.getState().listStatus).toBe('idle')
  })

  it('sets listStatus to error on failure', async () => {
    vi.mocked(apiClient.listSessions).mockRejectedValue(new Error('boom'))

    await useSessionStore.getState().fetchSessions()

    expect(useSessionStore.getState().listStatus).toBe('error')
  })
})

describe('fetchSession', () => {
  const detail: SessionDetail = { ...summary('s1'), transcript: [] }

  it('populates selected on success', async () => {
    vi.mocked(apiClient.getSession).mockResolvedValue(detail)

    await useSessionStore.getState().fetchSession('s1')

    expect(useSessionStore.getState().selected).toEqual(detail)
    expect(useSessionStore.getState().selectedStatus).toBe('idle')
  })

  it('sets selectedStatus to not_found on a session_not_found HttpError', async () => {
    vi.mocked(apiClient.getSession).mockRejectedValue(new HttpError(404, 'session_not_found', 'no such session'))

    await useSessionStore.getState().fetchSession('missing')

    expect(useSessionStore.getState().selectedStatus).toBe('not_found')
  })
})

describe('applySessionStarted', () => {
  it('inserts a new in-progress row at the top of the list', () => {
    useSessionStore.setState({ sessions: [summary('existing')], total: 1 })

    useSessionStore
      .getState()
      .applySessionStarted({ id: 'new1', client_ip: '10.0.0.9', started_at: '2026-01-01T01:00:00Z' })

    const state = useSessionStore.getState()
    expect(state.sessions).toHaveLength(2)
    expect(state.sessions[0].id).toBe('new1')
    expect(state.sessions[0].status).toBe('')
    expect(state.total).toBe(2)
  })
})

describe('applySessionCompleted', () => {
  it('updates the matching row in the list and the selected detail', () => {
    useSessionStore.setState({
      sessions: [summary('s1', { status: '' })],
      selected: { ...summary('s1', { status: '' }), transcript: [] },
    })

    useSessionStore.getState().applySessionCompleted({ id: 's1', status: 'completed', message_id: 'm1' })

    const state = useSessionStore.getState()
    expect(state.sessions[0].status).toBe('completed')
    expect(state.sessions[0].message_id).toBe('m1')
    expect(state.selected?.status).toBe('completed')
    expect(state.selected?.message_id).toBe('m1')
  })
})

describe('applySessionLine', () => {
  it('appends a line to the open session in wire order', () => {
    useSessionStore.setState({
      selected: {
        ...summary('s1', { status: '' }),
        transcript: [{ direction: 'S', line: '220 maelsink.test ESMTP maelsink', position: 0 }],
      },
    })

    useSessionStore.getState().applySessionLine({ session_id: 's1', direction: 'C', line: 'EHLO client', position: 1 })

    const transcript = useSessionStore.getState().selected?.transcript
    expect(transcript).toEqual([
      { direction: 'S', line: '220 maelsink.test ESMTP maelsink', position: 0 },
      { direction: 'C', line: 'EHLO client', position: 1 },
    ])
  })

  it('is a no-op when the event is for a session that is not currently open', () => {
    const selected = { ...summary('s1', { status: '' }), transcript: [] }
    useSessionStore.setState({ selected })

    useSessionStore.getState().applySessionLine({ session_id: 'other', direction: 'C', line: 'EHLO client', position: 0 })

    expect(useSessionStore.getState().selected).toEqual(selected)
  })

  it('is a no-op when no session is open', () => {
    useSessionStore.setState({ selected: null })

    useSessionStore.getState().applySessionLine({ session_id: 's1', direction: 'C', line: 'EHLO client', position: 0 })

    expect(useSessionStore.getState().selected).toBeNull()
  })

  it('dedupes a line already present at the same position instead of duplicating it', () => {
    useSessionStore.setState({
      selected: {
        ...summary('s1', { status: '' }),
        transcript: [{ direction: 'S', line: '220 maelsink.test ESMTP maelsink', position: 0 }],
      },
    })

    useSessionStore
      .getState()
      .applySessionLine({ session_id: 's1', direction: 'S', line: '220 maelsink.test ESMTP maelsink', position: 0 })

    expect(useSessionStore.getState().selected?.transcript).toHaveLength(1)
  })

  it('sorts out-of-order arrivals by position', () => {
    useSessionStore.setState({
      selected: { ...summary('s1', { status: '' }), transcript: [{ direction: 'S', line: 'first', position: 0 }] },
    })

    useSessionStore.getState().applySessionLine({ session_id: 's1', direction: 'C', line: 'third', position: 2 })
    useSessionStore.getState().applySessionLine({ session_id: 's1', direction: 'C', line: 'second', position: 1 })

    expect(useSessionStore.getState().selected?.transcript.map((l) => l.line)).toEqual(['first', 'second', 'third'])
  })
})

describe('deleteSessionOptimistic', () => {
  it('removes the row immediately and keeps it removed on success', async () => {
    useSessionStore.setState({ sessions: [summary('a'), summary('b')], total: 2 })
    vi.mocked(apiClient.deleteSession).mockResolvedValue(undefined)

    await useSessionStore.getState().deleteSessionOptimistic('a')

    expect(useSessionStore.getState().sessions.map((s) => s.id)).toEqual(['b'])
    expect(useSessionStore.getState().total).toBe(1)
  })

  it('rolls back and toasts on failure', async () => {
    useSessionStore.setState({ sessions: [summary('a'), summary('b')], total: 2 })
    vi.mocked(apiClient.deleteSession).mockRejectedValue(new HttpError(500, 'internal_error', 'boom'))

    await useSessionStore.getState().deleteSessionOptimistic('a')

    expect(useSessionStore.getState().sessions.map((s) => s.id)).toEqual(['a', 'b'])
    expect(useSessionStore.getState().total).toBe(2)
    expect(useUIStore.getState().toasts).toHaveLength(1)
    expect(useUIStore.getState().toasts[0].variant).toBe('danger')
  })

  it('is a no-op for an id not in the current list', async () => {
    useSessionStore.setState({ sessions: [summary('a')], total: 1 })

    await useSessionStore.getState().deleteSessionOptimistic('missing')

    expect(apiClient.deleteSession).not.toHaveBeenCalled()
    expect(useSessionStore.getState().sessions).toHaveLength(1)
  })
})

describe('clearAll', () => {
  it('clears the list and toasts success', async () => {
    useSessionStore.setState({ sessions: [summary('a'), summary('b')], total: 2, offset: 1 })
    vi.mocked(apiClient.clearSessions).mockResolvedValue(undefined)

    await useSessionStore.getState().clearAll()

    expect(useSessionStore.getState().sessions).toEqual([])
    expect(useSessionStore.getState().total).toBe(0)
    expect(useSessionStore.getState().offset).toBe(0)
    expect(useUIStore.getState().toasts[0].variant).toBe('success')
  })

  it('toasts danger on failure without clearing the list', async () => {
    useSessionStore.setState({ sessions: [summary('a')], total: 1 })
    vi.mocked(apiClient.clearSessions).mockRejectedValue(new HttpError(500, 'internal_error', 'boom'))

    await useSessionStore.getState().clearAll()

    expect(useSessionStore.getState().sessions).toHaveLength(1)
    expect(useUIStore.getState().toasts[0].variant).toBe('danger')
  })
})

describe('applySessionDeleted', () => {
  it('removes the row from the list', () => {
    useSessionStore.setState({ sessions: [summary('a'), summary('b')], total: 2 })

    useSessionStore.getState().applySessionDeleted('a')

    expect(useSessionStore.getState().sessions.map((s) => s.id)).toEqual(['b'])
    expect(useSessionStore.getState().total).toBe(1)
  })

  it('marks the open detail as not_found when its session is deleted', () => {
    useSessionStore.setState({
      sessions: [summary('a')],
      total: 1,
      selected: { ...summary('a'), transcript: [] },
      selectedStatus: 'idle',
    })

    useSessionStore.getState().applySessionDeleted('a')

    expect(useSessionStore.getState().selected).toBeNull()
    expect(useSessionStore.getState().selectedStatus).toBe('not_found')
  })
})

describe('applySessionsCleared', () => {
  it('empties the list and resets pagination', () => {
    useSessionStore.setState({ sessions: [summary('a'), summary('b')], total: 2, offset: 1 })

    useSessionStore.getState().applySessionsCleared()

    expect(useSessionStore.getState().sessions).toEqual([])
    expect(useSessionStore.getState().total).toBe(0)
    expect(useSessionStore.getState().offset).toBe(0)
  })

  it('marks an open detail as not_found', () => {
    useSessionStore.setState({
      sessions: [summary('a')],
      selected: { ...summary('a'), transcript: [] },
      selectedStatus: 'idle',
    })

    useSessionStore.getState().applySessionsCleared()

    expect(useSessionStore.getState().selected).toBeNull()
    expect(useSessionStore.getState().selectedStatus).toBe('not_found')
  })
})

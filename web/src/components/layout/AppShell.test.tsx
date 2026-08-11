import { render, screen, fireEvent, act } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import AppShell from './AppShell'
import { useMessageStore } from '../../stores/useMessageStore'
import { useUIStore } from '../../stores/useUIStore'
import * as apiClient from '../../lib/apiClient'
import * as uiApiClient from '../../lib/uiApiClient'
import * as wsClient from '../../lib/wsClient'

vi.mock('../../lib/apiClient')
vi.mock('../../lib/uiApiClient')
vi.mock('../../lib/wsClient', async () => {
  const actual = await vi.importActual<typeof wsClient>('../../lib/wsClient')
  return { ...actual, connectWs: vi.fn() }
})

function renderShell(initialEntries: string[] = ['/']) {
  return render(
    <MemoryRouter initialEntries={initialEntries}>
      <AppShell>
        <Routes>
          <Route path="/" element={<div>content</div>} />
          <Route path="/messages/:id" element={<div>message detail</div>} />
        </Routes>
      </AppShell>
    </MemoryRouter>,
  )
}

describe('AppShell', () => {
  beforeEach(() => {
    vi.mocked(apiClient.getStats).mockRejectedValue(new Error('offline'))
    vi.mocked(apiClient.listTags).mockRejectedValue(new Error('offline'))
    vi.mocked(uiApiClient.getInfo).mockRejectedValue(new Error('offline'))
    useUIStore.setState({ modal: null, wsStatus: 'connecting' })
    useMessageStore.setState({ messages: [], total: 0 })
    vi.mocked(wsClient.connectWs).mockClear()
    vi.mocked(wsClient.connectWs).mockReturnValue({ close: vi.fn() })
  })

  it('fetches messages once on mount', () => {
    const fetchMessages = vi.fn()
    useMessageStore.setState({ fetchMessages })
    renderShell()
    expect(fetchMessages).toHaveBeenCalledTimes(1)
  })

  it('renders a ConfirmDialog driven by useUIStore.modal', () => {
    renderShell()
    expect(screen.queryByText('Clear all messages?')).not.toBeInTheDocument()

    act(() => {
      useUIStore.getState().openConfirm({
        title: 'Clear all messages?',
        body: 'This cannot be undone.',
        danger: true,
        onConfirm: () => {},
      })
    })

    expect(screen.getByText('Clear all messages?')).toBeInTheDocument()
  })

  it('invokes the modal onConfirm and closes on confirm click', () => {
    renderShell()
    const onConfirm = vi.fn()
    act(() => {
      useUIStore.getState().openConfirm({ title: 'Sure?', body: 'Body', onConfirm })
    })

    fireEvent.click(screen.getByRole('button', { name: 'Confirm' }))

    expect(onConfirm).toHaveBeenCalledTimes(1)
    expect(useUIStore.getState().modal).toBeNull()
  })

  it('Escape navigates from a message detail route back to the Inbox', () => {
    renderShell(['/messages/abc123'])
    expect(screen.getByText('message detail')).toBeInTheDocument()

    fireEvent.keyDown(window, { key: 'Escape' })

    expect(screen.getByText('content')).toBeInTheDocument()
  })

  it('Escape does nothing when already on the Inbox route', () => {
    renderShell(['/'])
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(screen.getByText('content')).toBeInTheDocument()
  })

  it('Escape does not navigate while a confirm modal is open', () => {
    renderShell(['/messages/abc123'])
    act(() => {
      useUIStore.getState().openConfirm({ title: 'Sure?', body: 'Body', onConfirm: () => {} })
    })

    fireEvent.keyDown(window, { key: 'Escape' })

    expect(screen.getByText('message detail')).toBeInTheDocument()
  })

  it('connects to the WebSocket on mount and disconnects on unmount', () => {
    const close = vi.fn()
    vi.mocked(wsClient.connectWs).mockReturnValue({ close })

    const { unmount } = renderShell()
    expect(wsClient.connectWs).toHaveBeenCalledTimes(1)

    unmount()
    expect(close).toHaveBeenCalledTimes(1)
  })

  it('stays connected across a route change (Inbox -> Detail)', () => {
    const close = vi.fn()
    vi.mocked(wsClient.connectWs).mockReturnValue({ close })

    renderShell(['/messages/abc123'])
    expect(wsClient.connectWs).toHaveBeenCalledTimes(1)

    fireEvent.keyDown(window, { key: 'Escape' })

    expect(screen.getByText('content')).toBeInTheDocument()
    expect(close).not.toHaveBeenCalled()
    expect(wsClient.connectWs).toHaveBeenCalledTimes(1)
  })

  it('prepends a message when a message.created frame arrives, from any route', () => {
    useMessageStore.setState({ messages: [], total: 0 })
    let handleFrame: ((frame: wsClient.WsFrame) => void) | undefined
    vi.mocked(wsClient.connectWs).mockImplementation((opts) => {
      handleFrame = opts.onEvent
      return { close: vi.fn() }
    })

    renderShell(['/messages/abc123'])
    act(() => {
      handleFrame?.({
        type: 'message.created',
        payload: {
          id: 'msg_new',
          from: 'a@example.com',
          to: ['b@example.com'],
          cc: [],
          bcc: [],
          subject: 'New mail',
          size_bytes: 10,
          has_attachments: false,
          attachment_count: 0,
          received_at: '2026-01-01T00:00:00Z',
          parse_warning: false,
          read: false,
          tags: [],
          preview: '',
        },
      })
    })

    expect(useMessageStore.getState().messages.map((m) => m.id)).toEqual(['msg_new'])
  })

  it('removes a message when a message.deleted frame arrives', () => {
    useMessageStore.setState({
      messages: [
        {
          id: 'msg_old',
          from: 'a@example.com',
          to: ['b@example.com'],
          cc: [],
          bcc: [],
          subject: 'Old mail',
          size_bytes: 10,
          has_attachments: false,
          attachment_count: 0,
          received_at: '2026-01-01T00:00:00Z',
          parse_warning: false,
          read: false,
          tags: [],
          preview: '',
        },
      ],
      total: 1,
    })
    let handleFrame: ((frame: wsClient.WsFrame) => void) | undefined
    vi.mocked(wsClient.connectWs).mockImplementation((opts) => {
      handleFrame = opts.onEvent
      return { close: vi.fn() }
    })

    renderShell()
    act(() => {
      handleFrame?.({ type: 'message.deleted', payload: { id: 'msg_old' } })
    })

    expect(useMessageStore.getState().messages).toHaveLength(0)
  })

  it('clears messages when a messages.cleared frame arrives', () => {
    useMessageStore.setState({
      messages: [
        {
          id: 'msg_old',
          from: 'a@example.com',
          to: ['b@example.com'],
          cc: [],
          bcc: [],
          subject: 'Old mail',
          size_bytes: 10,
          has_attachments: false,
          attachment_count: 0,
          received_at: '2026-01-01T00:00:00Z',
          parse_warning: false,
          read: false,
          tags: [],
          preview: '',
        },
      ],
      total: 1,
    })
    let handleFrame: ((frame: wsClient.WsFrame) => void) | undefined
    vi.mocked(wsClient.connectWs).mockImplementation((opts) => {
      handleFrame = opts.onEvent
      return { close: vi.fn() }
    })

    renderShell()
    act(() => {
      handleFrame?.({ type: 'messages.cleared', payload: {} })
    })

    expect(useMessageStore.getState().messages).toHaveLength(0)
    expect(useMessageStore.getState().total).toBe(0)
  })

  it('refetches sidebar tags when a tag.renamed frame arrives', () => {
    useMessageStore.setState({ sidebarTags: [{ name: 'old', color: 'indigo', count: 2, last_used: null }] })
    let handleFrame: ((frame: wsClient.WsFrame) => void) | undefined
    vi.mocked(wsClient.connectWs).mockImplementation((opts) => {
      handleFrame = opts.onEvent
      return { close: vi.fn() }
    })
    vi.mocked(apiClient.listTags).mockResolvedValue([{ name: 'new', color: 'indigo', count: 2, last_used: null }])

    renderShell()
    act(() => {
      handleFrame?.({ type: 'tag.renamed', payload: { name: 'old', new_name: 'new', merged: false } })
    })

    // The old name is dropped optimistically; fetchSidebarData (async, not
    // awaited here) reconciles the final list including the new name.
    expect(useMessageStore.getState().sidebarTags.map((t) => t.name)).toEqual([])
  })

  it('patches a tag color in place when a tag.recolored frame arrives', () => {
    useMessageStore.setState({ sidebarTags: [{ name: 'x', color: 'indigo', count: 1, last_used: null }] })
    let handleFrame: ((frame: wsClient.WsFrame) => void) | undefined
    vi.mocked(wsClient.connectWs).mockImplementation((opts) => {
      handleFrame = opts.onEvent
      return { close: vi.fn() }
    })

    renderShell()
    act(() => {
      handleFrame?.({ type: 'tag.recolored', payload: { name: 'x', color: 'amber' } })
    })

    expect(useMessageStore.getState().sidebarTags[0].color).toBe('amber')
  })

  it('triggers a sidebar refetch when a tag.created frame arrives', () => {
    useMessageStore.setState({ sidebarTags: [] })
    let handleFrame: ((frame: wsClient.WsFrame) => void) | undefined
    vi.mocked(wsClient.connectWs).mockImplementation((opts) => {
      handleFrame = opts.onEvent
      return { close: vi.fn() }
    })
    vi.mocked(apiClient.listTags).mockResolvedValue([{ name: 'fresh', color: 'cyan', count: 0, last_used: null }])

    renderShell()
    act(() => {
      handleFrame?.({ type: 'tag.created', payload: { name: 'fresh', color: 'cyan' } })
    })

    expect(apiClient.listTags).toHaveBeenCalled()
  })

  it('removes a tag from the sidebar list when a tag.deleted frame arrives', () => {
    useMessageStore.setState({ sidebarTags: [{ name: 'gone', color: 'indigo', count: 1, last_used: null }] })
    let handleFrame: ((frame: wsClient.WsFrame) => void) | undefined
    vi.mocked(wsClient.connectWs).mockImplementation((opts) => {
      handleFrame = opts.onEvent
      return { close: vi.fn() }
    })

    renderShell()
    act(() => {
      handleFrame?.({ type: 'tag.deleted', payload: { name: 'gone' } })
    })

    expect(useMessageStore.getState().sidebarTags.map((t) => t.name)).toEqual([])
  })

  it('updates useUIStore.wsStatus on connection status changes', () => {
    vi.mocked(wsClient.connectWs).mockImplementation((opts) => {
      opts.onStatusChange?.('reconnecting')
      return { close: vi.fn() }
    })

    renderShell()

    expect(useUIStore.getState().wsStatus).toBe('reconnecting')
  })
})

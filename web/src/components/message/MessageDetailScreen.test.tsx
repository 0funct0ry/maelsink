import { render, screen, fireEvent, waitFor, act } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import MessageDetailScreen from './MessageDetailScreen'
import { useMessageStore } from '../../stores/useMessageStore'
import { useUIStore } from '../../stores/useUIStore'
import * as apiClient from '../../lib/apiClient'
import type { MessageDetail } from '../../lib/apiTypes'

vi.mock('../../lib/apiClient', async () => {
  const actual = await vi.importActual<typeof import('../../lib/apiClient')>('../../lib/apiClient')
  return {
    ...actual,
    getRaw: vi.fn(),
    getAttachmentBlob: vi.fn(),
    exportMessage: vi.fn(),
  }
})

const message: MessageDetail = {
  id: 'm1',
  from: 'a@example.com',
  to: ['b@example.com'],
  cc: [],
  bcc: [],
  subject: 'Hello world',
  size_bytes: 1200,
  has_attachments: false,
  attachment_count: 0,
  received_at: '2024-01-01T00:00:00Z',
  parse_warning: false,
  read: false,
  tags: [],
  preview: '',
  headers: [],
  text_body: 'body',
  html_body: '<p>hi</p>',
  attachments: [],
  raw_size_bytes: 1200,
}

function renderScreen(id = 'm1') {
  return render(
    <MemoryRouter initialEntries={[`/messages/${id}`]}>
      <Routes>
        <Route path="/messages/:id" element={<MessageDetailScreen />} />
        <Route path="/" element={<div>Inbox Home</div>} />
      </Routes>
    </MemoryRouter>,
  )
}

beforeEach(() => {
  vi.stubGlobal('URL', {
    ...URL,
    createObjectURL: vi.fn(() => 'blob:mock'),
    revokeObjectURL: vi.fn(),
  })
  useMessageStore.setState({
    selected: null,
    selectedStatus: 'idle',
    selectedError: null,
    fetchMessage: vi.fn(),
    markRead: vi.fn(),
    deleteMessageOptimistic: vi.fn(),
    clearSelected: vi.fn(),
  } as Partial<ReturnType<typeof useMessageStore.getState>>)
})

describe('MessageDetailScreen', () => {
  it('shows a loading skeleton while fetching', () => {
    useMessageStore.setState({ selectedStatus: 'loading' })
    renderScreen()
    expect(screen.queryByText('Hello world')).not.toBeInTheDocument()
  })

  it('shows DeletedMessageState for not_found', () => {
    useMessageStore.setState({ selectedStatus: 'not_found' })
    renderScreen()
    expect(screen.getByText(/This message no longer exists/)).toBeInTheDocument()
  })

  it('shows DeletedMessageState when a realtime message.deleted event arrives for the open message (M7.0)', () => {
    useMessageStore.setState({ selected: message, selectedStatus: 'idle' })
    renderScreen()
    expect(screen.getByText('Hello world')).toBeInTheDocument()

    act(() => {
      useMessageStore.getState().applyMessageDeleted(message.id)
    })

    expect(screen.getByText(/This message no longer exists/)).toBeInTheDocument()
  })

  it('shows a distinct generic error panel for error status', () => {
    useMessageStore.setState({ selectedStatus: 'error' })
    renderScreen()
    expect(screen.getByText(/Something went wrong loading this message/)).toBeInTheDocument()
  })

  it('renders message details when populated and marks it read', async () => {
    const markRead = vi.fn()
    useMessageStore.setState({ selected: message, selectedStatus: 'idle', markRead })
    renderScreen()
    expect(screen.getByText('Hello world')).toBeInTheDocument()
    await waitFor(() => expect(markRead).toHaveBeenCalledWith('m1'))
  })

  it('renders tag chips when the message has tags', async () => {
    useMessageStore.setState({
      selected: { ...message, tags: ['signup-flow', 'smoke'] },
      selectedStatus: 'idle',
      markRead: vi.fn(),
    })
    renderScreen()
    expect(await screen.findByText('signup-flow')).toBeInTheDocument()
    expect(screen.getByText('smoke')).toBeInTheDocument()
  })

  it('renders no tag chips when the message has no tags', async () => {
    useMessageStore.setState({ selected: message, selectedStatus: 'idle', markRead: vi.fn() })
    renderScreen()
    await screen.findByText('Hello world')
    expect(screen.queryByText('signup-flow')).not.toBeInTheDocument()
  })

  it('opens the tag edit modal from the "Edit tags" button', async () => {
    useMessageStore.setState({ selected: { ...message, tags: ['smoke'] }, selectedStatus: 'idle', markRead: vi.fn() })
    renderScreen()
    fireEvent.click(await screen.findByText('Edit tags'))
    expect(screen.getByRole('heading', { name: 'Edit tags' })).toBeInTheDocument()
  })

  it('shows a formatted absolute date alongside the relative time', async () => {
    useMessageStore.setState({
      selected: { ...message, received_at: '2026-08-07T14:32:10Z' },
      selectedStatus: 'idle',
      markRead: vi.fn(),
    })
    renderScreen()
    expect(await screen.findByText('(Fri, 07 Aug 2026 14:32:10 GMT)')).toBeInTheDocument()
  })

  it('does not call markRead again when already read', async () => {
    const markRead = vi.fn()
    useMessageStore.setState({ selected: { ...message, read: true }, selectedStatus: 'idle', markRead })
    renderScreen()
    await waitFor(() => expect(screen.getByText('Hello world')).toBeInTheDocument())
    expect(markRead).not.toHaveBeenCalled()
  })

  it('delete navigates back to inbox', async () => {
    const deleteMessageOptimistic = vi.fn().mockResolvedValue(undefined)
    useMessageStore.setState({ selected: message, selectedStatus: 'idle', deleteMessageOptimistic })
    renderScreen()
    fireEvent.click(screen.getAllByRole('button', { name: /Delete/ })[0])
    const confirmButtons = screen.getAllByRole('button', { name: 'Delete' })
    fireEvent.click(confirmButtons[confirmButtons.length - 1])
    await waitFor(() => expect(deleteMessageOptimistic).toHaveBeenCalledWith('m1'))
    await waitFor(() => expect(screen.getByText('Inbox Home')).toBeInTheDocument())
  })

  it('export triggers exportMessage and a blob download', async () => {
    vi.mocked(apiClient.exportMessage).mockResolvedValue(new Blob(['x']))
    useMessageStore.setState({ selected: message, selectedStatus: 'idle' })
    renderScreen()
    fireEvent.click(screen.getByText('Export .eml'))
    await waitFor(() => expect(apiClient.exportMessage).toHaveBeenCalledWith('m1'))
  })

  it('pushes a success toast once export completes', async () => {
    useUIStore.setState({ toasts: [] })
    vi.mocked(apiClient.exportMessage).mockResolvedValue(new Blob(['x']))
    useMessageStore.setState({ selected: message, selectedStatus: 'idle' })
    renderScreen()
    fireEvent.click(screen.getByText('Export .eml'))
    await waitFor(() =>
      expect(useUIStore.getState().toasts.some((t) => t.message === 'Message exported')).toBe(true),
    )
  })

  it('pushes a danger toast if export fails', async () => {
    useUIStore.setState({ toasts: [] })
    vi.mocked(apiClient.exportMessage).mockRejectedValue(new Error('boom'))
    useMessageStore.setState({ selected: message, selectedStatus: 'idle' })
    renderScreen()
    fireEvent.click(screen.getByText('Export .eml'))
    await waitFor(() =>
      expect(useUIStore.getState().toasts.some((t) => t.message === 'Failed to export message')).toBe(true),
    )
  })
})

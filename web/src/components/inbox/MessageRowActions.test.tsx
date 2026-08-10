import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import MessageRowActions from './MessageRowActions'
import { useMessageStore } from '../../stores/useMessageStore'
import { useUIStore } from '../../stores/useUIStore'
import * as apiClient from '../../lib/apiClient'
import type { MessageSummary } from '../../lib/apiTypes'

vi.mock('../../lib/apiClient', async () => {
  const actual = await vi.importActual<typeof import('../../lib/apiClient')>('../../lib/apiClient')
  return { ...actual, exportMessage: vi.fn() }
})

function makeMessage(overrides: Partial<MessageSummary> = {}): MessageSummary {
  return {
    id: 'msg-1',
    from: 'alice@example.com',
    to: ['bob@example.com'],
    cc: [],
    subject: 'Hello world',
    size_bytes: 2048,
    has_attachments: false,
    attachment_count: 0,
    received_at: new Date().toISOString(),
    parse_warning: false,
    read: true,
    tags: [],
    preview: '',
    ...overrides,
  }
}

describe('MessageRowActions', () => {
  beforeEach(() => {
    useMessageStore.setState({ deleteMessageOptimistic: vi.fn(), markRead: vi.fn() })
    useUIStore.setState({ modal: null, toasts: [] })
    vi.stubGlobal('URL', { ...URL, createObjectURL: vi.fn(() => 'blob:mock'), revokeObjectURL: vi.fn() })
  })

  it('exports the message and pushes a success toast', async () => {
    vi.mocked(apiClient.exportMessage).mockResolvedValue(new Blob(['x']))
    render(<MessageRowActions message={makeMessage()} onPreview={vi.fn()} />)
    fireEvent.click(screen.getByLabelText('Message actions'))
    fireEvent.click(screen.getByText('Export .eml'))
    await waitFor(() => expect(apiClient.exportMessage).toHaveBeenCalledWith('msg-1'))
    await waitFor(() => expect(useUIStore.getState().toasts.some((t) => t.variant === 'success')).toBe(true))
  })

  it('pushes a danger toast if export fails', async () => {
    vi.mocked(apiClient.exportMessage).mockRejectedValue(new Error('boom'))
    render(<MessageRowActions message={makeMessage()} onPreview={vi.fn()} />)
    fireEvent.click(screen.getByLabelText('Message actions'))
    fireEvent.click(screen.getByText('Export .eml'))
    await waitFor(() => expect(useUIStore.getState().toasts.some((t) => t.variant === 'danger')).toBe(true))
  })

  it('calls onPreview and closes the menu', () => {
    const onPreview = vi.fn()
    render(<MessageRowActions message={makeMessage()} onPreview={onPreview} />)
    fireEvent.click(screen.getByLabelText('Message actions'))
    fireEvent.click(screen.getByText('Preview'))
    expect(onPreview).toHaveBeenCalledTimes(1)
    expect(screen.queryByText('Preview')).not.toBeInTheDocument()
  })

  it('closes the menu on Escape', () => {
    render(<MessageRowActions message={makeMessage()} onPreview={vi.fn()} />)
    fireEvent.click(screen.getByLabelText('Message actions'))
    expect(screen.getByText('Preview')).toBeInTheDocument()
    fireEvent.keyDown(document, { key: 'Escape' })
    expect(screen.queryByText('Preview')).not.toBeInTheDocument()
  })
})

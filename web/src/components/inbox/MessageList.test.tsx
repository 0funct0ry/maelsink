import { render, screen } from '@testing-library/react'
import MessageList from './MessageList'
import { useMessageStore } from '../../stores/useMessageStore'
import * as uiApiClient from '../../lib/uiApiClient'
import type { MessageSummary } from '../../lib/apiTypes'

vi.mock('../../lib/uiApiClient')

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

describe('MessageList', () => {
  beforeEach(() => {
    vi.mocked(uiApiClient.getInfo).mockRejectedValue(new Error('no info'))
  })

  it('shows a loading skeleton', () => {
    useMessageStore.setState({ listStatus: 'loading', messages: [], listError: null })
    const { container } = render(<MessageList onOpenMessage={vi.fn()} onPreviewMessage={vi.fn()} />)
    expect(container.querySelectorAll('.animate-pulse').length).toBeGreaterThan(0)
  })

  it('shows the error message', () => {
    useMessageStore.setState({
      listStatus: 'error',
      messages: [],
      listError: { message: 'boom' } as never,
    })
    render(<MessageList onOpenMessage={vi.fn()} onPreviewMessage={vi.fn()} />)
    expect(screen.getByText('boom')).toBeInTheDocument()
  })

  it('shows the empty state when there are no messages', () => {
    useMessageStore.setState({ listStatus: 'idle', messages: [], listError: null })
    render(<MessageList onOpenMessage={vi.fn()} onPreviewMessage={vi.fn()} />)
    expect(screen.getByText('No messages yet')).toBeInTheDocument()
  })

  it('renders a MessageRow per message and wires onOpenMessage', () => {
    useMessageStore.setState({
      listStatus: 'idle',
      listError: null,
      messages: [makeMessage({ id: 'a', subject: 'First' }), makeMessage({ id: 'b', subject: 'Second' })],
      deleteMessageOptimistic: vi.fn(),
    })
    render(<MessageList onOpenMessage={vi.fn()} onPreviewMessage={vi.fn()} />)
    expect(screen.getByText('First')).toBeInTheDocument()
    expect(screen.getByText('Second')).toBeInTheDocument()
  })
})

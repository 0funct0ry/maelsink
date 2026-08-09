import { render, screen, fireEvent } from '@testing-library/react'
import MessageRow from './MessageRow'
import { useMessageStore } from '../../stores/useMessageStore'
import type { MessageSummary } from '../../lib/apiTypes'

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
    ...overrides,
  }
}

describe('MessageRow', () => {
  beforeEach(() => {
    useMessageStore.setState({ deleteMessageOptimistic: vi.fn() })
  })

  it('renders from, subject, size', () => {
    render(<MessageRow message={makeMessage()} onOpen={vi.fn()} />)
    expect(screen.getByText('alice@example.com')).toBeInTheDocument()
    expect(screen.getByText('Hello world')).toBeInTheDocument()
    expect(screen.getByText('2.0 KB')).toBeInTheDocument()
  })

  it('shows an unread indicator when unread', () => {
    render(<MessageRow message={makeMessage({ read: false })} onOpen={vi.fn()} />)
    expect(screen.getByLabelText('Unread')).toBeInTheDocument()
  })

  it('does not show an unread indicator when read', () => {
    render(<MessageRow message={makeMessage({ read: true })} onOpen={vi.fn()} />)
    expect(screen.queryByLabelText('Unread')).not.toBeInTheDocument()
  })

  it('shows a "+N more" badge when there are multiple recipients', () => {
    render(
      <MessageRow
        message={makeMessage({ to: ['bob@example.com', 'carol@example.com', 'dave@example.com'] })}
        onOpen={vi.fn()}
      />,
    )
    expect(screen.getByText('bob@example.com')).toBeInTheDocument()
    expect(screen.getByText('+2 more')).toBeInTheDocument()
  })

  it('shows attachment count when has_attachments', () => {
    render(<MessageRow message={makeMessage({ has_attachments: true, attachment_count: 3 })} onOpen={vi.fn()} />)
    expect(screen.getByText('3')).toBeInTheDocument()
  })

  it('shows a warning badge when parse_warning is set', () => {
    render(<MessageRow message={makeMessage({ parse_warning: true })} onOpen={vi.fn()} />)
    expect(screen.getByText('Parse warning')).toBeInTheDocument()
  })

  it('calls onOpen when the row is clicked', () => {
    const onOpen = vi.fn()
    render(<MessageRow message={makeMessage()} onOpen={onOpen} />)
    fireEvent.click(screen.getByText('Hello world'))
    expect(onOpen).toHaveBeenCalledTimes(1)
  })

  it('calls deleteMessageOptimistic and not onOpen when the delete button is clicked', () => {
    const onOpen = vi.fn()
    const deleteMessageOptimistic = vi.fn()
    useMessageStore.setState({ deleteMessageOptimistic })
    render(<MessageRow message={makeMessage()} onOpen={onOpen} />)
    fireEvent.click(screen.getByLabelText('Delete message'))
    expect(deleteMessageOptimistic).toHaveBeenCalledWith('msg-1')
    expect(onOpen).not.toHaveBeenCalled()
  })
})

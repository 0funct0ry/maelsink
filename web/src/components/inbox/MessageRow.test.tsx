import { render, screen, fireEvent } from '@testing-library/react'
import MessageRow from './MessageRow'
import { useMessageStore } from '../../stores/useMessageStore'
import { useUIStore } from '../../stores/useUIStore'
import type { MessageSummary } from '../../lib/apiTypes'

function makeMessage(overrides: Partial<MessageSummary> = {}): MessageSummary {
  return {
    id: 'msg-1',
    from: 'alice@example.com',
    to: ['bob@example.com'],
    cc: [],
    bcc: [],
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

function renderRow(overrides: Partial<MessageSummary> = {}, onOpen = vi.fn(), onPreview = vi.fn()) {
  render(<MessageRow message={makeMessage(overrides)} onOpen={onOpen} onPreview={onPreview} />)
  return { onOpen, onPreview }
}

describe('MessageRow', () => {
  beforeEach(() => {
    useMessageStore.setState({ deleteMessageOptimistic: vi.fn(), markRead: vi.fn() })
    useUIStore.setState({ modal: null })
  })

  it('renders from, subject, size', () => {
    renderRow()
    expect(screen.getByText('alice@example.com')).toBeInTheDocument()
    expect(screen.getByText('Hello world')).toBeInTheDocument()
    expect(screen.getByText('2.0 KB')).toBeInTheDocument()
  })

  it('shows an unread indicator when unread', () => {
    renderRow({ read: false })
    expect(screen.getByLabelText('Unread')).toBeInTheDocument()
  })

  it('does not show an unread indicator when read', () => {
    renderRow({ read: true })
    expect(screen.queryByLabelText('Unread')).not.toBeInTheDocument()
  })

  it('shows a "+N more" badge when there are multiple recipients', () => {
    renderRow({ to: ['bob@example.com', 'carol@example.com', 'dave@example.com'] })
    expect(screen.getByText('bob@example.com')).toBeInTheDocument()
    expect(screen.getByText('+2 more')).toBeInTheDocument()
  })

  it('shows attachment count when has_attachments', () => {
    renderRow({ has_attachments: true, attachment_count: 3 })
    expect(screen.getByText('3')).toBeInTheDocument()
  })

  it('shows a warning badge when parse_warning is set', () => {
    renderRow({ parse_warning: true })
    expect(screen.getByText('Parse warning')).toBeInTheDocument()
  })

  it('calls onOpen when the row is clicked', () => {
    const { onOpen } = renderRow()
    fireEvent.click(screen.getByText('Hello world'))
    expect(onOpen).toHaveBeenCalledTimes(1)
  })

  it('renders a colored chip per tag', () => {
    renderRow({ tags: ['smoke', 'release'] })
    expect(screen.getByText('smoke')).toBeInTheDocument()
    expect(screen.getByText('release')).toBeInTheDocument()
  })

  it('renders no tag chips when there are no tags', () => {
    renderRow({ tags: [] })
    expect(screen.queryByText('smoke')).not.toBeInTheDocument()
  })

  it('does not render a body preview snippet in the list row', () => {
    renderRow({ preview: 'hello there, this is the body' })
    expect(screen.queryByText('hello there, this is the body')).not.toBeInTheDocument()
  })

  it('opens the actions menu and calls onPreview, not onOpen, when Preview is clicked', () => {
    const { onOpen, onPreview } = renderRow()
    fireEvent.click(screen.getByLabelText('Message actions'))
    fireEvent.click(screen.getByText('Preview'))
    expect(onPreview).toHaveBeenCalledTimes(1)
    expect(onOpen).not.toHaveBeenCalled()
  })

  it('opens the actions menu and calls markRead(false) when marking an already-read message unread', () => {
    const markRead = vi.fn()
    useMessageStore.setState({ markRead })
    renderRow({ read: true })
    fireEvent.click(screen.getByLabelText('Message actions'))
    fireEvent.click(screen.getByText('Mark as unread'))
    expect(markRead).toHaveBeenCalledWith('msg-1', false)
  })

  it('opens the actions menu and calls markRead(true) when marking an unread message read', () => {
    const markRead = vi.fn()
    useMessageStore.setState({ markRead })
    renderRow({ read: false })
    fireEvent.click(screen.getByLabelText('Message actions'))
    fireEvent.click(screen.getByText('Mark as read'))
    expect(markRead).toHaveBeenCalledWith('msg-1', true)
  })

  it('opens the actions menu and opens a confirm dialog when Delete is clicked, not onOpen', () => {
    const { onOpen } = renderRow()
    fireEvent.click(screen.getByLabelText('Message actions'))
    fireEvent.click(screen.getByText('Delete'))
    expect(useUIStore.getState().modal).toMatchObject({ kind: 'confirm', danger: true })
    expect(onOpen).not.toHaveBeenCalled()
  })

  it('deletes via the confirm dialog', () => {
    const deleteMessageOptimistic = vi.fn()
    useMessageStore.setState({ deleteMessageOptimistic })
    renderRow()
    fireEvent.click(screen.getByLabelText('Message actions'))
    fireEvent.click(screen.getByText('Delete'))
    useUIStore.getState().modal?.onConfirm()
    expect(deleteMessageOptimistic).toHaveBeenCalledWith('msg-1')
  })

  it('closes the menu when clicking outside it', () => {
    renderRow()
    fireEvent.click(screen.getByLabelText('Message actions'))
    expect(screen.getByText('Preview')).toBeInTheDocument()
    fireEvent.mouseDown(document.body)
    expect(screen.queryByText('Preview')).not.toBeInTheDocument()
  })
})

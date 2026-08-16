import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import MessageListScreen from './MessageListScreen'
import * as composeApi from '../lib/composeApi'
import type { MessageSummary } from '../lib/composeApi'
import { useConnectionStore } from '../stores/useConnectionStore'

vi.mock('../lib/composeApi')

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
    received_at: new Date('2026-01-01T00:00:00Z').toISOString(),
    parse_warning: false,
    read: true,
    tags: [],
    preview: '',
    ...overrides,
  }
}

beforeEach(() => {
  // Action buttons are disabled while the connection status is red — assume
  // a healthy target unless a test cares about the disabled state.
  useConnectionStore.setState({ status: 'green' })
})

afterEach(() => {
  vi.restoreAllMocks()
})

describe('MessageListScreen', () => {
  it('renders messages newest-first as returned by the API', async () => {
    vi.mocked(composeApi.listMessages).mockResolvedValue({
      messages: [makeMessage({ id: 'a', subject: 'Newest' }), makeMessage({ id: 'b', subject: 'Oldest' })],
      total: 2,
      limit: 50,
      offset: 0,
    })

    render(
      <MemoryRouter>
        <MessageListScreen />
      </MemoryRouter>,
    )

    await waitFor(() => expect(screen.getByText('Newest')).toBeInTheDocument())
    const items = screen.getAllByText(/Newest|Oldest/)
    expect(items[0]).toHaveTextContent('Newest')
  })

  it('deletes a message via its row delete button', async () => {
    vi.mocked(composeApi.listMessages).mockResolvedValue({
      messages: [makeMessage({ id: 'a', subject: 'Delete me' })],
      total: 1,
      limit: 50,
      offset: 0,
    })
    vi.mocked(composeApi.deleteMessage).mockResolvedValue(undefined)

    render(
      <MemoryRouter>
        <MessageListScreen />
      </MemoryRouter>,
    )

    await waitFor(() => expect(screen.getByText('Delete me')).toBeInTheDocument())
    fireEvent.click(screen.getByRole('button', { name: 'Delete' }))

    // Deleting is a two-step action — a confirm modal gates the actual call.
    expect(composeApi.deleteMessage).not.toHaveBeenCalled()
    expect(screen.getByText('Delete this message?')).toBeInTheDocument()

    vi.mocked(composeApi.listMessages).mockResolvedValue({ messages: [], total: 0, limit: 20, offset: 0 })
    fireEvent.click(screen.getAllByRole('button', { name: 'Delete' })[1])

    expect(composeApi.deleteMessage).toHaveBeenCalledWith('a')
    await waitFor(() => expect(screen.queryByText('Delete me')).not.toBeInTheDocument())
  })

  it('only clears all after confirming the modal', async () => {
    vi.mocked(composeApi.listMessages).mockResolvedValue({
      messages: [makeMessage()],
      total: 1,
      limit: 50,
      offset: 0,
    })
    vi.mocked(composeApi.clearMessages).mockResolvedValue(undefined)

    render(
      <MemoryRouter>
        <MessageListScreen />
      </MemoryRouter>,
    )

    await waitFor(() => expect(screen.getByText('Hello world')).toBeInTheDocument())
    fireEvent.click(screen.getByRole('button', { name: 'Clear all' }))

    // Modal open — clear must not have fired yet.
    expect(composeApi.clearMessages).not.toHaveBeenCalled()
    expect(screen.getByText('Clear all messages?')).toBeInTheDocument()

    fireEvent.click(screen.getAllByRole('button', { name: 'Clear all' })[1])
    expect(composeApi.clearMessages).toHaveBeenCalledTimes(1)
  })

  it('paginates via the next/previous controls', async () => {
    vi.mocked(composeApi.listMessages).mockResolvedValue({
      messages: [makeMessage({ id: 'a', subject: 'Page one' })],
      total: 25,
      limit: 20,
      offset: 0,
    })

    render(
      <MemoryRouter>
        <MessageListScreen />
      </MemoryRouter>,
    )

    await waitFor(() => expect(screen.getByText('Page one')).toBeInTheDocument())
    expect(screen.getByText('1–20 of 25')).toBeInTheDocument()

    vi.mocked(composeApi.listMessages).mockResolvedValue({
      messages: [makeMessage({ id: 'b', subject: 'Page two' })],
      total: 25,
      limit: 20,
      offset: 20,
    })
    fireEvent.click(screen.getByRole('button', { name: 'Next page' }))

    await waitFor(() => expect(composeApi.listMessages).toHaveBeenCalledWith({ limit: 20, offset: 20 }))
    await waitFor(() => expect(screen.getByText('Page two')).toBeInTheDocument())
    expect(screen.getByText('21–25 of 25')).toBeInTheDocument()
  })
})

import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import ApiExplorerScreen from './ApiExplorerScreen'
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

function selectEndpoint(name: string) {
  const nav = screen.getByRole('navigation', { name: 'API endpoints' })
  fireEvent.click(within(nav).getByRole('button', { name }))
}

beforeEach(() => {
  useConnectionStore.setState({ status: 'green' })
})

afterEach(() => {
  vi.restoreAllMocks()
})

describe('ApiExplorerScreen', () => {
  it('shows a two-pane layout with the list endpoint selected by default', () => {
    render(<ApiExplorerScreen />)

    expect(screen.getByRole('navigation', { name: 'API endpoints' })).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'list' })).toBeInTheDocument()
    expect(screen.getByRole('main')).toHaveTextContent('/api/v1/messages')
  })

  it('list panel runs and shows the raw response', async () => {
    vi.mocked(composeApi.listMessages).mockResolvedValue({
      messages: [makeMessage()],
      total: 1,
      limit: 20,
      offset: 0,
    })

    render(<ApiExplorerScreen />)

    fireEvent.click(screen.getByRole('button', { name: 'Run' }))

    await waitFor(() => expect(composeApi.listMessages).toHaveBeenCalled())
    await waitFor(() => expect(screen.getByRole('main')).toHaveTextContent('Hello world'))
  })

  it('switching endpoints in the nav swaps the main panel', async () => {
    const detail = {
      ...makeMessage(),
      headers: [],
      text_body: 'hi',
      html_body: '',
      attachments: [],
      raw_size_bytes: 10,
    }
    vi.mocked(composeApi.getMessage).mockResolvedValue(detail)

    render(<ApiExplorerScreen />)
    selectEndpoint('get')

    expect(screen.getByRole('heading', { name: 'get' })).toBeInTheDocument()
    expect(screen.queryByPlaceholderText('RFC3339')).not.toBeInTheDocument() // list's filter fields are gone

    const idInput = screen.getByPlaceholderText('message ID')
    fireEvent.change(idInput, { target: { value: 'msg-1' } })
    fireEvent.click(screen.getByRole('button', { name: 'Run' }))

    await waitFor(() => expect(composeApi.getMessage).toHaveBeenCalledWith('msg-1'))
  })

  it('delete panel requires confirmation before firing', async () => {
    vi.mocked(composeApi.deleteMessage).mockResolvedValue(undefined)

    render(<ApiExplorerScreen />)
    selectEndpoint('delete')

    const idInput = screen.getByPlaceholderText('message ID')
    fireEvent.change(idInput, { target: { value: 'msg-1' } })
    fireEvent.click(screen.getByRole('button', { name: 'Run' }))

    expect(composeApi.deleteMessage).not.toHaveBeenCalled()
    expect(screen.getByText('Delete this message?')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Delete' }))
    await waitFor(() => expect(composeApi.deleteMessage).toHaveBeenCalledWith('msg-1'))
  })

  it('clear panel requires confirmation and cancelling sends no request', async () => {
    vi.mocked(composeApi.clearMessages).mockResolvedValue(undefined)

    render(<ApiExplorerScreen />)
    selectEndpoint('clear')

    fireEvent.click(screen.getByRole('button', { name: 'Run' }))

    expect(screen.getByText('Clear all messages?')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }))
    expect(composeApi.clearMessages).not.toHaveBeenCalled()

    fireEvent.click(screen.getByRole('button', { name: 'Run' }))
    fireEvent.click(screen.getByRole('button', { name: 'Clear all' }))
    await waitFor(() => expect(composeApi.clearMessages).toHaveBeenCalledTimes(1))
  })

  it('export panel downloads a file via the modal', async () => {
    vi.mocked(composeApi.exportMessages).mockResolvedValue({
      blob: new Blob(['zip'], { type: 'application/zip' }),
      filename: 'export.zip',
    })
    vi.mocked(composeApi.triggerDownload).mockImplementation(() => {})

    render(<ApiExplorerScreen />)
    selectEndpoint('export')

    fireEvent.click(screen.getByRole('button', { name: 'Export…' }))
    expect(screen.getByText('Export messages')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Download .zip' }))

    await waitFor(() => expect(composeApi.exportMessages).toHaveBeenCalled())
    await waitFor(() => expect(composeApi.triggerDownload).toHaveBeenCalled())
  })
})

import { afterEach, describe, expect, it, vi } from 'vitest'
import { useMessageStore } from './useMessageStore'
import { useUIStore } from './useUIStore'
import { HttpError, NetworkError } from '../lib/apiErrors'
import * as apiClient from '../lib/apiClient'
import type { MessageSummary } from '../lib/apiTypes'

vi.mock('../lib/apiClient')

function summary(id: string): MessageSummary {
  return {
    id,
    from: 'a@example.com',
    to: ['b@example.com'],
    cc: [],
    subject: 'hi',
    size_bytes: 10,
    has_attachments: false,
    attachment_count: 0,
    received_at: '2026-01-01T00:00:00Z',
    parse_warning: false,
    read: false,
    tags: [],
    preview: '',
  }
}

afterEach(() => {
  vi.clearAllMocks()
  useMessageStore.setState({
    messages: [],
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

describe('fetchMessages', () => {
  it('populates messages on success', async () => {
    vi.mocked(apiClient.listMessages).mockResolvedValue({
      messages: [summary('a'), summary('b')],
      total: 2,
      limit: 50,
      offset: 0,
    })

    await useMessageStore.getState().fetchMessages()

    expect(useMessageStore.getState().messages).toHaveLength(2)
    expect(useMessageStore.getState().listStatus).toBe('idle')
  })

  it('sets listStatus to error on failure', async () => {
    vi.mocked(apiClient.listMessages).mockRejectedValue(new NetworkError(new Error('offline')))

    await useMessageStore.getState().fetchMessages()

    expect(useMessageStore.getState().listStatus).toBe('error')
    expect(useMessageStore.getState().listError).toBeInstanceOf(NetworkError)
  })
})

describe('setQuery', () => {
  it('merges the query, resets offset, and refetches', async () => {
    vi.mocked(apiClient.listMessages).mockResolvedValue({ messages: [], total: 0, limit: 50, offset: 0 })
    useMessageStore.setState({ offset: 20 })

    useMessageStore.getState().setQuery({ q: 'invoice' })

    expect(useMessageStore.getState().query).toEqual({ q: 'invoice' })
    expect(useMessageStore.getState().offset).toBe(0)
    await vi.waitFor(() => expect(apiClient.listMessages).toHaveBeenCalled())
  })
})

describe('fetchMessage', () => {
  it('sets selectedStatus to not_found on a message_not_found error', async () => {
    vi.mocked(apiClient.getMessage).mockRejectedValue(new HttpError(404, 'message_not_found', 'gone'))

    await useMessageStore.getState().fetchMessage('abc')

    expect(useMessageStore.getState().selectedStatus).toBe('not_found')
  })

  it('sets selectedStatus to error on other failures', async () => {
    vi.mocked(apiClient.getMessage).mockRejectedValue(new HttpError(500, 'internal_error', 'boom'))

    await useMessageStore.getState().fetchMessage('abc')

    expect(useMessageStore.getState().selectedStatus).toBe('error')
  })

  it('populates selected on success', async () => {
    vi.mocked(apiClient.getMessage).mockResolvedValue({
      ...summary('abc'),
      headers: [],
      text_body: '',
      html_body: '',
      attachments: [],
      raw_size_bytes: 0,
    })

    await useMessageStore.getState().fetchMessage('abc')

    expect(useMessageStore.getState().selected?.id).toBe('abc')
    expect(useMessageStore.getState().selectedStatus).toBe('idle')
  })
})

describe('deleteMessageOptimistic', () => {
  it('removes the row immediately and keeps it removed on success', async () => {
    useMessageStore.setState({ messages: [summary('a'), summary('b')], total: 2 })
    vi.mocked(apiClient.deleteMessage).mockResolvedValue(undefined)

    await useMessageStore.getState().deleteMessageOptimistic('a')

    expect(useMessageStore.getState().messages.map((m) => m.id)).toEqual(['b'])
    expect(useMessageStore.getState().total).toBe(1)
  })

  it('rolls back and toasts on failure', async () => {
    useMessageStore.setState({ messages: [summary('a'), summary('b')], total: 2 })
    vi.mocked(apiClient.deleteMessage).mockRejectedValue(new HttpError(500, 'internal_error', 'boom'))

    await useMessageStore.getState().deleteMessageOptimistic('a')

    expect(useMessageStore.getState().messages.map((m) => m.id)).toEqual(['a', 'b'])
    expect(useMessageStore.getState().total).toBe(2)
    expect(useUIStore.getState().toasts).toHaveLength(1)
    expect(useUIStore.getState().toasts[0].variant).toBe('danger')
  })
})

describe('clearAll', () => {
  it('empties the list on success', async () => {
    useMessageStore.setState({ messages: [summary('a')], total: 1, offset: 10 })
    vi.mocked(apiClient.clearMessages).mockResolvedValue(undefined)

    await useMessageStore.getState().clearAll()

    expect(useMessageStore.getState().messages).toEqual([])
    expect(useMessageStore.getState().total).toBe(0)
    expect(useMessageStore.getState().offset).toBe(0)
    expect(useUIStore.getState().toasts.some((t) => t.variant === 'success')).toBe(true)
  })

  it('toasts on failure without clearing', async () => {
    useMessageStore.setState({ messages: [summary('a')], total: 1 })
    vi.mocked(apiClient.clearMessages).mockRejectedValue(new HttpError(500, 'internal_error', 'boom'))

    await useMessageStore.getState().clearAll()

    expect(useMessageStore.getState().messages).toHaveLength(1)
    expect(useUIStore.getState().toasts).toHaveLength(1)
  })
})

describe('markRead', () => {
  it('marks the row and selected message read on success', async () => {
    useMessageStore.setState({
      messages: [summary('a')],
      selected: { ...summary('a'), headers: [], text_body: '', html_body: '', attachments: [], raw_size_bytes: 0 },
    })
    vi.mocked(apiClient.markRead).mockResolvedValue(undefined)

    await useMessageStore.getState().markRead('a')

    expect(useMessageStore.getState().messages[0].read).toBe(true)
    expect(useMessageStore.getState().selected?.read).toBe(true)
  })

  it('marks the row unread when passed read=false', async () => {
    useMessageStore.setState({ messages: [{ ...summary('a'), read: true }] })
    vi.mocked(apiClient.markRead).mockResolvedValue(undefined)

    await useMessageStore.getState().markRead('a', false)

    expect(apiClient.markRead).toHaveBeenCalledWith('a', false)
    expect(useMessageStore.getState().messages[0].read).toBe(false)
  })

  it('pushes a danger toast if the request fails', async () => {
    useMessageStore.setState({ messages: [summary('a')] })
    vi.mocked(apiClient.markRead).mockRejectedValue(new Error('boom'))

    await useMessageStore.getState().markRead('a')

    expect(useUIStore.getState().toasts).toHaveLength(1)
  })
})

describe('clearSelected', () => {
  it('resets selected state', () => {
    useMessageStore.setState({ selected: { ...summary('a'), headers: [], text_body: '', html_body: '', attachments: [], raw_size_bytes: 0 }, selectedStatus: 'idle' })
    useMessageStore.getState().clearSelected()
    expect(useMessageStore.getState().selected).toBeNull()
  })
})

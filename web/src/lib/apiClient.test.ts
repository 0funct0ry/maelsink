import { afterEach, describe, expect, it, vi } from 'vitest'
import * as apiClient from './apiClient'
import { useUIStore } from '../stores/useUIStore'

function mockFetch(impl: typeof fetch) {
  vi.stubGlobal('fetch', impl)
}

afterEach(() => {
  vi.unstubAllGlobals()
  useUIStore.setState({ authToken: null, authRequired: false, pendingRetry: null })
})

describe('apiClient', () => {
  it('listMessages sends query params and parses the list envelope', async () => {
    let seenUrl = ''
    mockFetch(
      vi.fn(async (url) => {
        seenUrl = String(url)
        return new Response(JSON.stringify({ messages: [], total: 0, limit: 50, offset: 0 }), { status: 200 })
      }) as unknown as typeof fetch,
    )
    const result = await apiClient.listMessages({ q: 'hello', limit: 10 })
    expect(result.total).toBe(0)
    expect(seenUrl).toContain('q=hello')
    expect(seenUrl).toContain('limit=10')
  })

  it('listMessages sends repeatable tag params and tag_mode', async () => {
    let seenUrl = ''
    mockFetch(
      vi.fn(async (url) => {
        seenUrl = String(url)
        return new Response(JSON.stringify({ messages: [], total: 0, limit: 50, offset: 0 }), { status: 200 })
      }) as unknown as typeof fetch,
    )
    await apiClient.listMessages({ tag: ['a', 'b'], tag_mode: 'all' })
    const params = new URL(seenUrl, 'http://x').searchParams
    expect(params.getAll('tag')).toEqual(['a', 'b'])
    expect(params.get('tag_mode')).toBe('all')
  })

  it('updateMessageTags issues a PATCH to the tags endpoint', async () => {
    let seenUrl = ''
    let seenMethod = ''
    let seenBody = ''
    mockFetch(
      vi.fn(async (url, init) => {
        seenUrl = String(url)
        seenMethod = init?.method ?? ''
        seenBody = String(init?.body ?? '')
        return new Response(JSON.stringify({ id: 'abc', tags: ['release'] }), { status: 200 })
      }) as unknown as typeof fetch,
    )
    const result = await apiClient.updateMessageTags('abc', { add: ['release'], remove: ['smoke'] })
    expect(seenUrl).toContain('/api/v1/messages/abc/tags')
    expect(seenMethod).toBe('PATCH')
    expect(JSON.parse(seenBody)).toEqual({ add: ['release'], remove: ['smoke'] })
    expect(result.tags).toEqual(['release'])
  })

  it('exportAllUrl forwards repeatable tag params', () => {
    expect(apiClient.exportAllUrl({ tag: ['a', 'b'], tag_mode: 'all' })).toBe(
      '/api/v1/messages/export?tag=a&tag=b&tag_mode=all',
    )
  })

  it('getMessage fetches the detail endpoint', async () => {
    mockFetch(
      vi.fn(async () => new Response(JSON.stringify({ id: 'abc' }), { status: 200 })) as unknown as typeof fetch,
    )
    const msg = await apiClient.getMessage('abc')
    expect(msg.id).toBe('abc')
  })

  it('markRead issues a PATCH', async () => {
    let seenMethod = ''
    mockFetch(
      vi.fn(async (_url, init) => {
        seenMethod = init?.method ?? ''
        return new Response(null, { status: 204 })
      }) as unknown as typeof fetch,
    )
    await apiClient.markRead('abc')
    expect(seenMethod).toBe('PATCH')
  })

  it('deleteMessage issues a DELETE', async () => {
    let seenMethod = ''
    mockFetch(
      vi.fn(async (_url, init) => {
        seenMethod = init?.method ?? ''
        return new Response(null, { status: 204 })
      }) as unknown as typeof fetch,
    )
    await apiClient.deleteMessage('abc')
    expect(seenMethod).toBe('DELETE')
  })

  it('clearMessages passes confirm=true', async () => {
    let seenUrl = ''
    mockFetch(
      vi.fn(async (url) => {
        seenUrl = String(url)
        return new Response(null, { status: 204 })
      }) as unknown as typeof fetch,
    )
    await apiClient.clearMessages()
    expect(seenUrl).toContain('confirm=true')
  })

  it('getRaw returns text', async () => {
    mockFetch(vi.fn(async () => new Response('raw', { status: 200 })) as unknown as typeof fetch)
    expect(await apiClient.getRaw('abc')).toBe('raw')
  })

  it('getStats parses stats shape', async () => {
    mockFetch(
      vi.fn(
        async () =>
          new Response(
            JSON.stringify({ total_messages: 3, total_size_bytes: 100, oldest_received_at: null, newest_received_at: null }),
            { status: 200 },
          ),
      ) as unknown as typeof fetch,
    )
    const stats = await apiClient.getStats()
    expect(stats.total_messages).toBe(3)
  })

  it('getVersion parses version shape', async () => {
    mockFetch(
      vi.fn(
        async () => new Response(JSON.stringify({ version: '1.0.0', commit: 'abc', go: 'go1.23' }), { status: 200 }),
      ) as unknown as typeof fetch,
    )
    const version = await apiClient.getVersion()
    expect(version.version).toBe('1.0.0')
  })

  it('getAttachmentDownloadUrl builds a direct URL (no fetch)', () => {
    expect(apiClient.getAttachmentDownloadUrl('abc', 'att1')).toBe('/api/v1/messages/abc/attachments/att1')
  })

  it('exportAllUrl builds the bulk export URL (no fetch)', () => {
    expect(apiClient.exportAllUrl()).toBe('/api/v1/messages/export')
  })
})

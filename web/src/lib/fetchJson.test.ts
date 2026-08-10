import { afterEach, describe, expect, it, vi } from 'vitest'
import { fetchBlob, fetchJson, fetchText } from './fetchJson'
import { HttpError, NetworkError } from './apiErrors'
import { useUIStore } from '../stores/useUIStore'

function mockFetch(impl: typeof fetch) {
  vi.stubGlobal('fetch', impl)
}

afterEach(() => {
  vi.unstubAllGlobals()
  useUIStore.setState({ authToken: null, authRequired: false, pendingRetry: null })
})

describe('fetchJson', () => {
  it('parses a successful JSON response', async () => {
    mockFetch(
      vi.fn(async () => new Response(JSON.stringify({ ok: true }), { status: 200 })) as unknown as typeof fetch,
    )
    const result = await fetchJson<{ ok: boolean }>('/api/v1/health')
    expect(result).toEqual({ ok: true })
  })

  it('returns undefined for a 204 No Content response', async () => {
    mockFetch(vi.fn(async () => new Response(null, { status: 204 })) as unknown as typeof fetch)
    const result = await fetchJson('/api/v1/messages/abc')
    expect(result).toBeUndefined()
  })

  it('attaches the Authorization header when a token is stored', async () => {
    useUIStore.setState({ authToken: 'secret' })
    let seenHeaders: Headers | undefined
    mockFetch(
      vi.fn(async (_url, init) => {
        seenHeaders = new Headers(init?.headers)
        return new Response('{}', { status: 200 })
      }) as unknown as typeof fetch,
    )
    await fetchJson('/api/v1/messages')
    expect(seenHeaders?.get('Authorization')).toBe('Bearer secret')
  })

  it('throws HttpError with the parsed error envelope on a non-2xx response', async () => {
    mockFetch(
      vi.fn(
        async () =>
          new Response(JSON.stringify({ error: { code: 'message_not_found', message: 'no message with id x' } }), {
            status: 404,
          }),
      ) as unknown as typeof fetch,
    )
    await expect(fetchJson('/api/v1/messages/x')).rejects.toMatchObject({
      status: 404,
      code: 'message_not_found',
    })
  })

  it('falls back to unknown_error when the error body is not parseable JSON', async () => {
    mockFetch(vi.fn(async () => new Response('not json', { status: 500 })) as unknown as typeof fetch)
    const err = await fetchJson('/api/v1/messages').catch((e) => e)
    expect(err).toBeInstanceOf(HttpError)
    expect((err as HttpError).code).toBe('unknown_error')
  })

  it('throws NetworkError when fetch itself rejects', async () => {
    mockFetch(vi.fn(async () => {
      throw new TypeError('Failed to fetch')
    }) as unknown as typeof fetch)
    await expect(fetchJson('/api/v1/messages')).rejects.toBeInstanceOf(NetworkError)
  })

  it('flags authRequired on a 401 response', async () => {
    mockFetch(
      vi.fn(
        async () =>
          new Response(JSON.stringify({ error: { code: 'unauthorized', message: 'nope' } }), { status: 401 }),
      ) as unknown as typeof fetch,
    )
    await fetchJson('/api/v1/messages').catch(() => undefined)
    expect(useUIStore.getState().authRequired).toBe(true)
  })
})

describe('fetchBlob / fetchText', () => {
  it('fetchBlob resolves a Blob on success', async () => {
    mockFetch(vi.fn(async () => new Response(new Blob(['data']), { status: 200 })) as unknown as typeof fetch)
    const blob = await fetchBlob('/api/v1/messages/x/attachments/y')
    expect(blob).toBeTruthy()
    expect(typeof (blob as Blob).size).toBe('number')
    expect(typeof (blob as Blob).type).toBe('string')
  })

  it('fetchText resolves text on success', async () => {
    mockFetch(vi.fn(async () => new Response('raw source', { status: 200 })) as unknown as typeof fetch)
    const text = await fetchText('/api/v1/messages/x/raw')
    expect(text).toBe('raw source')
  })

  it('fetchBlob throws HttpError on failure', async () => {
    mockFetch(vi.fn(async () => new Response('{}', { status: 404 })) as unknown as typeof fetch)
    await expect(fetchBlob('/api/v1/messages/x/attachments/y')).rejects.toBeInstanceOf(HttpError)
  })
})

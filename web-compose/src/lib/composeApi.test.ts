import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import {
  ComposeApiError,
  clearMessages,
  deleteMessage,
  downloadAttachment,
  exportMessages,
  getFunctions,
  getMessage,
  getStats,
  getVersion,
  health,
  listMessages,
  listMessagesQuery,
  renderTemplate,
  sendTemplate,
  triggerDownload,
} from './composeApi'

function jsonResponse(body: unknown, init: { status?: number } = {}): Response {
  return new Response(JSON.stringify(body), {
    status: init.status ?? 200,
    headers: { 'Content-Type': 'application/json' },
  })
}

beforeEach(() => {
  vi.stubGlobal('fetch', vi.fn())
})

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('composeApi request helper (via health)', () => {
  it('returns parsed JSON on success', async () => {
    vi.mocked(fetch).mockResolvedValue(jsonResponse({ target_reachable: true, status: 'green' }))
    const resp = await health()
    expect(resp.status).toBe('green')
    expect(fetch).toHaveBeenCalledWith('/compose-api/v1/health', undefined)
  })

  it('throws a ComposeApiError with details parsed from a JSON error body', async () => {
    vi.mocked(fetch).mockResolvedValue(
      jsonResponse({ error: { code: 'bad_template', message: 'unexpected token', line: 3, column: 5 } }, { status: 400 }),
    )
    await expect(getMessage('abc')).rejects.toMatchObject({
      status: 400,
      code: 'bad_template',
      message: 'unexpected token',
      line: 3,
      column: 5,
    })
  })

  it('falls back to statusText when the error body is not JSON', async () => {
    vi.mocked(fetch).mockResolvedValue(new Response('not json', { status: 500, statusText: 'Server Error' }))
    await expect(getMessage('abc')).rejects.toMatchObject({ status: 500, code: 'unknown_error', message: 'Server Error' })
  })

  it('returns undefined for a 204 response', async () => {
    vi.mocked(fetch).mockResolvedValue(new Response(null, { status: 204 }))
    await expect(deleteMessage('abc')).resolves.toBeUndefined()
  })
})

describe('listMessagesQuery', () => {
  it('defaults sort and omits unset params', () => {
    const q = listMessagesQuery()
    expect(q.get('sort')).toBe('received_at_desc')
    expect(q.has('limit')).toBe(false)
  })

  it('includes every provided param', () => {
    const q = listMessagesQuery({
      limit: 10,
      offset: 5,
      q: 'hello',
      from: 'a@b.com',
      to: 'c@d.com',
      subject: 'hi',
      since: '2026-01-01',
      until: '2026-01-02',
      sort: 'received_at_asc',
    })
    expect(q.get('limit')).toBe('10')
    expect(q.get('offset')).toBe('5')
    expect(q.get('q')).toBe('hello')
    expect(q.get('from')).toBe('a@b.com')
    expect(q.get('to')).toBe('c@d.com')
    expect(q.get('subject')).toBe('hi')
    expect(q.get('since')).toBe('2026-01-01')
    expect(q.get('until')).toBe('2026-01-02')
    expect(q.get('sort')).toBe('received_at_asc')
  })
})

describe('composeApi endpoints', () => {
  it('listMessages hits the messages endpoint with the query string', async () => {
    vi.mocked(fetch).mockResolvedValue(jsonResponse({ messages: [], total: 0, limit: 20, offset: 0 }))
    await listMessages({ q: 'foo' })
    expect(vi.mocked(fetch).mock.calls[0][0]).toContain('/compose-api/v1/messages?')
  })

  it('clearMessages issues a DELETE with no id', async () => {
    vi.mocked(fetch).mockResolvedValue(new Response(null, { status: 204 }))
    await clearMessages()
    expect(fetch).toHaveBeenCalledWith('/compose-api/v1/messages', { method: 'DELETE' })
  })

  it('renderTemplate POSTs JSON', async () => {
    vi.mocked(fetch).mockResolvedValue(jsonResponse({ rendered: 'Subject: hi' }))
    const req = { template: 'x', format: 'eml' as const, vars: {} }
    await renderTemplate(req)
    expect(fetch).toHaveBeenCalledWith(
      '/compose-api/v1/render',
      expect.objectContaining({ method: 'POST', body: JSON.stringify(req) }),
    )
  })

  it('sendTemplate POSTs JSON', async () => {
    vi.mocked(fetch).mockResolvedValue(jsonResponse({ from: 'a@b.com', to: ['c@d.com'] }))
    const req = { template: 'x', format: 'json' as const, vars: {} }
    await sendTemplate(req)
    expect(fetch).toHaveBeenCalledWith(
      '/compose-api/v1/send',
      expect.objectContaining({ method: 'POST', body: JSON.stringify(req) }),
    )
  })

  it('getFunctions unwraps the functions field', async () => {
    vi.mocked(fetch).mockResolvedValue(jsonResponse({ functions: [{ name: 'upper', category: 'string', args: '', returns: '', description: '' }] }))
    const fns = await getFunctions()
    expect(fns).toHaveLength(1)
    expect(fns[0].name).toBe('upper')
  })

  it('getStats and getVersion return their bodies', async () => {
    vi.mocked(fetch).mockResolvedValueOnce(
      jsonResponse({ total_messages: 1, total_size_bytes: 2, oldest_received_at: null, newest_received_at: null }),
    )
    const stats = await getStats()
    expect(stats.total_messages).toBe(1)

    vi.mocked(fetch).mockResolvedValueOnce(
      jsonResponse({
        target: { version: '1', commit: 'a', build_date: 'b', go: 'c' },
        compose: { version: '1', commit: 'a', build_date: 'b', go: 'c' },
      }),
    )
    const version = await getVersion()
    expect(version.target.version).toBe('1')
  })
})

describe('blob downloads', () => {
  it('exportMessages fetches a blob and derives a filename from Content-Disposition', async () => {
    const blob = new Blob(['zipdata'])
    vi.mocked(fetch).mockResolvedValue(
      new Response(blob, { status: 200, headers: { 'Content-Disposition': 'attachment; filename="export-1.zip"' } }),
    )
    const result = await exportMessages({ q: 'foo' })
    expect(result.filename).toBe('export-1.zip')
    expect(vi.mocked(fetch).mock.calls[0][0]).not.toContain('limit')
  })

  it('falls back to the default filename when no Content-Disposition header is present', async () => {
    vi.mocked(fetch).mockResolvedValue(new Response(new Blob(['x']), { status: 200 }))
    const result = await exportMessages()
    expect(result.filename).toBe('export.zip')
  })

  it('downloadAttachment throws ComposeApiError on failure', async () => {
    vi.mocked(fetch).mockResolvedValue(jsonResponse({ error: { code: 'not_found', message: 'missing' } }, { status: 404 }))
    await expect(downloadAttachment('m1', 'a1', 'file.txt')).rejects.toBeInstanceOf(ComposeApiError)
  })

  it('triggerDownload creates and clicks a transient anchor', () => {
    const createObjectURL = vi.fn().mockReturnValue('blob:mock')
    const revokeObjectURL = vi.fn()
    vi.stubGlobal('URL', { ...URL, createObjectURL, revokeObjectURL })
    const clickSpy = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})

    triggerDownload({ blob: new Blob(['x']), filename: 'report.zip' })

    expect(createObjectURL).toHaveBeenCalled()
    expect(clickSpy).toHaveBeenCalled()
    expect(revokeObjectURL).toHaveBeenCalledWith('blob:mock')
    clickSpy.mockRestore()
  })
})

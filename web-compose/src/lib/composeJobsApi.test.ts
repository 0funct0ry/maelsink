import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { ComposeApiError, cancelJob, listJobs, openJobStream, startJob } from './composeJobsApi'

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

describe('composeJobsApi request helper', () => {
  it('startJob POSTs the kind and params', async () => {
    vi.mocked(fetch).mockResolvedValue(jsonResponse({ jobId: 'job-1' }))
    const result = await startJob('randmsg', { count: 5 })
    expect(result.jobId).toBe('job-1')
    expect(fetch).toHaveBeenCalledWith(
      '/compose-api/v1/jobs/randmsg',
      expect.objectContaining({ method: 'POST', body: JSON.stringify({ count: 5 }) }),
    )
  })

  it('cancelJob POSTs to the cancel endpoint', async () => {
    vi.mocked(fetch).mockResolvedValue(
      jsonResponse({
        jobId: 'job-1',
        kind: 'randmsg',
        status: 'cancelled',
        sent: 1,
        failed: 0,
        startedAt: '2026-01-01T00:00:00Z',
        elapsedSeconds: 1,
      }),
    )
    const snap = await cancelJob('job 1')
    expect(snap.status).toBe('cancelled')
    expect(fetch).toHaveBeenCalledWith('/compose-api/v1/jobs/job%201/cancel', { method: 'POST' })
  })

  it('listJobs unwraps the jobs field', async () => {
    vi.mocked(fetch).mockResolvedValue(jsonResponse({ jobs: [] }))
    await expect(listJobs()).resolves.toEqual([])
  })

  it('throws a ComposeApiError parsed from a JSON error body', async () => {
    vi.mocked(fetch).mockResolvedValue(jsonResponse({ error: { code: 'not_found', message: 'no such job' } }, { status: 404 }))
    await expect(cancelJob('missing')).rejects.toMatchObject({
      status: 404,
      code: 'not_found',
      message: 'no such job',
    })
    await expect(cancelJob('missing')).rejects.toBeInstanceOf(ComposeApiError)
  })

  it('falls back to statusText when the error body is not JSON', async () => {
    vi.mocked(fetch).mockResolvedValue(new Response('oops', { status: 500, statusText: 'Server Error' }))
    await expect(listJobs()).rejects.toMatchObject({ status: 500, code: 'unknown_error', message: 'Server Error' })
  })
})

describe('openJobStream', () => {
  class FakeWebSocket {
    static instances: FakeWebSocket[] = []
    url: string
    onmessage: ((event: { data: string }) => void) | null = null
    onclose: (() => void) | null = null
    closed = false

    constructor(url: string) {
      this.url = url
      FakeWebSocket.instances.push(this)
    }

    close() {
      this.closed = true
    }
  }

  beforeEach(() => {
    FakeWebSocket.instances = []
    vi.stubGlobal('WebSocket', FakeWebSocket)
  })

  it('builds a ws:// URL from the current location and dispatches parsed ticks', () => {
    const onTick = vi.fn()
    const onClose = vi.fn()
    const close = openJobStream('job-1', onTick, onClose)

    const ws = FakeWebSocket.instances[0]
    expect(ws.url).toBe(`ws://${window.location.host}/compose-api/v1/jobs/job-1/stream`)

    const tick = { jobId: 'job-1', kind: 'randmsg', status: 'running', sent: 1, failed: 0, startedAt: '', elapsedSeconds: 1 }
    ws.onmessage?.({ data: JSON.stringify(tick) })
    expect(onTick).toHaveBeenCalledWith(tick)

    ws.onclose?.()
    expect(onClose).toHaveBeenCalled()

    close()
    expect(ws.closed).toBe(true)
  })

  it('ignores malformed frames instead of throwing', () => {
    const onTick = vi.fn()
    openJobStream('job-1', onTick)
    const ws = FakeWebSocket.instances[0]
    expect(() => ws.onmessage?.({ data: 'not json' })).not.toThrow()
    expect(onTick).not.toHaveBeenCalled()
  })

  it('works without an onClose callback', () => {
    openJobStream('job-1', vi.fn())
    const ws = FakeWebSocket.instances[0]
    expect(() => ws.onclose?.()).not.toThrow()
  })
})

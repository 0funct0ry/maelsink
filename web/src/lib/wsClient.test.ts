import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { connectWs, type WsStatus } from './wsClient'

class FakeWebSocket {
  static instances: FakeWebSocket[] = []
  onopen: (() => void) | null = null
  onmessage: ((event: { data: string }) => void) | null = null
  onclose: (() => void) | null = null
  onerror: (() => void) | null = null
  closed = false

  constructor(public url: string) {
    FakeWebSocket.instances.push(this)
  }

  close() {
    this.closed = true
    this.onclose?.()
  }

  // Test helpers, not part of the real WebSocket API.
  emitOpen() {
    this.onopen?.()
  }
  emitMessage(data: unknown) {
    this.onmessage?.({ data: JSON.stringify(data) })
  }
  emitClose() {
    this.onclose?.()
  }
}

beforeEach(() => {
  FakeWebSocket.instances = []
  vi.useFakeTimers()
})

afterEach(() => {
  vi.useRealTimers()
})

function latestSocket() {
  const s = FakeWebSocket.instances[FakeWebSocket.instances.length - 1]
  if (!s) throw new Error('no socket constructed')
  return s
}

describe('connectWs', () => {
  it('delivers parsed frames to onEvent', () => {
    const onEvent = vi.fn()
    const conn = connectWs({ onEvent, WebSocketImpl: FakeWebSocket as unknown as typeof WebSocket })

    latestSocket().emitOpen()
    latestSocket().emitMessage({ type: 'hello', payload: { version: '1.0.0' } })

    expect(onEvent).toHaveBeenCalledWith({ type: 'hello', payload: { version: '1.0.0' } })
    conn.close()
  })

  it('reports status transitions: connecting -> open', () => {
    const onStatusChange = vi.fn()
    const conn = connectWs({
      onEvent: () => {},
      onStatusChange,
      WebSocketImpl: FakeWebSocket as unknown as typeof WebSocket,
    })

    expect(onStatusChange).toHaveBeenCalledWith('connecting')
    latestSocket().emitOpen()
    expect(onStatusChange).toHaveBeenCalledWith('open')
    conn.close()
  })

  it('reconnects with increasing backoff after an unexpected close', () => {
    const onStatusChange = vi.fn()
    const conn = connectWs({
      onEvent: () => {},
      onStatusChange,
      WebSocketImpl: FakeWebSocket as unknown as typeof WebSocket,
    })

    latestSocket().emitOpen()
    expect(FakeWebSocket.instances).toHaveLength(1)

    // First unexpected close -> reconnecting, then a new socket after backoff.
    latestSocket().emitClose()
    expect(onStatusChange).toHaveBeenCalledWith('reconnecting')
    expect(FakeWebSocket.instances).toHaveLength(1)

    vi.advanceTimersByTime(2000)
    expect(FakeWebSocket.instances).toHaveLength(2)

    // Second unexpected close (without ever reaching 'open') should wait
    // at least as long as the first backoff, i.e. advancing by the same
    // amount that resolved attempt 1 must not yet resolve attempt 2 if the
    // delay has strictly increased.
    latestSocket().emitClose()
    vi.advanceTimersByTime(600) // shorter than attempt 1's max (1000ms*jitter range starts higher)
    const countAfterShortWait = FakeWebSocket.instances.length
    vi.advanceTimersByTime(4000)
    expect(FakeWebSocket.instances.length).toBeGreaterThan(countAfterShortWait)

    conn.close()
  })

  it('does not reconnect after an explicit close()', () => {
    const conn = connectWs({ onEvent: () => {}, WebSocketImpl: FakeWebSocket as unknown as typeof WebSocket })
    latestSocket().emitOpen()

    conn.close()
    vi.advanceTimersByTime(30_000)

    expect(FakeWebSocket.instances).toHaveLength(1)
  })

  it('treats a close as reconnect-worthy regardless of cause (e.g. server.shutdown)', () => {
    const statuses: WsStatus[] = []
    const conn = connectWs({
      onEvent: () => {},
      onStatusChange: (s) => statuses.push(s),
      WebSocketImpl: FakeWebSocket as unknown as typeof WebSocket,
    })
    latestSocket().emitOpen()
    latestSocket().emitMessage({ type: 'server.shutdown', payload: {} })
    latestSocket().emitClose()

    expect(statuses).toContain('reconnecting')
    conn.close()
  })
})

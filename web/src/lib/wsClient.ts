import { wsUrl } from './apiBase'

export interface WsFrame {
  type: string
  payload: unknown
}

export type WsStatus = 'connecting' | 'open' | 'reconnecting' | 'closed'

export interface WsClientOptions {
  onEvent: (frame: WsFrame) => void
  onStatusChange?: (status: WsStatus) => void
  /** Overridable for tests; defaults to the real WebSocket constructor. */
  WebSocketImpl?: typeof WebSocket
}

const BASE_BACKOFF_MS = 500
const MAX_BACKOFF_MS = 15_000

/**
 * Connects to /ws (SPEC.md §5.5) and reconnects with exponential backoff +
 * jitter on any unexpected close, until close() is called explicitly. A
 * server.shutdown frame is treated the same as an unexpected close (the
 * server is telling clients it's going away, not erroring) — it still goes
 * through backoff so a mid-restart server isn't hammered.
 */
export function connectWs(opts: WsClientOptions): { close: () => void } {
  const WS = opts.WebSocketImpl ?? WebSocket
  let socket: WebSocket | null = null
  let closedByCaller = false
  let attempt = 0
  let timer: ReturnType<typeof setTimeout> | null = null

  const setStatus = (status: WsStatus) => opts.onStatusChange?.(status)

  const scheduleReconnect = () => {
    if (closedByCaller) return
    setStatus('reconnecting')
    const backoff = Math.min(BASE_BACKOFF_MS * 2 ** attempt, MAX_BACKOFF_MS)
    const jitter = backoff * (0.5 + Math.random() * 0.5)
    attempt += 1
    timer = setTimeout(connect, jitter)
  }

  function connect() {
    if (closedByCaller) return
    setStatus(attempt === 0 ? 'connecting' : 'reconnecting')
    socket = new WS(wsUrl())

    socket.onopen = () => {
      attempt = 0
      setStatus('open')
    }

    socket.onmessage = (event) => {
      try {
        const frame = JSON.parse(event.data as string) as WsFrame
        opts.onEvent(frame)
      } catch {
        // Ignore malformed frames rather than crashing the connection.
      }
    }

    socket.onclose = () => {
      socket = null
      if (closedByCaller) return
      scheduleReconnect()
    }

    socket.onerror = () => {
      socket?.close()
    }
  }

  connect()

  return {
    close: () => {
      closedByCaller = true
      if (timer) clearTimeout(timer)
      socket?.close()
      socket = null
      setStatus('closed')
    },
  }
}

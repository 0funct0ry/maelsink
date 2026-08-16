// Thin fetch wrapper for compose's own local proxy surface,
// /compose-api/v1/* — never the target's REST API directly (SPEC.md
// §7.7.3). All paths are relative so this works whether compose is served
// at "/" or (in a later milestone) behind a base path.

export interface MessageSummary {
  id: string
  from: string
  to: string[]
  cc: string[]
  bcc: string[]
  subject: string
  size_bytes: number
  has_attachments: boolean
  attachment_count: number
  received_at: string
  parse_warning: boolean
  read: boolean
  tags: string[]
  preview: string
}

export interface Header {
  name: string
  value: string
}

export interface AttachmentInfo {
  id: string
  filename: string
  content_type: string
  size_bytes: number
  content_id: string | null
}

export interface MessageDetail extends MessageSummary {
  headers: Header[]
  text_body: string
  html_body: string
  attachments: AttachmentInfo[]
  raw_size_bytes: number
}

export interface ListResponse {
  messages: MessageSummary[]
  total: number
  limit: number
  offset: number
}

export interface HealthResponse {
  target_reachable: boolean
  status: 'green' | 'yellow' | 'red'
  target_health?: unknown
  error?: string
}

export class ComposeApiError extends Error {
  status: number
  code: string

  constructor(status: number, code: string, message: string) {
    super(message)
    this.status = status
    this.code = code
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const resp = await fetch(path, init)
  if (!resp.ok) {
    let code = 'unknown_error'
    let message = resp.statusText
    try {
      const body = await resp.json()
      if (body?.error?.code) {
        code = body.error.code
        message = body.error.message ?? message
      }
    } catch {
      // non-JSON error body — fall back to statusText
    }
    throw new ComposeApiError(resp.status, code, message)
  }
  if (resp.status === 204) {
    return undefined as T
  }
  return (await resp.json()) as T
}

export function health(): Promise<HealthResponse> {
  return request<HealthResponse>('/compose-api/v1/health')
}

export interface ListMessagesParams {
  limit?: number
  offset?: number
}

export function listMessages(params: ListMessagesParams = {}): Promise<ListResponse> {
  const q = new URLSearchParams({ sort: 'received_at_desc' })
  if (params.limit != null) q.set('limit', String(params.limit))
  if (params.offset != null) q.set('offset', String(params.offset))
  return request<ListResponse>(`/compose-api/v1/messages?${q.toString()}`)
}

export function getMessage(id: string): Promise<MessageDetail> {
  return request<MessageDetail>(`/compose-api/v1/messages/${encodeURIComponent(id)}`)
}

export function deleteMessage(id: string): Promise<void> {
  return request<void>(`/compose-api/v1/messages/${encodeURIComponent(id)}`, { method: 'DELETE' })
}

export function clearMessages(): Promise<void> {
  return request<void>('/compose-api/v1/messages', { method: 'DELETE' })
}

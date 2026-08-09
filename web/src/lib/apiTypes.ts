// Mirrors internal/api's JSON DTOs (SPEC.md §5.1) and internal/webui/uiapi's
// /ui-api/v1/info response. Keep these in lockstep with the Go structs in
// internal/api/handlers.go and internal/webui/uiapi/uiapi.go.

export interface MessageSummary {
  id: string
  from: string
  to: string[]
  cc: string[]
  subject: string
  size_bytes: number
  has_attachments: boolean
  attachment_count: number
  received_at: string
  parse_warning: boolean
  read: boolean
}

export interface HeaderEntry {
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
  headers: HeaderEntry[]
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

export interface Stats {
  total_messages: number
  total_size_bytes: number
  oldest_received_at: string | null
  newest_received_at: string | null
}

export interface Version {
  version: string
  commit: string
  go: string
}

export interface ErrorEnvelope {
  error: {
    code: string
    message: string
  }
}

export interface UiInfo {
  smtp: {
    host: string
    port: number
  }
  auth_enabled: boolean
}

export interface ListMessagesParams {
  q?: string
  from?: string
  to?: string
  subject?: string
  limit?: number
  offset?: number
  since?: string
  until?: string
  sort?: 'received_at_desc' | 'received_at_asc'
}

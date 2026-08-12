// Mirrors internal/api's JSON DTOs (SPEC.md §5.1) and internal/webui/uiapi's
// /ui-api/v1/info response. Keep these in lockstep with the Go structs in
// internal/api/handlers.go and internal/webui/uiapi/uiapi.go.

export interface MessageSummary {
  id: string
  from: string
  from_name?: string
  to: string[]
  to_names?: string[]
  cc: string[]
  cc_names?: string[]
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
  /** The SMTP session that produced this message (M8.4), for the Message
   * Detail -> Session Detail cross-link. Absent for messages saved outside
   * a tracked SMTP session. */
  session_id?: string
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
  unread_count: number
  attachment_count: number
  parse_warning_count: number
}

export interface TagStats {
  name: string
  color: string
  count: number
  last_used: string | null
}

export interface ConfigSource {
  layer: 'default' | 'file' | 'env' | 'flag'
  origin: string
}

export interface ConfigEntry {
  section: string
  key: string
  value: unknown
  source: ConfigSource
}

export interface Version {
  version: string
  commit: string
  build_date: string
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
  cc?: string
  bcc?: string
  subject?: string
  limit?: number
  offset?: number
  since?: string
  until?: string
  sort?: 'received_at_desc' | 'received_at_asc'
  tag?: string[]
  tag_mode?: 'any' | 'all'
  read?: boolean
  has_attachments?: boolean
  parse_warning?: boolean
}

// SMTP session logging (M8.4). Mirrors internal/api/handlers.go's
// sessionSummaryJSON/sessionDetailJSON.
export interface SessionSummary {
  id: string
  client_ip: string
  client_helo: string
  started_at: string
  ended_at: string | null
  // 'completed' | 'rejected' | 'aborted' | 'timeout' once finished; '' means
  // still in progress (a session.started row not yet completed).
  status: string
  message_id: string | null
}

export interface TranscriptLine {
  direction: 'C' | 'S'
  line: string
  position: number
}

export interface SessionDetail extends SessionSummary {
  transcript: TranscriptLine[]
}

export interface ListSessionsParams {
  status?: string
  client_ip?: string
  limit?: number
  offset?: number
  since?: string
  until?: string
  sort?: 'started_at_desc' | 'started_at_asc'
}

export interface ListSessionsResponse {
  sessions: SessionSummary[]
  total: number
  limit: number
  offset: number
}

import { apiUrl } from './apiBase'
import { fetchBlob, fetchJson, fetchText } from './fetchJson'
import type {
  ListMessagesParams,
  ListResponse,
  MessageDetail,
  MessageSummary,
  Stats,
  TagStats,
  Version,
} from './apiTypes'

function buildQueryString(params: Record<string, string | number | boolean | string[] | undefined> = {}): string {
  const search = new URLSearchParams()
  for (const [key, value] of Object.entries(params)) {
    if (value === undefined || value === '') continue
    if (Array.isArray(value)) {
      for (const v of value) {
        if (v !== '') search.append(key, v)
      }
      continue
    }
    search.set(key, String(value))
  }
  const qs = search.toString()
  return qs ? `?${qs}` : ''
}

export function listMessages(params: ListMessagesParams = {}): Promise<ListResponse> {
  return fetchJson<ListResponse>('/api/v1/messages', { query: { ...params } })
}

export function getMessage(id: string): Promise<MessageDetail> {
  return fetchJson<MessageDetail>(`/api/v1/messages/${id}`)
}

export function markRead(id: string, read = true): Promise<void> {
  return fetchJson<void>(`/api/v1/messages/${id}/read`, { method: 'PATCH', body: { read } })
}

export function updateMessageTags(id: string, body: { add: string[]; remove: string[] }): Promise<MessageSummary> {
  return fetchJson<MessageSummary>(`/api/v1/messages/${id}/tags`, { method: 'PATCH', body })
}

export function deleteMessage(id: string): Promise<void> {
  return fetchJson<void>(`/api/v1/messages/${id}`, { method: 'DELETE' })
}

export function clearMessages(): Promise<void> {
  return fetchJson<void>('/api/v1/messages', { method: 'DELETE', query: { confirm: 'true' } })
}

export function getRaw(id: string): Promise<string> {
  return fetchText(`/api/v1/messages/${id}/raw`)
}

export function getAttachmentBlob(id: string, attachmentId: string): Promise<Blob> {
  return fetchBlob(`/api/v1/messages/${id}/attachments/${attachmentId}`)
}

export function getAttachmentDownloadUrl(id: string, attachmentId: string): string {
  return apiUrl(`/api/v1/messages/${id}/attachments/${attachmentId}`)
}

export function exportMessage(id: string): Promise<Blob> {
  return fetchBlob(`/api/v1/messages/${id}/export`)
}

/**
 * Builds the Export All download URL, forwarding the same filter params the
 * active list query is using (M6.1) so Export All respects the current
 * search/filter instead of always exporting every message.
 */
export function exportAllUrl(params: ListMessagesParams = {}): string {
  return apiUrl('/api/v1/messages/export') + buildQueryString({ ...params })
}

export function getStats(): Promise<Stats> {
  return fetchJson<Stats>('/api/v1/stats')
}

export function listTags(): Promise<TagStats[]> {
  return fetchJson<TagStats[]>('/api/v1/tags')
}

export function createTag(name: string, color: string): Promise<TagStats> {
  return fetchJson<TagStats>('/api/v1/tags', { method: 'POST', body: { name, color } })
}

export function renameTag(name: string, newName: string): Promise<TagStats> {
  return fetchJson<TagStats>(`/api/v1/tags/${encodeURIComponent(name)}`, { method: 'PATCH', body: { name: newName } })
}

export function recolorTag(name: string, color: string): Promise<TagStats> {
  return fetchJson<TagStats>(`/api/v1/tags/${encodeURIComponent(name)}`, { method: 'PATCH', body: { color } })
}

export function deleteTag(name: string): Promise<void> {
  return fetchJson<void>(`/api/v1/tags/${encodeURIComponent(name)}`, { method: 'DELETE' })
}

export function deleteTagWithMessages(name: string): Promise<void> {
  return fetchJson<void>(`/api/v1/tags/${encodeURIComponent(name)}/messages`, { method: 'DELETE' })
}

export function getVersion(): Promise<Version> {
  return fetchJson<Version>('/api/v1/version')
}

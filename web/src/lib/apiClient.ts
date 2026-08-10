import { apiUrl } from './apiBase'
import { fetchBlob, fetchJson, fetchText } from './fetchJson'
import type { ListMessagesParams, ListResponse, MessageDetail, Stats, TagCount, Version } from './apiTypes'

function buildQueryString(params: Record<string, string | number | boolean | undefined> = {}): string {
  const search = new URLSearchParams()
  for (const [key, value] of Object.entries(params)) {
    if (value === undefined || value === '') continue
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

export function getTags(): Promise<TagCount[]> {
  return fetchJson<TagCount[]>('/api/v1/tags')
}

export function getVersion(): Promise<Version> {
  return fetchJson<Version>('/api/v1/version')
}

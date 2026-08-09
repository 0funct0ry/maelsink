import { apiUrl } from './apiBase'
import { fetchBlob, fetchJson, fetchText } from './fetchJson'
import type { ListMessagesParams, ListResponse, MessageDetail, Stats, Version } from './apiTypes'

export function listMessages(params: ListMessagesParams = {}): Promise<ListResponse> {
  return fetchJson<ListResponse>('/api/v1/messages', { query: { ...params } })
}

export function getMessage(id: string): Promise<MessageDetail> {
  return fetchJson<MessageDetail>(`/api/v1/messages/${id}`)
}

export function markRead(id: string): Promise<void> {
  return fetchJson<void>(`/api/v1/messages/${id}/read`, { method: 'PATCH' })
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

export function exportAllUrl(): string {
  return apiUrl('/api/v1/messages/export')
}

export function getStats(): Promise<Stats> {
  return fetchJson<Stats>('/api/v1/stats')
}

export function getVersion(): Promise<Version> {
  return fetchJson<Version>('/api/v1/version')
}

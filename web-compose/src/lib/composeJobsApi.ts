// Thin client for compose's Jobs Panel surface, /compose-api/v1/jobs/*
// (SPEC.md §7.7.4.3, M13.3). Mirrors composeApi.ts's conventions (relative
// paths, ComposeApiError on non-2xx) but adds a WebSocket helper for a
// job's live progress stream.

import { ComposeApiError } from './composeApi'

export { ComposeApiError }

export type JobKind = 'randmsg' | 'intmsg' | 'weirdmsg' | 'blast' | 'deluge'

export type JobStatus = 'running' | 'completed' | 'cancelled' | 'failed'

export interface JobSnapshot {
  jobId: string
  kind: string
  status: JobStatus
  sent: number
  failed: number
  startedAt: string
  elapsedSeconds: number
  error?: string
}

async function jobRequest<T>(path: string, init?: RequestInit): Promise<T> {
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
  return (await resp.json()) as T
}

export interface ContentParams {
  to?: string
  from?: string
  cc?: string[]
  bcc?: string[]
  subject?: string
  body?: 'text' | 'html' | 'both' | 'random'
  attachments?: number
  attachmentSize?: string
  tags?: string[]
  scenario?: string
}

export interface RandMsgParams extends ContentParams {
  count?: number
  concurrency?: number
}

export interface IntMsgParams extends ContentParams {
  intervalMs?: number
  rate?: number
  jitter?: string
  profile?: 'steady' | 'poisson' | 'bursty'
  burstSize?: number
  burstIntervalMs?: number
  count?: number
  durationMs?: number
  untilError?: boolean
}

export interface WeirdMsgParams {
  kind?: 'bounce' | 'malformed' | 'huge' | 'unicode' | 'spoof' | 'thread' | 'invite' | 'random'
  size?: string
  depth?: number
  to?: string
  from?: string
}

export interface BlastParams extends ContentParams {
  recipients?: number
  split?: 'to' | 'cc' | 'bcc' | 'mixed'
}

export interface DelugeParams extends ContentParams {
  count?: number
  concurrency?: number
}

export type JobParams<K extends JobKind> = K extends 'randmsg'
  ? RandMsgParams
  : K extends 'intmsg'
    ? IntMsgParams
    : K extends 'weirdmsg'
      ? WeirdMsgParams
      : K extends 'blast'
        ? BlastParams
        : DelugeParams

export function startJob<K extends JobKind>(kind: K, params: JobParams<K>): Promise<{ jobId: string }> {
  return jobRequest<{ jobId: string }>(`/compose-api/v1/jobs/${kind}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(params),
  })
}

export function cancelJob(jobId: string): Promise<JobSnapshot> {
  return jobRequest<JobSnapshot>(`/compose-api/v1/jobs/${encodeURIComponent(jobId)}/cancel`, { method: 'POST' })
}

export function listJobs(): Promise<JobSnapshot[]> {
  return jobRequest<{ jobs: JobSnapshot[] }>('/compose-api/v1/jobs').then((r) => r.jobs)
}

// openJobStream opens the job's WS progress stream, calling onTick for
// every tick (including the final one) and onClose once the socket closes
// (whether the job reached a terminal state or the connection just
// dropped). Returns a function to close the socket early (e.g. on
// component unmount).
export function openJobStream(jobId: string, onTick: (tick: JobSnapshot) => void, onClose?: () => void): () => void {
  const proto = window.location.protocol === 'https:' ? 'wss' : 'ws'
  const ws = new WebSocket(`${proto}://${window.location.host}/compose-api/v1/jobs/${encodeURIComponent(jobId)}/stream`)

  ws.onmessage = (event) => {
    try {
      onTick(JSON.parse(event.data) as JobSnapshot)
    } catch {
      // ignore malformed frames
    }
  }
  ws.onclose = () => onClose?.()

  return () => ws.close()
}

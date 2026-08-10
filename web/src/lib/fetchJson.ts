import { apiUrl } from './apiBase'
import { HttpError, NetworkError } from './apiErrors'
import type { ErrorEnvelope } from './apiTypes'
import { useUIStore } from '../stores/useUIStore'

export interface FetchJsonOptions {
  method?: string
  query?: Record<string, string | number | boolean | string[] | undefined>
  /** JSON-serializable request body; sets Content-Type: application/json. */
  body?: unknown
  /** Skip attaching Authorization — never needed today, kept for completeness. */
  skipAuth?: boolean
}

function buildQuery(query?: FetchJsonOptions['query']): string {
  if (!query) return ''
  const params = new URLSearchParams()
  for (const [key, value] of Object.entries(query)) {
    if (value === undefined || value === '') continue
    if (Array.isArray(value)) {
      for (const v of value) {
        if (v !== '') params.append(key, v)
      }
      continue
    }
    params.set(key, String(value))
  }
  const qs = params.toString()
  return qs ? `?${qs}` : ''
}

async function classifyErrorResponse(res: Response): Promise<HttpError> {
  let code = 'unknown_error'
  let message = res.statusText || `request failed with status ${res.status}`
  try {
    const body = (await res.json()) as ErrorEnvelope
    if (body?.error?.code) code = body.error.code
    if (body?.error?.message) message = body.error.message
  } catch {
    // Body wasn't valid JSON (or empty) — keep the unknown_error fallback.
  }
  return new HttpError(res.status, code, message)
}

/**
 * Shared low-level fetch wrapper used by apiClient and uiApiClient: builds
 * the request against the runtime base path, injects the bearer token when
 * present, and classifies failures into NetworkError (fetch itself threw) vs
 * HttpError (a response came back with a non-2xx status). On a 401 it flags
 * useUIStore.authRequired so an ApiKeyModal can prompt — retry orchestration
 * lives in the caller (store actions), not here.
 */
export async function fetchJson<T>(path: string, opts: FetchJsonOptions = {}): Promise<T> {
  const url = apiUrl(path) + buildQuery(opts.query)
  const headers: Record<string, string> = {}

  if (!opts.skipAuth) {
    const token = useUIStore.getState().authToken
    if (token) headers.Authorization = `Bearer ${token}`
  }

  if (opts.body !== undefined) {
    headers['Content-Type'] = 'application/json'
  }

  let res: Response
  try {
    res = await fetch(url, {
      method: opts.method ?? 'GET',
      headers,
      body: opts.body !== undefined ? JSON.stringify(opts.body) : undefined,
    })
  } catch (cause) {
    throw new NetworkError(cause)
  }

  if (res.status === 401) {
    useUIStore.getState().setAuthRequired(true)
  }

  if (!res.ok) {
    throw await classifyErrorResponse(res)
  }

  if (res.status === 204 || res.headers.get('content-length') === '0') {
    return undefined as T
  }

  return (await res.json()) as T
}

/** Like fetchJson, but returns the raw Blob (attachment/raw-source downloads). */
export async function fetchBlob(path: string): Promise<Blob> {
  const url = apiUrl(path)
  const token = useUIStore.getState().authToken
  const headers: Record<string, string> = {}
  if (token) headers.Authorization = `Bearer ${token}`

  let res: Response
  try {
    res = await fetch(url, { headers })
  } catch (cause) {
    throw new NetworkError(cause)
  }

  if (res.status === 401) {
    useUIStore.getState().setAuthRequired(true)
  }
  if (!res.ok) {
    throw await classifyErrorResponse(res)
  }
  return res.blob()
}

/** Like fetchJson, but returns the raw text body (used for /raw). */
export async function fetchText(path: string): Promise<string> {
  const url = apiUrl(path)
  const token = useUIStore.getState().authToken
  const headers: Record<string, string> = {}
  if (token) headers.Authorization = `Bearer ${token}`

  let res: Response
  try {
    res = await fetch(url, { headers })
  } catch (cause) {
    throw new NetworkError(cause)
  }

  if (res.status === 401) {
    useUIStore.getState().setAuthRequired(true)
  }
  if (!res.ok) {
    throw await classifyErrorResponse(res)
  }
  return res.text()
}

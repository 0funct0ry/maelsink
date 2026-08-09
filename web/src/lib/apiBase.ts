// The Go server (internal/webui) rewrites the __MAELSINK_BASE_PATH__
// placeholder in index.html to the request's resolved reverse-proxy base
// path (SPEC.md §3.4). Nothing in the frontend should hardcode "/" — every
// API/WS/router call should go through this module instead.
declare global {
  interface Window {
    __MAELSINK_BASE__?: string
  }
}

const PLACEHOLDER = '__MAELSINK_BASE_PATH__'

/** Resolved reverse-proxy base path, e.g. "/maelsink", or "" at the root. */
export function basePath(): string {
  const raw = window.__MAELSINK_BASE__
  if (!raw || raw === PLACEHOLDER) return ''
  return raw.endsWith('/') ? raw.slice(0, -1) : raw
}

/** Joins a path onto the resolved base path, e.g. apiUrl("/api/v1/messages"). */
export function apiUrl(path: string): string {
  return `${basePath()}${path.startsWith('/') ? path : `/${path}`}`
}

/** Resolves the WebSocket URL for the /ws endpoint under the base path. */
export function wsUrl(): string {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  return `${protocol}//${window.location.host}${apiUrl('/ws')}`
}

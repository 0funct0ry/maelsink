/** Base class for every error thrown by fetchJson/apiClient/uiApiClient. */
export class ApiClientError extends Error {}

/**
 * Thrown when `fetch` itself rejects (offline, connection refused, CORS
 * block, DNS failure) — no response was ever received, so this is distinct
 * from a server that responded with an error status (HttpError).
 */
export class NetworkError extends ApiClientError {
  constructor(cause: unknown) {
    super('Unable to reach the server')
    this.cause = cause
  }
}

/**
 * Thrown when the server responded with a non-2xx status. `code` is the
 * error envelope's `error.code` (SPEC.md §5.3) when the body parsed as one,
 * else "unknown_error".
 */
export class HttpError extends ApiClientError {
  status: number
  code: string

  constructor(status: number, code: string, message: string) {
    super(message)
    this.status = status
    this.code = code
  }
}

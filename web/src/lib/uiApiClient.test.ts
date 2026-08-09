import { afterEach, describe, expect, it, vi } from 'vitest'
import { getInfo } from './uiApiClient'
import { useUIStore } from '../stores/useUIStore'

afterEach(() => {
  vi.unstubAllGlobals()
  useUIStore.setState({ authToken: null, authRequired: false, pendingRetry: null })
})

describe('uiApiClient.getInfo', () => {
  it('parses the info response', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(
        async () =>
          new Response(JSON.stringify({ smtp: { host: '127.0.0.1', port: 1025 }, auth_enabled: false }), {
            status: 200,
          }),
      ),
    )
    const info = await getInfo()
    expect(info.smtp.host).toBe('127.0.0.1')
    expect(info.smtp.port).toBe(1025)
    expect(info.auth_enabled).toBe(false)
  })
})

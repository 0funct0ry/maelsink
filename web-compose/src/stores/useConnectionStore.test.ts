import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { useConnectionStore } from './useConnectionStore'
import * as composeApi from '../lib/composeApi'

vi.mock('../lib/composeApi', async () => {
  const actual = await vi.importActual<typeof composeApi>('../lib/composeApi')
  return { ...actual, health: vi.fn() }
})

beforeEach(() => {
  vi.useFakeTimers()
  useConnectionStore.setState({ status: 'red', lastChecked: null, lastError: null })
})

afterEach(() => {
  vi.useRealTimers()
  vi.restoreAllMocks()
})

describe('useConnectionStore', () => {
  it('transitions to green when the target is healthy', async () => {
    vi.mocked(composeApi.health).mockResolvedValue({ target_reachable: true, status: 'green' })
    await useConnectionStore.getState().poll()
    expect(useConnectionStore.getState().status).toBe('green')
  })

  it('transitions to red when the request fails', async () => {
    vi.mocked(composeApi.health).mockRejectedValue(new Error('network error'))
    await useConnectionStore.getState().poll()
    expect(useConnectionStore.getState().status).toBe('red')
    expect(useConnectionStore.getState().lastError).toBe('network error')
  })

  it('polls repeatedly on the configured interval', async () => {
    vi.mocked(composeApi.health).mockResolvedValue({ target_reachable: true, status: 'green' })
    const stop = useConnectionStore.getState().startPolling()
    await vi.advanceTimersByTimeAsync(0)
    expect(composeApi.health).toHaveBeenCalledTimes(1)

    vi.mocked(composeApi.health).mockResolvedValue({ target_reachable: false, status: 'red' })
    await vi.advanceTimersByTimeAsync(5000)
    expect(composeApi.health).toHaveBeenCalledTimes(2)
    expect(useConnectionStore.getState().status).toBe('red')

    stop()
  })
})

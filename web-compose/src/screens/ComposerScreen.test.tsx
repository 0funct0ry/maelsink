import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import ComposerScreen from './ComposerScreen'
import * as composeApi from '../lib/composeApi'
import { useVarsStore } from '../stores/useVarsStore'
import { useDraftsStore } from '../stores/useDraftsStore'

vi.mock('../lib/composeApi', async () => {
  const actual = await vi.importActual<typeof composeApi>('../lib/composeApi')
  return { ...actual, renderTemplate: vi.fn(), sendTemplate: vi.fn(), getFunctions: vi.fn() }
})

function makeMemoryStorage(): Storage {
  const data = new Map<string, string>()
  return {
    getItem: (key: string) => (data.has(key) ? data.get(key)! : null),
    setItem: (key: string, value: string) => data.set(key, value),
    removeItem: (key: string) => data.delete(key),
    clear: () => data.clear(),
    key: (index: number) => Array.from(data.keys())[index] ?? null,
    get length() {
      return data.size
    },
  } as Storage
}

beforeEach(() => {
  vi.stubGlobal('localStorage', makeMemoryStorage())
  useVarsStore.setState({ vars: {} })
  useDraftsStore.setState({ drafts: {} })
  vi.mocked(composeApi.getFunctions).mockResolvedValue([])
  vi.mocked(composeApi.renderTemplate).mockResolvedValue({ rendered: 'rendered output' })
  vi.useFakeTimers({ shouldAdvanceTime: true })
})

afterEach(() => {
  vi.useRealTimers()
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

describe('ComposerScreen', () => {
  it('renders a preview from the current template', async () => {
    render(<ComposerScreen />)

    await vi.advanceTimersByTimeAsync(500)
    await waitFor(() => expect(composeApi.renderTemplate).toHaveBeenCalled())
    await waitFor(() => expect(screen.getByText('rendered output')).toBeInTheDocument())
  })

  it('shows an inline error with position info on a bad render', async () => {
    vi.mocked(composeApi.renderTemplate).mockRejectedValue(
      new composeApi.ComposeApiError(400, 'render_failed', 'unexpected EOF', 5, 3),
    )
    render(<ComposerScreen />)

    await vi.advanceTimersByTimeAsync(500)
    await waitFor(() => expect(screen.getByText(/unexpected EOF/)).toBeInTheDocument())
    expect(screen.getByText(/Line 5, column 3/)).toBeInTheDocument()
  })

  it('inserts a var reference at the cursor via the vars panel', async () => {
    useVarsStore.setState({ vars: { myVar: 'hello' } })
    render(<ComposerScreen />)

    fireEvent.click(screen.getByRole('button', { name: 'myVar' }))

    await vi.advanceTimersByTimeAsync(500)
    await waitFor(() => expect(composeApi.renderTemplate).toHaveBeenCalled())
    const lastCall = vi.mocked(composeApi.renderTemplate).mock.calls.at(-1)?.[0]
    expect(lastCall?.template).toContain('{{.myVar}}')
  })

  it('inserts a function snippet via the function picker', async () => {
    vi.mocked(composeApi.getFunctions).mockResolvedValue([
      { name: 'fakeEmail', category: 'generate', args: '', returns: 'string', description: 'a fake email' },
    ])
    render(<ComposerScreen />)

    await waitFor(() => expect(screen.getByRole('button', { name: 'fakeEmail' })).toBeInTheDocument())
    fireEvent.click(screen.getByRole('button', { name: 'fakeEmail' }))

    await vi.advanceTimersByTimeAsync(500)
    await waitFor(() => expect(composeApi.renderTemplate).toHaveBeenCalled())
    const lastCall = vi.mocked(composeApi.renderTemplate).mock.calls.at(-1)?.[0]
    expect(lastCall?.template).toContain('{{ fakeEmail }}')
  })

  it('sends the current template and records a recent send', async () => {
    vi.mocked(composeApi.sendTemplate).mockResolvedValue({ from: 'a@example.com', to: ['b@example.com'] })
    render(<ComposerScreen />)

    fireEvent.click(screen.getByRole('button', { name: 'Send' }))

    await waitFor(() => expect(composeApi.sendTemplate).toHaveBeenCalled())
    await waitFor(() => expect(screen.getByText(/Sent to b@example.com/)).toBeInTheDocument())
  })

  it('adds an eml attachment and includes it in render/send calls, but not in json mode', async () => {
    vi.mocked(composeApi.sendTemplate).mockResolvedValue({ from: 'a@example.com', to: ['b@example.com'] })
    render(<ComposerScreen />)

    fireEvent.click(screen.getByRole('button', { name: '+ Add attachment' }))
    fireEvent.change(screen.getByPlaceholderText('{{ fPDF }}'), { target: { value: '{{ fCSV }}' } })
    fireEvent.change(screen.getByPlaceholderText('filename'), { target: { value: 'report.csv' } })

    await vi.advanceTimersByTimeAsync(500)
    await waitFor(() => expect(composeApi.renderTemplate).toHaveBeenCalled())
    let lastCall = vi.mocked(composeApi.renderTemplate).mock.calls.at(-1)?.[0]
    expect(lastCall?.attachments).toEqual([{ path: '{{ fCSV }}', filename: 'report.csv' }])

    fireEvent.click(screen.getByRole('button', { name: 'Send' }))
    await waitFor(() => expect(composeApi.sendTemplate).toHaveBeenCalled())
    const sendCall = vi.mocked(composeApi.sendTemplate).mock.calls.at(-1)?.[0]
    expect(sendCall?.attachments).toEqual([{ path: '{{ fCSV }}', filename: 'report.csv' }])

    // Switching to JSON mode must not carry the eml-only attachments along.
    fireEvent.click(screen.getByRole('button', { name: 'JSON' }))
    expect(screen.queryByRole('button', { name: '+ Add attachment' })).not.toBeInTheDocument()

    await vi.advanceTimersByTimeAsync(500)
    await waitFor(() => {
      lastCall = vi.mocked(composeApi.renderTemplate).mock.calls.at(-1)?.[0]
      expect(lastCall?.format).toBe('json')
    })
    expect(lastCall?.attachments).toBeUndefined()
  })

  it('shows resolved attachments from the render response', async () => {
    vi.mocked(composeApi.renderTemplate).mockResolvedValue({
      rendered: 'rendered output',
      attachments: [{ path: '/tmp/generated.csv', filename: 'report.csv' }],
    })
    render(<ComposerScreen />)

    await vi.advanceTimersByTimeAsync(500)
    await waitFor(() => expect(screen.getByText('report.csv')).toBeInTheDocument())
    expect(screen.getByText('Attachments (1)')).toBeInTheDocument()
  })
})

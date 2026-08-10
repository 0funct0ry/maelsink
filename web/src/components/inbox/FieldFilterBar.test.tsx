import { render, screen, fireEvent, act } from '@testing-library/react'
import FieldFilterBar from './FieldFilterBar'
import { useMessageStore } from '../../stores/useMessageStore'
import type { MessageSummary } from '../../lib/apiTypes'

function makeMessage(overrides: Partial<MessageSummary> = {}): MessageSummary {
  return {
    id: 'msg-1',
    from: 'alice@example.com',
    to: ['bob@example.com'],
    cc: [],
    bcc: [],
    subject: 'Welcome',
    size_bytes: 1024,
    has_attachments: false,
    attachment_count: 0,
    received_at: new Date().toISOString(),
    parse_warning: false,
    read: false,
    tags: [],
    preview: '',
    ...overrides,
  }
}

function resetStore() {
  useMessageStore.setState({ query: {}, messages: [] })
}

function openPanel() {
  fireEvent.click(screen.getByRole('button', { name: /filter by from, to, cc, bcc, subject/i }))
}

describe('FieldFilterBar', () => {
  beforeEach(() => {
    resetStore()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('renders closed by default', () => {
    render(<FieldFilterBar />)
    expect(screen.getByRole('button', { name: /filter by from, to, cc, bcc, subject/i })).toHaveAttribute(
      'aria-expanded',
      'false',
    )
    expect(screen.queryByLabelText('Filter by from')).not.toBeInTheDocument()
  })

  it('opens the panel and shows current query values in each field', () => {
    useMessageStore.setState({ query: { from: 'a@x.com', cc: 'b@y.com' } })
    render(<FieldFilterBar />)

    openPanel()

    expect(screen.getByLabelText('Filter by from')).toHaveValue('a@x.com')
    expect(screen.getByLabelText('Filter by cc')).toHaveValue('b@y.com')
    expect(screen.getByLabelText('Filter by to')).toHaveValue('')
  })

  it('shows a count badge for active filters', () => {
    useMessageStore.setState({ query: { from: 'a@x.com', bcc: 'c@z.com' } })
    render(<FieldFilterBar />)
    expect(screen.getByText('2')).toBeInTheDocument()
  })

  it('debounces each field independently before calling setQuery', () => {
    vi.useFakeTimers()
    const setQuery = vi.fn()
    useMessageStore.setState({ setQuery })
    render(<FieldFilterBar />)

    openPanel()
    fireEvent.change(screen.getByLabelText('Filter by cc'), { target: { value: 'qa@example.com' } })
    expect(setQuery).not.toHaveBeenCalled()

    act(() => {
      vi.advanceTimersByTime(299)
    })
    expect(setQuery).not.toHaveBeenCalled()

    act(() => {
      vi.advanceTimersByTime(1)
    })
    expect(setQuery).toHaveBeenCalledWith({ cc: 'qa@example.com' })
    expect(setQuery).toHaveBeenCalledTimes(1)
  })

  it('only fires once per field for rapid keystrokes within the debounce window', () => {
    vi.useFakeTimers()
    const setQuery = vi.fn()
    useMessageStore.setState({ setQuery })
    render(<FieldFilterBar />)

    openPanel()
    const input = screen.getByLabelText('Filter by bcc')
    fireEvent.change(input, { target: { value: 'b' } })
    act(() => {
      vi.advanceTimersByTime(100)
    })
    fireEvent.change(input, { target: { value: 'bc' } })
    act(() => {
      vi.advanceTimersByTime(100)
    })
    fireEvent.change(input, { target: { value: 'bcc@x.com' } })
    act(() => {
      vi.advanceTimersByTime(300)
    })

    expect(setQuery).toHaveBeenCalledTimes(1)
    expect(setQuery).toHaveBeenCalledWith({ bcc: 'bcc@x.com' })
  })

  it('offers the distinct values found in the loaded messages as dropdown options', () => {
    useMessageStore.setState({
      messages: [
        makeMessage({ id: 'a', from: 'alice@example.com' }),
        makeMessage({ id: 'b', from: 'zoe@example.com' }),
        makeMessage({ id: 'c', from: 'alice@example.com' }),
      ],
    })
    render(<FieldFilterBar />)

    openPanel()
    fireEvent.focus(screen.getByLabelText('Filter by from'))

    const options = screen.getAllByRole('option')
    expect(options.map((o) => o.textContent)).toEqual(['alice@example.com', 'zoe@example.com'])
  })

  it('flattens array fields and skips empty values when building options', () => {
    useMessageStore.setState({
      messages: [
        makeMessage({ id: 'a', cc: ['qa@example.com', 'ops@example.com'] }),
        makeMessage({ id: 'b', cc: ['qa@example.com'] }),
      ],
    })
    render(<FieldFilterBar />)

    openPanel()
    fireEvent.focus(screen.getByLabelText('Filter by cc'))

    expect(screen.getAllByRole('option').map((o) => o.textContent)).toEqual([
      'ops@example.com',
      'qa@example.com',
    ])
  })

  it('applies a picked dropdown value immediately, without waiting for the debounce', () => {
    vi.useFakeTimers()
    const setQuery = vi.fn()
    useMessageStore.setState({ setQuery, messages: [makeMessage({ from: 'alice@example.com' })] })
    render(<FieldFilterBar />)

    openPanel()
    fireEvent.focus(screen.getByLabelText('Filter by from'))
    fireEvent.click(screen.getByRole('button', { name: 'alice@example.com' }))

    expect(setQuery).toHaveBeenCalledWith({ from: 'alice@example.com' })
    expect(screen.queryByRole('option')).not.toBeInTheDocument()
  })

  it('filters dropdown options by what has been typed', () => {
    useMessageStore.setState({
      messages: [makeMessage({ id: 'a', from: 'alice@example.com' }), makeMessage({ id: 'b', from: 'zoe@example.com' })],
    })
    render(<FieldFilterBar />)

    openPanel()
    fireEvent.change(screen.getByLabelText('Filter by from'), { target: { value: 'zo' } })

    expect(screen.getAllByRole('option').map((o) => o.textContent)).toEqual(['zoe@example.com'])
  })

  it('shows no dropdown when the loaded messages have no values for that field', () => {
    useMessageStore.setState({ messages: [makeMessage({ bcc: [] })] })
    render(<FieldFilterBar />)

    openPanel()
    fireEvent.focus(screen.getByLabelText('Filter by bcc'))

    expect(screen.queryByRole('option')).not.toBeInTheDocument()
  })

  it('"Clear filters" resets all five fields but not q or sidebar filters', () => {
    const setQuery = vi.fn()
    useMessageStore.setState({
      setQuery,
      query: { q: 'keep-me', tag: ['keep-me-too'], from: 'a@x.com', to: 'b@x.com', cc: 'c@x.com', bcc: 'd@x.com', subject: 'hi' },
    })
    render(<FieldFilterBar />)

    openPanel()
    fireEvent.click(screen.getByRole('button', { name: 'Clear filters' }))

    expect(setQuery).toHaveBeenCalledWith({ from: '', to: '', cc: '', bcc: '', subject: '' })
  })

  it('disables "Clear filters" when no field filter is active', () => {
    render(<FieldFilterBar />)
    openPanel()
    expect(screen.getByRole('button', { name: 'Clear filters' })).toBeDisabled()
  })
})

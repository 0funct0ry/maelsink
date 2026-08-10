import { render, screen, fireEvent } from '@testing-library/react'
import ActiveFilterChips from './ActiveFilterChips'
import { useMessageStore } from '../../stores/useMessageStore'

describe('ActiveFilterChips', () => {
  beforeEach(() => {
    useMessageStore.setState({ query: {} })
  })

  it('renders nothing when no field filter is active', () => {
    const { container } = render(<ActiveFilterChips />)
    expect(container).toBeEmptyDOMElement()
  })

  it('ignores q and sidebar filters when deciding what to show', () => {
    useMessageStore.setState({ query: { q: 'hello', tag: ['smoke'], read: false } })
    const { container } = render(<ActiveFilterChips />)
    expect(container).toBeEmptyDOMElement()
  })

  it('renders one chip per active field filter', () => {
    useMessageStore.setState({ query: { from: 'billing@x.com', bcc: 'ops@x.com' } })
    render(<ActiveFilterChips />)

    expect(screen.getByText('from: billing@x.com')).toBeInTheDocument()
    expect(screen.getByText('bcc: ops@x.com')).toBeInTheDocument()
  })

  it('removing a chip clears only that field', () => {
    const setQuery = vi.fn()
    useMessageStore.setState({ setQuery, query: { q: 'keep-me', tag: ['keep-me-too'], from: 'billing@x.com' } })
    render(<ActiveFilterChips />)

    fireEvent.click(screen.getByLabelText('Remove from filter'))

    expect(setQuery).toHaveBeenCalledWith({ from: '' })
    expect(setQuery).toHaveBeenCalledTimes(1)
  })
})

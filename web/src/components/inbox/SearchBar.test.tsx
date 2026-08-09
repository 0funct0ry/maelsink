import { render, screen, fireEvent, act } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import SearchBar from './SearchBar'
import { useMessageStore } from '../../stores/useMessageStore'

function resetStore() {
  useMessageStore.setState({ query: {} })
}

function renderSearchBar() {
  return render(
    <MemoryRouter>
      <SearchBar />
    </MemoryRouter>,
  )
}

describe('SearchBar', () => {
  beforeEach(() => {
    resetStore()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('renders with the current query value', () => {
    useMessageStore.setState({ query: { q: 'hello' } })
    renderSearchBar()
    expect(screen.getByRole('searchbox')).toHaveValue('hello')
  })

  it('debounces calls to setQuery while typing', () => {
    vi.useFakeTimers()
    const setQuery = vi.fn()
    useMessageStore.setState({ setQuery })
    renderSearchBar()

    const input = screen.getByRole('searchbox')
    fireEvent.change(input, { target: { value: 'foo' } })
    expect(setQuery).not.toHaveBeenCalled()

    act(() => {
      vi.advanceTimersByTime(299)
    })
    expect(setQuery).not.toHaveBeenCalled()

    act(() => {
      vi.advanceTimersByTime(1)
    })
    expect(setQuery).toHaveBeenCalledWith({ q: 'foo' })
    expect(setQuery).toHaveBeenCalledTimes(1)
  })

  it('only fires once for rapid keystrokes within the debounce window', () => {
    vi.useFakeTimers()
    const setQuery = vi.fn()
    useMessageStore.setState({ setQuery })
    renderSearchBar()

    const input = screen.getByRole('searchbox')
    fireEvent.change(input, { target: { value: 'f' } })
    act(() => {
      vi.advanceTimersByTime(100)
    })
    fireEvent.change(input, { target: { value: 'fo' } })
    act(() => {
      vi.advanceTimersByTime(100)
    })
    fireEvent.change(input, { target: { value: 'foo' } })
    act(() => {
      vi.advanceTimersByTime(300)
    })

    expect(setQuery).toHaveBeenCalledTimes(1)
    expect(setQuery).toHaveBeenCalledWith({ q: 'foo' })
  })

  it('clears immediately without debounce and shows/hides the clear button', () => {
    const setQuery = vi.fn()
    useMessageStore.setState({ setQuery, query: { q: 'abc' } })
    renderSearchBar()

    expect(screen.getByLabelText('Clear search')).toBeInTheDocument()
    fireEvent.click(screen.getByLabelText('Clear search'))

    expect(setQuery).toHaveBeenCalledWith({ q: '' })
    expect(screen.getByRole('searchbox')).toHaveValue('')
    expect(screen.queryByLabelText('Clear search')).not.toBeInTheDocument()
  })
})

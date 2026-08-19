import { render, screen, fireEvent, act } from '@testing-library/react'
import TagBadge from './TagBadge'
import { useMessageStore } from '../../stores/useMessageStore'

describe('TagBadge', () => {
  afterEach(() => {
    useMessageStore.setState({ sidebarTags: [] })
  })

  it('renders the tag text', () => {
    render(<TagBadge tag="smoke" />)
    expect(screen.getByText('smoke')).toBeInTheDocument()
  })

  it('uses the persisted color token from the store, not a name hash, once a tag is loaded', () => {
    // "smoke" hashes to a fixed palette entry via tagColor() — pick a
    // different persisted token so a pass here can only mean paletteByToken
    // (the store-backed lookup) won, not the hash fallback coincidentally
    // agreeing with it.
    useMessageStore.setState({
      sidebarTags: [{ name: 'smoke', color: 'rose', count: 1, last_used: null }],
    })
    render(<TagBadge tag="smoke" />)
    const badge = screen.getByText('smoke').closest('span')
    expect(badge?.className).toContain('bg-rose-100')
  })

  it('recoloring the persisted tag updates the badge without remounting', () => {
    useMessageStore.setState({
      sidebarTags: [{ name: 'smoke', color: 'indigo', count: 1, last_used: null }],
    })
    render(<TagBadge tag="smoke" />)
    expect(screen.getByText('smoke').closest('span')?.className).toContain('bg-indigo-100')

    act(() => {
      useMessageStore.setState({
        sidebarTags: [{ name: 'smoke', color: 'rose', count: 1, last_used: null }],
      })
    })
    expect(screen.getByText('smoke').closest('span')?.className).toContain('bg-rose-100')
  })

  it('falls back to the name-hash color for a tag not yet in the store', () => {
    render(<TagBadge tag="unloaded-tag" />)
    expect(screen.getByText('unloaded-tag')).toBeInTheDocument()
  })

  it('renders no remove control by default', () => {
    render(<TagBadge tag="smoke" />)
    expect(screen.queryByLabelText('Remove tag smoke')).not.toBeInTheDocument()
  })

  it('calls onRemove when the remove control is clicked', () => {
    const onRemove = vi.fn()
    render(<TagBadge tag="smoke" onRemove={onRemove} />)
    fireEvent.click(screen.getByLabelText('Remove tag smoke'))
    expect(onRemove).toHaveBeenCalledTimes(1)
  })
})

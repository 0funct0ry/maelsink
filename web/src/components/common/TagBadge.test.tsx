import { render, screen, fireEvent } from '@testing-library/react'
import TagBadge from './TagBadge'

describe('TagBadge', () => {
  it('renders the tag text', () => {
    render(<TagBadge tag="smoke" />)
    expect(screen.getByText('smoke')).toBeInTheDocument()
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

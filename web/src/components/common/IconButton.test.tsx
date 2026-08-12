import { render, screen, fireEvent } from '@testing-library/react'
import { Trash2 } from 'lucide-react'
import IconButton from './IconButton'

describe('IconButton', () => {
  it('renders with the given aria-label', () => {
    render(<IconButton icon={<Trash2 />} aria-label="Delete message" />)
    expect(screen.getByRole('button', { name: 'Delete message' })).toBeInTheDocument()
  })

  it('calls onClick when clicked', () => {
    const onClick = vi.fn()
    render(<IconButton icon={<Trash2 />} aria-label="Delete" onClick={onClick} />)
    fireEvent.click(screen.getByRole('button', { name: 'Delete' }))
    expect(onClick).toHaveBeenCalledTimes(1)
  })

  it('applies danger variant classes', () => {
    render(<IconButton icon={<Trash2 />} aria-label="Delete" variant="danger" />)
    expect(screen.getByRole('button', { name: 'Delete' }).className).toMatch(/danger/)
  })

  it('applies sm size classes', () => {
    render(<IconButton icon={<Trash2 />} aria-label="Delete" size="sm" />)
    expect(screen.getByRole('button', { name: 'Delete' }).className).toMatch(/h-7/)
  })

  // `aria-label` is a required prop on IconButtonProps (not optional), so
  // omitting it — e.g. `<IconButton icon={<Trash2 />} />` — is a
  // compile-time TypeScript error, not something to assert at runtime.

  it('renders a visible tooltip with the same text as the aria-label', () => {
    render(<IconButton icon={<Trash2 />} aria-label="Delete message" />)
    expect(screen.getByRole('tooltip')).toHaveTextContent('Delete message')
  })
})

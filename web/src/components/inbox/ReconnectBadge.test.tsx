import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import ReconnectBadge from './ReconnectBadge'

describe('ReconnectBadge', () => {
  it('renders nothing when open', () => {
    const { container } = render(<ReconnectBadge status="open" />)
    expect(container).toBeEmptyDOMElement()
  })

  it('renders nothing when connecting', () => {
    const { container } = render(<ReconnectBadge status="connecting" />)
    expect(container).toBeEmptyDOMElement()
  })

  it('renders nothing when closed', () => {
    const { container } = render(<ReconnectBadge status="closed" />)
    expect(container).toBeEmptyDOMElement()
  })

  it('renders the reconnecting pill when reconnecting', () => {
    render(<ReconnectBadge status="reconnecting" />)
    expect(screen.getByText('Reconnecting…')).toBeInTheDocument()
  })
})

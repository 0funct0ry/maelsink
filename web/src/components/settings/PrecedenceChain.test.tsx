import { render, screen } from '@testing-library/react'
import PrecedenceChain from './PrecedenceChain'

describe('PrecedenceChain', () => {
  it('renders the four precedence stages in order', () => {
    render(<PrecedenceChain />)
    expect(screen.getByText('Default')).toBeInTheDocument()
    expect(screen.getByText('Config file')).toBeInTheDocument()
    expect(screen.getByText('Environment variable')).toBeInTheDocument()
    expect(screen.getByText('CLI flag')).toBeInTheDocument()
  })

  it('makes clear this is a static legend, not live data', () => {
    render(<PrecedenceChain />)
    expect(screen.getByText(/legend — not live data/i)).toBeInTheDocument()
    expect(screen.getByText(/does not currently track which layer resolved/i)).toBeInTheDocument()
  })
})

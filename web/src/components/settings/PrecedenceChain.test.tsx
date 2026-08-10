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

  it('no longer disclaims the data as illustrative-only, now that provenance is live', () => {
    render(<PrecedenceChain />)
    expect(screen.queryByText(/legend — not live data/i)).not.toBeInTheDocument()
    expect(screen.queryByText(/does not currently track which layer resolved/i)).not.toBeInTheDocument()
    expect(screen.getByText(/shows exactly/i)).toBeInTheDocument()
  })
})

import { render, screen } from '@testing-library/react'
import FilteredEmptyState from './FilteredEmptyState'

describe('FilteredEmptyState', () => {
  it('renders copy distinct from the first-run empty state', () => {
    render(<FilteredEmptyState />)
    expect(screen.getByText('No matching messages')).toBeInTheDocument()
    expect(screen.queryByText(/SMTP/)).not.toBeInTheDocument()
  })
})

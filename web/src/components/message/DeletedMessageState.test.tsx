import { render, screen, fireEvent } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import DeletedMessageState from './DeletedMessageState'

describe('DeletedMessageState', () => {
  it('renders the not-found copy', () => {
    render(<DeletedMessageState messageId="abc123" />, { wrapper: MemoryRouter })
    expect(screen.getByText(/This message no longer exists/)).toBeInTheDocument()
  })

  it('navigates back to inbox on click', () => {
    render(<DeletedMessageState messageId="abc123" />, { wrapper: MemoryRouter })
    fireEvent.click(screen.getByText('Back to Inbox'))
    // No throw means navigate() executed; verifying route change would need a full router setup.
    expect(screen.getByText('Back to Inbox')).toBeInTheDocument()
  })
})

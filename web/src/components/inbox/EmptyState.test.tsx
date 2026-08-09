import { render, screen, waitFor } from '@testing-library/react'
import EmptyState from './EmptyState'
import * as uiApiClient from '../../lib/uiApiClient'

vi.mock('../../lib/uiApiClient')

describe('EmptyState', () => {
  it('shows generic copy while info is loading / unavailable', async () => {
    vi.mocked(uiApiClient.getInfo).mockRejectedValue(new Error('network error'))
    render(<EmptyState />)
    expect(screen.getByText('No messages yet')).toBeInTheDocument()
    await waitFor(() => {
      expect(screen.getByText(/Send mail through the SMTP server/)).toBeInTheDocument()
    })
  })

  it('shows the SMTP host:port once info resolves', async () => {
    vi.mocked(uiApiClient.getInfo).mockResolvedValue({
      smtp: { host: '127.0.0.1', port: 1025 },
      auth_enabled: false,
    })
    render(<EmptyState />)
    await waitFor(() => {
      expect(screen.getByText('127.0.0.1:1025')).toBeInTheDocument()
    })
  })
})

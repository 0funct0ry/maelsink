import { render, screen, waitFor } from '@testing-library/react'
import ConnectionInfoCard from './ConnectionInfoCard'
import { getInfo } from '../../lib/uiApiClient'

vi.mock('../../lib/uiApiClient')

describe('ConnectionInfoCard', () => {
  it('shows a loading state before data arrives', () => {
    vi.mocked(getInfo).mockReturnValue(new Promise(() => {}))
    render(<ConnectionInfoCard />)
    expect(screen.getByTestId('connection-loading')).toBeInTheDocument()
  })

  it('renders host:port and auth enabled badge', async () => {
    vi.mocked(getInfo).mockResolvedValue({
      smtp: { host: '127.0.0.1', port: 1025 },
      auth_enabled: true,
      db_filename: 'maelsink.db',
    })
    render(<ConnectionInfoCard />)
    await waitFor(() => expect(screen.getByText('127.0.0.1:1025')).toBeInTheDocument())
    expect(screen.getByText('Enabled')).toBeInTheDocument()
  })

  it('renders auth disabled badge', async () => {
    vi.mocked(getInfo).mockResolvedValue({
      smtp: { host: '0.0.0.0', port: 1025 },
      auth_enabled: false,
      db_filename: 'maelsink.db',
    })
    render(<ConnectionInfoCard />)
    await waitFor(() => expect(screen.getByText('Disabled')).toBeInTheDocument())
  })

  it('shows an inline error without crashing on failure', async () => {
    vi.mocked(getInfo).mockRejectedValue(new Error('boom'))
    render(<ConnectionInfoCard />)
    await waitFor(() =>
      expect(screen.getByText(/failed to load connection info/i)).toBeInTheDocument(),
    )
  })
})

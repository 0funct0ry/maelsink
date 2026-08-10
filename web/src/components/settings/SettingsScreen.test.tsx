import { render, screen, waitFor } from '@testing-library/react'
import SettingsScreen from './SettingsScreen'
import { getStats, getVersion } from '../../lib/apiClient'
import { getConfig, getInfo } from '../../lib/uiApiClient'

vi.mock('../../lib/apiClient')
vi.mock('../../lib/uiApiClient')

describe('SettingsScreen', () => {
  beforeEach(() => {
    vi.mocked(getStats).mockResolvedValue({
      total_messages: 1,
      total_size_bytes: 100,
      unread_count: 0,
      attachment_count: 0,
      parse_warning_count: 0,
      oldest_received_at: null,
      newest_received_at: null,
    })
    vi.mocked(getInfo).mockResolvedValue({
      smtp: { host: '127.0.0.1', port: 1025 },
      auth_enabled: false,
    })
    vi.mocked(getConfig).mockResolvedValue([])
  })

  it('renders the Settings heading', () => {
    vi.mocked(getVersion).mockReturnValue(new Promise(() => {}))
    render(<SettingsScreen />)
    expect(screen.getByRole('heading', { name: 'Settings' })).toBeInTheDocument()
  })

  it('renders version info once loaded', async () => {
    vi.mocked(getVersion).mockResolvedValue({ version: '1.2.3', commit: 'abc123', go: 'go1.26.4' })
    render(<SettingsScreen />)
    await waitFor(() => expect(screen.getByText('1.2.3')).toBeInTheDocument())
    expect(screen.getByText('abc123')).toBeInTheDocument()
    expect(screen.queryByText('go1.26.4')).not.toBeInTheDocument()
  })

  it('shows an inline version error without crashing the rest of the screen', async () => {
    vi.mocked(getVersion).mockRejectedValue(new Error('boom'))
    render(<SettingsScreen />)
    await waitFor(() =>
      expect(screen.getByText(/failed to load version info/i)).toBeInTheDocument(),
    )
    // Other independent sections still render fine.
    expect(await screen.findByText('127.0.0.1:1025')).toBeInTheDocument()
  })
})

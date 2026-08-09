import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import ConfigTable from './ConfigTable'
import { getInfo } from '../../lib/uiApiClient'

vi.mock('../../lib/uiApiClient')

describe('ConfigTable', () => {
  it('shows a loading state before data arrives', () => {
    vi.mocked(getInfo).mockReturnValue(new Promise(() => {}))
    render(<ConfigTable />)
    expect(screen.getByTestId('config-loading')).toBeInTheDocument()
  })

  it('renders the smtp host/port and auth rows', async () => {
    vi.mocked(getInfo).mockResolvedValue({
      smtp: { host: '127.0.0.1', port: 1025 },
      auth_enabled: true,
    })
    render(<ConfigTable />)
    await waitFor(() => expect(screen.getByText('SMTP Host')).toBeInTheDocument())
    expect(screen.getByText('127.0.0.1')).toBeInTheDocument()
    expect(screen.getByText('SMTP Port')).toBeInTheDocument()
    expect(screen.getByText('1025')).toBeInTheDocument()
    expect(screen.getByText('API Auth Enabled')).toBeInTheDocument()
    expect(screen.getByText('true')).toBeInTheDocument()
  })

  it('filters rows by label substring, case-insensitively', async () => {
    vi.mocked(getInfo).mockResolvedValue({
      smtp: { host: '127.0.0.1', port: 1025 },
      auth_enabled: false,
    })
    render(<ConfigTable />)
    await waitFor(() => expect(screen.getByText('SMTP Host')).toBeInTheDocument())

    fireEvent.change(screen.getByLabelText(/filter config fields/i), { target: { value: 'auth' } })

    expect(screen.queryByText('SMTP Host')).not.toBeInTheDocument()
    expect(screen.getByText('API Auth Enabled')).toBeInTheDocument()
  })

  it('shows an inline error without crashing on failure', async () => {
    vi.mocked(getInfo).mockRejectedValue(new Error('boom'))
    render(<ConfigTable />)
    await waitFor(() => expect(screen.getByText(/failed to load config/i)).toBeInTheDocument())
  })
})

import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import ConfigTable from './ConfigTable'
import { getConfig } from '../../lib/uiApiClient'
import type { ConfigEntry } from '../../lib/apiTypes'

vi.mock('../../lib/uiApiClient')

const sampleEntries: ConfigEntry[] = [
  {
    section: 'smtp',
    key: 'smtp.host',
    value: '127.0.0.1',
    secret: false,
    source: { layer: 'default', origin: '' },
  },
  {
    section: 'smtp',
    key: 'smtp.port',
    value: 1025,
    secret: false,
    source: { layer: 'file', origin: 'maelsink.yaml' },
  },
  {
    section: 'smtp',
    key: 'smtp.auth.password',
    value: true,
    secret: true,
    source: { layer: 'flag', origin: '--smtp-auth-password=***' },
  },
  {
    section: 'web',
    key: 'web.host',
    value: '0.0.0.0',
    secret: false,
    source: { layer: 'env', origin: 'MAELSINK_WEB_HOST' },
  },
  {
    section: 'api',
    key: 'api.port',
    value: 9999,
    secret: false,
    source: { layer: 'flag', origin: '--api-port=9999' },
  },
  {
    section: 'storage',
    key: 'storage.path',
    value: '',
    secret: false,
    source: { layer: 'default', origin: '' },
  },
]

describe('ConfigTable', () => {
  it('shows a loading state before data arrives', () => {
    vi.mocked(getConfig).mockReturnValue(new Promise(() => {}))
    render(<ConfigTable />)
    expect(screen.getByTestId('config-loading')).toBeInTheDocument()
  })

  it('renders entries grouped by section with key/value/source/origin', async () => {
    vi.mocked(getConfig).mockResolvedValue(sampleEntries)
    render(<ConfigTable />)
    await waitFor(() => expect(screen.getByText('smtp.host')).toBeInTheDocument())

    expect(screen.getByText('smtp')).toBeInTheDocument()
    expect(screen.getByText('web')).toBeInTheDocument()
    expect(screen.getByText('api')).toBeInTheDocument()

    expect(screen.getByText('127.0.0.1')).toBeInTheDocument()
    expect(screen.getByText('1025')).toBeInTheDocument()
    expect(screen.getByText('maelsink.yaml')).toBeInTheDocument()
    expect(screen.getByText('MAELSINK_WEB_HOST')).toBeInTheDocument()
    expect(screen.getByText('--api-port=9999')).toBeInTheDocument()

    expect(screen.getAllByText('Default').length).toBeGreaterThan(0)
    expect(screen.getByText('Config file')).toBeInTheDocument()
    expect(screen.getByText('Environment variable')).toBeInTheDocument()
    expect(screen.getAllByText('CLI flag').length).toBeGreaterThan(0)
  })

  it('filters rows by key substring, case-insensitively', async () => {
    vi.mocked(getConfig).mockResolvedValue(sampleEntries)
    render(<ConfigTable />)
    await waitFor(() => expect(screen.getByText('smtp.host')).toBeInTheDocument())

    fireEvent.change(screen.getByLabelText(/filter config fields/i), { target: { value: 'web' } })

    expect(screen.queryByText('smtp.host')).not.toBeInTheDocument()
    expect(screen.getByText('web.host')).toBeInTheDocument()
  })

  it('filters rows by source chip', async () => {
    vi.mocked(getConfig).mockResolvedValue(sampleEntries)
    render(<ConfigTable />)
    await waitFor(() => expect(screen.getByText('smtp.host')).toBeInTheDocument())

    fireEvent.click(screen.getByRole('button', { name: 'Flag' }))

    expect(screen.queryByText('smtp.host')).not.toBeInTheDocument()
    expect(screen.getByText('api.port')).toBeInTheDocument()
  })

  it('shows "No matching fields" when the filter excludes everything', async () => {
    vi.mocked(getConfig).mockResolvedValue(sampleEntries)
    render(<ConfigTable />)
    await waitFor(() => expect(screen.getByText('smtp.host')).toBeInTheDocument())

    fireEvent.change(screen.getByLabelText(/filter config fields/i), { target: { value: 'nonexistent' } })

    expect(screen.getByText('No matching fields.')).toBeInTheDocument()
  })

  it('shows an inline error without crashing on failure', async () => {
    vi.mocked(getConfig).mockRejectedValue(new Error('boom'))
    render(<ConfigTable />)
    await waitFor(() => expect(screen.getByText(/failed to load config/i)).toBeInTheDocument())
  })

  it('shows "(in-memory)" for an empty storage.path instead of a blank cell', async () => {
    vi.mocked(getConfig).mockResolvedValue(sampleEntries)
    render(<ConfigTable />)
    await waitFor(() => expect(screen.getByText('storage.path')).toBeInTheDocument())

    expect(screen.getByText('(in-memory)')).toBeInTheDocument()
  })

  it('masks secret entries instead of showing their value', async () => {
    vi.mocked(getConfig).mockResolvedValue(sampleEntries)
    render(<ConfigTable />)
    await waitFor(() => expect(screen.getByText('smtp.auth.password')).toBeInTheDocument())

    expect(screen.getByText('Secret')).toBeInTheDocument()
    expect(screen.getByText('••••••••')).toBeInTheDocument()
    expect(screen.queryByText('true')).not.toBeInTheDocument()
  })
})

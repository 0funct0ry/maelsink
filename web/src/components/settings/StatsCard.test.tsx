import { render, screen, waitFor } from '@testing-library/react'
import StatsCard from './StatsCard'
import { getStats } from '../../lib/apiClient'

vi.mock('../../lib/apiClient')

describe('StatsCard', () => {
  it('shows a loading state before data arrives', () => {
    vi.mocked(getStats).mockReturnValue(new Promise(() => {}))
    render(<StatsCard />)
    expect(screen.getByTestId('stats-loading')).toBeInTheDocument()
  })

  it('renders populated stats', async () => {
    vi.mocked(getStats).mockResolvedValue({
      total_messages: 42,
      total_size_bytes: 1536,
      oldest_received_at: '2024-01-01T00:00:00Z',
      newest_received_at: '2024-01-02T00:00:00Z',
    })
    render(<StatsCard />)
    await waitFor(() => expect(screen.getByText('42')).toBeInTheDocument())
    expect(screen.getByText('1.5 KB')).toBeInTheDocument()
  })

  it('renders "—" for null oldest/newest timestamps', async () => {
    vi.mocked(getStats).mockResolvedValue({
      total_messages: 0,
      total_size_bytes: 0,
      oldest_received_at: null,
      newest_received_at: null,
    })
    render(<StatsCard />)
    await waitFor(() => expect(screen.getAllByText('—')).toHaveLength(2))
  })

  it('shows an inline error without crashing on failure', async () => {
    vi.mocked(getStats).mockRejectedValue(new Error('boom'))
    render(<StatsCard />)
    await waitFor(() =>
      expect(screen.getByText(/failed to load message stats/i)).toBeInTheDocument(),
    )
  })
})

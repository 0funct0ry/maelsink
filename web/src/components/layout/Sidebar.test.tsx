import { render, screen, fireEvent, waitFor, act } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import Sidebar from './Sidebar'
import { useMessageStore } from '../../stores/useMessageStore'
import { useUIStore } from '../../stores/useUIStore'
import * as apiClient from '../../lib/apiClient'

vi.mock('../../lib/apiClient')

function renderSidebar() {
  return render(
    <MemoryRouter>
      <Sidebar />
    </MemoryRouter>,
  )
}

describe('Sidebar', () => {
  beforeEach(() => {
    useMessageStore.setState({ total: 5 })
    useUIStore.setState({ modal: null })
  })

  it('shows the All messages count from the store', () => {
    vi.mocked(apiClient.getStats).mockRejectedValue(new Error('offline'))
    renderSidebar()
    expect(screen.getByText('All messages')).toBeInTheDocument()
    expect(screen.getByText('5')).toBeInTheDocument()
  })

  it('renders the storage card once stats load', async () => {
    vi.mocked(apiClient.getStats).mockResolvedValue({
      total_messages: 5,
      total_size_bytes: 2048,
      oldest_received_at: null,
      newest_received_at: null,
    })
    renderSidebar()
    await waitFor(() => expect(screen.getByText('Storage used')).toBeInTheDocument())
    expect(screen.getByText('2.0 KB')).toBeInTheDocument()
  })

  it('does not render the storage card when stats fail to load', async () => {
    vi.mocked(apiClient.getStats).mockRejectedValue(new Error('offline'))
    renderSidebar()
    await waitFor(() => expect(apiClient.getStats).toHaveBeenCalled())
    expect(screen.queryByText('Storage used')).not.toBeInTheDocument()
  })

  it('opens a confirm dialog via useUIStore when Clear all messages is clicked', () => {
    vi.mocked(apiClient.getStats).mockRejectedValue(new Error('offline'))
    renderSidebar()
    fireEvent.click(screen.getByText('Clear all messages'))
    expect(useUIStore.getState().modal).toMatchObject({ kind: 'confirm', danger: true })
  })

  it('calls clearAll when the opened confirm is invoked', () => {
    vi.mocked(apiClient.getStats).mockRejectedValue(new Error('offline'))
    const clearAll = vi.fn().mockResolvedValue(undefined)
    useMessageStore.setState({ clearAll })
    renderSidebar()
    fireEvent.click(screen.getByText('Clear all messages'))
    act(() => {
      useUIStore.getState().modal?.onConfirm()
    })
    expect(clearAll).toHaveBeenCalledTimes(1)
  })
})

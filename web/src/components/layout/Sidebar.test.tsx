import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { MemoryRouter, Route, Routes, useLocation } from 'react-router-dom'
import Sidebar from './Sidebar'
import { useMessageStore } from '../../stores/useMessageStore'
import * as apiClient from '../../lib/apiClient'

vi.mock('../../lib/apiClient')

// jsdom's localStorage is flaky under this test runner's sandboxing (see
// useUIStore.test.ts), so stub it with a plain in-memory implementation —
// Sidebar's saved-searches group reads/writes it on mount.
function makeMemoryStorage(): Storage {
  const data = new Map<string, string>()
  return {
    getItem: (key: string) => (data.has(key) ? data.get(key)! : null),
    setItem: (key: string, value: string) => data.set(key, value),
    removeItem: (key: string) => data.delete(key),
    clear: () => data.clear(),
    key: (index: number) => Array.from(data.keys())[index] ?? null,
    get length() {
      return data.size
    },
  } as Storage
}

function renderSidebar() {
  return render(
    <MemoryRouter>
      <Sidebar />
    </MemoryRouter>,
  )
}

function LocationProbe() {
  return <div data-testid="location">{useLocation().pathname}</div>
}

function renderSidebarAt(initialPath: string) {
  return render(
    <MemoryRouter initialEntries={[initialPath]}>
      <Sidebar />
      <Routes>
        <Route path="*" element={<LocationProbe />} />
      </Routes>
    </MemoryRouter>,
  )
}

describe('Sidebar', () => {
  beforeEach(() => {
    vi.stubGlobal('localStorage', makeMemoryStorage())
    useMessageStore.setState({ total: 5, query: {}, setQuery: vi.fn() })
    vi.mocked(apiClient.getTags).mockResolvedValue([])
  })

  afterEach(() => {
    vi.unstubAllGlobals()
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
      unread_count: 0,
      attachment_count: 0,
      parse_warning_count: 0,
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

  it('clicking Unread sets query.read=false', () => {
    vi.mocked(apiClient.getStats).mockRejectedValue(new Error('offline'))
    const setQuery = vi.fn()
    useMessageStore.setState({ setQuery })
    renderSidebar()
    fireEvent.click(screen.getByText('Unread'))
    expect(setQuery).toHaveBeenCalledWith(
      expect.objectContaining({ read: false, has_attachments: undefined, parse_warning: undefined }),
    )
  })

  it('clicking With attachments sets query.has_attachments=true', () => {
    vi.mocked(apiClient.getStats).mockRejectedValue(new Error('offline'))
    const setQuery = vi.fn()
    useMessageStore.setState({ setQuery })
    renderSidebar()
    fireEvent.click(screen.getByText('With attachments'))
    expect(setQuery).toHaveBeenCalledWith(expect.objectContaining({ has_attachments: true }))
  })

  it('clicking Parse warnings sets query.parse_warning=true', () => {
    vi.mocked(apiClient.getStats).mockRejectedValue(new Error('offline'))
    const setQuery = vi.fn()
    useMessageStore.setState({ setQuery })
    renderSidebar()
    fireEvent.click(screen.getByText('Parse warnings'))
    expect(setQuery).toHaveBeenCalledWith(expect.objectContaining({ parse_warning: true }))
  })

  it('navigates back to the Inbox when a mailbox filter is clicked from Message Detail', () => {
    vi.mocked(apiClient.getStats).mockRejectedValue(new Error('offline'))
    useMessageStore.setState({ setQuery: vi.fn() })
    renderSidebarAt('/messages/abc123')
    expect(screen.getByTestId('location').textContent).toBe('/messages/abc123')
    fireEvent.click(screen.getByText('Unread'))
    expect(screen.getByTestId('location').textContent).toBe('/')
  })

  it('navigates back to the Inbox when a tag is clicked from Message Detail', async () => {
    vi.mocked(apiClient.getStats).mockRejectedValue(new Error('offline'))
    vi.mocked(apiClient.getTags).mockResolvedValue([{ tag: 'smoke', count: 4 }])
    useMessageStore.setState({ setQuery: vi.fn() })
    renderSidebarAt('/messages/abc123')
    await screen.findByText('smoke')
    fireEvent.click(screen.getByText('smoke'))
    expect(screen.getByTestId('location').textContent).toBe('/')
  })

  it('shows mailbox counts once stats load', async () => {
    vi.mocked(apiClient.getStats).mockResolvedValue({
      total_messages: 5,
      total_size_bytes: 2048,
      unread_count: 3,
      attachment_count: 2,
      parse_warning_count: 1,
      oldest_received_at: null,
      newest_received_at: null,
    })
    renderSidebar()
    await waitFor(() => expect(screen.getByText('3')).toBeInTheDocument())
    expect(screen.getByText('2')).toBeInTheDocument()
    expect(screen.getByText('1')).toBeInTheDocument()
  })

  it('keeps the All messages count at the unfiltered stats total, not the active filter\'s result count', async () => {
    vi.mocked(apiClient.getStats).mockResolvedValue({
      total_messages: 10,
      total_size_bytes: 2048,
      unread_count: 9,
      attachment_count: 0,
      parse_warning_count: 0,
      oldest_received_at: null,
      newest_received_at: null,
    })
    // Simulate the Unread filter being active: the store's `total` reflects
    // only the filtered result set (7), which must not leak into the
    // "All messages" badge.
    useMessageStore.setState({ total: 7, query: { read: false }, setQuery: vi.fn() })
    renderSidebar()
    await waitFor(() => expect(screen.getByText('10')).toBeInTheDocument())
    expect(screen.queryByText('7')).not.toBeInTheDocument()
  })

  it('renders tags from getTags and filters by tag on click', async () => {
    vi.mocked(apiClient.getStats).mockRejectedValue(new Error('offline'))
    vi.mocked(apiClient.getTags).mockResolvedValue([{ tag: 'smoke', count: 4 }])
    const setQuery = vi.fn()
    useMessageStore.setState({ setQuery })
    renderSidebar()
    await waitFor(() => expect(screen.getByText('smoke')).toBeInTheDocument())
    fireEvent.click(screen.getByText('smoke'))
    expect(setQuery).toHaveBeenCalledWith(expect.objectContaining({ tag: 'smoke' }))
  })

  it('does not render a Tags section when there are no tags', async () => {
    vi.mocked(apiClient.getStats).mockRejectedValue(new Error('offline'))
    vi.mocked(apiClient.getTags).mockResolvedValue([])
    renderSidebar()
    await waitFor(() => expect(apiClient.getTags).toHaveBeenCalled())
    expect(screen.queryByText('Tags')).not.toBeInTheDocument()
  })

  it('saves a search and can reapply it', () => {
    vi.mocked(apiClient.getStats).mockRejectedValue(new Error('offline'))
    const setQuery = vi.fn()
    useMessageStore.setState({ query: { subject: 'invoice' }, setQuery })
    renderSidebar()

    fireEvent.change(screen.getByLabelText('Saved search name'), { target: { value: 'Invoices' } })
    fireEvent.click(screen.getByText('Save'))

    const savedButton = screen.getByText('Invoices')
    fireEvent.click(savedButton)
    expect(setQuery).toHaveBeenCalledWith({ subject: 'invoice' })
  })

  it('deletes a saved search', () => {
    vi.mocked(apiClient.getStats).mockRejectedValue(new Error('offline'))
    useMessageStore.setState({ query: { subject: 'invoice' } })
    renderSidebar()

    fireEvent.change(screen.getByLabelText('Saved search name'), { target: { value: 'Invoices' } })
    fireEvent.click(screen.getByText('Save'))
    expect(screen.getByText('Invoices')).toBeInTheDocument()

    fireEvent.click(screen.getByLabelText('Delete saved search Invoices'))
    expect(screen.queryByText('Invoices')).not.toBeInTheDocument()
  })
})

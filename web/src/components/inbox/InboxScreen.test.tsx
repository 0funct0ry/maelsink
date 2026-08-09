import { render, screen, fireEvent } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import InboxScreen from './InboxScreen'
import { useMessageStore } from '../../stores/useMessageStore'
import * as uiApiClient from '../../lib/uiApiClient'

vi.mock('../../lib/uiApiClient')

function renderScreen() {
  return render(
    <MemoryRouter>
      <InboxScreen />
    </MemoryRouter>,
  )
}

describe('InboxScreen', () => {
  beforeEach(() => {
    vi.mocked(uiApiClient.getInfo).mockRejectedValue(new Error('no info'))
    useMessageStore.setState({
      messages: [],
      total: 3,
      limit: 50,
      offset: 0,
      query: {},
      listStatus: 'idle',
      listError: null,
      fetchMessages: vi.fn(),
      setQuery: vi.fn(),
    })
  })

  it('fetches messages on mount', () => {
    const fetchMessages = vi.fn()
    useMessageStore.setState({ fetchMessages })
    renderScreen()
    expect(fetchMessages).toHaveBeenCalledTimes(1)
  })

  it('shows the live message count in the header', () => {
    renderScreen()
    expect(screen.getByText('3 captured')).toBeInTheDocument()
  })

  it('renders an Export All link pointing at exportAllUrl', () => {
    renderScreen()
    const link = screen.getByTitle('Export all messages as .zip')
    expect(link).toHaveAttribute('download')
    expect(link).toHaveAttribute('href', expect.stringContaining('/api/v1/messages/export'))
  })

  it('opening the sort menu and picking Oldest updates the query', () => {
    const setQuery = vi.fn()
    useMessageStore.setState({ setQuery })
    renderScreen()
    fireEvent.click(screen.getByText(/Sort: Newest/))
    fireEvent.click(screen.getByText('Oldest'))
    expect(setQuery).toHaveBeenCalledWith({ sort: 'received_at_asc' })
  })
})

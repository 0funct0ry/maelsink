import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import TopBar from './TopBar'
import * as uiApiClient from '../../lib/uiApiClient'

vi.mock('../../lib/uiApiClient')

function renderTopBar() {
  return render(
    <MemoryRouter>
      <TopBar />
    </MemoryRouter>,
  )
}

describe('TopBar', () => {
  it('renders the wordmark', () => {
    vi.mocked(uiApiClient.getInfo).mockRejectedValue(new Error('offline'))
    renderTopBar()
    expect(screen.getByText('maelsink')).toBeInTheDocument()
  })

  it('shows the live SMTP connection pill once info loads', async () => {
    vi.mocked(uiApiClient.getInfo).mockResolvedValue({
      smtp: { host: '127.0.0.1', port: 1025 },
      auth_enabled: false,
    })
    renderTopBar()
    await waitFor(() => expect(screen.getByText(/smtp:\/\/127\.0\.0\.1:1025/)).toBeInTheDocument())
  })

  it('does not render the connection pill when info fails to load', async () => {
    vi.mocked(uiApiClient.getInfo).mockRejectedValue(new Error('offline'))
    renderTopBar()
    await waitFor(() => expect(uiApiClient.getInfo).toHaveBeenCalled())
    expect(screen.queryByText(/smtp:\/\//)).not.toBeInTheDocument()
  })

  it('renders a global search box and a settings shortcut', () => {
    vi.mocked(uiApiClient.getInfo).mockRejectedValue(new Error('offline'))
    renderTopBar()
    expect(screen.getByRole('searchbox')).toBeInTheDocument()
    expect(screen.getByLabelText('Settings')).toBeInTheDocument()
  })
})

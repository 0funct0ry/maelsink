import { render, screen, fireEvent, act } from '@testing-library/react'
import { MemoryRouter, useLocation } from 'react-router-dom'
import TopBar from './TopBar'
import { useMessageStore } from '../../stores/useMessageStore'
import { useUIStore } from '../../stores/useUIStore'

function renderTopBar() {
  return render(
    <MemoryRouter>
      <TopBar />
    </MemoryRouter>,
  )
}

function LocationProbe() {
  return <div data-testid="location">{useLocation().pathname}</div>
}

describe('TopBar', () => {
  beforeEach(() => {
    useUIStore.setState({ modal: null, wsStatus: 'connecting' })
  })

  it('renders the wordmark', () => {
    renderTopBar()
    expect(screen.getByText('maelsink')).toBeInTheDocument()
  })

  it('renders a global search box, a clear-all shortcut, a tags shortcut, and a settings shortcut', () => {
    renderTopBar()
    expect(screen.getByRole('searchbox')).toBeInTheDocument()
    expect(screen.getByLabelText('Clear all messages')).toBeInTheDocument()
    expect(screen.getByLabelText('Manage tags')).toBeInTheDocument()
    expect(screen.getByLabelText('Settings')).toBeInTheDocument()
  })

  it('navigates to /tags when the tags icon is clicked', () => {
    render(
      <MemoryRouter initialEntries={['/']}>
        <TopBar />
        <LocationProbe />
      </MemoryRouter>,
    )
    fireEvent.click(screen.getByLabelText('Manage tags'))
    expect(screen.getByTestId('location').textContent).toBe('/tags')
  })

  it('opens a confirm dialog via useUIStore when the clear-all icon is clicked', () => {
    renderTopBar()
    fireEvent.click(screen.getByLabelText('Clear all messages'))
    expect(useUIStore.getState().modal).toMatchObject({ kind: 'confirm', danger: true })
  })

  it('calls clearAll when the opened confirm is invoked', () => {
    const clearAll = vi.fn().mockResolvedValue(undefined)
    useMessageStore.setState({ clearAll })
    renderTopBar()
    fireEvent.click(screen.getByLabelText('Clear all messages'))
    act(() => {
      useUIStore.getState().modal?.onConfirm()
    })
    expect(clearAll).toHaveBeenCalledTimes(1)
  })

  it('shows the reconnecting badge when useUIStore.wsStatus is reconnecting (M7.0)', () => {
    useUIStore.setState({ wsStatus: 'reconnecting' })
    renderTopBar()
    expect(screen.getByText('Reconnecting…')).toBeInTheDocument()
  })

  it('hides the reconnecting badge once wsStatus returns to open', () => {
    useUIStore.setState({ wsStatus: 'open' })
    renderTopBar()
    expect(screen.queryByText('Reconnecting…')).not.toBeInTheDocument()
  })

  it('opens the mobile navigation drawer with the sidebar content', () => {
    renderTopBar()
    expect(screen.queryByText('Mailbox')).not.toBeInTheDocument()
    fireEvent.click(screen.getByLabelText('Open navigation'))
    expect(screen.getByText('Mailbox')).toBeInTheDocument()
    expect(screen.getByText('All messages')).toBeInTheDocument()
  })

  it('closes the drawer on navigation', () => {
    renderTopBar()
    fireEvent.click(screen.getByLabelText('Open navigation'))
    expect(screen.getByText('Mailbox')).toBeInTheDocument()
    fireEvent.click(screen.getByLabelText('Manage tags'))
    expect(screen.queryByText('Mailbox')).not.toBeInTheDocument()
  })
})

import { render, screen, fireEvent, act } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import AppShell from './AppShell'
import { useMessageStore } from '../../stores/useMessageStore'
import { useUIStore } from '../../stores/useUIStore'
import * as apiClient from '../../lib/apiClient'
import * as uiApiClient from '../../lib/uiApiClient'

vi.mock('../../lib/apiClient')
vi.mock('../../lib/uiApiClient')

function renderShell() {
  return render(
    <MemoryRouter>
      <AppShell>
        <div>content</div>
      </AppShell>
    </MemoryRouter>,
  )
}

describe('AppShell', () => {
  beforeEach(() => {
    vi.mocked(apiClient.getStats).mockRejectedValue(new Error('offline'))
    vi.mocked(uiApiClient.getInfo).mockRejectedValue(new Error('offline'))
    useUIStore.setState({ modal: null })
  })

  it('fetches messages once on mount', () => {
    const fetchMessages = vi.fn()
    useMessageStore.setState({ fetchMessages })
    renderShell()
    expect(fetchMessages).toHaveBeenCalledTimes(1)
  })

  it('renders a ConfirmDialog driven by useUIStore.modal', () => {
    renderShell()
    expect(screen.queryByText('Clear all messages?')).not.toBeInTheDocument()

    act(() => {
      useUIStore.getState().openConfirm({
        title: 'Clear all messages?',
        body: 'This cannot be undone.',
        danger: true,
        onConfirm: () => {},
      })
    })

    expect(screen.getByText('Clear all messages?')).toBeInTheDocument()
  })

  it('invokes the modal onConfirm and closes on confirm click', () => {
    renderShell()
    const onConfirm = vi.fn()
    act(() => {
      useUIStore.getState().openConfirm({ title: 'Sure?', body: 'Body', onConfirm })
    })

    fireEvent.click(screen.getByRole('button', { name: 'Confirm' }))

    expect(onConfirm).toHaveBeenCalledTimes(1)
    expect(useUIStore.getState().modal).toBeNull()
  })
})

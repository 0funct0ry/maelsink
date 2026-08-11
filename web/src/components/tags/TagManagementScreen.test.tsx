import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import TagManagementScreen from './TagManagementScreen'
import ConfirmDialog from '../common/ConfirmDialog'
import { useMessageStore } from '../../stores/useMessageStore'
import { useUIStore } from '../../stores/useUIStore'
import * as apiClient from '../../lib/apiClient'

vi.mock('../../lib/apiClient')

// TagManagementScreen relies on the globally-mounted ConfirmDialog pattern
// (AppShell in production); render one here too so merge/delete
// confirmations actually appear and can be interacted with.
function renderScreen() {
  return render(
    <>
      <TagManagementScreen />
      <ConfirmDialogHost />
    </>,
  )
}

function ConfirmDialogHost() {
  const modal = useUIStore((state) => state.modal)
  const closeModal = useUIStore((state) => state.closeModal)
  return (
    <ConfirmDialog
      open={modal?.kind === 'confirm'}
      onClose={closeModal}
      onConfirm={modal?.onConfirm ?? (() => {})}
      title={modal?.title ?? ''}
      body={modal?.body ?? ''}
      confirmLabel={modal?.confirmLabel}
      danger={modal?.danger}
    />
  )
}

const TAGS = [
  { name: 'smoke', color: 'indigo', count: 3, last_used: '2026-01-02T00:00:00Z' },
  { name: 'release', color: 'emerald', count: 1, last_used: '2026-01-01T00:00:00Z' },
]

describe('TagManagementScreen', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useMessageStore.setState({ sidebarTags: TAGS, fetchSidebarData: vi.fn().mockResolvedValue(undefined) })
    useUIStore.setState({ modal: null, toasts: [] })
    vi.mocked(apiClient.listTags).mockResolvedValue(TAGS)
  })

  it('renders every tag with its count and last-used date', () => {
    renderScreen()
    expect(screen.getByText('smoke')).toBeInTheDocument()
    expect(screen.getByText('release')).toBeInTheDocument()
    expect(screen.getByText('3')).toBeInTheDocument()
    expect(screen.getByText('1')).toBeInTheDocument()
  })

  it('sorts by count descending by default, and toggles order on header click', () => {
    renderScreen()
    const rows = screen.getAllByRole('row').slice(1) // skip header row
    expect(rows[0]).toHaveTextContent('smoke')

    fireEvent.click(screen.getByText(/Count/))
    const rowsAsc = screen.getAllByRole('row').slice(1)
    expect(rowsAsc[0]).toHaveTextContent('release')
  })

  it('creates a new tag', async () => {
    vi.mocked(apiClient.createTag).mockResolvedValue({ name: 'fresh', color: 'indigo', count: 0, last_used: null })
    renderScreen()
    fireEvent.change(screen.getByLabelText('New tag'), { target: { value: 'fresh' } })
    fireEvent.click(screen.getByText('Add tag'))
    await waitFor(() => expect(apiClient.createTag).toHaveBeenCalledWith('fresh', 'indigo'))
  })

  it('shows an error when creating a duplicate tag', async () => {
    vi.mocked(apiClient.createTag).mockRejectedValue(new Error('conflict'))
    renderScreen()
    fireEvent.change(screen.getByLabelText('New tag'), { target: { value: 'smoke' } })
    fireEvent.click(screen.getByText('Add tag'))
    await waitFor(() => expect(screen.getByText(/already exists/)).toBeInTheDocument())
  })

  it('renames a tag with no collision directly, without a confirm dialog', async () => {
    vi.mocked(apiClient.renameTag).mockResolvedValue({ name: 'smokey', color: 'indigo', count: 3, last_used: null })
    renderScreen()
    fireEvent.click(screen.getByText('smoke'))
    const input = screen.getByLabelText('Rename tag smoke')
    fireEvent.change(input, { target: { value: 'smokey' } })
    fireEvent.keyDown(input, { key: 'Enter' })
    await waitFor(() => expect(apiClient.renameTag).toHaveBeenCalledWith('smoke', 'smokey'))
    expect(screen.queryByText('Merge tags?')).not.toBeInTheDocument()
  })

  it('shows a merge confirmation when renaming a tag to an existing tag\'s name, and merges on confirm', async () => {
    vi.mocked(apiClient.renameTag).mockResolvedValue({ name: 'release', color: 'emerald', count: 4, last_used: null })
    renderScreen()
    fireEvent.click(screen.getByText('smoke'))
    const input = screen.getByLabelText('Rename tag smoke')
    fireEvent.change(input, { target: { value: 'release' } })
    fireEvent.keyDown(input, { key: 'Enter' })

    expect(screen.getByText('Merge tags?')).toBeInTheDocument()
    expect(apiClient.renameTag).not.toHaveBeenCalled()

    fireEvent.click(screen.getByText('Merge'))
    await waitFor(() => expect(apiClient.renameTag).toHaveBeenCalledWith('smoke', 'release'))
  })

  it('recolors a tag via the swatch picker', async () => {
    vi.mocked(apiClient.recolorTag).mockResolvedValue({ name: 'smoke', color: 'amber', count: 3, last_used: null })
    renderScreen()
    fireEvent.click(screen.getByLabelText('Recolor tag smoke'))
    const amberSwatches = screen.getAllByLabelText('amber')
    fireEvent.click(amberSwatches[amberSwatches.length - 1])
    await waitFor(() => expect(apiClient.recolorTag).toHaveBeenCalledWith('smoke', 'amber'))
  })

  it('closes the color picker when clicking outside it', () => {
    renderScreen()
    // The "Add tag" section always has one swatch picker; opening a row's
    // recolor popover adds a second.
    expect(screen.getAllByRole('group', { name: 'Tag color' })).toHaveLength(1)

    fireEvent.click(screen.getByLabelText('Recolor tag smoke'))
    expect(screen.getAllByRole('group', { name: 'Tag color' })).toHaveLength(2)

    fireEvent.pointerDown(document.body)
    expect(screen.getAllByRole('group', { name: 'Tag color' })).toHaveLength(1)
  })

  it('removes a tag (untag-only) after confirming', async () => {
    vi.mocked(apiClient.deleteTag).mockResolvedValue(undefined)
    renderScreen()
    fireEvent.click(screen.getByLabelText('Remove tag smoke'))
    expect(screen.getByText('Remove tag from all messages?')).toBeInTheDocument()
    fireEvent.click(screen.getByText('Remove tag'))
    await waitFor(() => expect(apiClient.deleteTag).toHaveBeenCalledWith('smoke'))
  })

  it('deletes a tag and its messages after confirming, showing the message count in the confirmation', async () => {
    vi.mocked(apiClient.deleteTagWithMessages).mockResolvedValue(undefined)
    renderScreen()
    fireEvent.click(screen.getByLabelText('Delete tag smoke and its messages'))
    expect(screen.getByText(/permanently deletes 3 messages/)).toBeInTheDocument()
    fireEvent.click(screen.getByText('Delete messages'))
    await waitFor(() => expect(apiClient.deleteTagWithMessages).toHaveBeenCalledWith('smoke'))
  })

  it('never calls window.confirm/alert/prompt', () => {
    const confirmSpy = vi.spyOn(window, 'confirm')
    const alertSpy = vi.spyOn(window, 'alert')
    renderScreen()
    fireEvent.click(screen.getByLabelText('Remove tag smoke'))
    fireEvent.click(screen.getByLabelText('Delete tag release and its messages'))
    expect(confirmSpy).not.toHaveBeenCalled()
    expect(alertSpy).not.toHaveBeenCalled()
  })
})

import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import TagEditModal from './TagEditModal'
import { useMessageStore } from '../../stores/useMessageStore'
import { useUIStore } from '../../stores/useUIStore'
import type { MessageSummary } from '../../lib/apiTypes'

function makeMessage(overrides: Partial<MessageSummary> = {}): MessageSummary {
  return {
    id: 'msg-1',
    from: 'alice@example.com',
    to: ['bob@example.com'],
    cc: [],
    bcc: [],
    subject: 'Hello world',
    size_bytes: 2048,
    has_attachments: false,
    attachment_count: 0,
    received_at: new Date().toISOString(),
    parse_warning: false,
    read: true,
    tags: [],
    preview: '',
    ...overrides,
  }
}

describe('TagEditModal', () => {
  beforeEach(() => {
    useUIStore.setState({ toasts: [] })
  })

  it('renders current tags as badges', () => {
    useMessageStore.setState({ updateTagsOptimistic: vi.fn(), sidebarTags: [] })
    render(<TagEditModal open onClose={vi.fn()} message={makeMessage({ tags: ['smoke', 'release'] })} />)
    expect(screen.getByText('smoke')).toBeInTheDocument()
    expect(screen.getByText('release')).toBeInTheDocument()
  })

  it('removes a tag when its remove control is clicked', () => {
    const updateTagsOptimistic = vi.fn().mockResolvedValue(undefined)
    useMessageStore.setState({ updateTagsOptimistic, sidebarTags: [] })
    render(<TagEditModal open onClose={vi.fn()} message={makeMessage({ tags: ['smoke'] })} />)
    fireEvent.click(screen.getByLabelText('Remove tag smoke'))
    expect(updateTagsOptimistic).toHaveBeenCalledWith('msg-1', [], ['smoke'])
  })

  it('adds a tag via the input and Enter', () => {
    const updateTagsOptimistic = vi.fn().mockResolvedValue(undefined)
    useMessageStore.setState({ updateTagsOptimistic, sidebarTags: [] })
    render(<TagEditModal open onClose={vi.fn()} message={makeMessage()} />)
    const input = screen.getByLabelText('Add a tag')
    fireEvent.change(input, { target: { value: 'release' } })
    fireEvent.keyDown(input, { key: 'Enter' })
    expect(updateTagsOptimistic).toHaveBeenCalledWith('msg-1', ['release'], [])
  })

  it('adds a tag via the Add button and clears the input', () => {
    const updateTagsOptimistic = vi.fn().mockResolvedValue(undefined)
    useMessageStore.setState({ updateTagsOptimistic, sidebarTags: [] })
    render(<TagEditModal open onClose={vi.fn()} message={makeMessage()} />)
    const input = screen.getByLabelText('Add a tag') as HTMLInputElement
    fireEvent.change(input, { target: { value: 'release' } })
    fireEvent.click(screen.getByText('Add'))
    expect(updateTagsOptimistic).toHaveBeenCalledWith('msg-1', ['release'], [])
    expect(input.value).toBe('')
  })

  it('ignores an empty add', () => {
    const updateTagsOptimistic = vi.fn()
    useMessageStore.setState({ updateTagsOptimistic, sidebarTags: [] })
    render(<TagEditModal open onClose={vi.fn()} message={makeMessage()} />)
    fireEvent.click(screen.getByText('Add'))
    expect(updateTagsOptimistic).not.toHaveBeenCalled()
  })

  it('offers autocomplete suggestions from sidebarTags, excluding tags already on the message', () => {
    useMessageStore.setState({
      updateTagsOptimistic: vi.fn(),
      sidebarTags: [
        { tag: 'smoke', count: 3 },
        { tag: 'release', count: 1 },
      ],
    })
    render(<TagEditModal open onClose={vi.fn()} message={makeMessage({ tags: ['smoke'] })} />)
    const options = document.querySelectorAll('#tag-edit-suggestions option')
    const values = Array.from(options).map((o) => (o as HTMLOptionElement).value)
    expect(values).toEqual(['release'])
  })

  it('shows a danger toast on a failed update', async () => {
    const updateTagsOptimistic = vi.fn().mockImplementation(async () => {
      useUIStore.getState().pushToast('danger', 'Failed to update tags')
    })
    useMessageStore.setState({ updateTagsOptimistic, sidebarTags: [] })
    render(<TagEditModal open onClose={vi.fn()} message={makeMessage({ tags: ['smoke'] })} />)
    fireEvent.click(screen.getByLabelText('Remove tag smoke'))
    await waitFor(() =>
      expect(useUIStore.getState().toasts.some((t) => t.message === 'Failed to update tags')).toBe(true),
    )
  })

  it('calls onClose when Done is clicked', () => {
    const onClose = vi.fn()
    useMessageStore.setState({ updateTagsOptimistic: vi.fn(), sidebarTags: [] })
    render(<TagEditModal open onClose={onClose} message={makeMessage()} />)
    fireEvent.click(screen.getByText('Done'))
    expect(onClose).toHaveBeenCalledTimes(1)
  })
})

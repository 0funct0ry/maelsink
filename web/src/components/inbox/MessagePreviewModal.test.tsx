import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import MessagePreviewModal from './MessagePreviewModal'
import * as apiClient from '../../lib/apiClient'
import type { MessageDetail } from '../../lib/apiTypes'

vi.mock('../../lib/apiClient', async () => {
  const actual = await vi.importActual<typeof import('../../lib/apiClient')>('../../lib/apiClient')
  return { ...actual, getMessage: vi.fn(), getRaw: vi.fn() }
})

const detail: MessageDetail = {
  id: 'm1',
  from: 'a@example.com',
  to: ['b@example.com'],
  cc: [],
  bcc: [],
  subject: 'Preview me',
  size_bytes: 1200,
  has_attachments: false,
  attachment_count: 0,
  received_at: '2024-01-01T00:00:00Z',
  parse_warning: false,
  read: false,
  tags: [],
  preview: '',
  headers: [],
  text_body: 'body',
  html_body: '<p>hi</p>',
  attachments: [],
  raw_size_bytes: 1200,
}

function renderModal(messageId: string | null, onClose = vi.fn()) {
  return render(
    <MemoryRouter>
      <Routes>
        <Route path="*" element={<MessagePreviewModal messageId={messageId} onClose={onClose} />} />
      </Routes>
    </MemoryRouter>,
  )
}

describe('MessagePreviewModal', () => {
  it('renders nothing when messageId is null', () => {
    renderModal(null)
    expect(screen.queryByText('Close')).not.toBeInTheDocument()
  })

  it('shows a loading state while fetching', () => {
    vi.mocked(apiClient.getMessage).mockReturnValue(new Promise(() => {}))
    const { container } = renderModal('m1')
    expect(container.querySelectorAll('.animate-pulse').length).toBeGreaterThan(0)
  })

  it('renders the message once loaded', async () => {
    vi.mocked(apiClient.getMessage).mockResolvedValue(detail)
    renderModal('m1')
    expect(await screen.findByText('Preview me')).toBeInTheDocument()
    expect(screen.getByText(/a@example.com/)).toBeInTheDocument()
  })

  it('shows an error state when the fetch fails', async () => {
    vi.mocked(apiClient.getMessage).mockRejectedValue(new Error('boom'))
    renderModal('m1')
    expect(await screen.findByText('Failed to load this message.')).toBeInTheDocument()
  })

  it('calls onClose when Close is clicked', async () => {
    vi.mocked(apiClient.getMessage).mockResolvedValue(detail)
    const onClose = vi.fn()
    renderModal('m1', onClose)
    await screen.findByText('Preview me')
    fireEvent.click(screen.getByText('Close'))
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('refetches when messageId changes', async () => {
    vi.mocked(apiClient.getMessage).mockResolvedValue(detail)
    const { rerender } = renderModal('m1')
    await waitFor(() => expect(apiClient.getMessage).toHaveBeenCalledWith('m1'))

    vi.mocked(apiClient.getMessage).mockResolvedValue({ ...detail, id: 'm2', subject: 'Second' })
    rerender(
      <MemoryRouter>
        <Routes>
          <Route path="*" element={<MessagePreviewModal messageId="m2" onClose={vi.fn()} />} />
        </Routes>
      </MemoryRouter>,
    )
    await waitFor(() => expect(apiClient.getMessage).toHaveBeenCalledWith('m2'))
  })
})

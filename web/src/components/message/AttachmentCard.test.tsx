import { render, screen, waitFor } from '@testing-library/react'
import AttachmentCard from './AttachmentCard'
import * as apiClient from '../../lib/apiClient'

vi.mock('../../lib/apiClient', () => ({
  getAttachmentBlob: vi.fn(),
  getAttachmentDownloadUrl: vi.fn((id: string, attId: string) => `/api/v1/messages/${id}/attachments/${attId}`),
}))

beforeEach(() => {
  vi.stubGlobal('URL', {
    ...URL,
    createObjectURL: vi.fn(() => 'blob:mock'),
    revokeObjectURL: vi.fn(),
  })
})

describe('AttachmentCard', () => {
  it('renders filename, size, and a download link', () => {
    render(
      <AttachmentCard
        messageId="m1"
        attachment={{ id: 'a1', filename: 'notes.txt', content_type: 'text/plain', size_bytes: 2048, content_id: null }}
      />,
    )
    expect(screen.getByText('notes.txt')).toBeInTheDocument()
    expect(screen.getByText('2.0 KB')).toBeInTheDocument()
    const link = screen.getByLabelText('Download notes.txt')
    expect(link).toHaveAttribute('href', '/api/v1/messages/m1/attachments/a1')
    expect(link).toHaveAttribute('download', 'notes.txt')
  })

  it('does not fetch a blob for non-previewable types', () => {
    render(
      <AttachmentCard
        messageId="m1"
        attachment={{ id: 'a1', filename: 'notes.txt', content_type: 'text/plain', size_bytes: 10, content_id: null }}
      />,
    )
    expect(apiClient.getAttachmentBlob).not.toHaveBeenCalled()
  })

  it('fetches and previews an image attachment', async () => {
    vi.mocked(apiClient.getAttachmentBlob).mockResolvedValue(new Blob(['x']))
    render(
      <AttachmentCard
        messageId="m1"
        attachment={{ id: 'a1', filename: 'pic.png', content_type: 'image/png', size_bytes: 10, content_id: null }}
      />,
    )
    await waitFor(() => expect(screen.getByAltText('pic.png')).toBeInTheDocument())
  })

  it('handles a failed blob fetch gracefully', async () => {
    vi.mocked(apiClient.getAttachmentBlob).mockRejectedValue(new Error('boom'))
    render(
      <AttachmentCard
        messageId="m1"
        attachment={{ id: 'a1', filename: 'pic.png', content_type: 'image/png', size_bytes: 10, content_id: null }}
      />,
    )
    await waitFor(() => expect(apiClient.getAttachmentBlob).toHaveBeenCalled())
    expect(screen.queryByAltText('pic.png')).not.toBeInTheDocument()
  })
})

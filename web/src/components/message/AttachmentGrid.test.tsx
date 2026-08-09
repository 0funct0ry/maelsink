import { render, screen } from '@testing-library/react'
import AttachmentGrid from './AttachmentGrid'

vi.mock('../../lib/apiClient', () => ({
  getAttachmentBlob: vi.fn().mockResolvedValue(new Blob()),
  getAttachmentDownloadUrl: vi.fn(() => '/download'),
}))

beforeEach(() => {
  vi.stubGlobal('URL', {
    ...URL,
    createObjectURL: vi.fn(() => 'blob:mock'),
    revokeObjectURL: vi.fn(),
  })
})

describe('AttachmentGrid', () => {
  it('renders nothing when there are no attachments', () => {
    const { container } = render(<AttachmentGrid messageId="m1" attachments={[]} />)
    expect(container).toBeEmptyDOMElement()
  })

  it('renders a heading with the count and a card per attachment', () => {
    render(
      <AttachmentGrid
        messageId="m1"
        attachments={[
          { id: 'a1', filename: 'one.txt', content_type: 'text/plain', size_bytes: 10, content_id: null },
          { id: 'a2', filename: 'two.txt', content_type: 'text/plain', size_bytes: 20, content_id: null },
        ]}
      />,
    )
    expect(screen.getByText('Attachments (2)')).toBeInTheDocument()
    expect(screen.getByText('one.txt')).toBeInTheDocument()
    expect(screen.getByText('two.txt')).toBeInTheDocument()
  })
})
